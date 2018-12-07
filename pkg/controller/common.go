package controller

import (
	"fmt"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	batchv1listers "k8s.io/client-go/listers/batch/v1"

	batch "k8s.io/api/batch/v1"
)

func deleteJobsFromLabelSelector(namespace string, jobsSelector labels.Selector, lister batchv1listers.JobLister, jobControl JobControlInterface) error {
	jobs, err := lister.Jobs(namespace).List(jobsSelector)
	if err != nil {
		return fmt.Errorf("unable to retrieve jobs to remove from labelSelector %s in namespace %s: %v", jobsSelector.String(), namespace, err)
	}
	errs := []error{}
	for i := range jobs {
		jobToBeRemoved := jobs[i].DeepCopy()
		if IsJobFinished(jobToBeRemoved) { // don't remove already finished job
			glog.V(4).Infof("skipping job %s since finished", jobToBeRemoved.Name)
			continue
		}
		if err := jobControl.DeleteJob(jobToBeRemoved.Namespace, jobToBeRemoved.Name, jobToBeRemoved); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func compareJobSpecMD5Hash(hash string, job *batch.Job) bool {
	if val, ok := job.Annotations[DaemonSetJobMD5AnnotationKey]; ok && val == hash {
		return true
	}
	return false
}
