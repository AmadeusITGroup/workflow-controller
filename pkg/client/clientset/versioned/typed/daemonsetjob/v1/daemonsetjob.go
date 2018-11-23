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
	v1 "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	scheme "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// DaemonSetJobsGetter has a method to return a DaemonSetJobInterface.
// A group's client should implement this interface.
type DaemonSetJobsGetter interface {
	DaemonSetJobs(namespace string) DaemonSetJobInterface
}

// DaemonSetJobInterface has methods to work with DaemonSetJob resources.
type DaemonSetJobInterface interface {
	Create(*v1.DaemonSetJob) (*v1.DaemonSetJob, error)
	Update(*v1.DaemonSetJob) (*v1.DaemonSetJob, error)
	UpdateStatus(*v1.DaemonSetJob) (*v1.DaemonSetJob, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.DaemonSetJob, error)
	List(opts meta_v1.ListOptions) (*v1.DaemonSetJobList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.DaemonSetJob, err error)
	DaemonSetJobExpansion
}

// daemonSetJobs implements DaemonSetJobInterface
type daemonSetJobs struct {
	client rest.Interface
	ns     string
}

// newDaemonSetJobs returns a DaemonSetJobs
func newDaemonSetJobs(c *DaemonsetjobV1Client, namespace string) *daemonSetJobs {
	return &daemonSetJobs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the daemonSetJob, and returns the corresponding daemonSetJob object, and an error if there is any.
func (c *daemonSetJobs) Get(name string, options meta_v1.GetOptions) (result *v1.DaemonSetJob, err error) {
	result = &v1.DaemonSetJob{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of DaemonSetJobs that match those selectors.
func (c *daemonSetJobs) List(opts meta_v1.ListOptions) (result *v1.DaemonSetJobList, err error) {
	result = &v1.DaemonSetJobList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested daemonSetJobs.
func (c *daemonSetJobs) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a daemonSetJob and creates it.  Returns the server's representation of the daemonSetJob, and an error, if there is any.
func (c *daemonSetJobs) Create(daemonSetJob *v1.DaemonSetJob) (result *v1.DaemonSetJob, err error) {
	result = &v1.DaemonSetJob{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		Body(daemonSetJob).
		Do().
		Into(result)
	return
}

// Update takes the representation of a daemonSetJob and updates it. Returns the server's representation of the daemonSetJob, and an error, if there is any.
func (c *daemonSetJobs) Update(daemonSetJob *v1.DaemonSetJob) (result *v1.DaemonSetJob, err error) {
	result = &v1.DaemonSetJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		Name(daemonSetJob.Name).
		Body(daemonSetJob).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *daemonSetJobs) UpdateStatus(daemonSetJob *v1.DaemonSetJob) (result *v1.DaemonSetJob, err error) {
	result = &v1.DaemonSetJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		Name(daemonSetJob.Name).
		SubResource("status").
		Body(daemonSetJob).
		Do().
		Into(result)
	return
}

// Delete takes name of the daemonSetJob and deletes it. Returns an error if one occurs.
func (c *daemonSetJobs) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *daemonSetJobs) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("daemonsetjobs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched daemonSetJob.
func (c *daemonSetJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.DaemonSetJob, err error) {
	result = &v1.DaemonSetJob{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("daemonsetjobs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
