package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	"github.com/heptiolabs/healthcheck"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	apiv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/reference"
	"k8s.io/client-go/util/workqueue"

	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	winformers "github.com/amadeusitgroup/workflow-controller/pkg/client/informers/externalversions"
	wlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/workflow/v1"
)

// WorkflowControllerConfig contains info to customize Workflow controller behaviour
type WorkflowControllerConfig struct {
	RemoveInvalidWorkflow bool
	NumberOfWorkers       int
}

// WorkflowStepLabelKey defines the key of label to be injected by workflow controller
const (
	WorkflowLabelKey     = "kubernetes.io/workflow"
	WorkflowStepLabelKey = "kubernetes.io/workflow-step"
)

// WorkflowController represents the Workflow controller
type WorkflowController struct {
	Client     wclientset.Interface
	KubeClient clientset.Interface

	WorkflowLister wlisters.WorkflowLister
	WorkflowSynced cache.InformerSynced

	JobLister batchv1listers.JobLister
	JobSynced cache.InformerSynced // returns true if job has been synced. Added as member for testing

	JobControl JobControlInterface

	updateHandler func(*wapi.Workflow) error // callback to upate Workflow. Added as member for testing
	syncHandler   func(string) error

	queue workqueue.RateLimitingInterface // Workflows to be synced

	Recorder record.EventRecorder

	config WorkflowControllerConfig
}

// NewWorkflowController creates and initializes the WorkflowController instance
func NewWorkflowController(
	client wclientset.Interface,
	kubeClient clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	workflowInformerFactory winformers.SharedInformerFactory) *WorkflowController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	workflowInformer := workflowInformerFactory.Workflow().V1().Workflows()

	wc := &WorkflowController{
		Client:         client,
		KubeClient:     kubeClient,
		WorkflowLister: workflowInformer.Lister(),
		WorkflowSynced: workflowInformer.Informer().HasSynced,
		JobLister:      jobInformer.Lister(),
		JobSynced:      jobInformer.Informer().HasSynced,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workflow"),
		config: WorkflowControllerConfig{
			RemoveInvalidWorkflow: true,
			NumberOfWorkers:       1,
		},

		Recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "workflow-controller"}),
	}
	workflowInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    wc.onAddWorkflow,
			UpdateFunc: wc.onUpdateWorkflow,
			DeleteFunc: wc.onDeleteWorkflow,
		},
	)

	jobInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    wc.onAddJob,
			UpdateFunc: wc.onUpdateJob,
			DeleteFunc: wc.onDeleteJob,
		},
	)
	wc.JobControl = &WorkflowJobControl{kubeClient, wc.Recorder}
	wc.updateHandler = wc.updateWorkflow
	wc.syncHandler = wc.sync

	return wc
}

// AddHealthCheck add Readiness and Liveness Checks to the handler
func (w *WorkflowController) AddHealthCheck(h healthcheck.Handler) {
	h.AddReadinessCheck("Workflow_cache_sync", func() error {
		if w.WorkflowSynced() {
			return nil
		}
		return fmt.Errorf("Workflow cache not sync")
	})

	h.AddReadinessCheck("Job_cache_sync", func() error {
		if w.JobSynced() {
			return nil
		}
		return fmt.Errorf("Job cache not sync")
	})
}

