package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/pkg/api"

	"github.com/sdminonne/workflow-controller/pkg/workflow"
	"github.com/sdminonne/workflow-controller/test/e2e/framework"
)

var _ = Describe("Workflow CRUD", func() {

	It("should create a workflow", func() {
		workflowClient, _ := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow(framework.FrameworkContext.ResourceGroup, framework.FrameworkContext.ResourceVersion, "hello-workflow", ns)
		Eventually(func() error {
			_, err := workflowClient.Workflows(ns).Create(myWorkflow)
			return err
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")
		defer func() {
			workflowClient.Workflows(ns).Delete(myWorkflow.Name, nil)
			By("Workflow deleted")
		}()

		Eventually(func() error {
			curr, err := workflowClient.Workflows(ns).Get(myWorkflow.Name)
			if err != nil {
				return err
			}
			if workflow.IsWorkflowFinished(curr) {
				return nil
			}
			framework.Logf("Workflow not yet finished %v", curr)
			return fmt.Errorf("workflow %s not finished", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())

	})
