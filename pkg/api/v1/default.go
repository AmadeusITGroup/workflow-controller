package v1

import (
	reflect "reflect"

	batchv1 "k8s.io/api/batch/v1"
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
		SetDefaults_Job(dummyJob)
		step.JobTemplate.Spec = dummyJob.Spec
		w.Spec.Steps[i] = step
	}
	return w
}

// SetDefaults_Job copied here function from k8s.io/kubernetes/pkg/apis/batch/v1/defaults.go
func SetDefaults_Job(obj *batchv1.Job) {
	// For a non-parallel job, you can leave both `.spec.completions` and
	// `.spec.parallelism` unset.  When both are unset, both are defaulted to 1.
	if obj.Spec.Completions == nil && obj.Spec.Parallelism == nil {
		obj.Spec.Completions = new(int32)
		*obj.Spec.Completions = 1
		obj.Spec.Parallelism = new(int32)
		*obj.Spec.Parallelism = 1
	}
	if obj.Spec.Parallelism == nil {
		obj.Spec.Parallelism = new(int32)
		*obj.Spec.Parallelism = 1
	}
	labels := obj.Spec.Template.Labels
	if labels != nil && len(obj.Labels) == 0 {
		obj.Labels = labels
	}
}
