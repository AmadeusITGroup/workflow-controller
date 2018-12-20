package controller

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/golang/glog"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"

	dapi "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
)

// JobControlInterface defines interface to control Jobs life-cycle.
// Interface is needed to mock it in unit tests
type JobControlInterface interface {
	// CreateJobFromWorkflow
	CreateJobFromWorkflow(namespace string, template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, key string) (*batch.Job, error)
	// CreateJobFromDaemonSetJob
	CreateJobFromDaemonSetJob(namespace string, template *batchv2.JobTemplateSpec, daemonsetjob *dapi.DaemonSetJob, nodeName string) (*batch.Job, error)
	// DeleteJob
	DeleteJob(namespace, name string, object runtime.Object) error
}

// WorkflowJobControl implements JobControlInterface
type WorkflowJobControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

var _ JobControlInterface = &WorkflowJobControl{}

// CreateJobFromWorkflow creates a Job According to a specific JobTemplateSpec
func (w WorkflowJobControl) CreateJobFromWorkflow(namespace string, template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, stepName string) (*batch.Job, error) {
	labels, err := getJobLabelsSetFromWorkflow(workflow, template, stepName)
	if err != nil {
		return nil, err
	}
	owner := buildOwnerReferenceForDaemonsetJob(&workflow.ObjectMeta)
	return w.CreateJob(namespace, stepName, &workflow.ObjectMeta, template, &labels, &owner)
}

// CreateJobFromDaemonSetJob creates a Job According to a specific JobTemplateSpec
func (w WorkflowJobControl) CreateJobFromDaemonSetJob(namespace string, template *batchv2.JobTemplateSpec, daemonsetjob *dapi.DaemonSetJob, nodeName string) (*batch.Job, error) {
	labels, err := getJobLabelsSetFromDaemonSetJob(daemonsetjob, template)
	if err != nil {
		return nil, err
	}
	templateCopy := template.DeepCopy()
	templateCopy.Spec.Template.Spec.NodeName = nodeName

	// Generate a MD5 representing the JobTemplateSpec send
	hash, err := generateMD5JobSpec(&daemonsetjob.Spec.JobTemplate.Spec)
	if err != nil {
		return nil, fmt.Errorf("unable to generates the JobSpec MD5, %v", err)
	}
	objMeta := daemonsetjob.ObjectMeta.DeepCopy()
	if objMeta.Annotations == nil {
		objMeta.Annotations = map[string]string{}
	}
	objMeta.Annotations[DaemonSetJobMD5AnnotationKey] = hash

	owner := buildOwnerReferenceForDaemonsetJob(&daemonsetjob.ObjectMeta)
	job, err := w.CreateJob(namespace, nodeName, objMeta, templateCopy, &labels, &owner)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// CreateJob creates a Job According to a specific JobTemplateSpec
func (w WorkflowJobControl) CreateJob(namespace string, subName string, obj *metav1.ObjectMeta, template *batchv2.JobTemplateSpec, labelsset *labels.Set, owner *metav1.OwnerReference) (*batch.Job, error) {
	job, err := initJob(template, obj, subName, labelsset, owner)
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

	if err := w.KubeClient.Batch().Jobs(namespace).Delete(jobName, CascadeDeleteOptions(0)); err != nil {
		w.Recorder.Eventf(object, api.EventTypeWarning, "FailedDelete", "Error deleting: %v", err)
		return fmt.Errorf("unable to delete job: %v", err)
	}

	glog.V(6).Infof("Controller %v deleted job %v", accessor.GetName(), jobName)
	w.Recorder.Eventf(object, api.EventTypeNormal, "SuccessfulDelete", "Deleted job: %v", jobName)
	return nil
}

func initJob(template *batchv2.JobTemplateSpec, obj *metav1.ObjectMeta, subName string, labelsset *labels.Set, owner *metav1.OwnerReference) (*batch.Job, error) {
	jobGeneratedName := fmt.Sprintf("wfl-%s-%s-", obj.Name, subName)
	desiredAnnotations, err := getJobAnnotationsSet(obj, template)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize Job %s/%s: %v", obj.Namespace, subName, err)
	}

	job := &batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:       *labelsset,
			Annotations:  desiredAnnotations,
			GenerateName: jobGeneratedName, // TODO: double check prefix
		},
	}

	job.ObjectMeta.OwnerReferences = append(job.ObjectMeta.OwnerReferences, *owner)
	job.Spec = *template.Spec.DeepCopy()

	return job, nil
}

// FakeJobControl implements WorkflowJobControl interface for testing purpose
type FakeJobControl struct {
}

// CreateJobFromWorkflow mocks job creation
func (f *FakeJobControl) CreateJobFromWorkflow(namespace string, template *batchv2.JobTemplateSpec, workflow *wapi.Workflow, key string) (*batch.Job, error) {
	return nil, nil
}

// CreateJobFromDaemonSetJob mocks job creation
func (f *FakeJobControl) CreateJobFromDaemonSetJob(namespace string, template *batchv2.JobTemplateSpec, workflow *dapi.DaemonSetJob, nodeName string) (*batch.Job, error) {
	return nil, nil
}

// DeleteJob mocks job deletion
func (f *FakeJobControl) DeleteJob(namespace, name string, object runtime.Object) error {
	return nil
}

// generateMD5JobSpec used to generate the PodSpec MD5 hash
func generateMD5JobSpec(spec *batch.JobSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}
	hash := md5.New()
	io.Copy(hash, bytes.NewReader(b))
	return hex.EncodeToString(hash.Sum(nil)), nil
}
