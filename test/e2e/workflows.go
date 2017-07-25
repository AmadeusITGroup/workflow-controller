package e2e

import (
	"fmt"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	wapi "github.com/sdminonne/workflow-controller/pkg/api/v1"
	"github.com/sdminonne/workflow-controller/pkg/controller"
	"github.com/sdminonne/workflow-controller/test/e2e/framework"
)

func deleteWorkflow(workflowClient *rest.RESTClient, workflow *wapi.Workflow) {
	result := wapi.Workflow{}
	workflowClient.Delete().Resource(wapi.ResourcePlural).Namespace(workflow.Namespace).Do().Into(&result)
	By("Workflow deleted")
}

func deleteAllJobs(kubeClient clientset.Interface, workflow *wapi.Workflow) {
	jobs, err := kubeClient.Batch().Jobs(workflow.Namespace).List(metav1.ListOptions{})
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
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow1", ns, nil)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return err
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			if workflows.Items[0].Status.StartTime != nil {
				return nil
			}
			framework.Logf("Workflow %s not updated", myWorkflow.Name)
			return fmt.Errorf("workflow %s not updated", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())
	})

	It("should default workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow2", ns, nil)
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return nil
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
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

	It("should validate workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow1", ns, nil)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return err
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			if workflows.Items[0].Status.Validated {
				framework.Logf("Workflow %s validated", myWorkflow.Name)
				return nil
			}
			framework.Logf("Workflow %s not validated", myWorkflow.Name)
			return fmt.Errorf("workflow %s not validated", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())
	})

	It("should run to finish a workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "workflow1", ns, nil)
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return err
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			if controller.IsWorkflowFinished(&workflows.Items[0]) {
				framework.Logf("Workflow %s finished", myWorkflow.Name)
				return nil
			}
			framework.Logf("Workflow %s not finished", myWorkflow.Name)
			return fmt.Errorf("workflow %s not finished", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())
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
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return err
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) != 1 {
				return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
			}
			if controller.IsWorkflowFinished(&workflows.Items[0]) &&
				framework.IsWorkflowFailedDueDeadline(&workflows.Items[0]) {
				framework.Logf("Workflow %s finished due deadline", myWorkflow.Name)
				return nil
			}
			framework.Logf("Workflow %s not finished", myWorkflow.Name)
			return fmt.Errorf("workflow %s not finished", myWorkflow.Name)
		}, "40s", "5s").ShouldNot(HaveOccurred())
	})

	It("should remove an invalid workflow", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflowWithLoop("dag.example.com", "v1", "loopworkflow", ns)

		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return err
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) == 0 {
				// OK workflows removed or never created
				return nil
			}
			return fmt.Errorf("workflow %s not yet removed", myWorkflow.Name)
		}, "40s", "1s").ShouldNot(HaveOccurred())
	})

	It("should remove workflow created non empty status", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myWorkflow := framework.NewWorkflow("dag.example.com", "v1", "nonemptystatus", ns, nil)
		myWorkflow.Status = wapi.WorkflowStatus{
			Validated: true,
		}
		defer func() {
			deleteWorkflow(workflowClient, myWorkflow)
			deleteAllJobs(kubeClient, myWorkflow)
		}()
		Eventually(func() error {
			result := wapi.Workflow{}
			err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(ns).Body(myWorkflow).Do().Into(&result)
			if err != nil {
				glog.Warningf("cannot create Workflow %s/%s: %v", ns, myWorkflow.Name, err)
				return err
			}
			return nil
		}, "5s", "1s").ShouldNot(HaveOccurred())
		framework.Logf("Workflow is created")

		Eventually(func() error {
			workflows := wapi.WorkflowList{}
			err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(ns).Do().Into(&workflows)
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			if len(workflows.Items) == 0 {
				// OK workflows removed or never created
				return nil
			}
			return fmt.Errorf("workflow %s not yet removed", myWorkflow.Name)
		}, "40s", "1s").ShouldNot(HaveOccurred())
	})
})
