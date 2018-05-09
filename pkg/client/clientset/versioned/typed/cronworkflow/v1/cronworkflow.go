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
	v1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	scheme "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CronWorkflowsGetter has a method to return a CronWorkflowInterface.
// A group's client should implement this interface.
type CronWorkflowsGetter interface {
	CronWorkflows(namespace string) CronWorkflowInterface
}

// CronWorkflowInterface has methods to work with CronWorkflow resources.
type CronWorkflowInterface interface {
	Create(*v1.CronWorkflow) (*v1.CronWorkflow, error)
	Update(*v1.CronWorkflow) (*v1.CronWorkflow, error)
	UpdateStatus(*v1.CronWorkflow) (*v1.CronWorkflow, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.CronWorkflow, error)
	List(opts meta_v1.ListOptions) (*v1.CronWorkflowList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.CronWorkflow, err error)
	CronWorkflowExpansion
}

// cronWorkflows implements CronWorkflowInterface
type cronWorkflows struct {
	client rest.Interface
	ns     string
}

// newCronWorkflows returns a CronWorkflows
func newCronWorkflows(c *CronworkflowV1Client, namespace string) *cronWorkflows {
	return &cronWorkflows{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the cronWorkflow, and returns the corresponding cronWorkflow object, and an error if there is any.
func (c *cronWorkflows) Get(name string, options meta_v1.GetOptions) (result *v1.CronWorkflow, err error) {
	result = &v1.CronWorkflow{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("cronworkflows").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CronWorkflows that match those selectors.
func (c *cronWorkflows) List(opts meta_v1.ListOptions) (result *v1.CronWorkflowList, err error) {
	result = &v1.CronWorkflowList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("cronworkflows").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cronWorkflows.
func (c *cronWorkflows) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("cronworkflows").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a cronWorkflow and creates it.  Returns the server's representation of the cronWorkflow, and an error, if there is any.
func (c *cronWorkflows) Create(cronWorkflow *v1.CronWorkflow) (result *v1.CronWorkflow, err error) {
	result = &v1.CronWorkflow{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("cronworkflows").
		Body(cronWorkflow).
		Do().
		Into(result)
	return
}

// Update takes the representation of a cronWorkflow and updates it. Returns the server's representation of the cronWorkflow, and an error, if there is any.
func (c *cronWorkflows) Update(cronWorkflow *v1.CronWorkflow) (result *v1.CronWorkflow, err error) {
	result = &v1.CronWorkflow{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("cronworkflows").
		Name(cronWorkflow.Name).
		Body(cronWorkflow).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *cronWorkflows) UpdateStatus(cronWorkflow *v1.CronWorkflow) (result *v1.CronWorkflow, err error) {
	result = &v1.CronWorkflow{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("cronworkflows").
		Name(cronWorkflow.Name).
		SubResource("status").
		Body(cronWorkflow).
		Do().
		Into(result)
	return
}

// Delete takes name of the cronWorkflow and deletes it. Returns an error if one occurs.
func (c *cronWorkflows) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("cronworkflows").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *cronWorkflows) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("cronworkflows").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched cronWorkflow.
func (c *cronWorkflows) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.CronWorkflow, err error) {
	result = &v1.CronWorkflow{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("cronworkflows").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
