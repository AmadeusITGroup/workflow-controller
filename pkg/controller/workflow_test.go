package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	batch "k8s.io/api/batch/v1"
	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wclient "github.com/amadeusitgroup/workflow-controller/pkg/client"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	winformers "github.com/amadeusitgroup/workflow-controller/pkg/client/informers/externalversions"
	utiltesting "github.com/amadeusitgroup/workflow-controller/pkg/util/testing"
)

// utility function to create a basic Workflow with steps
func newWorkflow(count int32, startTime *metav1.Time) *wapi.Workflow {
	workflow := wapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: workflow.GroupName + "/v1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mydag",
			Namespace: api.NamespaceDefault,
		},
		Spec: wapi.WorkflowSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"workflow": "example-selector",
				},
			},
		},
	}
	// update workflow status
	workflow.Status.StartTime = startTime
	// populate steps
	workflow.Spec.Steps = make([]wapi.WorkflowStep, count)
	for i := range workflow.Spec.Steps {
		workflow.Spec.Steps[i].Name = fmt.Sprintf("step-%v", i)
		workflow.Spec.Steps[i].JobTemplate = utiltesting.ValidFakeTemplateSpec()
	}
	return &workflow
}

// create count jobs with the given state (Active, Complete, Failed) for the given workflow
func newJobList(count int32, fromIndex int32, status batch.JobConditionType, workflow *wapi.Workflow) []batch.Job {
	var succeededPods, failedPods, activePods int32
	var condition batch.JobCondition
	switch status {
	case batch.JobComplete:
		succeededPods = 1
		condition.Type = batch.JobComplete
		condition.Status = api.ConditionTrue
	case batch.JobFailed:
		failedPods = 1
		condition.Type = batch.JobFailed
		condition.Status = api.ConditionTrue
	default:
		activePods = 1
	}
	jobs := []batch.Job{}
	for i := int32(0); i < count; i++ {
		// set step name
		stepIndex := i + fromIndex
		stepName := fmt.Sprintf("step-%v", stepIndex)
		// get labels
		labelset, _ := getJobLabelsSetFromWorkflow(workflow, workflow.Spec.Steps[stepIndex].JobTemplate, stepName)
		labels := map[string]string{}
		for k, v := range labelset {
			labels[k] = v
		}
		// create Job
		newJob := batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      stepName,
				Labels:    labels,
				Namespace: workflow.Namespace,
				SelfLink:  "/apiv1s/extensions/v1beta1/namespaces/default/jobs/job",
			},
			Status: batch.JobStatus{
				Conditions: []batch.JobCondition{condition},
				Active:     activePods,
				Failed:     failedPods,
				Succeeded:  succeededPods,
			},
		}
		jobs = append(jobs, newJob)
	}
	return jobs
}

func newWorkflowControllerFromClients(kubeClient clientset.Interface, workflowClient versioned.Interface) (*WorkflowController, kubeinformers.SharedInformerFactory, winformers.SharedInformerFactory) {
	kubeInformers := kubeinformers.NewSharedInformerFactory(kubeClient, 0)
	workflowInformers := winformers.NewSharedInformerFactory(workflowClient, 0)
	wc := NewWorkflowController(workflowClient, kubeClient, kubeInformers, workflowInformers)
	wc.JobControl = &FakeJobControl{}
	wc.JobSynced = func() bool { return true }

	return wc, kubeInformers, workflowInformers
}

