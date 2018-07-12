package controller

import (
	"context"
	"time"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	cwapi "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/workflow-controller/pkg/util"
)

var controllerKind = cwapi.SchemeGroupVersion.WithKind(cwapi.ResourceKind)

// CronWorkflowControllerConfig contains info to customize CronWorkflow controller behaviour
type CronWorkflowControllerConfig struct {
	RemoveIvalidCronWorkflow bool
	NumberOfWorkers          int
}

// CronWorkflowController represents the CronWorkflow controller
type CronWorkflowController struct {
	CronWorkflowClient wclientset.Interface
	WorkflowClient     wclientset.Interface
	KubeClient         clientset.Interface

	updateHandler func(*cwapi.CronWorkflow) error // callback to upate CronWorkflow. Added as member for testing

	Recorder record.EventRecorder
	config   CronWorkflowControllerConfig
}

// NewCronWorkflowController creates and initializes the CronWorkflowController instance
func NewCronWorkflowController(
	cronWorkflowClient wclientset.Interface,
	workflowClient wclientset.Interface,
	kubeClient clientset.Interface) *CronWorkflowController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	wc := &CronWorkflowController{
		CronWorkflowClient: cronWorkflowClient,
		WorkflowClient:     workflowClient,
		KubeClient:         kubeClient,

		config: CronWorkflowControllerConfig{
			RemoveIvalidCronWorkflow: true,
			NumberOfWorkers:          1,
		},

		Recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "cronworkflow-controller"}),
	}

	wc.updateHandler = wc.updateCronWorkflow
	return wc
}

// Run simply runs the controller. Assuming Informers are already running: via factories
func (w *CronWorkflowController) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	glog.Infof("Starting cronWorkflow-controller")

	wait.Until(w.syncAll, 10*time.Second, ctx.Done())

	glog.Infof("Stopping cronWorkflow-controller")
	return ctx.Err()
}

// SyncAll syncs all cronWorkflow
func (w *CronWorkflowController) syncAll() {
	// List All Workflows (using label selector)
	ws, err := w.WorkflowClient.WorkflowV1().Workflows(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("Unable to list workflows: %v", err)
		return
	}
	workflows := ws.Items
	glog.V(4).Infof("Found %d workflows", len(workflows))

	cws, err := w.CronWorkflowClient.CronworkflowV1().CronWorkflows(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("Unable to list cronWorkflows: %v", err)
		return
	}
	cronWorkflows := cws.Items
	glog.V(4).Infof("Found %d cronworkflows", len(cronWorkflows))

	workflowsByCronWorkflows := groupWorkflowwsByCronWorkflows(workflows)
	for _, cw := range cronWorkflows {
		w.syncOne(&cw, workflowsByCronWorkflows[cw.UID], time.Now())
		// cleanupOldWorkflow
	}
}

func (w *CronWorkflowController) syncOne(cronWorkflow *cwapi.CronWorkflow, workflows []wapi.Workflow, now time.Time) {
	childrenWorkflows := make(map[types.UID]bool)
	for _, wfl := range workflows {
		childrenWorkflows[wfl.ObjectMeta.UID] = true
		found := inActiveList(cronWorkflow, wfl.ObjectMeta.UID)
		workflowFinished := IsWorkflowFinished(&wfl)
		switch {
		case found && workflowFinished:
			deleteFromActiveList(cronWorkflow, wfl.ObjectMeta.UID)
			w.Recorder.Eventf(cronWorkflow, apiv1.EventTypeNormal, "CompltedWorkflow", "Completed workflow: %s/%s", wfl.Namespace, wfl.Name)
		}
	}

	for _, w := range cronWorkflow.Status.Active {
		if found := childrenWorkflows[w.UID]; !found {
			deleteFromActiveList(cronWorkflow, w.UID)
		}
	}

	updatedCronwWorkflow, err := w.CronWorkflowClient.CronworkflowV1().CronWorkflows(cronWorkflow.Namespace).UpdateStatus(cronWorkflow)
	if err != nil {
		glog.Errorf("Unable to update cronWorkflow %s/%s", cronWorkflow.Namespace, cronWorkflow.Name)
		return
	}

	*cronWorkflow = *updatedCronwWorkflow

	// Now looks to create Workflow if needed
	if cronWorkflow.DeletionTimestamp != nil {
		return // being deleted
	}
	if cronWorkflow.Spec.Suspend != nil && *cronWorkflow.Spec.Suspend {
		glog.V(4).Infof("CronWorkflow %s/%s suspended", cronWorkflow.Namespace, cronWorkflow.Name)
		return
	}

	times, err := getRecentUnmetScheduleTimes(cronWorkflow, now)
	if err != nil {
		glog.Errorf("Unable to determine schedule times from cronWorkflow %s/%s: %v", cronWorkflow.Namespace, cronWorkflow.Name, err)
		return
	}

	switch {
	case len(times) == 0:
		glog.V(2).Infof("No workflow need to be scheduled for cronworkflow %s/%s", cronWorkflow.Namespace, cronWorkflow.Name)
		return
	case len(times) > 1:
		glog.Warningf("Multiple workflows should be started for cronworkflow %s/%s. Only the most recent will be started", cronWorkflow.Namespace, cronWorkflow.Name)
	}
	scheduledTime := times[len(times)-1]

	//TODO: handle policy

	workflowToBeCreated, err := getWorkflowFromCronWorkflow(cronWorkflow, scheduledTime)
	if err != nil {
		glog.Errorf("Unable to get workflow from template for cronWorkflow %s/%s", cronWorkflow.Namespace, cronWorkflow.Name)
		return
	}

	_, err = w.WorkflowClient.WorkflowV1().Workflows(cronWorkflow.Namespace).Create(workflowToBeCreated)
	if err != nil {
		glog.Errorf("Unable to create workflow for cronWorkflow %s/%s: %v", cronWorkflow.Namespace, cronWorkflow.Name, err)
		return
	}
}

