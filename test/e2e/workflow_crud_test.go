package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/pkg/api"

	wapitesting "github.com/sdminonne/workflow-controller/pkg/api/testing"
	"github.com/sdminonne/workflow-controller/test/e2e/framework"
)

var _ = Describe("Workflowc CRUD", func() {

	It("should create a workflow", func() {
		workflowClient, _ := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := wapitesting.NewWorkflow(framework.FrameworkContext.ResourceGroup, framework.FrameworkContext.ResourceVersion, "mydag", ns, nil)
		Eventually(func() error {
			_, err := workflowClient.Workflows(ns).Create(myWorkflow)
			return err
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")
		defer func() {
			workflowClient.Workflows(ns).Delete(myWorkflow.Name, nil)
			By("Workflow deleted")
		}()
	})
	It("should fail to create a workflow", func() {
		//workflowClient, kubeClient := framework.BuildAndSetClients()
		//ns := api.NamespaceDefault
		//myWorkflow := wapitesting.NewWorkflow(framework.FrameworkContext.ResourceGroup, framework.FrameworkContext.ResourceVersion, "mydag", ns, nil)
	})
	It("should check defaulting", func() {
		//workflowClient, kubeClient := framework.BuildAndSetClients()
		//fmt.Printf("%v %v\n", workflowClient, kubeClient)
		//fmt.Printf("Three!!")
	})
})
