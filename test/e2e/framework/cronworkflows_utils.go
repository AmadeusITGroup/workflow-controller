package framework

import (
	"fmt"

	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//HOCheckCronWorkflowRegistration check wether CronWorkflow is defined
func HOCheckCronWorkflowRegistration(workflowClient versioned.Interface, namespace string) func() error {
	return func() error {
		//workflows := wapi.WorkflowList{}
		workflows, err := workflowClient.WorkflowV1().Workflows(workflow.Namespace).List(metav1.ListOptions{})
		if err != nil {
			Logf("Cannot list workflows:%v", err)
			return err
		}
		if len(workflows.Items) != 1 {
			return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
		}
		if workflows.Items[0].Status.StartTime != nil {
			Logf("Workflow started")
			return nil
		}
		Logf("Workflow %s not updated", workflow.Name)
		return fmt.Errorf("workflow %s not updated", workflow.Name)
	}
}
