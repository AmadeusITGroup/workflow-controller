package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/heptiolabs/healthcheck"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	cwapi "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/workflow-controller/pkg/util"
)

const (
	CronWorkflowLabelKey = "kubernetes.io/cronworkflow"
)

var controllerKind = cwapi.SchemeGroupVersion.WithKind(cwapi.ResourceKind)

type cronWorkflowControlInterface interface {
	UpdateStatus(sj *cwapi.CronWorkflow) (*cwapi.CronWorkflow, error)
}

// realCronWorkflowControl is the default implementation of cronWorkflowControlInterface
type realCronWorkflowControl struct {
	client wclientset.Interface
}

var _ cronWorkflowControlInterface = &realCronWorkflowControl{}

func (c *realCronWorkflowControl) UpdateStatus(cw *cwapi.CronWorkflow) (*cwapi.CronWorkflow, error) {
	return c.client.CronworkflowV1().CronWorkflows(cw.Namespace).UpdateStatus(cw)
}

// fakeCronWorkflowControl is the default implementation of cronWorkflowControlInterface.
type fakeCronWorkflowControl struct {
	Updates []cwapi.CronWorkflow
}

var _ cronWorkflowControlInterface = &fakeCronWorkflowControl{}

func (c *fakeCronWorkflowControl) UpdateStatus(cw *cwapi.CronWorkflow) (*cwapi.CronWorkflow, error) {
	c.Updates = append(c.Updates, *cw)
	return cw, nil
}

// workflowControlInterface is an interface which handles Workflow CRUD
type workflowControlInterface interface {
	// GetWorkflow retrieves a Workflow.
	GetWorkflow(namespace, name string) (*wapi.Workflow, error)
	// CreateWorkflow creates new Workflows according to the spec.
	CreateWorkflow(namespace string, job *wapi.Workflow) (*wapi.Workflow, error)
	// UpdateWorkflow updates a Workflow.
	UpdateWorkflow(namespace string, job *wapi.Workflow) (*wapi.Workflow, error)
	// TODO: implements patchWorkflow
	// DeleteWorkflow deletes the Workflow identified by name.
	DeleteWorkflow(namespace string, name string) error
}

type realWorkflowControl struct {
	c wclientset.Interface
}

var _ workflowControlInterface = &realWorkflowControl{}

// NewWorkflowControl creates a realWorkflowControl
func NewWorkflowControl(client wclientset.Interface) *realWorkflowControl {
	return &realWorkflowControl{c: client}
}

func (w *realWorkflowControl) GetWorkflow(namespace, name string) (*wapi.Workflow, error) {
	return w.c.WorkflowV1().Workflows(namespace).Get(name, metav1.GetOptions{})
}

func (w *realWorkflowControl) CreateWorkflow(namespace string, workflow *wapi.Workflow) (*wapi.Workflow, error) {
	return w.c.WorkflowV1().Workflows(namespace).Create(workflow)
}

func (w *realWorkflowControl) UpdateWorkflow(namespace string, job *wapi.Workflow) (*wapi.Workflow, error) {
	glog.Warningf("realWorkflowControl.UpdateWorkflow not yet implemented")
	return nil, nil
}

func (w *realWorkflowControl) DeleteWorkflow(namespace string, name string) error {
	glog.Warningf("realWorkflowControl.DeleteWorkflow not yet implemented")
	return nil
}

type fakeWorkflowControl struct {
}

var _ workflowControlInterface = &fakeWorkflowControl{}

func (w *fakeWorkflowControl) GetWorkflow(namespace, name string) (*wapi.Workflow, error) {
	return nil, nil
}

func (w *fakeWorkflowControl) CreateWorkflow(namespace string, job *wapi.Workflow) (*wapi.Workflow, error) {
	return nil, nil
}

func (w *fakeWorkflowControl) UpdateWorkflow(namespace string, job *wapi.Workflow) (*wapi.Workflow, error) {
	return nil, nil
}

func (w *fakeWorkflowControl) DeleteWorkflow(namespace string, name string) error {
	return nil
}

// CronWorkflowControllerConfig contains info to customize Workflow controller behaviour
type CronWorkflowControllerConfig struct {
	RemoveIvalidCronWorkflow bool
	NumberOfWorkers          int
}

// CronWorkflowController represents the Workflow controller
type CronWorkflowController struct {
	Client     wclientset.Interface
	KubeClient clientset.Interface
	queue      workqueue.RateLimitingInterface // Workflows to be synced

	//CronWorkflowLister cwlisters.CronWorkflowLister
	//CronWorkflowSynced cache.InformerSynced

	//WorkflowLister wlisters.WorkflowLister
	//WorkflowSynced cache.InformerSynced // returns true if job has been synced. Added as member for testing

	cronWorkflowControl cronWorkflowControlInterface
	workflowControl     workflowControlInterface
	//updateHandler       func(*cwapi.CronWorkflow) error // callback to upate Workflow. Added as member for testing
	//syncHandler         func(string) error

	Recorder record.EventRecorder
	config   CronWorkflowControllerConfig
}

