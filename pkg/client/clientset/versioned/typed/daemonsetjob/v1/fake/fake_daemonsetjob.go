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
	daemonsetjob_v1 "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDaemonSetJobs implements DaemonSetJobInterface
type FakeDaemonSetJobs struct {
	Fake *FakeDaemonsetjobV1
	ns   string
}

var daemonsetjobsResource = schema.GroupVersionResource{Group: "daemonsetjob", Version: "v1", Resource: "daemonsetjobs"}

var daemonsetjobsKind = schema.GroupVersionKind{Group: "daemonsetjob", Version: "v1", Kind: "DaemonSetJob"}

// Get takes name of the daemonSetJob, and returns the corresponding daemonSetJob object, and an error if there is any.
func (c *FakeDaemonSetJobs) Get(name string, options v1.GetOptions) (result *daemonsetjob_v1.DaemonSetJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(daemonsetjobsResource, c.ns, name), &daemonsetjob_v1.DaemonSetJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*daemonsetjob_v1.DaemonSetJob), err
}

// List takes label and field selectors, and returns the list of DaemonSetJobs that match those selectors.
func (c *FakeDaemonSetJobs) List(opts v1.ListOptions) (result *daemonsetjob_v1.DaemonSetJobList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(daemonsetjobsResource, daemonsetjobsKind, c.ns, opts), &daemonsetjob_v1.DaemonSetJobList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &daemonsetjob_v1.DaemonSetJobList{}
	for _, item := range obj.(*daemonsetjob_v1.DaemonSetJobList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested daemonSetJobs.
func (c *FakeDaemonSetJobs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(daemonsetjobsResource, c.ns, opts))

}

// Create takes the representation of a daemonSetJob and creates it.  Returns the server's representation of the daemonSetJob, and an error, if there is any.
func (c *FakeDaemonSetJobs) Create(daemonSetJob *daemonsetjob_v1.DaemonSetJob) (result *daemonsetjob_v1.DaemonSetJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(daemonsetjobsResource, c.ns, daemonSetJob), &daemonsetjob_v1.DaemonSetJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*daemonsetjob_v1.DaemonSetJob), err
}

// Update takes the representation of a daemonSetJob and updates it. Returns the server's representation of the daemonSetJob, and an error, if there is any.
func (c *FakeDaemonSetJobs) Update(daemonSetJob *daemonsetjob_v1.DaemonSetJob) (result *daemonsetjob_v1.DaemonSetJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(daemonsetjobsResource, c.ns, daemonSetJob), &daemonsetjob_v1.DaemonSetJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*daemonsetjob_v1.DaemonSetJob), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeDaemonSetJobs) UpdateStatus(daemonSetJob *daemonsetjob_v1.DaemonSetJob) (*daemonsetjob_v1.DaemonSetJob, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(daemonsetjobsResource, "status", c.ns, daemonSetJob), &daemonsetjob_v1.DaemonSetJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*daemonsetjob_v1.DaemonSetJob), err
}

// Delete takes name of the daemonSetJob and deletes it. Returns an error if one occurs.
func (c *FakeDaemonSetJobs) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(daemonsetjobsResource, c.ns, name), &daemonsetjob_v1.DaemonSetJob{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDaemonSetJobs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(daemonsetjobsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &daemonsetjob_v1.DaemonSetJobList{})
	return err
}

// Patch applies the patch and returns the patched daemonSetJob.
func (c *FakeDaemonSetJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *daemonsetjob_v1.DaemonSetJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(daemonsetjobsResource, c.ns, name, data, subresources...), &daemonsetjob_v1.DaemonSetJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*daemonsetjob_v1.DaemonSetJob), err
}
