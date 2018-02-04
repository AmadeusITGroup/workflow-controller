// Generated file, do not modify manually!
package fake

import (
	v1 "github.com/amadeusitgroup/workflow-controller/pkg/client/versioned/typed/cronworkflow/v1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeCronworkflowV1 struct {
	*testing.Fake
}

func (c *FakeCronworkflowV1) CronWorkflows(namespace string) v1.CronWorkflowInterface {
	return &FakeCronWorkflows{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCronworkflowV1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
