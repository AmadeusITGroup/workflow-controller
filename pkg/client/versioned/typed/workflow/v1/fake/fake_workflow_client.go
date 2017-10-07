// Generated file, do not modify manually!
package fake

import (
	v1 "github.com/sdminonne/workflow-controller/pkg/client/versioned/typed/workflow/v1"
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