func TestControllerSyncWorkflow(t *testing.T) {
	testCases := map[string]struct {
		// workflow setup
		deleting   bool
		stepsCount int32
		startTime  *metav1.Time
		// jobs setup
		activeJobs    int32
		succeededJobs int32
		failedJobs    int32
		// helper functions
		workflowTweak       func(*wapi.Workflow) *wapi.Workflow
		customUpdateHandler func(*wapi.Workflow) error // custom update func
		// expectations
		expectedWorkflowFinished bool
		expectedWorkflowFailed   bool
	}{
		"workflow default": { // it tests if the workflow is defaulted
			deleting:   false,
			stepsCount: 1,
			startTime:  nil,
			activeJobs: 0, succeededJobs: 0, failedJobs: 0,
			customUpdateHandler: func(w *wapi.Workflow) error {
				if !wapi.IsWorkflowDefaulted(w) {
					return fmt.Errorf("workflow %q not defaulted", w.Name)
				}
				return nil
			},
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return w
			},
		},
		"workflow validated": {
			deleting:   false,
			stepsCount: 1,
			startTime:  nil,
			activeJobs: 0, succeededJobs: 0, failedJobs: 0,
			customUpdateHandler: func(w *wapi.Workflow) error {
				errs := wapi.ValidateWorkflow(w)
				if len(errs) > 0 {
					return fmt.Errorf("workflow %q not valid", w.Name)
				}
				return nil
			},
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w) // workflow must be defaulted to be validated
			},
		},
		"workflow is running, no completed steps": {
			deleting:   false,
			stepsCount: 10,
			startTime:  &metav1.Time{time.Now().Truncate(time.Minute * (-1))},
			activeJobs: 10, succeededJobs: 0, failedJobs: 0,
			customUpdateHandler: nil,
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w)
			},
		},
		"workflow is running and with some completed steps": {
			deleting:   false,
			stepsCount: 10,
			startTime:  &metav1.Time{time.Now().Truncate(time.Minute * (-1))},
			activeJobs: 5, succeededJobs: 5, failedJobs: 0,
			customUpdateHandler: nil,
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w)
			},
		},
		"workflow is running and with some completed and some failed steps": {
			deleting:   false,
			stepsCount: 10,
			startTime:  &metav1.Time{time.Now().Truncate(time.Minute * (-1))},
			activeJobs: 5, succeededJobs: 3, failedJobs: 2,
			customUpdateHandler: nil,
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w)
			},
		},
		"workflow completed with succeeded steps": {
			deleting:   false,
			stepsCount: 10,
			startTime:  &metav1.Time{time.Now().Truncate(time.Minute * (-1))},
			activeJobs: 0, succeededJobs: 10, failedJobs: 0,
			customUpdateHandler: nil,
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w)
			},
			expectedWorkflowFinished: true,
		},
		"workflow completed with failed steps": {
			deleting:   false,
			stepsCount: 10,
			startTime:  &metav1.Time{time.Now().Truncate(time.Minute * (-1))},
			activeJobs: 0, succeededJobs: 8, failedJobs: 2,
			customUpdateHandler: nil,
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w)
			},
			expectedWorkflowFinished: true,
		},
	}
	for name, tc := range testCases {
		// print test case nae
		fmt.Printf("Running '%s' test case ...\n", name)
		// workflow controller setup
		restConfig := &rest.Config{Host: "localhost"}
		workflowClient, err := wclient.NewWorkflowClient(restConfig)
		if err != nil {
			t.Fatalf("%s:%v", name, err)
		}
		kubeClient := clientset.NewForConfigOrDie(restConfig)
		controller, kubeInformerFactory, workflowInformerFactory := newWorkflowControllerFromClients(kubeClient, workflowClient)

		// workflow & jobs setup
		workflow := newWorkflow(tc.stepsCount, tc.startTime)
		key, err := cache.MetaNamespaceKeyFunc(workflow)
		if err != nil {
			t.Fatalf("%s - unable to get key from workflow:%v", name, err)
		}
		if tc.deleting {
			now := metav1.Now()
			workflow.DeletionTimestamp = &now
		}
		tweakedWorkflow := tc.workflowTweak(workflow)
		workflowInformerFactory.Workflow().V1().Workflows().Informer().GetStore().Add(tweakedWorkflow)

		jobIndexer := kubeInformerFactory.Batch().V1().Jobs().Informer().GetIndexer()
		for _, job := range newJobList(tc.activeJobs, 0, "", tweakedWorkflow) {
			jobIndexer.Add(job.DeepCopy())
		}
		for _, job := range newJobList(tc.succeededJobs, tc.activeJobs, batch.JobComplete, tweakedWorkflow) {
			jobIndexer.Add(job.DeepCopy())
		}
		for _, job := range newJobList(tc.failedJobs, tc.activeJobs+tc.succeededJobs, batch.JobFailed, tweakedWorkflow) {
			jobIndexer.Add(job.DeepCopy())
		}

		if tc.customUpdateHandler != nil {
			controller.updateHandler = tc.customUpdateHandler
		} else {
			controller.updateHandler = func(w *wapi.Workflow) error {
				// update workflow in store only
				if err := workflowInformerFactory.Workflow().V1().Workflows().Informer().GetStore().Update(w); err != nil {
					t.Errorf("%s - %v", name, err)
				}
				return nil
			}
		}

		// run sync twice: first time to update workflow steps, second to update status
		if err := controller.sync(key); err != nil {
			t.Errorf("%s - %v", name, err)
		}
		if err := controller.sync(key); err != nil {
			t.Errorf("%s - %v", name, err)
		}

		// get workflow from store
		if tweakedWorkflow, err = controller.getWorkflowByKey(key); err != nil {
			t.Errorf("%s - %v", name, err)
		}

		// validate expectations
		if IsWorkflowFinished(tweakedWorkflow) != tc.expectedWorkflowFinished {
			t.Errorf("%s - expected workflow FINISHED to be equal to '%v'", name, tc.expectedWorkflowFinished)
		}
	}
}

