package util

import (
	"fmt"

	cwapi "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
)

// GetStepByName returns a pointer to WorkflowStep
func GetStepByName(w *wapi.Workflow, stepName string) *wapi.WorkflowStep {
	for i := range w.Spec.Steps {
		if w.Spec.Steps[i].Name == stepName {
			return &w.Spec.Steps[i]
		}
	}
	return nil
}

// GetStepStatusByName return a pointer to WorkflowStepStatus
func GetStepStatusByName(w *wapi.Workflow, stepName string) *wapi.WorkflowStepStatus {
	for i := range w.Status.Statuses {
		if w.Status.Statuses[i].Name == stepName {
			return &w.Status.Statuses[i]
		}
	}
	return nil
}

// RemoveStepFromSpec removes Step from Workflow Spec
func RemoveStepFromSpec(w *wapi.Workflow, stepName string) error {
	for i := range w.Spec.Steps {
		if w.Spec.Steps[i].Name == stepName {
			w.Spec.Steps = w.Spec.Steps[:i+copy(w.Spec.Steps[i:], w.Spec.Steps[i+1:])]
			return nil
		}
	}
	return fmt.Errorf("unable to find step %q in workflow", stepName)
}

func CopyMap(m map[string]string) map[string]string {
	l := make(map[string]string)
	for k, v := range m {
		l[k] = v
	}
	return l
}

func GetWorkflowSpecFromWorkflowTemplate(template *cwapi.WorkflowTemplateSpec) *wapi.WorkflowSpec {
	return template.Spec.DeepCopy()
}
