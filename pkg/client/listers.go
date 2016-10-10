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

package client

import (
	"fmt"
	"reflect"

	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	kcache "k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"

	"github.com/golang/glog"
	"github.com/sdminonne/workflow-controller/pkg/api"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	wcodec "github.com/sdminonne/workflow-controller/pkg/api/codec"
)

// StoreToWorkflowLister adds Exists and List method to cache.Store
type StoreToWorkflowLister struct {
	kcache.Store
}

// Exists checks whether StoreToWorkflowLister contains a certain Workflow.
func (s *StoreToWorkflowLister) Exists(workflow *api.Workflow) (bool, error) {
	_, exists, err := s.Store.Get(workflow)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// List lists all stored Workflows.
func (s *StoreToWorkflowLister) List() (workflows api.WorkflowList, err error) {
	for i := range s.Store.List() {
		obj, ok := s.Store.List()[i].(*runtime.Unstructured)
		if !ok {
			glog.Errorf("Expected Workflow type. Unexpected object of type: %v %#v", reflect.TypeOf(obj), obj)
			continue
		}
		w, err := wcodec.UnstructuredToWorkflow(obj)
		if err != nil {
			glog.Errorf("Unable to encode listed object: %v", err)
			continue
		}
		workflows.Items = append(workflows.Items, *w)
	}
	return workflows, nil
}

// GetJobWorkflows return the Workflows which may control the job
func (s *StoreToWorkflowLister) GetJobWorkflows(job *batch.Job) (workflows []wapi.Workflow, err error) {
	if len(job.Labels) == 0 {
		err = fmt.Errorf("no workflows found for job %v because it has no labels", job.Name)
		return
	}
	workflowList, err := s.List()
	if err != nil {
		return workflows, err
	}
	for i := range workflowList.Items {
		if workflowList.Items[i].Namespace != job.Namespace {
			continue
		}
		selector, _ := unversioned.LabelSelectorAsSelector(workflowList.Items[i].Spec.Selector)
		if selector.Matches(labels.Set(job.Labels)) {
			workflows = append(workflows, workflowList.Items[i])
		}
	}
	return
}
