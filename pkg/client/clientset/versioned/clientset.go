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
package versioned

import (
	cronworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/cronworkflow/v1"
	daemonsetjobv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/daemonsetjob/v1"
	workflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/workflow/v1"
	glog "github.com/golang/glog"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	CronworkflowV1() cronworkflowv1.CronworkflowV1Interface
	// Deprecated: please explicitly pick a version if possible.
	Cronworkflow() cronworkflowv1.CronworkflowV1Interface
	DaemonsetjobV1() daemonsetjobv1.DaemonsetjobV1Interface
	// Deprecated: please explicitly pick a version if possible.
	Daemonsetjob() daemonsetjobv1.DaemonsetjobV1Interface
	WorkflowV1() workflowv1.WorkflowV1Interface
	// Deprecated: please explicitly pick a version if possible.
	Workflow() workflowv1.WorkflowV1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	cronworkflowV1 *cronworkflowv1.CronworkflowV1Client
	daemonsetjobV1 *daemonsetjobv1.DaemonsetjobV1Client
	workflowV1     *workflowv1.WorkflowV1Client
}

// CronworkflowV1 retrieves the CronworkflowV1Client
func (c *Clientset) CronworkflowV1() cronworkflowv1.CronworkflowV1Interface {
	return c.cronworkflowV1
}

// Deprecated: Cronworkflow retrieves the default version of CronworkflowClient.
// Please explicitly pick a version.
func (c *Clientset) Cronworkflow() cronworkflowv1.CronworkflowV1Interface {
	return c.cronworkflowV1
}

// DaemonsetjobV1 retrieves the DaemonsetjobV1Client
func (c *Clientset) DaemonsetjobV1() daemonsetjobv1.DaemonsetjobV1Interface {
	return c.daemonsetjobV1
}

// Deprecated: Daemonsetjob retrieves the default version of DaemonsetjobClient.
// Please explicitly pick a version.
func (c *Clientset) Daemonsetjob() daemonsetjobv1.DaemonsetjobV1Interface {
	return c.daemonsetjobV1
}

// WorkflowV1 retrieves the WorkflowV1Client
func (c *Clientset) WorkflowV1() workflowv1.WorkflowV1Interface {
	return c.workflowV1
}

// Deprecated: Workflow retrieves the default version of WorkflowClient.
// Please explicitly pick a version.
func (c *Clientset) Workflow() workflowv1.WorkflowV1Interface {
	return c.workflowV1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.cronworkflowV1, err = cronworkflowv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.daemonsetjobV1, err = daemonsetjobv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.workflowV1, err = workflowv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the DiscoveryClient: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.cronworkflowV1 = cronworkflowv1.NewForConfigOrDie(c)
	cs.daemonsetjobV1 = daemonsetjobv1.NewForConfigOrDie(c)
	cs.workflowV1 = workflowv1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.cronworkflowV1 = cronworkflowv1.New(c)
	cs.daemonsetjobV1 = daemonsetjobv1.New(c)
	cs.workflowV1 = workflowv1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
