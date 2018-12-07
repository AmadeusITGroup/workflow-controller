package testing

import (
	batchv1 "k8s.io/api/batch/v1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cronworkflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	workflowv1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
)

// ValidFakeTemplateSpec creates and initializes a valid but fake (without a real container image) JobTemplateSpec
func ValidFakeTemplateSpec() *batchv2alpha1.JobTemplateSpec {
	return &batchv2alpha1.JobTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batchv1.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:            "baz",
							Image:           "foo/bar",
							ImagePullPolicy: "IfNotPresent",
						},
					},
					RestartPolicy: "OnFailure",
					DNSPolicy:     "Default",
				},
			},
		},
	}
}

// NewWorkflowTemplateSpec creates and initializes a WorkflowTemplateSpec
func NewWorkflowTemplateSpec() cronworkflowv1.WorkflowTemplateSpec {
	return cronworkflowv1.WorkflowTemplateSpec{
		Spec: workflowv1.WorkflowSpec{
			Steps: []workflowv1.WorkflowStep{
				{
					Name:        "one",
					JobTemplate: ValidFakeTemplateSpec(),
				},
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
		},
	}
}

// NewCronWorkflow creates and initializes a CronWorkflow
func NewCronWorkflow() *cronworkflowv1.CronWorkflow {
	return &cronworkflowv1.CronWorkflow{
		Spec: *NewCronWorkflowSpec(),
	}
}

// NewCronWorkflowSpec creates and initializes a CronWorkflowSpec
func NewCronWorkflowSpec() *cronworkflowv1.CronWorkflowSpec {
	return &cronworkflowv1.CronWorkflowSpec{
		Schedule:          "* * * * *",
		ConcurrencyPolicy: batchv2alpha1.AllowConcurrent,
		WorkflowTemplate:  NewWorkflowTemplateSpec(),
	}
}

// DeleteStepStatusByName remove a step status by name or panic
func DeleteStepStatusByName(statuses *[]workflowv1.WorkflowStepStatus, stepName string) {
	for i := range *statuses {
		if (*statuses)[i].Name == stepName {
			*statuses = (*statuses)[:i+copy((*statuses)[i:], (*statuses)[i+1:])]
			return
		}
	}
	panic("cannot find stepStatus")
}
