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
package v1

import (
	v1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	scheme "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// WorkflowsGetter has a method to return a WorkflowInterface.
// A group's client should implement this interface.
type WorkflowsGetter interface {
	Workflows(namespace string) WorkflowInterface
}

// WorkflowInterface has methods to work with Workflow resources.
type WorkflowInterface interface {
	Create(*v1.Workflow) (*v1.Workflow, error)
	Update(*v1.Workflow) (*v1.Workflow, error)
	UpdateStatus(*v1.Workflow) (*v1.Workflow, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.Workflow, error)
	List(opts meta_v1.ListOptions) (*v1.WorkflowList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Workflow, err error)
	WorkflowExpansion
}

// workflows implements WorkflowInterface
type workflows struct {
	client rest.Interface
	ns     string
}

// newWorkflows returns a Workflows
func newWorkflows(c *WorkflowV1Client, namespace string) *workflows {
	return &workflows{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the workflow, and returns the corresponding workflow object, and an error if there is any.
func (c *workflows) Get(name string, options meta_v1.GetOptions) (result *v1.Workflow, err error) {
	result = &v1.Workflow{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("workflows").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Workflows that match those selectors.
func (c *workflows) List(opts meta_v1.ListOptions) (result *v1.WorkflowList, err error) {
	result = &v1.WorkflowList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("workflows").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested workflows.
func (c *workflows) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("workflows").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a workflow and creates it.  Returns the server's representation of the workflow, and an error, if there is any.
func (c *workflows) Create(workflow *v1.Workflow) (result *v1.Workflow, err error) {
	result = &v1.Workflow{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("workflows").
		Body(workflow).
		Do().
		Into(result)
	return
}

// Update takes the representation of a workflow and updates it. Returns the server's representation of the workflow, and an error, if there is any.
func (c *workflows) Update(workflow *v1.Workflow) (result *v1.Workflow, err error) {
	result = &v1.Workflow{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("workflows").
		Name(workflow.Name).
		Body(workflow).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *workflows) UpdateStatus(workflow *v1.Workflow) (result *v1.Workflow, err error) {
	result = &v1.Workflow{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("workflows").
		Name(workflow.Name).
		SubResource("status").
		Body(workflow).
		Do().
		Into(result)
	return
}

// Delete takes name of the workflow and deletes it. Returns an error if one occurs.
func (c *workflows) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("workflows").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *workflows) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("workflows").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched workflow.
func (c *workflows) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.Workflow, err error) {
	result = &v1.Workflow{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("workflows").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