// NewCronWorkflowController creates and initializes the CronWorkflowController instance
func NewCronWorkflowController(client wclientset.Interface, kubeClient clientset.Interface) *CronWorkflowController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	// TODO: check rate limit usage here
	/*
		if cronWorkflowClient != nil && cronWorkflowClient.CronworkflowV1().RESTClient().GetRateLimiter() != nil {
			if err := metrics.RegisterMetricAndTrackRateLimiterUsage("cronworkflow_controller", kubeClient.CoreV1().RESTClient().GetRateLimiter()); err != nil {
				return nil, err
			}
		}
	*/

	wc := &CronWorkflowController{
		Client:              client,
		KubeClient:          kubeClient,
		cronWorkflowControl: &realCronWorkflowControl{client},
		workflowControl:     NewWorkflowControl(client),
		config: CronWorkflowControllerConfig{
			RemoveIvalidCronWorkflow: true,
			NumberOfWorkers:          1,
		},
		Recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "cronworkflow-controller"}),
	}

	return wc
}

// AddHealthCheck add Readiness and Liveness Checks to the handler
func (w *CronWorkflowController) AddHealthCheck(h healthcheck.Handler) error {
	h.AddReadinessCheck("Workflow_registered", func() error {
		if _, err := w.Client.WorkflowV1().Workflows(metav1.NamespaceAll).List(metav1.ListOptions{}); err != nil {
			return err
		}
		return nil
	})

	h.AddReadinessCheck("CronWorkflow_registered", func() error {
		if _, err := w.Client.CronworkflowV1().CronWorkflows(metav1.NamespaceAll).List(metav1.ListOptions{}); err != nil {
			return err
		}
		return nil
	})
	return nil
}

// Run simply runs the controller. Assuming Informers are already running: via factories
func (w *CronWorkflowController) Run(ctx context.Context) error {
	defer utilruntime.HandleCrash()
	glog.Infof("Starting cronWorkflow controller")

	wait.Until(w.syncAll, 10*time.Second, ctx.Done())

	glog.Infof("Stopping cronWorkflow-controller")
	return ctx.Err()
}

// SyncAll syncs all cronWorkflow
func (w *CronWorkflowController) syncAll() {
	// List All Workflows TODO: use label selector
	ws, err := w.Client.WorkflowV1().Workflows(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("Unable to list workflows: %v", err)
		return
	}
	workflows := ws.Items
	glog.V(4).Infof("Found %d workflows", len(workflows))

	cws, err := w.Client.CronworkflowV1().CronWorkflows(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("Unable to list cronWorkflows: %v", err)
		return
	}
	cronWorkflows := cws.Items
	glog.V(2).Infof("Found %d cronWorkflows", len(cronWorkflows))

	workflowsByCronWorkflows := groupWorkflowsByCronWorkflows(workflows)
	for _, cw := range cronWorkflows {
		syncOne(&cw, workflowsByCronWorkflows[cw.UID], time.Now(), w.Client, w.workflowControl, w.Recorder)
		cleanupOldFinishedWorkflow(&cw, workflowsByCronWorkflows[cw.UID], w.workflowControl, w.Recorder)
	}
}

