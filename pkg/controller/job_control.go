package controller

import (
	"fmt"

	"github.com/golang/glog"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	wapi "github.com/sdminonne/workflow-controller/pkg/api/workflow/v1"
)

// JobControlInterface defines interface to control Jobs life-cycle.
// Interface is needed to mock it in unit tests
type JobControlInterface interface {
	// CreateJob
	CreateJob(namespace string, template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, key string) (*batch.Job, error)
	// DeleteJob
	DeleteJob(namespace, name string, object runtime.Object) error
}

// WorkflowJobControl implements JobControlInterface
type WorkflowJobControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

var _ JobControlInterface = &WorkflowJobControl{}

// CreateJob creates a Job According to a specific JobTemplateSpec
func (w WorkflowJobControl) CreateJob(namespace string, template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, stepName string) (*batch.Job, error) {
	job, err := initJob(template, workflow, stepName)
	if err != nil {
		return nil, fmt.Errorf("cannot create job: %v", err)
	}
	return w.KubeClient.BatchV1().Jobs(namespace).Create(job)
}

// DeleteJob deletes a Job
func (w WorkflowJobControl) DeleteJob(namespace, jobName string, object runtime.Object) error {
	accessor, err := meta.Accessor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}

	if err := w.KubeClient.Batch().Jobs(namespace).Delete(jobName, cascadeDeleteOptions(0)); err != nil {
		w.Recorder.Eventf(object, api.EventTypeWarning, "FailedDelete", "Error deleting: %v", err)
		return fmt.Errorf("unable to delete job: %v", err)
	}

	glog.V(6).Infof("Controller %v deleted job %v", accessor.GetName(), jobName)
	w.Recorder.Eventf(object, api.EventTypeNormal, "SuccessfulDelete", "Deleted job: %v", jobName)
	return nil
}

func initJob(template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, stepName string) (*batch.Job, error) {
	desiredLabels, err := getJobLabelsSet(workflow, template, stepName)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Job for Workflow %s/%s step %s: %v", workflow.Namespace, workflow.Name, stepName, err)
	}
	desiredAnnotations, err := getJobAnnotationsSet(workflow, template)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Job for Workflow %s/%s step %s: %v", workflow.Namespace, workflow.Name, stepName, err)
	}
	jobGeneratedName := fmt.Sprintf("wfl-%s-%s-", workflow.Name, stepName)
	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       desiredLabels,
			Annotations:  desiredAnnotations,
			GenerateName: jobGeneratedName, // TODO: double check prefix
		},
	}
	job.Spec = *template.Spec.DeepCopy()
	return job, nil
}

// FakeJobControl implements WorkflowJobControl interface for testing purpose
type FakeJobControl struct {
}

// CreateJob mocks job creation
func (f *FakeJobControl) CreateJob(namespace string, template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, key string) (*batch.Job, error) {
	return nil, nil
}

// DeleteJob mocks job deletion
func (f *FakeJobControl) DeleteJob(namespace, name string, object runtime.Object) error {
	return nil
}
