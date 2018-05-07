/*
   Copyright 2018 Amadeus SAS

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

// Generated file, do not modify manually!
package fake

import (
	workflow_v1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeWorkflows implements WorkflowInterface
type FakeWorkflows struct {
	Fake *FakeWorkflowV1
	ns   string
}

var workflowsResource = schema.GroupVersionResource{Group: "workflow", Version: "v1", Resource: "workflows"}

var workflowsKind = schema.GroupVersionKind{Group: "workflow", Version: "v1", Kind: "Workflow"}

// Get takes name of the workflow, and returns the corresponding workflow object, and an error if there is any.
func (c *FakeWorkflows) Get(name string, options v1.GetOptions) (result *workflow_v1.Workflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(workflowsResource, c.ns, name), &workflow_v1.Workflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*workflow_v1.Workflow), err
}

// List takes label and field selectors, and returns the list of Workflows that match those selectors.
func (c *FakeWorkflows) List(opts v1.ListOptions) (result *workflow_v1.WorkflowList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(workflowsResource, workflowsKind, c.ns, opts), &workflow_v1.WorkflowList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &workflow_v1.WorkflowList{}
	for _, item := range obj.(*workflow_v1.WorkflowList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested workflows.
func (c *FakeWorkflows) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(workflowsResource, c.ns, opts))

}

// Create takes the representation of a workflow and creates it.  Returns the server's representation of the workflow, and an error, if there is any.
func (c *FakeWorkflows) Create(workflow *workflow_v1.Workflow) (result *workflow_v1.Workflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(workflowsResource, c.ns, workflow), &workflow_v1.Workflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*workflow_v1.Workflow), err
}

// Update takes the representation of a workflow and updates it. Returns the server's representation of the workflow, and an error, if there is any.
func (c *FakeWorkflows) Update(workflow *workflow_v1.Workflow) (result *workflow_v1.Workflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(workflowsResource, c.ns, workflow), &workflow_v1.Workflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*workflow_v1.Workflow), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeWorkflows) UpdateStatus(workflow *workflow_v1.Workflow) (*workflow_v1.Workflow, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(workflowsResource, "status", c.ns, workflow), &workflow_v1.Workflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*workflow_v1.Workflow), err
}

// Delete takes name of the workflow and deletes it. Returns an error if one occurs.
func (c *FakeWorkflows) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(workflowsResource, c.ns, name), &workflow_v1.Workflow{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeWorkflows) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(workflowsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &workflow_v1.WorkflowList{})
	return err
}

// Patch applies the patch and returns the patched workflow.
func (c *FakeWorkflows) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *workflow_v1.Workflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(workflowsResource, c.ns, name, data, subresources...), &workflow_v1.Workflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*workflow_v1.Workflow), err
}
