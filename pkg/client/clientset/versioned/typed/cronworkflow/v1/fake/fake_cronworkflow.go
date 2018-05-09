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
	cronworkflow_v1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCronWorkflows implements CronWorkflowInterface
type FakeCronWorkflows struct {
	Fake *FakeCronworkflowV1
	ns   string
}

var cronworkflowsResource = schema.GroupVersionResource{Group: "cronworkflow", Version: "v1", Resource: "cronworkflows"}

var cronworkflowsKind = schema.GroupVersionKind{Group: "cronworkflow", Version: "v1", Kind: "CronWorkflow"}

// Get takes name of the cronWorkflow, and returns the corresponding cronWorkflow object, and an error if there is any.
func (c *FakeCronWorkflows) Get(name string, options v1.GetOptions) (result *cronworkflow_v1.CronWorkflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(cronworkflowsResource, c.ns, name), &cronworkflow_v1.CronWorkflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronworkflow_v1.CronWorkflow), err
}

// List takes label and field selectors, and returns the list of CronWorkflows that match those selectors.
func (c *FakeCronWorkflows) List(opts v1.ListOptions) (result *cronworkflow_v1.CronWorkflowList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(cronworkflowsResource, cronworkflowsKind, c.ns, opts), &cronworkflow_v1.CronWorkflowList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &cronworkflow_v1.CronWorkflowList{}
	for _, item := range obj.(*cronworkflow_v1.CronWorkflowList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cronWorkflows.
func (c *FakeCronWorkflows) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(cronworkflowsResource, c.ns, opts))

}

// Create takes the representation of a cronWorkflow and creates it.  Returns the server's representation of the cronWorkflow, and an error, if there is any.
func (c *FakeCronWorkflows) Create(cronWorkflow *cronworkflow_v1.CronWorkflow) (result *cronworkflow_v1.CronWorkflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(cronworkflowsResource, c.ns, cronWorkflow), &cronworkflow_v1.CronWorkflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronworkflow_v1.CronWorkflow), err
}

// Update takes the representation of a cronWorkflow and updates it. Returns the server's representation of the cronWorkflow, and an error, if there is any.
func (c *FakeCronWorkflows) Update(cronWorkflow *cronworkflow_v1.CronWorkflow) (result *cronworkflow_v1.CronWorkflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(cronworkflowsResource, c.ns, cronWorkflow), &cronworkflow_v1.CronWorkflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronworkflow_v1.CronWorkflow), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeCronWorkflows) UpdateStatus(cronWorkflow *cronworkflow_v1.CronWorkflow) (*cronworkflow_v1.CronWorkflow, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(cronworkflowsResource, "status", c.ns, cronWorkflow), &cronworkflow_v1.CronWorkflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronworkflow_v1.CronWorkflow), err
}

// Delete takes name of the cronWorkflow and deletes it. Returns an error if one occurs.
func (c *FakeCronWorkflows) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(cronworkflowsResource, c.ns, name), &cronworkflow_v1.CronWorkflow{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCronWorkflows) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(cronworkflowsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &cronworkflow_v1.CronWorkflowList{})
	return err
}

// Patch applies the patch and returns the patched cronWorkflow.
func (c *FakeCronWorkflows) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *cronworkflow_v1.CronWorkflow, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(cronworkflowsResource, c.ns, name, data, subresources...), &cronworkflow_v1.CronWorkflow{})

	if obj == nil {
		return nil, err
	}
	return obj.(*cronworkflow_v1.CronWorkflow), err
}
