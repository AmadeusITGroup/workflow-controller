package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	"github.com/heptiolabs/healthcheck"

	apiv1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	cwapi "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	cwlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/cronworkflow/v1"
	wlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/workflow/v1"
)

// CronWorkflowControllerConfig contains info to customize Workflow controller behaviour
type CronWorkflowControllerConfig struct {
	RemoveIvalidCronWorkflow bool
	NumberOfWorkers          int
}

// CronWorkflowController represents the Workflow controller
type CronWorkflowController struct {
	CronWorkflowClient wclientset.Interface
	KubeClient         clientset.Interface
	queue              workqueue.RateLimitingInterface // Workflows to be synced

	CronWorkflowLister cwlisters.CronWorkflowLister
	CronWorkflowSynced cache.InformerSynced

	WorkflowLister wlisters.WorkflowLister
	WorkflowSynced cache.InformerSynced // returns true if job has been synced. Added as member for testing

	updateHandler func(*cwapi.CronWorkflow) error // callback to upate Workflow. Added as member for testing
	syncHandler   func(string) error

	Recorder record.EventRecorder
	config   CronWorkflowControllerConfig
}

// NewCronWorkflowController creates and initializes the CronWorkflowController instance
func NewCronWorkflowController(
	cronWorkflowClient wclientset.Interface,
	kubeClient clientset.Interface) *CronWorkflowController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	wc := &CronWorkflowController{
		CronWorkflowClient: cronWorkflowClient,
		KubeClient:         kubeClient,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "cronworkflow"),
		config: CronWorkflowControllerConfig{
			RemoveIvalidCronWorkflow: true,
			NumberOfWorkers:          1,
		},

		Recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "cronworkflow-controller"}),
	}

	wc.updateHandler = wc.updateCronWorkflow
	wc.syncHandler = wc.sync

	return wc
}

// AddHealthCheck add Readiness and Liveness Checks to the handler
func (w *CronWorkflowController) AddHealthCheck(h healthcheck.Handler) {
	h.AddReadinessCheck("Workflow_cache_sync", func() error {
		if w.WorkflowSynced() {
			return nil
		}
		return fmt.Errorf("Workflow cache not sync")
	})

	h.AddReadinessCheck("CronWorkflow_cache_sync", func() error {
		if w.CronWorkflowSynced() {
			return nil
		}
		return fmt.Errorf("CronWorkflow cache not sync")
	})
}

