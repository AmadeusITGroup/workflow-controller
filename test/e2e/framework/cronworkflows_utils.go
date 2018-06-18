package framework

import (
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cronworkflowV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
)

//HOCheckCronWorkflowRegistration check wether CronWorkflow is defined
func HOCheckCronWorkflowRegistration(extensionClient apiextensionsclient.Clientset, namespace string) func() error {
	return func() error {
		cronWorkflowResourceName := cronworkflowV1.ResourcePlural + "." + workflow.GroupName
		_, err := extensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(cronWorkflowResourceName, metav1.GetOptions{})
		return err
	}
}