func syncOne(cronWorkflow *cwapi.CronWorkflow, workflows []wapi.Workflow, now time.Time /*cronWorkflowControl cronWorkflowControlInterface*/, client wclientset.Interface, workflowControl workflowControlInterface, recorder record.EventRecorder) {
	childrenWorkflows := make(map[types.UID]bool)
	for _, wfl := range workflows {
		childrenWorkflows[wfl.ObjectMeta.UID] = true
		found := inActiveList(cronWorkflow, wfl.ObjectMeta.UID)
		switch {
		case found && IsWorkflowFinished(&wfl):
			deleteFromActiveList(cronWorkflow, wfl.ObjectMeta.UID)
			recorder.Eventf(cronWorkflow, apiv1.EventTypeNormal, "CompletedWorkflow", "Completed workflow: %s/%s", wfl.Namespace, wfl.Name)
		}
	}

	for _, w := range cronWorkflow.Status.Active {
		if found := childrenWorkflows[w.UID]; !found {
			deleteFromActiveList(cronWorkflow, w.UID)
		}
	}

	// Should updateStatus only
	//updatedCronwWorkflow, err := cronWorkflowControl.UpdateStatus(cronWorkflow)
	updatedCronWorkflow, err := client.CronworkflowV1().CronWorkflows(cronWorkflow.Namespace).Update(cronWorkflow)
	if err != nil {
		glog.Errorf("Unable to update cronWorkflow %s/%s (rv = %s): %v", cronWorkflow.Namespace, cronWorkflow.Name, updatedCronWorkflow.ResourceVersion, err)
		return
	}

	// Now looks to create Workflow if needed
	if updatedCronWorkflow.DeletionTimestamp != nil {
		return // being deleted
	}
	if updatedCronWorkflow.Spec.Suspend != nil && *updatedCronWorkflow.Spec.Suspend {
		glog.V(4).Infof("CronWorkflow %s/%s suspended", updatedCronWorkflow.Namespace, updatedCronWorkflow.Name)
		return
	}

	times, err := getRecentUnmetScheduleTimes(updatedCronWorkflow, now)
	if err != nil {
		glog.Errorf("Unable to determine schedule times from cronWorkflow %s/%s: %v", updatedCronWorkflow.Namespace, updatedCronWorkflow.Name, err)
		return
	}

	switch {
	case len(times) == 0:
		glog.V(2).Infof("No workflow need to be scheduled for cronworkflow %s/%s", updatedCronWorkflow.Namespace, updatedCronWorkflow.Name)
		return
	case len(times) > 1:
		glog.Warningf("Multiple workflows should be started for cronworkflow %s/%s. Only the most recent will be started", cronWorkflow.Namespace, cronWorkflow.Name)
	}
	scheduledTime := times[len(times)-1]

	//TODO: handle policy

	workflowToBeCreated, err := inferrWorkflowDefinitionFromCronWorkflow(updatedCronWorkflow, scheduledTime)
	if err != nil {
		glog.Errorf("Unable to get workflow from template for cronWorkflow %s/%s", updatedCronWorkflow.Namespace, updatedCronWorkflow.Name)
		return
	}

	workflow, err := workflowControl.CreateWorkflow(updatedCronWorkflow.Namespace, workflowToBeCreated)
	if err != nil {
		glog.Errorf("Unable to create workflow for cronWorkflow %s/%s: %v", updatedCronWorkflow.Namespace, updatedCronWorkflow.Name, err)
		return
	}

	glog.Infof("Cronworkflow %s/%s created workflow %s/%s", updatedCronWorkflow.Namespace, updatedCronWorkflow.Name, workflow.Namespace, workflow.Name)

}

func cleanupOldFinishedWorkflow(cronWorkflow *cwapi.CronWorkflow, workflows []wapi.Workflow, workflowControl workflowControlInterface, recorder record.EventRecorder) {
	// TODO
	glog.Warning("TODO: cronWorkflowController.cleanupOldFinishedWorkflow -> IMPLEMENT IT!")
}

// getTimeHash returns Unix Epoch Time
func getTimeHash(scheduledTime time.Time) int64 {
	return scheduledTime.Unix()
}

func inferrWorkflowDefinitionFromCronWorkflow(cronWorkflow *cwapi.CronWorkflow, scheduledTime time.Time) (*wapi.Workflow, error) {
	labels := util.CopyMap(cronWorkflow.Spec.WorkflowTemplate.Labels)
	annotations := util.CopyMap(cronWorkflow.Spec.WorkflowTemplate.Annotations)
	// TODO: add validation to see if the spec.workflowTemplate.labels contains a CronWorklowLabelKey/value pair
	labels[CronWorkflowLabelKey] = cronWorkflow.Name
	workflowGeneratedName := fmt.Sprintf("%s-%d", cronWorkflow.Name, getTimeHash(scheduledTime)) // similarly to kube/kube cronJob

	workflow := &wapi.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          labels,
			Annotations:     annotations,
			Name:            workflowGeneratedName,
			OwnerReferences: []metav1.OwnerReference{},
		},
		Spec: *util.GetWorkflowSpecFromWorkflowTemplate(&cronWorkflow.Spec.WorkflowTemplate),
	}
	return wapi.DefaultWorkflow(workflow), nil
}

func groupWorkflowsByCronWorkflows(workflows []wapi.Workflow) map[types.UID][]wapi.Workflow {
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
	if _, err := w.Client.CronworkflowV1().CronWorkflows(cwfl.Namespace).Update(cwfl); err != nil {
		glog.V(6).Infof("Workflow %s/%s updated", cwfl.Namespace, cwfl.Name)
		return err
	}
	return nil
}

func (w *CronWorkflowController) deleteCronWorkflow(namespace, name string) error {
	if err := w.Client.WorkflowV1().Workflows(namespace).Delete(name, nil); err != nil {
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
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	w.queue.Add(key)
}

func (w *CronWorkflowController) manageCronWorkflow(workflow *cwapi.CronWorkflow) error {
	return nil
}

func (w *CronWorkflowController) getCronWorkflowsFromWorkflow(job *wapi.Workflow) ([]*cwapi.CronWorkflow, error) {
	cronWorkflows := []*cwapi.CronWorkflow{}
	return cronWorkflows, nil
}
*/
