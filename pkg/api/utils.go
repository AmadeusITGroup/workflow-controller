package api

import "fmt"

// RemoveStepFromSpec remove Step from Workflow Spec
func (w *Workflow) RemoveStepFromSpec(stepName string) error {
	for i := range w.Spec.Steps {
		if w.Spec.Steps[i].Name == stepName {
			w.Spec.Steps = w.Spec.Steps[:i+copy(w.Spec.Steps[i:], w.Spec.Steps[i+1:])]
			return nil
		}
	}
	return fmt.Errorf("unable to find step %q in workflow", stepName)
}

// GetStepByName returns a pointer to Workflow Step
func (w *Workflow) GetStepByName(stepName string) *WorkflowStep {
	for i := range w.Spec.Steps {
		if w.Spec.Steps[i].Name == stepName {
			return &w.Spec.Steps[i]
		}
	}
	return nil
}

// GetStepStatusByName returns a pointer to Workflow Step
func (w *Workflow) GetStepStatusByName(stepName string) *WorkflowStepStatus {
	for i := range w.Status.Statuses {
		if w.Status.Statuses[i].Name == stepName {
			return &w.Status.Statuses[i]
		}
	}
	return nil
}