// Run simply runs the controller. Assuming Informers are already running: via factories
func (w *CronWorkflowController) Run(ctx context.Context) error {
	glog.Infof("Starting workflow controller")

	if !cache.WaitForCacheSync(ctx.Done(), w.WorkflowSynced, w.CronWorkflowSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := 0; i < w.config.NumberOfWorkers; i++ {
		go wait.Until(w.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	return ctx.Err()
}

func (w *CronWorkflowController) runWorker() {
	for w.processNextItem() {
	}
}

func (w *CronWorkflowController) processNextItem() bool {
	key, quit := w.queue.Get()
	if quit {
		return false
	}
	defer w.queue.Done(key)
	err := w.syncHandler(key.(string))
	if err == nil {
		w.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing workflow: %v", err))
	w.queue.AddRateLimited(key)

	return true
}

// enqueue adds key in the controller queue
func (w *CronWorkflowController) enqueue(workflow *cwapi.CronWorkflow) {
	key, err := cache.MetaNamespaceKeyFunc(workflow)
	if err != nil {
		glog.Errorf("CronWorkflowController:enqueue: couldn't get key for Workflow   %s/%s", workflow.Namespace, workflow.Name)
		return
	}
	w.queue.Add(key)
}

// get workflow by key method
func (w *CronWorkflowController) getCronWorkflowByKey(key string) (*cwapi.CronWorkflow, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	glog.V(6).Infof("Syncing %s/%s", namespace, name)
	workflow, err := w.CronWorkflowLister.CronWorkflows(namespace).Get(name)
	if err != nil {
		glog.Errorf("unable to get Workflow %s/%s: %v. Maybe deleted", namespace, name, err)
		return nil, err
	}
	return workflow, nil
}

// workflow sync method
func (w *CronWorkflowController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(6).Infof("Finished syncing CronWorkflow %q (%v", key, time.Now().Sub(startTime))
	}()

	var sharedWorkflow *cwapi.CronWorkflow
	var err error
	if sharedWorkflow, err = w.getCronWorkflowByKey(key); err != nil {
		glog.Errorf("unable to get Workflow %s: %v. Maybe deleted", key, err)
		return nil
	}
	namespace := sharedWorkflow.Namespace
	name := sharedWorkflow.Name

	if !cwapi.IsCronWorkflowDefaulted(sharedWorkflow) {
		defaultedWorkflow := cwapi.DefaultCronWorkflow(sharedWorkflow)
		if err := w.updateHandler(defaultedWorkflow); err != nil {
			glog.Errorf("CronWorkflowController.sync unable to default Workflow %s/%s: %v", namespace, name, err)
			return fmt.Errorf("unable to default Workflow %s/%s: %v", namespace, name, err)
		}
		glog.V(6).Infof("CronWorkflowController.sync Defaulted %s/%s", namespace, name)
		return nil
	}

	// Validation. We always revalidate CronWorkflow
	if errs := cwapi.ValidateCronWorkflow(sharedWorkflow); errs != nil && len(errs) > 0 {
		glog.Errorf("CronWorkflowController.sync Worfklow %s/%s not valid: %v", namespace, name, errs)
		if w.config.RemoveIvalidCronWorkflow {
			glog.Errorf("Invalid workflow %s/%s is going to be removed", namespace, name)
			if err := w.deleteCronWorkflow(namespace, name); err != nil {
				glog.Errorf("unable to delete invalid workflow %s/%s: %v", namespace, name, err)
				return fmt.Errorf("unable to delete invalid workflow %s/%s: %v", namespace, name, err)
			}
		}
		return nil
	}

	// TODO: add test the case of graceful deletion
	if sharedWorkflow.DeletionTimestamp != nil {
		return nil
	}

	workflow := sharedWorkflow.DeepCopy()

	return w.manageCronWorkflow(workflow)
}

func (w *CronWorkflowController) onAddCronWorkflow(obj interface{}) {
	workflow, ok := obj.(*cwapi.CronWorkflow)
	if !ok {
		glog.Errorf("adding workflow, expected workflow object. Got: %+v", obj)
		return
	}
	if !reflect.DeepEqual(workflow.Status, wapi.WorkflowStatus{}) {
		glog.Errorf("workflow %s/%s created with non empty status. Going to be removed", workflow.Namespace, workflow.Name)
		if _, err := cache.MetaNamespaceKeyFunc(workflow); err != nil {
			glog.Errorf("couldn't get key for Workflow (to be deleted) %s/%s: %v", workflow.Namespace, workflow.Name, err)
			return
		}
		// TODO: how to remove a workflow created with an invalid or even with a valid status. What in case of error for this delete?
		if err := w.deleteCronWorkflow(workflow.Namespace, workflow.Name); err != nil {
			glog.Errorf("unable to delete non empty status Workflow %s/%s: %v. No retry will be performed.", workflow.Namespace, workflow.Name, err)
		}
		return
	}
	w.enqueue(workflow)
}

func (w *CronWorkflowController) onUpdateCronWorkflow(oldObj, newObj interface{}) {
	workflow, ok := newObj.(*cwapi.CronWorkflow)
	if !ok {
		glog.Errorf("Expected workflow object. Got: %+v", newObj)
		return
	}
	w.enqueue(workflow)
}

func (w *CronWorkflowController) onDeleteCronWorkflow(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	w.queue.Add(key)
}

func (w *CronWorkflowController) updateCronWorkflow(cwfl *cwapi.CronWorkflow) error {
	if _, err := w.CronWorkflowClient.CronworkflowV1().CronWorkflows(cwfl.Namespace).Update(cwfl); err != nil {
		glog.V(6).Infof("Workflow %s/%s updated", cwfl.Namespace, cwfl.Name)
		return err
	}
	return nil
}

func (w *CronWorkflowController) deleteCronWorkflow(namespace, name string) error {
	if err := w.CronWorkflowClient.WorkflowV1().Workflows(namespace).Delete(name, nil); err != nil {
		return fmt.Errorf("unable to delete Workflow %s/%s: %v", namespace, name, err)
	}

	glog.V(6).Infof("Workflow %s/%s deleted", namespace, name)
	return nil
}

func (w *CronWorkflowController) onAddWorkflow(obj interface{}) {
	workflow := obj.(*wapi.Workflow)
	cronWorkflows, err := w.getCronWorkflowsFromWorkflow(workflow)
	if err != nil {
		glog.Errorf("unable to get CronWorkflows from Workflow %s/%s: %v", workflow.Namespace, workflow.Name, err)
		return
	}
	for i := range cronWorkflows {
		w.enqueue(cronWorkflows[i])
	}
}

func (w *CronWorkflowController) onUpdateWorkflow(oldObj, newObj interface{}) {
	oldWorkflow := oldObj.(*wapi.Workflow)
	newWorkflow := newObj.(*wapi.Workflow)
	if oldWorkflow.ResourceVersion == newWorkflow.ResourceVersion { // Since periodic resync will send update events for all known jobs.
		return
	}
	glog.V(6).Infof("onUpdateJob old=%v, cur=%v ", oldWorkflow.Name, newWorkflow.Name)
	workflows, err := w.getCronWorkflowsFromWorkflow(newWorkflow)
	if err != nil {
		glog.Errorf("CronWorkflowController.onUpdateJob cannot get workflows for job %s/%s: %v", newWorkflow.Namespace, newWorkflow.Name, err)
		return
	}
	for i := range workflows {
		w.enqueue(workflows[i])
	}

	// TODO: in case of relabelling ?
	// TODO: in case of labelSelector relabelling?
}

func (w *CronWorkflowController) onDeleteWorkflow(obj interface{}) {
}

func (w *CronWorkflowController) manageCronWorkflow(workflow *cwapi.CronWorkflow) error {
	return nil
}

func (w *CronWorkflowController) getCronWorkflowsFromWorkflow(job *wapi.Workflow) ([]*cwapi.CronWorkflow, error) {
	cronWorkflows := []*cwapi.CronWorkflow{}
	return cronWorkflows, nil
}
