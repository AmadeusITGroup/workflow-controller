package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/pkg/api"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	"github.com/sdminonne/workflow-controller/pkg/client"
	"github.com/sdminonne/workflow-controller/pkg/workflow"
	"github.com/sdminonne/workflow-controller/test/e2e/framework"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
)

func deleteWorkflow(workflowClient client.Interface, workflow *wapi.Workflow) {
	workflowClient.Workflows(workflow.Namespace).Delete(workflow.Name, nil)
	By("Workflow deleted")
}

func deleteAllJobs(kubeClient clientset.Interface, workflow *wapi.Workflow) {
	jobs, err := kubeClient.Batch().Jobs(workflow.Namespace).List(api.ListOptions{})
	if err != nil {
		return
	}
	for i := range jobs.Items {
		kubeClient.Batch().Jobs(workflow.Namespace).Delete(jobs.Items[i].Name, nil)
	}
	By("Jobs delete")
}

var _ = Describe("Workflow CRUD", func() {

	It("should create a workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow(framework.FrameworkContext.ResourceGroup, framework.FrameworkContext.ResourceVersion, "hello-workflow", ns, nil)
		Eventually(func() error {
			_, err := workflowClient.Workflows(ns).Create(myWorkflow)
			return err
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
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

	It("should exceed deadline", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		deadline := int64(3)
		myWorkflow := framework.NewWorkflow(framework.FrameworkContext.ResourceGroup, framework.FrameworkContext.ResourceVersion, "mydag", ns, &deadline)
		Eventually(func() error {
			_, err := workflowClient.Workflows(ns).Create(myWorkflow)
			return err
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(func() error {
			curr, err := workflowClient.Workflows(ns).Get(myWorkflow.Name)
			if err != nil {
				return err
			}
			if workflow.IsWorkflowFinished(curr) {
				for i := range curr.Status.Conditions {
					fmt.Printf("Condition Reason -> %s", curr.Status.Conditions[i].Reason)
					if curr.Status.Conditions[i].Reason == "DeadlineExceeded" {
						return nil
					}
				}
				return fmt.Errorf("Workflow finished but not deadline exceeded")
			}
			framework.Logf("Workflow not yet finished %v", curr)
			return fmt.Errorf("workflow %s not finished", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())
	})

	/*
		It("should check defaulting", func() {
			//workflowClient, kubeClient := framework.BuildAndSetClients()
			//fmt.Printf("%v %v\n", workflowClient, kubeClient)
			//fmt.Printf("Three!!")
		})
	*/
})
