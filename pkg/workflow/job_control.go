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
	"sync"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/apis/batch"

	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/record"

	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
)

// JobControlInterface defines interface for JobControl
type JobControlInterface interface {
	// CreateJob
	CreateJob(namespace string, template *batch.JobTemplateSpec, object runtime.Object, key string) error
	// DeleteJob
	DeleteJob(namespace, name string, object runtime.Object) error
}

// WorkflowJobControl implements JobControlInterface
type WorkflowJobControl struct {
	KubeClient clientset.Interface
	Recorder   record.EventRecorder
}

var _ JobControlInterface = &WorkflowJobControl{}

func getJobsPrefix(controllerName string) string {
	prefix := fmt.Sprintf("%s-", controllerName)
	if errs := validation.ValidateReplicationControllerName(prefix, true); len(errs) > 0 {
		prefix = controllerName
	}
	return prefix
}

func getJobsAnnotationSet(template *batch.JobTemplateSpec, object runtime.Object) (labels.Set, error) {
	workflow := *object.(*wapi.Workflow)
	desiredAnnotations := make(labels.Set)
	for k, v := range workflow.Annotations {
		desiredAnnotations[k] = v
	}
	createdByRef, err := api.GetReference(object)
	if err != nil {
		return desiredAnnotations, fmt.Errorf("unable to get controller reference: %v", err)
	}

	//TODO: codec  hardcoded to v1 for the moment.
	codec := api.Codecs.LegacyCodec(unversioned.GroupVersion{Group: api.GroupName, Version: "v1"})

	createdByRefJSON, err := runtime.Encode(codec, &api.SerializedReference{
		Reference: *createdByRef,
	})
	if err != nil {
		return desiredAnnotations, fmt.Errorf("unable to serialize controller reference: %v", err)
	}
	desiredAnnotations[api.CreatedByAnnotation] = string(createdByRefJSON)
	return desiredAnnotations, nil
}

func getWorkflowJobLabelSet(workflow *wapi.Workflow, template *batch.JobTemplateSpec, stepName string) labels.Set {
	desiredLabels := make(labels.Set)
	for k, v := range workflow.Labels {
		desiredLabels[k] = v
	}
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}
	desiredLabels[WorkflowStepLabelKey] = stepName // @sdminonne: TODO double check this
	return desiredLabels
}

// CreateWorkflowJobLabelSelector creates job label selector
func CreateWorkflowJobLabelSelector(workflow *wapi.Workflow, template *batch.JobTemplateSpec, stepName string) labels.Selector {
	return labels.SelectorFromSet(getWorkflowJobLabelSet(workflow, template, stepName))
}

// CreateJob creates a Job According to a specific JobTemplateSpec
func (w WorkflowJobControl) CreateJob(namespace string, template *batch.JobTemplateSpec, object runtime.Object, stepName string) error {
	workflow := object.(*wapi.Workflow)
	desiredLabels := getWorkflowJobLabelSet(workflow, template, stepName)
	desiredAnnotations, err := getJobsAnnotationSet(template, object)
	if err != nil {
		return err
	}
	meta, err := api.ObjectMetaFor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}
	prefix := getJobsPrefix(meta.Name)
	job := &batch.Job{
		ObjectMeta: api.ObjectMeta{
			Labels:       desiredLabels,
			Annotations:  desiredAnnotations,
			GenerateName: prefix,
		},
	}

	if err := api.Scheme.Convert(&(template.Spec), &job.Spec, nil); err != nil {
		return fmt.Errorf("unable to convert job template: %v", err)
	}

	newJob, err := w.KubeClient.Batch().Jobs(namespace).Create(job)
	if err != nil {
		w.Recorder.Eventf(object, api.EventTypeWarning, "FailedCreate", "Error creating: %v", err)
		return fmt.Errorf("unable to create job: %v", err)
	}
	glog.V(4).Infof("Controller %v created job %v", meta.Name, newJob.Name)
	return nil
}

// DeleteJob deletes a Job
func (w WorkflowJobControl) DeleteJob(namespace, jobName string, object runtime.Object) error {
	accessor, err := meta.Accessor(object)
	if err != nil {
		return fmt.Errorf("object does not have ObjectMeta, %v", err)
	}

	if err := w.KubeClient.Batch().Jobs(namespace).Delete(jobName, nil); err != nil {
		w.Recorder.Eventf(object, api.EventTypeWarning, "FailedDelete", "Error deleting: %v", err)
		return fmt.Errorf("unable to delete job: %v", err)
	}

	glog.V(4).Infof("Controller %v deleted job %v", accessor.GetName(), jobName)
	w.Recorder.Eventf(object, api.EventTypeNormal, "SuccessfulDelete", "Deleted job: %v", jobName)

	return nil
}

// FakeJobControl implements data structure to mock JobControl
type FakeJobControl struct {
	sync.Mutex
	CreatedJobTemplates []batch.JobTemplateSpec
	DeletedJobNames     []string
	Err                 error
}

// FakeJobControl implements JobControlInterface
var _ JobControlInterface = &FakeJobControl{}

// CreateJob add JobTemplateSpec to the list of the CreatedJobTemplates
func (f *FakeJobControl) CreateJob(namespace string, template *batch.JobTemplateSpec, object runtime.Object, key string) error {
	f.Lock()
	defer f.Unlock()
	f.CreatedJobTemplates = append(f.CreatedJobTemplates, *template)
	return nil
}

// DeleteJob add Job name to DeletedJobNames
func (f *FakeJobControl) DeleteJob(namespace, name string, object runtime.Object) error {
	f.Lock()
	defer f.Unlock()
	if f.Err != nil {
		return f.Err
	}
	f.DeletedJobNames = append(f.DeletedJobNames, name)
	return nil
}
