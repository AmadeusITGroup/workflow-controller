package framework

import (
	"strings"

	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/pkg/apis/extensions"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

	"github.com/sdminonne/workflow-controller/pkg/client"
)

// WorkflowClient creates a Workflow client for e2e framework
func (f *Framework) workflowClient(resource *extensions.ThirdPartyResource) (client.Interface, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(f.KubeMasterURL, f.KubeConfigPath)
	if err != nil {
		return nil, err
	}
	return client.NewForConfigOrDie(resource, kubeconfig), nil
}

// WorkflowClient defines client for e2e tests
type WorkflowClient struct {
	f *Framework
	client.Interface
}

func (f *Framework) kubeClient() (clientset.Interface, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(f.KubeMasterURL, f.KubeConfigPath)
	if err != nil {
		Failf("Unable to initialize KubeClient: %v", err)
		return nil, err
	}
	kubeClient := clientset.NewForConfigOrDie(restclient.AddUserAgent(kubeconfig, "e2e-workflow"))
	return kubeClient, err
}

// BuildAndSetClients builds and initilize workflow and kube client
func BuildAndSetClients() (client.Interface, clientset.Interface) {
	f := NewFramework()
	Ω(f).ShouldNot(BeNil())
	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())
	Logf("Check wether Workflow resource is registered...")
	Eventually(func() bool {
		r, err := client.IsWorkflowRegistered(kubeClient, f.ResourceName, f.ResourceGroup, f.ResourceVersion)
		if err != nil {
			Logf("Error: %v", err)
		}
		return (r && err == nil)
	}, "5s", "1s").Should(BeTrue())
	Logf("It is!")
	resourceName := strings.Join([]string{f.ResourceName, f.ResourceGroup}, ".")
	thirdPartyResource, err := kubeClient.Extensions().ThirdPartyResources().Get(resourceName)
	workflowClient, err := f.workflowClient(thirdPartyResource)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(workflowClient).ShouldNot(BeNil())
	return workflowClient, kubeClient
}
