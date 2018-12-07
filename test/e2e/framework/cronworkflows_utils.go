package framework

import (
	"fmt"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	batchv2 "k8s.io/api/batch/v2alpha1"

	cwapiV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	wapiV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/golang/glog"
)

//HOCheckCronWorkflowRegistration check wether CronWorkflow is defined
func HOCheckCronWorkflowRegistration(extensionClient apiextensionsclient.Clientset, namespace string) func() error {
	return func() error {
		cronWorkflowResourceName := cwapiV1.ResourcePlural + "." + workflow.GroupName
		_, err := extensionClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(cronWorkflowResourceName, metav1.GetOptions{})
		return err
	}
}

func everyMinuteSchedule() string {
	return "* * * * *"
}

// NewOneStepCronWorkflow creates a CronWorkflow generating single step workflows
func NewOneStepCronWorkflow(version, name, namespace string) *cwapiV1.CronWorkflow {
	cw := newEmptyCronWorkflow(version, name, namespace)
	step1Name := "one"
	step1Dependencies := []string{}
	step1 := NewWorkflowStep(step1Name, step1Dependencies)
	cw.Spec.WorkflowTemplate.Spec.Steps = append(cw.Spec.WorkflowTemplate.Spec.Steps, *step1)
	return cw
}

// NewTwoStepsCronWorkflow creates a CronWorkflow generating two steps workflows
func NewTwoStepsCronWorkflow(version, name, namespace string) *cwapiV1.CronWorkflow {
	c2 := NewOneStepCronWorkflow(version, name, namespace)
	step2Name := "two"
	step2Dependencies := []string{"one"}
	step2 := NewWorkflowStep(step2Name, step2Dependencies)
	c2.Spec.WorkflowTemplate.Spec.Steps = append(c2.Spec.WorkflowTemplate.Spec.Steps, *step2)
	return c2
}

func newEmptyCronWorkflow(version, name, namespace string) *cwapiV1.CronWorkflow {
	return &cwapiV1.CronWorkflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       cwapiV1.ResourceKind,
			APIVersion: workflow.GroupName + "/" + version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cwapiV1.CronWorkflowSpec{
			Schedule:          everyMinuteSchedule(),
			ConcurrencyPolicy: batchv2.AllowConcurrent,
			WorkflowTemplate: cwapiV1.WorkflowTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: wapiV1.WorkflowSpec{
					Steps: []wapiV1.WorkflowStep{},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"workflow": "empty",
						},
					},
				},
			},
		},
	}
}

// HOCreateCronWorkflow is an higher order func that returns the func to create a CronWorkflow
func HOCreateCronWorkflow(cronWorkflowClient versioned.Interface, cronWorkflow *cwapiV1.CronWorkflow, namespace string) func() error {
	return func() error {
		created, err := cronWorkflowClient.CronworkflowV1().CronWorkflows(namespace).Create(cronWorkflow)
		if err != nil {
			glog.Warningf("cannot create Workflow %s/%s: %v", namespace, cronWorkflow.Name, err)
			return err
		}
		Logf("CronWorkflow %s/%s created", created.Namespace, created.Name)
		return nil
	}
}

// HOIsWorkflowCreatedFromCronWorkflow is an higher order func that returns the func to check wheter a Workflow has been created from a CronWorkflow
func HOIsWorkflowCreatedFromCronWorkflow(workflowClient versioned.Interface, cronWorkflow *cwapiV1.CronWorkflow, namespace string) func() error {
	return func() error {
		workflows, err := workflowClient.WorkflowV1().Workflows(namespace).List(metav1.ListOptions{
			LabelSelector: labels.Set(cronWorkflow.Spec.WorkflowTemplate.Labels).AsSelector().String(),
		})

		if err != nil {
			return err
		}
		if len(workflows.Items) == 0 {
			return fmt.Errorf("No workflows found for cronWorkflow %s/%s", namespace, cronWorkflow.Name)
		}
		return nil
	}
}
