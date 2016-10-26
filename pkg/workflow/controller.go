/*
Copyright 2016 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workflow

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/extensions"

	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	unversionedcore "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/unversioned"
	"k8s.io/kubernetes/pkg/client/record"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/thirdpartyresourcedata"
	"k8s.io/kubernetes/pkg/runtime"
	utilruntime "k8s.io/kubernetes/pkg/util/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/util/workqueue"
	"k8s.io/kubernetes/pkg/watch"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	wapivalidation "github.com/sdminonne/workflow-controller/pkg/api/validation"
	wclient "github.com/sdminonne/workflow-controller/pkg/client"
)

// WorkflowStepLabelKey defines the key of label to be injected by workflow controller
const WorkflowStepLabelKey = "kubernetes.io/workflow"

// Controller is a useless struct I created just as a placeholder
type Controller struct {

	// resource contains the ThirdPartyResource handled by the controller
	resource *extensions.ThirdPartyResource

	// kubeClient  is needed to retrieve kubernetes Objects
	kubeClient clientset.Interface

	// wfClient is needed to retrieve Workflows Objects
	wfClient wclient.Interface

	//jobControl is needed to create/delete Jobs
	jobControl JobControlInterface

	// To allow injection of updateWorkflowStatus for testing.
	updateHandler func(workflow *wapi.Workflow) error
	syncHandler   func(workflowKey string) error
	// jobStoreSynced returns true if the jod store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	jobStoreSynced func() bool

	// A TTLCache of job creates/deletes each rc expects to see
	expectations controller.ControllerExpectationsInterface

	// A store of workflow, populated by the cacheController
	workflowStore wclient.StoreToWorkflowLister

	// Watches changes to all workflows
	workflowController *cache.Controller

	// A store of job, populated by the jobController
	jobStore cache.StoreToJobLister

	// Watches changes to all jobs. It doesn't take any actions since
	jobController *cache.Controller

	// Workflows to be updated
	queue *workqueue.Type

	recorder record.EventRecorder
}

// GetGroupVersionKind returns GroupVersionKind for the thirdpartyresourcedata to be handled
func (w *Controller) GetGroupVersionKind() *unversioned.GroupVersionKind {
	kind, group, err := thirdpartyresourcedata.ExtractApiGroupAndKind(w.resource)
	if err != nil {
		glog.Errorf("cannot extract API Group and Kind: %v", err)
		return nil
	}
	return &unversioned.GroupVersionKind{
		Group:   group,
		Version: w.resource.Versions[0].Name,
		Kind:    kind,
	}
}

// NewController creates and doesn't initialize the workflow controller
func NewController(kubeClient clientset.Interface, wfClient wclient.Interface, resource *extensions.ThirdPartyResource, resyncPeriod controller.ResyncPeriodFunc) *Controller {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)

	eventBroadcaster.StartRecordingToSink(&unversionedcore.EventSinkImpl{Interface: kubeClient.Core().Events("")})

	wc := &Controller{
		resource:   resource,
		kubeClient: kubeClient,
		wfClient:   wfClient,
		jobControl: WorkflowJobControl{
			KubeClient: kubeClient,
			Recorder:   eventBroadcaster.NewRecorder(api.EventSource{Component: "workflow-controller"}),
		},
		expectations: controller.NewControllerExpectations(),
		queue:        workqueue.New(),
		recorder:     eventBroadcaster.NewRecorder(api.EventSource{Component: "workflow-controller"}),
	}

	wc.jobStore.Store, wc.jobController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return wc.kubeClient.Batch().Jobs(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return wc.kubeClient.Batch().Jobs(api.NamespaceAll).Watch(options)
			},
		},
		&batch.Job{},
		resyncPeriod(),
		cache.ResourceEventHandlerFuncs{
			AddFunc:    wc.onAddJob,
			UpdateFunc: wc.onUpdateJob,
			DeleteFunc: wc.onDeleteJob,
		},
	)

	wc.jobStoreSynced = wc.jobController.HasSynced

	wc.workflowStore.Store, wc.workflowController = cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return wc.wfClient.Workflows(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return wc.wfClient.Workflows(api.NamespaceAll).Watch(options)
			},
		},
		&wapi.Workflow{},
		resyncPeriod(),
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(cur interface{}) {
				w, ok := cur.(*wapi.Workflow)
				if !ok {
					glog.Errorf("not workflow: %t", cur)
					return
				}

				removeInvalidWorkflow := true // TODO: @sdminonne it should be configurable
				if err := wc.defaultAndValidateWorkflow(w, removeInvalidWorkflow); err != nil {
					glog.Errorf("Unable to default and validate workflow: %v", err)
					return
				}
				wc.enqueueController(w)
			},
			UpdateFunc: func(old, cur interface{}) {
				wc.enqueueController(cur)
			},
			DeleteFunc: wc.enqueueController,
		},
	)
	wc.updateHandler = wc.updateWorkflowStatus
	wc.syncHandler = wc.syncWorkflow
	return wc
}

func (w *Controller) defaultAndValidateWorkflow(workflow *wapi.Workflow, removeInvalidWorkflow bool) error {
	defaultedWorkflow, err := wapi.NewBasicDefaulter().Default(workflow)
	if err != nil {
		return fmt.Errorf("couldn't default Workflow %q: %v", workflow.Name, err)
	}

	errs := wapivalidation.ValidateWorkflow(defaultedWorkflow)
	if len(errs) != 0 {
		validationErrors := fmt.Errorf("Invalid workflow %q: %v", workflow.Name, errs.ToAggregate())
		if removeInvalidWorkflow {
			err := w.wfClient.Workflows(workflow.Namespace).Delete(workflow.Name, nil)
			if err != nil {
				return fmt.Errorf("%v: unable to remove it: %v", validationErrors, err)
			}
			return fmt.Errorf("%v: removed", validationErrors)
		}
		// TODO: annotate invalid workflow if not removed: user may want to patch it and feedback may come from annotation
		return validationErrors
	}
	if _, err = w.wfClient.Workflows(workflow.GetNamespace()).Update(defaultedWorkflow); err != nil {
		return fmt.Errorf("unable to default workflow %q: %v", workflow.Name, err)
	}
	glog.Infof("Worklow %q defaulted", workflow.Name)
	return nil
}

// Run runs main goroutine responsible for watching and syncing workflows.
func (w *Controller) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer w.queue.ShutDown()

	go w.jobController.Run(stopCh)
	go w.workflowController.Run(stopCh)

	for i := 0; i < workers; i++ {
		go wait.Until(w.worker, time.Second, stopCh)
	}

	<-stopCh
	glog.Infof("Shutting down Workflow Controller")
}

// getJobWorkflow return the workflow managing the given job
func (w *Controller) getJobWorkflow(job *batch.Job) *wapi.Workflow {
	workflows, err := w.workflowStore.GetJobWorkflows(job)
	if err != nil {
		glog.V(4).Infof("No workflows found for job %q: %v", job.Name, err)
		return nil
	}
	if len(workflows) == 0 {
		glog.V(4).Infof("No workflows found for job %q", job.Name)
		return nil
	}
	if len(workflows) > 1 {
		glog.Warningf("More than one workflow is selecting jobs with labels: %+v", job.Labels)
		// TODO: create a better policy
		//sort.Sort(byCreationTimestamp(workflows))
	}

	return &workflows[0]
}

// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (w *Controller) worker() {
	for {
		func() {
			key, quit := w.queue.Get()
			if quit {
				return
			}
			defer w.queue.Done(key)
			err := w.syncHandler(key.(string))
			if err != nil {
				glog.Errorf("Error syncing workflow: %v", err)
			}
		}()
	}
}

func (w *Controller) syncWorkflow(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(6).Infof("Finished syncing workflow %q (%v)", key, time.Now().Sub(startTime))
	}()

	if !w.jobStoreSynced() {
		time.Sleep(100 * time.Millisecond) // @sdminonne: TODO remove hard coded value
		glog.Infof("Waiting for job controller to sync, requeuing workflow %v", key)
		w.queue.Add(key)
		return nil
	}

	obj, exists, err := w.workflowStore.GetByKey(key)
	if !exists {
		glog.V(4).Infof("Workflow has been deleted: %v", key)
		w.expectations.DeleteExpectations(key)
		return nil
	}
	if err != nil {
		glog.Errorf("Unable to retrieve workflow %v from store: %v", key, err)
		w.queue.Add(key)
		return err
	}

	workflow, ok := obj.(*wapi.Workflow)
	if !ok {
		glog.Errorf("Couldn't obtain workflow from %t", obj)
	}
	workflowKey, err := controller.KeyFunc(workflow)
	if err != nil {
		glog.Errorf("Couldn't get key for workflow: %v", err)
		return err
	}

	if workflow.Status.Statuses == nil {
		workflow.Status.Statuses = make([]wapi.WorkflowStepStatus, 0)
		now := unversioned.Now()
		workflow.Status.StartTime = &now
		glog.V(4).Infof("Workflow.Status.Statuses initialized for %q", workflow.Name)
	}

	workflowNeedsSync := w.expectations.SatisfiedExpectations(workflowKey)
	if !workflowNeedsSync {
		glog.V(4).Infof("Workflow %v doesn't need synch", workflow.Name)
		return nil
	}

	if isWorkflowFinished(workflow) {
		return nil
	}

	if pastActiveDeadline(workflow) {
		// @sdminonne: TODO delete jobs & write error for the ExternalReference
		now := unversioned.Now()
		condition := wapi.WorkflowCondition{
			Type:               wapi.WorkflowFailed,
			Status:             api.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
			Reason:             "DeadlineExceeded",
			Message:            "Workflow was active longer than specified deadline",
		}
		workflow.Status.Conditions = append(workflow.Status.Conditions, condition)
		workflow.Status.CompletionTime = &now
		w.recorder.Event(workflow, api.EventTypeNormal, "DeadlineExceeded", "Workflow was active longer than specified deadline")
		if err := w.updateHandler(workflow); err != nil {
			glog.Errorf("Failed to update workflow %v, requeuing.  Error: %v", workflow.Name, err)
			w.enqueueController(workflow)
		}
		return nil
	}

	if w.manageWorkflow(workflow) {
		if err = w.updateHandler(workflow); err != nil {
			glog.Errorf("Failed to update workflow %v, requeuing.  Error: %v", workflow.Name, err)
			w.enqueueController(workflow)
			return nil
		}
		glog.Infof("Workflow %q status updated", workflow.Name)
	}
	return nil
}

// pastActiveDeadline checks if workflow has ActiveDeadlineSeconds field set and if it is exceeded.
func pastActiveDeadline(workflow *wapi.Workflow) bool {
	if workflow.Spec.ActiveDeadlineSeconds == nil || workflow.Status.StartTime == nil {
		return false
	}
	now := unversioned.Now()
	start := workflow.Status.StartTime.Time
	duration := now.Time.Sub(start)
	allowedDuration := time.Duration(*workflow.Spec.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

func (w *Controller) updateWorkflowStatus(workflow *wapi.Workflow) error {
	// todo @sdminonne: client to support UpdateStatus ??
	_, err := w.wfClient.Workflows(workflow.GetNamespace()).Update(workflow)
	return err
}

// A workflow is finished if one of its condition is Complete or Failed.
func isWorkflowFinished(w *wapi.Workflow) bool {
	for _, c := range w.Status.Conditions {
		if c.Status == api.ConditionTrue && (c.Type == wapi.WorkflowComplete || c.Type == wapi.WorkflowFailed) {
			return true
		}
	}
	return false
}

// enqueueController adds key in the controller queue
func (w *Controller) enqueueController(obj interface{}) {
	key, err := controller.KeyFunc(obj)
	if err != nil {
		glog.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	w.queue.Add(key)
}

// When a job is created, enqueue the controller tha manages it and update
// expectations
func (w *Controller) onAddJob(obj interface{}) {
	job := obj.(*batch.Job)
	glog.V(4).Infof("onAddJob %v", job.Name)
	if workflow := w.getJobWorkflow(job); workflow != nil {
		key, err := controller.KeyFunc(workflow)
		if err != nil {
			glog.Errorf("No key for workflow %#v: %v", workflow, err)
			return
		}
		w.expectations.CreationObserved(key)
		w.enqueueController(workflow)
	}
}

func (w *Controller) onUpdateJob(old, cur interface{}) {
	oldJob := old.(*batch.Job)
	curJob := cur.(*batch.Job)
	glog.V(4).Infof("onUpdateJob old=%v, cur=%v ", oldJob.Name, curJob.Name)
	if api.Semantic.DeepEqual(old, cur) {
		glog.V(4).Infof("\t nothing to update")
		return
	}
	if workflow := w.getJobWorkflow(curJob); workflow != nil {
		w.enqueueController(workflow)
	}
	// in case of relabelling
	if !reflect.DeepEqual(oldJob.Labels, curJob.Labels) {
		if oldWorkflow := w.getJobWorkflow(oldJob); oldWorkflow != nil {
			w.enqueueController(oldWorkflow)
		}
	}
	// TODO: in case of labelSelector relabelling?
}

func (w *Controller) onDeleteJob(obj interface{}) {
	job, ok := obj.(*batch.Job)
	glog.V(4).Infof("onDeleteJob old=%v", job.Name)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %+v, could take up to %v before a workflow recreates a job", obj, controller.ExpectationsTimeout)
			return
		}
		job, ok = tombstone.Obj.(*batch.Job)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a job %+v, could take up to %v before a workflow recreates a job", obj, controller.ExpectationsTimeout)
			return
		}
	}
	if workflow := w.getJobWorkflow(job); workflow != nil {
		key, err := controller.KeyFunc(obj)
		if err != nil {
			glog.Errorf("Couldn't get key for workflow %#v: %v", workflow, err)
			return
		}
		w.expectations.DeletionObserved(key)
		w.enqueueController(workflow)
	}
}

func (w *Controller) manageWorkflow(workflow *wapi.Workflow) bool {
	needsStatusUpdate := false
	glog.V(4).Infof("manage Workflow -> %v", workflow.Name)

	workflowComplete := true
	for i := range workflow.Spec.Steps {
		stepName := workflow.Spec.Steps[i].Name
		stepStatus := workflow.GetStepStatusByName(stepName)
		if stepStatus != nil && stepStatus.Complete {
			//if stepStatus, ok := workflow.Status.Statuses[stepName]; ok && stepStatus.Complete {
			glog.V(6).Infof("Step %q completed.", stepName)
			continue
		}
		workflowComplete = false // if here during the loop workflow not completed
		switch {
		case workflow.Spec.Steps[i].JobTemplate != nil: // Job step
			needsStatusUpdate = w.manageWorkflowJobStep(workflow, stepName, &(workflow.Spec.Steps[i])) || needsStatusUpdate
			//case workflow.Spec.Steps[i].ExternalRef != nil: // TODO handle: external object reference
			//	needsStatusUpdate = w.manageWorkflowReference(workflow, stepName, &step) || needsStatusUpdate
		}
	}

	if workflowComplete {
		now := unversioned.Now()
		condition := wapi.WorkflowCondition{
			Type:               wapi.WorkflowComplete,
			Status:             api.ConditionTrue,
			LastProbeTime:      now,
			LastTransitionTime: now,
		}
		workflow.Status.Conditions = append(workflow.Status.Conditions, condition)
		workflow.Status.CompletionTime = &now
		needsStatusUpdate = true
	}

	return needsStatusUpdate
}

func (w *Controller) retrieveJobsStep(workflow *wapi.Workflow, template *batch.JobTemplateSpec, stepName string) ([]batch.Job, error) {
	var jobs []batch.Job
	jobSelector := CreateWorkflowJobLabelSelector(workflow, template, stepName)
	jobList, err := w.jobStore.List()
	if err != nil {
		return jobs, err
	}
	for _, job := range jobList.Items {
		if jobSelector.Matches(labels.Set(job.Labels)) {
			job := job
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

// manageWorkflowJobStep handle a workflow step for JobTemplate.
func (w *Controller) manageWorkflowJobStep(workflow *wapi.Workflow, stepName string, step *wapi.WorkflowStep) bool {
	for _, dependencyName := range step.Dependencies {
		dependencyStatus := workflow.GetStepStatusByName(dependencyName)
		if dependencyStatus == nil || !dependencyStatus.Complete {
			glog.V(4).Infof("Dependecy %q not satisfied for %q", dependencyName, stepName)
			return false
		}
	}
	// all dependency satisfied (or missing) need action: update or create step
	key, err := controller.KeyFunc(workflow)
	if err != nil {
		glog.Errorf("Couldn't get key for workflow %#v: %v", workflow, err)
		return false
	}

	jobs, err := w.retrieveJobsStep(workflow, step.JobTemplate, stepName)
	if err != nil {
		glog.Errorf("Error getting jobs for step %q in workflow %q: %v", stepName, key, err)
		w.queue.Add(key)
		return false
	}
	switch len(jobs) {
	case 0: // create job
		err := w.jobControl.CreateJob(workflow.Namespace, step.JobTemplate, workflow, stepName)
		if err != nil {
			glog.Errorf("Couldn't create job: %v : %v", err, step.JobTemplate)
			defer utilruntime.HandleError(err)
			w.expectations.CreationObserved(key)
			return false
		}
		glog.V(4).Infof("Job created for step %q", stepName)
	case 1: // update status
		job := jobs[0]
		reference, err := api.GetReference(&job)
		if err != nil || reference == nil {
			glog.Errorf("Unable to get reference from %v: %v", job.Name, err)
			return false
		}
		jobFinished := IsJobFinished(&job)
		stepStatus := workflow.GetStepStatusByName(stepName)
		if stepStatus == nil {
			workflow.Status.Statuses = append(workflow.Status.Statuses, wapi.WorkflowStepStatus{
				Name:      stepName,
				Complete:  jobFinished,
				Reference: *reference})
		} else {
			stepStatus.Complete = jobFinished
		}
	default: // reconciliate
		glog.Errorf("Workflow.manageWorkfloJob %v too many jobs reported... Need reconciliation", workflow.Name)
		return false
	}
	return true
}

// IsJobFinished returns true whether or not a job is finished
func IsJobFinished(j *batch.Job) bool {
	for _, c := range j.Status.Conditions {
		if (c.Type == batch.JobComplete || c.Type == batch.JobFailed) && c.Status == api.ConditionTrue {
			return true
		}
	}
	return false
}
