package v1

import (
	reflect "reflect"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/common"
)

// IsDaemonSetJobDefaulted check wether
func IsDaemonSetJobDefaulted(d *DaemonSetJob) bool {
	defaultedDaemonSetJob := DefaultDaemonSetJob(d)
	return reflect.DeepEqual(d.Spec, defaultedDaemonSetJob.Spec)
}

// DefaultDaemonSetJob defaults workflow
func DefaultDaemonSetJob(undefaultedDaemonSetJob *DaemonSetJob) *DaemonSetJob {
	d := undefaultedDaemonSetJob.DeepCopy()
	if d.Spec.JobTemplate == nil {
		return d
	}
	dummyJob := &batchv1.Job{
		Spec: d.Spec.JobTemplate.Spec,
	}
	common.SetDefaults_Job(dummyJob)
	d.Spec.JobTemplate.Spec = dummyJob.Spec
	return d
}
