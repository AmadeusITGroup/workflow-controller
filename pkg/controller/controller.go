package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/golang/glog"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	v1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/client-go/util/workqueue"

	wapi "github.com/sdminonne/workflow-controller/pkg/api/v1"
)

// WorkflowControllerConfig contains info to customize Workflow controller behaviour
type WorkflowControllerConfig struct {
	RemoveIvalidWorkflow bool
	NumberOfThreads      int
}

// WorkflowStepLabelKey defines the key of label to be injected by workflow controller
const (
	WorkflowLabelKey     = "kubernetes.io/workflow"
	WorkflowStepLabelKey = "kubernetes.io/workflow-step"
)

// WorkflowController represents the Workflow controller
type WorkflowController struct {
	WorkflowClient *rest.RESTClient
	WorkflowScheme *runtime.Scheme

	KubeClient       clientset.Interface
	workflowStore    cache.Store
	workflowInformer cache.Controller

	JobInformer    cache.SharedIndexInformer
	JobLister      batchv1listers.JobLister
	JobStoreSynced cache.InformerSynced // returns true if job stroe has been synced. Added as member for testing

	JobControl JobControlInterface

	updateHandler func(*wapi.Workflow) error // callback to upate Workflow. Added as member for testing

	queue workqueue.RateLimitingInterface // Workflows to be synced

	cleanQueue workqueue.RateLimitingInterface // Workflows to be removed

	Recorder record.EventRecorder

	config WorkflowControllerConfig
}

// NewWorkflowController creates and initializes the WorklfowController instance
func NewWorkflowController(workflowClient *rest.RESTClient, workflowScheme *runtime.Scheme, kubeClient clientset.Interface) *WorkflowController {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	wc := &WorkflowController{
		WorkflowClient: workflowClient,
		WorkflowScheme: workflowScheme,
		KubeClient:     kubeClient,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workflow"),
		cleanQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workflow-clean"),
		config: WorkflowControllerConfig{
			RemoveIvalidWorkflow: true,
			NumberOfThreads:      1,
		},
		Recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "workflow-controller"}),
	}
	wc.workflowStore, wc.workflowInformer = cache.NewInformer(
		cache.NewListWatchFromClient(
			workflowClient,
			wapi.ResourcePlural,
			apiv1.NamespaceAll,
			fields.Everything(),
		),
		&wapi.Workflow{},
		time.Duration(2*time.Minute),
		cache.ResourceEventHandlerFuncs{
			AddFunc:    wc.onAddWorkflow,
			UpdateFunc: wc.onUpdateWorkflow,
			DeleteFunc: wc.onDeleteWorkflow,
		},
	)

	wc.JobControl = &WorkflowJobControl{kubeClient, wc.Recorder}

	wc.JobInformer = cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return wc.KubeClient.BatchV1().Jobs(apiv1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return wc.KubeClient.BatchV1().Jobs(apiv1.NamespaceAll).Watch(options)
			},
		},
		&batch.Job{},
		time.Duration(1*time.Minute),
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	wc.JobInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    wc.onAddJob,
			UpdateFunc: wc.onUpdateJob,
			DeleteFunc: wc.onDeleteJob,
		})

	wc.JobLister = v1.NewJobLister(wc.JobInformer.GetIndexer())
	wc.updateHandler = wc.updateWorkflow

	return wc
}

