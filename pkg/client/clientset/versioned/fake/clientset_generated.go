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
	clientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	cronworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/cronworkflow/v1"
	fakecronworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/cronworkflow/v1/fake"
	daemonsetjobv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/daemonsetjob/v1"
	fakedaemonsetjobv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/daemonsetjob/v1/fake"
	workflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/workflow/v1"
	fakeworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/workflow/v1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	fakePtr := testing.Fake{}
	fakePtr.AddReactor("*", "*", testing.ObjectReaction(o))
	fakePtr.AddWatchReactor("*", testing.DefaultWatchReactor(watch.NewFake(), nil))

	return &Clientset{fakePtr, &fakediscovery.FakeDiscovery{Fake: &fakePtr}}
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ clientset.Interface = &Clientset{}

// CronworkflowV1 retrieves the CronworkflowV1Client
func (c *Clientset) CronworkflowV1() cronworkflowv1.CronworkflowV1Interface {
	return &fakecronworkflowv1.FakeCronworkflowV1{Fake: &c.Fake}
}

// Cronworkflow retrieves the CronworkflowV1Client
func (c *Clientset) Cronworkflow() cronworkflowv1.CronworkflowV1Interface {
	return &fakecronworkflowv1.FakeCronworkflowV1{Fake: &c.Fake}
}

// DaemonsetjobV1 retrieves the DaemonsetjobV1Client
func (c *Clientset) DaemonsetjobV1() daemonsetjobv1.DaemonsetjobV1Interface {
	return &fakedaemonsetjobv1.FakeDaemonsetjobV1{Fake: &c.Fake}
}

// Daemonsetjob retrieves the DaemonsetjobV1Client
func (c *Clientset) Daemonsetjob() daemonsetjobv1.DaemonsetjobV1Interface {
	return &fakedaemonsetjobv1.FakeDaemonsetjobV1{Fake: &c.Fake}
}

// WorkflowV1 retrieves the WorkflowV1Client
func (c *Clientset) WorkflowV1() workflowv1.WorkflowV1Interface {
	return &fakeworkflowv1.FakeWorkflowV1{Fake: &c.Fake}
}

// Workflow retrieves the WorkflowV1Client
func (c *Clientset) Workflow() workflowv1.WorkflowV1Interface {
	return &fakeworkflowv1.FakeWorkflowV1{Fake: &c.Fake}
}