func getWorkflowFromCronWorkflow(cronWorkflow *cwapi.CronWorkflow, scheduledTime time.Time) (*wapi.Workflow, error) {
	labels := util.CopyMap(cronWorkflow.Spec.WorkflowTemplate.Labels)
	annotations := util.CopyMap(cronWorkflow.Spec.WorkflowTemplate.Annotations)

	workflow := &wapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Annotations:     annotations,
			Name:            "test",
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cronWorkflow, controllerKind)},
		},
		Spec: *util.GetWorkflowSpecFromWorkflowTemplate(&cronWorkflow.Spec.WorkflowTemplate),
	}
	return workflow, nil
}

func createWorkflow(workflow *wapi.Workflow) (*wapi.Workflow, error) {
	return nil, nil
}

func groupWorkflowwsByCronWorkflows(workflows []wapi.Workflow) map[types.UID][]wapi.Workflow {
	workflowsByCronWorkflows := make(map[types.UID][]wapi.Workflow)
	for _, w := range workflows {
		cRef := metav1.GetControllerOf(&w)
		if cRef == nil {
			glog.V(4).Infof("No parent uid for workflow %s/%s", w.Namespace, w.Name)
			continue
		}
		if cRef.Kind != cwapi.ResourceKind {
			glog.V(4).Infof("Workflow %s/%s not created by CronWorkflow", w.Namespace, w.Name)
			continue
		}
		workflowsByCronWorkflows[cRef.UID] = append(workflowsByCronWorkflows[cRef.UID], w)
	}
	return workflowsByCronWorkflows
}

func deleteFromActiveList(cw *cwapi.CronWorkflow, uid types.UID) {
	newActive := []apiv1.ObjectReference{}
	for _, j := range cw.Status.Active {
		if j.UID != uid {
			newActive = append(newActive, j)
		}
	}
	cw.Status.Active = newActive
}

func inActiveList(cw *cwapi.CronWorkflow, uid types.UID) bool {
	for _, j := range cw.Status.Active {
		if j.UID == uid {
			return true
		}
	}
	return false
}

/*
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
*/
/*
// enqueue adds key in the controller queue
func (w *CronWorkflowController) enqueue(cronWorkflow *cwapi.CronWorkflow) {
	key, err := cache.MetaNamespaceKeyFunc(cronWorkflow)
	if err != nil {
		glog.Errorf("CronWorkflowController:enqueue: couldn't get key for CronWorkflow   %s/%s", cronWorkflow.Namespace, cronWorkflow.Name)
		return
	}
	w.queue.Add(key)
}
*/