// Run simply runs the controller
func (w *WorkflowController) Run(ctx context.Context) error {
	glog.Infof("Starting workflow controller")

	go w.JobInformer.Run(ctx.Done())
	go w.workflowInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), w.JobInformer.HasSynced, w.workflowInformer.HasSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := 0; i < w.config.NumberOfThreads; i++ {
		go wait.Until(w.cleanWorker, 5*time.Minute, ctx.Done())
	}

	for i := 0; i < w.config.NumberOfThreads; i++ {
		go wait.Until(w.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	return ctx.Err()
}

func (w *WorkflowController) runWorker() {
	for w.processNextItem() {
	}
}

func (w *WorkflowController) cleanWorker() {
	for w.cleanNextItem() {
	}
}

func (w *WorkflowController) cleanNextItem() bool {
	key, quit := w.cleanQueue.Get()
	if quit {
		return false
	}
	defer w.cleanQueue.Done(key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key.(string))
	if err != nil {
		glog.Errorf("unable to dequeu %s from cleanQueue (queue to remove garbage workflows):%v", key, err)
		w.cleanQueue.AddRateLimited(key)
		return true
	}
	if err := w.deleteWorkflow(namespace, name); err != nil {
		w.cleanQueue.AddRateLimited(key)
		return true
	}
	w.cleanQueue.Forget(key)
	return true
}

func (w *WorkflowController) processNextItem() bool {
	key, quit := w.queue.Get()
	if quit {
		return false
	}
	defer w.queue.Done(key)
	err := w.sync(key.(string))
	if err == nil {
		w.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing workflow: %v", err))
	w.queue.AddRateLimited(key)

	return true
}

// enqueue adds key in the controller queue
func (w *WorkflowController) enqueue(workflow *wapi.Workflow) {
	key, err := cache.MetaNamespaceKeyFunc(workflow)
	if err != nil {
		glog.Errorf("WorkflowController:enqueue: couldn't get key for Workflow   %s/%s", workflow.Namespace, workflow.Name)
		return
	}
	w.queue.Add(key)
}

// workflow sync method
func (w *WorkflowController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(6).Infof("Finished syncing Workflow %q (%v", key, time.Now().Sub(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	glog.V(6).Infof("Syncing %s/%s", namespace, name)

	obj, exists, err := w.workflowStore.GetByKey(key)
	if err != nil {
		glog.Errorf("unable to get Workflow %s/%s:%v", namespace, name, err)
		return nil
	}
	if !exists {
		glog.V(6).Info("unable to get Workflow %s/%s: %v (may be deleted?)", namespace, name)
		return nil
	}
	sharedWorkflow := obj.(*wapi.Workflow)
	// Defaulting...
	if !sharedWorkflow.Status.Validated && // a Workflow validated has been already defaulted
		!wapi.IsWorkflowDefaulted(sharedWorkflow) {
		defaultedWorkflow := wapi.DefaultWorkflow(sharedWorkflow)
		if err := w.updateHandler(defaultedWorkflow); err != nil {
			glog.Errorf("WorkflowController.sync unable to default Workflow %s/%s:%v", namespace, name, err)
			return fmt.Errorf("unable to default Workflow %s/%s:%v", namespace, name, err)
		}
		glog.V(6).Infof("WorkflowController.sync Defaulted %s/%s", namespace, name)
		return nil
	}

	// Validation...
	if errs := wapi.ValidateWorkflow(sharedWorkflow); errs != nil && len(errs) > 0 {
		glog.Errorf("WorkflowController.sync Worfklow %s/%s not valid:%v", namespace, name, errs)
		if w.config.RemoveIvalidWorkflow {
			glog.Errorf("Workflow %s/%s going to be removed", namespace, name)
			w.cleanQueue.Add(key)
		}
		return nil
	}

	workflow := sharedWorkflow.DeepCopy()

	// Only valid workflow should follow this point
	if !workflow.Status.Validated {
		workflow.Status.Validated = true
		if err := w.updateHandler(workflow); err != nil {
			glog.Errorf("WorkflowController.sync Workflow %s/%s unable to set status.Validated to true:%v", namespace, name, err)
			return fmt.Errorf("unable to set status.Validated to true:%s/%s:%v", namespace, name, err)
		}
		return nil
	}

	// Init status.StartTime
	if workflow.Status.StartTime == nil {
		workflow.Status.Statuses = make([]wapi.WorkflowStepStatus, 0)
		now := metav1.Now()
		workflow.Status.StartTime = &now
		if err := w.updateHandler(workflow); err != nil {
			glog.Errorf("WorkflowController.sync Workflow %s/%s unable init startTime:%v", namespace, name, err)
			return err
		}
		glog.V(4).Infof("WorkflowController.sync Workflow %s/%s startTime updated", namespace, name)
		return nil
	}

	if pastActiveDeadline(workflow, startTime) {
		now := metav1.Now()
		workflow.Status.Conditions = append(workflow.Status.Conditions, newDeadlineExceededCondition())
		workflow.Status.CompletionTime = &now
		if err := w.updateHandler(workflow); err != nil {
			glog.Errorf("Workflow %s/%s unable to set DeadlineExceeded:%v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
			return fmt.Errorf("unable to set DeadlineExceeded for Workflow %s/%s:%v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
		}
		if err := w.deleteWorkflowJobs(workflow); err != nil {
			glog.Errorf("Workflow %s/%s unable to cleanup jobs:%v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
			return fmt.Errorf("unable to cleanup jobs for %s/%s:%v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
		}
		return nil
	}
	return w.manageWorkflow(workflow)
}

func newDeadlineExceededCondition() wapi.WorkflowCondition {
	return newCondition(wapi.WorkflowFailed, "DeadlineExceeded", "Workflow was active longer than specified deadline")
}

func newCompletedCondition() wapi.WorkflowCondition {
	return newCondition(wapi.WorkflowComplete, "", "")
}

func newCondition(conditionType wapi.WorkflowConditionType, reason, message string) wapi.WorkflowCondition {
	return wapi.WorkflowCondition{
		Type:               conditionType,
		Status:             apiv1.ConditionTrue,
		LastProbeTime:      metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func (w *WorkflowController) onAddWorkflow(obj interface{}) {
	workflow, ok := obj.(*wapi.Workflow)
	if !ok {
		glog.Errorf("WorkflowController.onAddWorkflow, expected workflow object. Got: %+v", obj)
		return
	}
	if !reflect.DeepEqual(workflow.Status, wapi.WorkflowStatus{}) {
		glog.Errorf("Workflow %s/%s created with non empty status. Going to be removed", workflow.Namespace, workflow.Name)
		key, err := cache.MetaNamespaceKeyFunc(workflow)
		if err != nil {
			glog.Errorf("WorkflowController:onAddWorkflow: couldn't get key for Workflow (to be deleted) %s/%s", workflow.Namespace, workflow.Name)
			return
		}
		w.cleanQueue.Add(key)
		return
	}
	w.enqueue(workflow)
}

func (w *WorkflowController) onUpdateWorkflow(oldObj, newObj interface{}) {
	workflow, ok := newObj.(*wapi.Workflow)
	if !ok {
		glog.Errorf("WorkflowController.onUpdateWorkflow, expected workflow object. Got: %+v", newObj)
		return
	}
	if IsWorkflowFinished(workflow) {
		glog.Warning("Update event received on complete Workflow:%s/%s", workflow.Namespace, workflow.Name)
		return
	}
	w.enqueue(workflow)
}

func (w *WorkflowController) onDeleteWorkflow(obj interface{}) {
	workflow, ok := obj.(*wapi.Workflow)
	if !ok {
		glog.Errorf("WorkflowController.onDeleteWorkflow, expected workflow object. Got: %+v", obj)
		return
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("WorkflowController.onDeleteWorkflow, unable to get key for %s/%s:%v", workflow.Namespace, workflow.Name, err)
		return
	}
	w.queue.Add(key)
}

func (w *WorkflowController) updateWorkflow(wfl *wapi.Workflow) error {
	if err := w.WorkflowClient.Put().
		Name(wfl.ObjectMeta.Name).
		Namespace(wfl.ObjectMeta.Namespace).
		Resource(wapi.ResourcePlural).
		Body(wfl).
		Do().
		Error(); err != nil {
		return fmt.Errorf("unable to update Workflow:%v", err)
	}
	glog.V(6).Infof("Workflow %s/%s updated", wfl.ObjectMeta.Namespace, wfl.ObjectMeta.Name)
	return nil
}

func (w *WorkflowController) deleteWorkflow(namespace, name string) error {
	if err := w.WorkflowClient.Delete().
		Name(name).
		Namespace(namespace).
		Resource(wapi.ResourcePlural).
		Do().
		Error(); err != nil {
		return fmt.Errorf("unable to delete Workflow:%v", err)
	}
	glog.V(6).Infof("Workflow %s/%s deleted", namespace, name)
	return nil
}

func (w *WorkflowController) onAddJob(obj interface{}) {
	job := obj.(*batch.Job)
	workflows, err := w.getWorkflowsFromJob(job)
	if err != nil {
		glog.Errorf("WorkflowController.onAddJob:%v", err)
		return
	}
	for i := range workflows {
		w.enqueue(workflows[i])
	}
}

func (w *WorkflowController) onUpdateJob(oldObj, newObj interface{}) {
	oldJob := oldObj.(*batch.Job)
	newJob := newObj.(*batch.Job)

	if equality.Semantic.DeepEqual(oldJob, newJob) {
		return
	}
	glog.V(6).Infof("onUpdateJob old=%v, cur=%v ", oldJob.Name, newJob.Name)
	workflows, err := w.getWorkflowsFromJob(newJob)
	if err != nil {
		glog.Errorf("WorkflowController.onUpdateJob cannot get workflows for job %s/%s:%v", newJob.Namespace, newJob.Name, err)
		return
	}
	for i := range workflows {
		w.enqueue(workflows[i])
	}

	// TODO: in case of relabelling ?
	// TODO: in case of labelSelector relabelling?
}

func (w *WorkflowController) onDeleteJob(obj interface{}) {
	job, ok := obj.(*batch.Job)
	glog.V(6).Infof("onDeleteJob old=%v", job.Name)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %+v", obj)
			return
		}
		job, ok = tombstone.Obj.(*batch.Job)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a job %+v", obj)
			return
		}
	}
	workflows, err := w.getWorkflowsFromJob(job)
	if err != nil {
		glog.Errorf("WorkflowController.onDeleteJob:%v", err)
		return
	}
	for i := range workflows {
		//ww.expectations.DeleteiontObjserved(keyFunc(workflows[i])
		w.enqueue(workflows[i])
	}
}

func (w *WorkflowController) getWorkflowsFromJob(job *batch.Job) ([]*wapi.Workflow, error) {
	workflows := []*wapi.Workflow{}
	if len(job.Labels) == 0 {
		return workflows, fmt.Errorf("no workflows found for job. Job %v has no labels", job.Name)
	}
	workflowList := w.workflowStore.List()
	for i := range workflowList {
		tmp := workflowList[i]
		workflow, ok := tmp.(*wapi.Workflow)
		if !ok {
			glog.Errorf("unable to convert to Workflow %+v", tmp)
			continue
		}
		if workflow.Namespace != job.Namespace {
			continue
		}
		if labels.SelectorFromSet(workflow.Spec.Selector.MatchLabels).Matches(labels.Set(job.Labels)) {
			workflows = append(workflows, workflow)
		}
	}
	return workflows, nil
}

func pastActiveDeadline(workflow *wapi.Workflow, now time.Time) bool {
	if workflow.Spec.ActiveDeadlineSeconds == nil || workflow.Status.StartTime == nil {
		return false
	}
	start := workflow.Status.StartTime.Time
	duration := now.Sub(start)
	allowedDuration := time.Duration(*workflow.Spec.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

func (w *WorkflowController) manageWorkflow(workflow *wapi.Workflow) error {
	workflowToBeUpdated := false
	workflowComplete := true
	for i := range workflow.Spec.Steps {
		stepName := workflow.Spec.Steps[i].Name
		stepStatus := wapi.GetStepStatusByName(workflow, stepName)
		if stepStatus != nil && stepStatus.Complete {
			continue
		}

		workflowComplete = false
		switch {
		case workflow.Spec.Steps[i].JobTemplate != nil: // Job step
			if w.manageWorkflowJobStep(workflow, stepName, &(workflow.Spec.Steps[i])) {
				workflowToBeUpdated = true
				break
			}
		case workflow.Spec.Steps[i].ExternalRef != nil: // TODO handle: external object reference
			if w.manageWorkflowReferenceStep(workflow, stepName, &(workflow.Spec.Steps[i])) {
				workflowToBeUpdated = true
				break
			}
		}
	}
	if workflowComplete {
		now := metav1.Now()
		workflow.Status.Conditions = append(workflow.Status.Conditions, newCompletedCondition())
		workflow.Status.CompletionTime = &now
		glog.Infof("Workflow %s/%s complete.", workflow.Namespace, workflow.Name)
		workflowToBeUpdated = true
	}

	if workflowToBeUpdated {
		if err := w.updateHandler(workflow); err != nil {
			utilruntime.HandleError(err)
			w.enqueue(workflow)
			return err
		}
	}
	return nil
}

func (w *WorkflowController) manageWorkflowJobStep(workflow *wapi.Workflow, stepName string, step *wapi.WorkflowStep) bool {
	workflowUpdated := false
	for _, dependencyName := range step.Dependencies {
		dependencyStatus := GetStepStatusByName(workflow, dependencyName)
		if dependencyStatus == nil || !dependencyStatus.Complete {
			glog.V(4).Infof("Workflow %s/%s: dependecy %q not satisfied for %q", workflow.Namespace, workflow.Name, dependencyName, stepName)
			return workflowUpdated
		}
	}
	glog.V(6).Infof("Workflow %s/%s: All dependecy satisfied for %q", workflow.Namespace, workflow.Name, stepName)
	jobs, err := w.retrieveJobsStep(workflow, step.JobTemplate, stepName)
	if err != nil {
		glog.Errorf("unable to retrieve step jobs for Workflow %s/%s, step:%q:%v", workflow.Namespace, workflow.Name, stepName, err)
		w.enqueue(workflow)
		return workflowUpdated
	}
	switch len(jobs) {
	case 0: // create job
		_, err := w.JobControl.CreateJob(workflow.Namespace, step.JobTemplate, workflow, stepName)
		if err != nil {
			glog.Errorf("Couldn't create job: %v : %v", err, step.JobTemplate)
			w.enqueue(workflow)
			defer utilruntime.HandleError(err)
			return workflowUpdated
		}
		//w.expectations.CreationObserved(key)
		workflowUpdated = true
		glog.V(4).Infof("Job created for step %q", stepName)
	case 1: // update status
		job := jobs[0]
		reference, err := reference.GetReference(scheme.Scheme, job)
		if err != nil || reference == nil {
			glog.Errorf("Unable to get reference from %v: %v", job.Name, err)
			return false
		}
		jobFinished := IsJobFinished(job)
		stepStatus := GetStepStatusByName(workflow, stepName)
		if stepStatus == nil {
			workflow.Status.Statuses = append(workflow.Status.Statuses, wapi.WorkflowStepStatus{
				Name:      stepName,
				Complete:  jobFinished,
				Reference: *reference})
		} else {
			stepStatus.Complete = jobFinished
		}
		workflowUpdated = true
	default: // reconciliate
		glog.Errorf("Workflow.manageWorkflowJobStep %v too many jobs reported... Need reconciliation", workflow.Name)
		return false
	}
	return workflowUpdated
}

func (w *WorkflowController) manageWorkflowReferenceStep(workflow *wapi.Workflow, stepName string, step *wapi.WorkflowStep) bool {
	return true
}

func (w *WorkflowController) retrieveJobsStep(workflow *wapi.Workflow, template *batchv2.JobTemplateSpec, stepName string) ([]*batch.Job, error) {
	jobSelector := createWorkflowJobLabelSelector(workflow, template, stepName)
	jobs, err := w.JobLister.Jobs(workflow.Namespace).List(jobSelector)
	if err != nil {
		return jobs, err
	}
	return jobs, nil
}

func (w *WorkflowController) deleteWorkflowJobs(workflow *wapi.Workflow) error {
	jobsSelector := inferrWorkflowLabelSelectorForJobs(workflow)
	jobs, err := w.JobLister.Jobs(workflow.Namespace).List(jobsSelector)
	if err != nil {
		return fmt.Errorf("workflow %s/%s, unable to retrieve jobs to remove:%v", workflow.Namespace, workflow.Name, err)
	}
	errs := []error{}
	for i := range jobs {
		jobToBeRemoved := jobs[i].DeepCopy()
		if err := w.JobControl.DeleteJob(jobToBeRemoved.Namespace, jobToBeRemoved.Name, jobToBeRemoved); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}