// Run simply runs the controller. Assuming Informers are already running: via factories
func (w *WorkflowController) Run(ctx context.Context) error {
	glog.Infof("Starting workflow controller")

	if !cache.WaitForCacheSync(ctx.Done(), w.JobSynced, w.WorkflowSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := 0; i < w.config.NumberOfWorkers; i++ {
		go wait.Until(w.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	return ctx.Err()
}

func (w *WorkflowController) runWorker() {
	for w.processNextItem() {
	}
}

func (w *WorkflowController) processNextItem() bool {
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
func (w *WorkflowController) enqueue(workflow *wapi.Workflow) {
	key, err := cache.MetaNamespaceKeyFunc(workflow)
	if err != nil {
		glog.Errorf("WorkflowController:enqueue: couldn't get key for Workflow   %s/%s", workflow.Namespace, workflow.Name)
		return
	}
	w.queue.Add(key)
}

// get workflow by key method
func (w *WorkflowController) getWorkflowByKey(key string) (*wapi.Workflow, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	glog.V(6).Infof("Syncing %s/%s", namespace, name)
	workflow, err := w.WorkflowLister.Workflows(namespace).Get(name)
	if err != nil {
		glog.V(4).Infof("unable to get Workflow %s/%s: %v. Maybe deleted", namespace, name, err)
		return nil, err
	}
	return workflow, nil
}

// workflow sync method
func (w *WorkflowController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(6).Infof("Finished syncing Workflow %q (%v", key, time.Now().Sub(startTime))
	}()

	var sharedWorkflow *wapi.Workflow
	var err error
	if sharedWorkflow, err = w.getWorkflowByKey(key); err != nil {
		glog.V(4).Infof("unable to get Workflow %s: %v. Maybe deleted", key, err)
		return nil
	}
	namespace := sharedWorkflow.Namespace
	name := sharedWorkflow.Name

	if !wapi.IsWorkflowDefaulted(sharedWorkflow) {
		defaultedWorkflow := wapi.DefaultWorkflow(sharedWorkflow)
		if err := w.updateHandler(defaultedWorkflow); err != nil {
			glog.Errorf("WorkflowController.sync unable to default Workflow %s/%s: %v", namespace, name, err)
			return fmt.Errorf("unable to default Workflow %s/%s: %v", namespace, name, err)
		}
		glog.V(6).Infof("WorkflowController.sync Defaulted %s/%s", namespace, name)
		return nil
	}

	// Validation. We always revalidate Workflow since any update must be re-checked.
	if errs := wapi.ValidateWorkflow(sharedWorkflow); errs != nil && len(errs) > 0 {
		glog.Errorf("WorkflowController.sync Worfklow %s/%s not valid: %v", namespace, name, errs)
		if w.config.RemoveInvalidWorkflow {
			glog.Errorf("Invalid workflow %s/%s is going to be removed", namespace, name)
			if err := w.deleteWorkflow(namespace, name); err != nil {
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

	// Init status.StartTime
	if workflow.Status.StartTime == nil {
		workflow.Status.Statuses = make([]wapi.WorkflowStepStatus, 0)
		now := metav1.Now()
		workflow.Status.StartTime = &now
		if err := w.updateHandler(workflow); err != nil {
			glog.Errorf("workflow %s/%s: unable init startTime: %v", namespace, name, err)
			return err
		}
		glog.V(4).Infof("workflow %s/%s: startTime updated", namespace, name)
		return nil
	}

	if pastActiveDeadline(workflow, startTime) {
		now := metav1.Now()
		workflow.Status.Conditions = append(workflow.Status.Conditions, newDeadlineExceededCondition())
		workflow.Status.CompletionTime = &now
		if err := w.updateHandler(workflow); err != nil {
			glog.Errorf("workflow %s/%s unable to set DeadlineExceeded: %v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
			return fmt.Errorf("unable to set DeadlineExceeded for Workflow %s/%s: %v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
		}
		if err := w.deleteWorkflowJobs(workflow); err != nil {
			glog.Errorf("workflow %s/%s: unable to cleanup jobs: %v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
			return fmt.Errorf("workflow %s/%s: unable to cleanup jobs: %v", workflow.ObjectMeta.Namespace, workflow.ObjectMeta.Name, err)
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
		if err := w.deleteWorkflow(workflow.Namespace, workflow.Name); err != nil {
			glog.Errorf("unable to delete non empty status Workflow %s/%s: %v. No retry will be performed.", workflow.Namespace, workflow.Name, err)
		}
		return
	}
	w.enqueue(workflow)
}

func (w *WorkflowController) onUpdateWorkflow(oldObj, newObj interface{}) {
	workflow, ok := newObj.(*wapi.Workflow)
	if !ok {
		glog.Errorf("Expected workflow object. Got: %+v", newObj)
		return
	}
	if IsWorkflowFinished(workflow) {
		glog.Warningf("Update event received on complete Workflow: %s/%s", workflow.Namespace, workflow.Name)
		return
	}
	w.enqueue(workflow)
}

func (w *WorkflowController) onDeleteWorkflow(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	w.queue.Add(key)
}

func (w *WorkflowController) updateWorkflow(wfl *wapi.Workflow) error {
	if _, err := w.Client.WorkflowV1().Workflows(wfl.Namespace).Update(wfl); err != nil {
		glog.V(6).Infof("Workflow %s/%s updated", wfl.Namespace, wfl.Name)
	}
	return nil
}

func (w *WorkflowController) deleteWorkflow(namespace, name string) error {
	if err := w.Client.WorkflowV1().Workflows(namespace).Delete(name, nil); err != nil {
		return fmt.Errorf("unable to delete Workflow %s/%s: %v", namespace, name, err)
	}

	glog.V(6).Infof("Workflow %s/%s deleted", namespace, name)
	return nil
}

func (w *WorkflowController) onAddJob(obj interface{}) {
	job := obj.(*batch.Job)
	workflows, err := w.getWorkflowsFromJob(job)
	if err != nil {
		glog.Errorf("unable to get workflows from job %s/%s: %v", job.Namespace, job.Name, err)
		return
	}
	for i := range workflows {
		w.enqueue(workflows[i])
	}
}

func (w *WorkflowController) onUpdateJob(oldObj, newObj interface{}) {
	oldJob := oldObj.(*batch.Job)
	newJob := newObj.(*batch.Job)
	if oldJob.ResourceVersion == newJob.ResourceVersion { // Since periodic resync will send update events for all known jobs.
		return
	}
	glog.V(6).Infof("onUpdateJob old=%v, cur=%v ", oldJob.Name, newJob.Name)
	workflows, err := w.getWorkflowsFromJob(newJob)
	if err != nil {
		glog.Errorf("WorkflowController.onUpdateJob cannot get workflows for job %s/%s: %v", newJob.Namespace, newJob.Name, err)
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
		glog.Errorf("WorkflowController.onDeleteJob: %v", err)
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
		return workflows, fmt.Errorf("no workflows found for job. Job %s/%s has no labels", job.Namespace, job.Name)
	}
	workflowList, err := w.WorkflowLister.List(labels.Everything())
	if err != nil {
		return workflows, fmt.Errorf("no workflows found for job. Job %s/%s. Cannot list workflows: %v", job.Namespace, job.Name, err)
	}
	//workflowStore.List()
	for i := range workflowList {
		workflow := workflowList[i]
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
			glog.V(4).Infof("Workflow %s/%s: dependency %q not satisfied for %q", workflow.Namespace, workflow.Name, dependencyName, stepName)
			return workflowUpdated
		}
	}
	glog.V(6).Infof("Workflow %s/%s: All dependency satisfied for %q", workflow.Namespace, workflow.Name, stepName)
	jobs, err := w.retrieveJobsStep(workflow, step.JobTemplate, stepName)
	if err != nil {
		glog.Errorf("unable to retrieve step jobs for Workflow %s/%s, step:%q: %v", workflow.Namespace, workflow.Name, stepName, err)
		w.enqueue(workflow)
		return workflowUpdated
	}
	switch len(jobs) {
	case 0: // create job
		_, err := w.JobControl.CreateJobFromWorkflow(workflow.Namespace, step.JobTemplate, workflow, stepName)
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
			stepStatus = &wapi.WorkflowStepStatus{
				Name:      stepName,
				Complete:  jobFinished,
				Reference: *reference,
			}
			workflow.Status.Statuses = append(workflow.Status.Statuses, *stepStatus)
			workflowUpdated = true
		}
		if jobFinished {
			stepStatus.Complete = jobFinished
			glog.V(4).Infof("Workflow %s/%s Job finished for step %q", workflow.Namespace, workflow.Name, stepName)
			workflowUpdated = true
		}
	default: // reconciliate
		glog.Errorf("manageWorkflowJobStep %v too many jobs reported... Need reconciliation", workflow.Name)
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
	glog.V(6).Infof("deleting all jobs for workflow %s/%s", workflow.Namespace, workflow.Name)

	jobsSelector := inferrWorkflowLabelSelectorForJobs(workflow)
	return deleteJobsFromLabelSelector(workflow.Namespace, jobsSelector, w.JobLister, w.JobControl)
}
