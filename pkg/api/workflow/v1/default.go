package v1

import (
	reflect "reflect"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/common"
)

// IsWorkflowDefaulted check wether
func IsWorkflowDefaulted(w *Workflow) bool {
	defaultedWorkflow := DefaultWorkflow(w)
	return reflect.DeepEqual(w.Spec, defaultedWorkflow.Spec)
}

// DefaultWorkflow defaults workflow
func DefaultWorkflow(undefaultedWorkflow *Workflow) *Workflow {
	w := undefaultedWorkflow.DeepCopy()
	for i := range w.Spec.Steps {
		step := w.Spec.Steps[i]
		if step.JobTemplate == nil {
			continue
		}
		dummyJob := &batchv1.Job{
			Spec: step.JobTemplate.Spec,
		}
		common.SetDefaults_Job(dummyJob)
		step.JobTemplate.Spec = dummyJob.Spec
		w.Spec.Steps[i] = step
	}
	return w
}
