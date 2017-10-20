package e2e

import (
	"fmt"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	wapi "github.com/sdminonne/workflow-controller/pkg/api/workflow/v1"
	"github.com/sdminonne/workflow-controller/pkg/client/versioned"
	"github.com/sdminonne/workflow-controller/test/e2e/framework"
)

func deleteWorkflow(workflowClient versioned.Interface, workflow *wapi.Workflow) {
	workflowClient.WorkflowV1().Workflows(workflow.Namespace).Delete(workflow.Name, nil)
	By("Workflow deleted")
}

func cascadeDeleteOptions(gracePeriodSeconds int64) *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		GracePeriodSeconds: func(t int64) *int64 { return &t }(gracePeriodSeconds),
		PropagationPolicy: func() *metav1.DeletionPropagation {
			foreground := metav1.DeletePropagationForeground
			return &foreground
		}(),
	}
}

func deleteAllJobs(kubeClient clientset.Interface, workflow *wapi.Workflow) {
	jobs, err := kubeClient.Batch().Jobs(workflow.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return
	}
	for i := range jobs.Items {
		kubeClient.Batch().Jobs(workflow.Namespace).Delete(jobs.Items[i].Name /*cascadeDeleteOptions(0)*/, nil)
	}
	By("Jobs delete")
}

var _ = Describe("Workflow CRUD", func() {
	It("should create a workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow1", ns, nil)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowStarted(workflowClient, myWorkflow, ns), "40s", "5s").ShouldNot(HaveOccurred())
	})

	It("should default workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow2", ns, nil)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(func() error {
			workflows, err := workflowClient.WorkflowV1().Workflows(ns).List(metav1.ListOptions{})
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			if wapi.IsWorkflowDefaulted(&workflows.Items[0]) {
				return nil
			}
			framework.Logf("Workflow %s not defaulted", myWorkflow.Name)
			return fmt.Errorf("workflow %s not defaulted", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())
	})

	It("should run to finish a workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow3", ns, nil)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowFinished(workflowClient, myWorkflow, ns), "60s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOChekcAllStepsFinished(workflowClient, myWorkflow, ns), "60s", "5s").ShouldNot(HaveOccurred())
	})

	It("should be able to update workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflowWithThreeSteps("dag.example.com", "v1", "workflow3", ns)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowStarted(workflowClient, myWorkflow, ns), "40s", "5s").ShouldNot(HaveOccurred())

		// Edit Workflow adding a step "four"
		Eventually(func() error {
			workflows, err := workflowClient.WorkflowV1().Workflows(ns).List(metav1.ListOptions{})
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			workflow := workflows.Items[0].DeepCopy()
			step4 := framework.NewWorkflowStep("four", []string{"three"})
			workflow.Spec.Steps = append(workflow.Spec.Steps, *step4)
			if _, err = workflowClient.WorkflowV1().Workflows(ns).Update(workflow); err != nil {
				return fmt.Errorf("unable to update workflow %s/%s: %v", workflow.Namespace, workflow.Name, err)
			}
			framework.Logf("workflow update")
			return nil
		}, "40s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowFinished(workflowClient, myWorkflow, ns), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOChekcAllStepsFinished(workflowClient, myWorkflow, ns), "40s", "5s").ShouldNot(HaveOccurred())

	})

	It("should exceed deadline", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "deadlineworkflow", ns, nil)
		threeSecs := int64(3)
		myWorkflow.Spec.ActiveDeadlineSeconds = &threeSecs // Set deadline
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowFinished(workflowClient, myWorkflow, ns), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(func() error {
			workflows, err := workflowClient.WorkflowV1().Workflows(ns).List(metav1.ListOptions{})
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			if framework.IsWorkflowFailedDueDeadline(&workflows.Items[0]) {
				framework.Logf("Workflow %s finished due deadline", myWorkflow.Name)
				return nil
			}
			return fmt.Errorf("workflow %s not finished to deadline", myWorkflow.Name)
		}, "40s", "1s").ShouldNot(HaveOccurred())
	})

	It("should remove an invalid workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflowWithLoop("dag.example.com", "v1", "loopworkflow", ns)

		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HONoWorkflowsShouldRemains(workflowClient, ns), "40s", "1s").ShouldNot(HaveOccurred())
	})

	It("should remove workflow created non empty status", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "nonemptystatus", ns, nil)
		now := metav1.Now()
		myWorkflow.Status = wapi.WorkflowStatus{ // add a non empty status
			StartTime: &now,
		}
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HONoWorkflowsShouldRemains(workflowClient, ns), "40s", "1s").ShouldNot(HaveOccurred())
	})

})

var _ = Describe("Workflow Garbage Collection", func() {
	It("should delete all jobs for deleted Workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault

		myWorkflow := framework.NewWorkflowWithThreeSteps("dag.example.com", "v1", "workflow-long", ns)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(framework.HOCreateWorkflow(workflowClient, myWorkflow, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowStarted(workflowClient, myWorkflow, ns), "40s", "5s").ShouldNot(HaveOccurred())

		// as soon "one" finished...
		Eventually(framework.HOCheckStepFinished(workflowClient, myWorkflow, "one", ns), "40s", "1s").ShouldNot(HaveOccurred())

		// Delete Workflow
		Eventually(framework.HODeleteWorkflow(workflowClient, myWorkflow, ns), "40s", "1s").ShouldNot(HaveOccurred())

		// Check all Jobs have been deleted
		Eventually(framework.HONoJobsShouldRemains(kubeClient, ns), "90s", "5s").ShouldNot(HaveOccurred())

	})
})
