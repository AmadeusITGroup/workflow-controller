package api

import (

	//"k8s.io/kubernetes/pkg/apis/batch"
	"fmt"

	"k8s.io/kubernetes/pkg/api"
	_ "k8s.io/kubernetes/pkg/api/install"

	_ "k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	_ "k8s.io/kubernetes/pkg/apis/batch/install" // to register types
	"k8s.io/kubernetes/pkg/apis/batch/v1"
	utilerrors "k8s.io/kubernetes/pkg/util/errors"
)

// Defaulter defines interface for a Workflow defaulter
type Defaulter interface {
	// Default returns a defaulted Workflow
	Default(workflow *Workflow) (*Workflow, error)
}

// BasicDefaulter implements the basic defaulting for Workflow
type BasicDefaulter struct{}

// NewBasicDefaulter builds a BasicDefaulter
func NewBasicDefaulter() *BasicDefaulter {
	return &BasicDefaulter{}
}

// Default implements defaulting for BasicDefaulter
func (d *BasicDefaulter) Default(w *Workflow) (*Workflow, error) {
	errlist := []error{}
	for i := range w.Spec.Steps {
		step := w.Spec.Steps[i]
		if step.JobTemplate == nil {
			continue
		}
		dummyJob := &batch.Job{
			Spec: step.JobTemplate.Spec,
		}
		// Conversion to Version includes defaulting
		objV1, err := api.Scheme.ConvertToVersion(dummyJob, v1.SchemeGroupVersion)
		if err != nil {
			errlist = append(errlist, err)
			continue
		}
		dummyV1Job, ok := objV1.(*v1.Job)
		if !ok {
			errlist = append(errlist, fmt.Errorf("unable to convert object to v1.Job"))
			continue
		}
		// Convert back to internal format (apis/batch/...)
		objUnv, err := api.Scheme.ConvertToVersion(dummyV1Job, batch.SchemeGroupVersion)
		if err != nil {
			errlist = append(errlist, err)
			continue
		}
		defaultedObj, ok := objUnv.(*batch.Job)
		if !ok {
			errlist = append(errlist, fmt.Errorf("unable to convert object to batch.Job"))
			continue
		}
		step.JobTemplate.Spec = defaultedObj.Spec
		w.Spec.Steps[i] = step
	}
	return w, utilerrors.NewAggregate(errlist)
}