func TestControllerRun(t *testing.T) {
	testCases := map[string]struct {
		keys           []string
		JobSynced      cache.InformerSynced
		WorkflowSynced cache.InformerSynced
		syncHandler    func(string) error
		// expected
		expectedError        bool
		expectedErrorMessage string
	}{
		"happy run": {
			keys:           []string{"default/wf1", "default/wf2", "default/wf3", "default/last"},
			JobSynced:      func() bool { return true },
			WorkflowSynced: func() bool { return true },
		},
		"no sync for Job": {
			keys:                 []string{"default/wf1", "default/wf2", "default/wf3", "default/last"},
			JobSynced:            func() bool { return false },
			WorkflowSynced:       func() bool { return true },
			expectedError:        true,
			expectedErrorMessage: "Timed out waiting for caches to sync",
		},
		"no sync for Workflow": {
			keys:                 []string{"default/wf1", "default/wf2", "default/wf3", "default/last"},
			JobSynced:            func() bool { return true },
			WorkflowSynced:       func() bool { return false },
			expectedError:        true,
			expectedErrorMessage: "Timed out waiting for caches to sync",
		},
		"error during sync": {
			keys:                 []string{"default/wf1", "default/wf2", "default/error", "default/last"},
			JobSynced:            func() bool { return true },
			WorkflowSynced:       func() bool { return true },
			expectedError:        true,
			expectedErrorMessage: "WorkflowController.sync - Workflow default/error not valid",
		},
	}
	for name, tc := range testCases {
		restConfig := &rest.Config{Host: "localhost"}
		workflowClient, err := wclient.NewWorkflowClient(restConfig)
		if err != nil {
			t.Fatalf("%s:%v", name, err)
		}
		kubeClient := clientset.NewForConfigOrDie(restConfig)
		controller, _, _ := newWorkflowControllerFromClients(kubeClient, workflowClient)
		controller.JobSynced = tc.JobSynced
		controller.WorkflowSynced = tc.WorkflowSynced
		expectedErrorMessage := tc.expectedErrorMessage
		controller.syncHandler = func(key string) error {
			if key == "default/last" {
				controller.queue.ShutDown()
			} else if key == "default/error" {
				return fmt.Errorf(expectedErrorMessage)
			}
			return nil
		}
		for _, k := range tc.keys {
			controller.queue.Add(k)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := controller.Run(ctx); err != nil && err != context.DeadlineExceeded {
			if tc.expectedError {
				if err.Error() != expectedErrorMessage {
					t.Errorf("%s: %v", name, err)
				}
			} else {
				t.Errorf("%s: %v", name, err)
			}
		}
	}
}
