package framework

import (
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cronworkflowV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
)

func newSchedule() string {
	return "* * * * ?"
}

func newAntiSchedule() string {
	return ""
}

// NewCronWorkflow creates a CronWorkflow
func NewCronWorkflow(name, namespace string) *cronworkflowV1.CronWorkflow {
	return &cronworkflowV1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronWorkflow",
			APIVersion: workflow.GroupName + "/" + cronworkflowV1.ResourceVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: NewCronWorkflowSpec(),
	}
}

func NewCronWorkflowSpec() cronworkflowV1.CronWorkflowSpec {
	return cronworkflowV1.CronWorkflowSpec{
		Schedule:         newSchedule(),
		WorkflowTemplate: cronworkflowV1.WorkflowTemplateSpec{},
	}
}

//HOCheckCronWorkflowRegistration check wether CronWorkflow is defined
func HOCheckCronWorkflowRegistration(extensionClient apiextensionsclient.Clientset, namespace string) func() error {
	return func() error {
		cronWorkflowResourceName := cronworkflowV1.ResourcePlural + "." + workflow.GroupName
		_, err := extensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(cronWorkflowResourceName, metav1.GetOptions{})
		return err
	}
}