/*
// get workflow by key method
func (w *CronWorkflowController) getCronWorkflowByKey(key string) (*cwapi.CronWorkflow, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	glog.V(6).Infof("Syncing %s/%s", namespace, name)
	workflow, err := w.CronWorkflowLister.CronWorkflows(namespace).Get(name)
	if err != nil {
		glog.Errorf("Unable to get CronWorkflow %s/%s: %v. Maybe deleted", namespace, name, err)
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

	var sharedCronWorkflow *cwapi.CronWorkflow
	var err error
	if sharedCronWorkflow, err = w.getCronWorkflowByKey(key); err != nil {
		glog.Errorf("Unable to get CronWorkflow %s: %v. Maybe deleted", key, err)
		return nil
	}
	namespace := sharedCronWorkflow.Namespace
	name := sharedCronWorkflow.Name

	if !cwapi.IsCronWorkflowDefaulted(sharedCronWorkflow) {
		defaultedWorkflow := cwapi.DefaultCronWorkflow(sharedCronWorkflow)
		if err := w.updateHandler(defaultedWorkflow); err != nil {
			glog.Errorf("CronWorkflowController.sync unable to default CronWorkflow %s/%s: %v", namespace, name, err)
			return fmt.Errorf("unable to default CronWorkflow %s/%s: %v", namespace, name, err)
		}
		glog.V(6).Infof("CronWorkflowController.sync Defaulted %s/%s", namespace, name)
		return nil
	}

	// Validation. We always revalidate CronWorkflow
	if errs := cwvalidation.ValidateCronWorkflow(sharedCronWorkflow); errs != nil && len(errs) > 0 {
		glog.Errorf("CronWorkflowController.sync CronWorfklow %s/%s not valid: %v", namespace, name, errs)
		if w.config.RemoveIvalidCronWorkflow {
			glog.Errorf("Invalid workflow %s/%s is going to be removed", namespace, name)
			if err := w.deleteCronWorkflow(namespace, name); err != nil {
				glog.Errorf("Unable to delete invalid CronWorkflow %s/%s: %v", namespace, name, err)
				return fmt.Errorf("unable to delete invalid CronWorkflow %s/%s: %v", namespace, name, err)
			}
		}
		return nil
	}

	// TODO: add test the case of graceful deletion
	if sharedCronWorkflow.DeletionTimestamp != nil {
		return nil
	}

	cronWorkflow := sharedCronWorkflow.DeepCopy()
	return w.manageCronWorkflow(cronWorkflow)
}

/*
func (w *CronWorkflowController) onAddCronWorkflow(obj interface{}) {
	workflow, ok := obj.(*cwapi.CronWorkflow)
	if !ok {
		glog.Errorf("Adding ConrWorkflow, expected CronWorkflow object. Got: %+v", obj)
		return
	}
	if !reflect.DeepEqual(workflow.Status, cwapi.CronWorkflowStatus{}) {
		glog.Errorf("CronWorkflow %s/%s created with non empty status. Going to be removed", workflow.Namespace, workflow.Name)
		if _, err := cache.MetaNamespaceKeyFunc(workflow); err != nil {
			glog.Errorf("Couldn't get key for Workflow (to be deleted) %s/%s: %v", workflow.Namespace, workflow.Name, err)
			return
		}
		// TODO: how to remove a workflow created with an invalid or even with a valid status. What in case of error for this delete?
		if err := w.deleteCronWorkflow(workflow.Namespace, workflow.Name); err != nil {
			glog.Errorf("Unable to delete non empty status Workflow %s/%s: %v. No retry will be performed.", workflow.Namespace, workflow.Name, err)
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
*/

func (w *CronWorkflowController) updateCronWorkflow(cwfl *cwapi.CronWorkflow) error {
	if _, err := w.CronWorkflowClient.CronworkflowV1().CronWorkflows(cwfl.Namespace).Update(cwfl); err != nil {
		glog.V(6).Infof("Workflow %s/%s updated", cwfl.Namespace, cwfl.Name)
		return err
	}
	return nil
}

/*
func (w *CronWorkflowController) deleteCronWorkflow(namespace, name string) error {
	if err := w.CronWorkflowClient.WorkflowV1().Workflows(namespace).Delete(name, nil); err != nil {
		return fmt.Errorf("Unable to delete Workflow %s/%s: %v", namespace, name, err)
	}

	glog.V(6).Infof("Workflow %s/%s deleted", namespace, name)
	return nil
}

func (w *CronWorkflowController) onAddWorkflow(obj interface{}) {
	workflow := obj.(*wapi.Workflow)
	cronWorkflows, err := w.getCronWorkflowsFromWorkflow(workflow)
	if err != nil {
		glog.Errorf("Unable to get CronWorkflows from Workflow %s/%s: %v", workflow.Namespace, workflow.Name, err)
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
*/
