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
	v1 "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/typed/workflow/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeWorkflowV1 struct {
	*testing.Fake
}

func (c *FakeWorkflowV1) Workflows(namespace string) v1.WorkflowInterface {
	return &FakeWorkflows{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeWorkflowV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}