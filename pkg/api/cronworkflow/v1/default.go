package v1

import (
	reflect "reflect"
)

// IsCronWorkflowDefaulted checks wether cronWorkflow is defaulted
func IsCronWorkflowDefaulted(w *CronWorkflow) bool {
	defaultedWorkflow := DefaultCronWorkflow(w)
	return reflect.DeepEqual(w.Spec, defaultedWorkflow.Spec)
}

// DefaultCronWorkflow defaults cronWorkflow
func DefaultCronWorkflow(undefaultedWorkflow *CronWorkflow) *CronWorkflow {
	w := undefaultedWorkflow.DeepCopy()
	return w
}
