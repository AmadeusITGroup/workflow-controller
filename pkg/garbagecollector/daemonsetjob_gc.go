package garbagecollector

import (
	"fmt"
	"path"

	"github.com/golang/glog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	kclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	wclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	winformers "github.com/amadeusitgroup/workflow-controller/pkg/client/informers/externalversions"
	wlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/daemonsetjob/v1"

	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
)

// DaemonSetJobGarbageCollector represents a DaemonSetJob Garbage Collector.
// It collects orphaned Jobs
type DaemonSetJobGarbageCollector struct {
	KubeClient         kclientset.Interface
	DaemonSetJobClient wclientset.Interface
	DaemonSetJobLister wlisters.DaemonSetJobLister
	DaemonSetJobSynced cache.InformerSynced
}

// NewGarbageCollector builds initializes and returns a GarbageCollector
func NewDaemonSetJobGarbageCollector(daemonSetJobClient wclientset.Interface, kubeClient kclientset.Interface, daemonSetJobInformerFactory winformers.SharedInformerFactory) *DaemonSetJobGarbageCollector {
	return &DaemonSetJobGarbageCollector{
		KubeClient:         kubeClient,
		DaemonSetJobClient: daemonSetJobClient,
		DaemonSetJobLister: daemonSetJobInformerFactory.Daemonsetjob().V1().DaemonSetJobs().Lister(),
		DaemonSetJobSynced: daemonSetJobInformerFactory.Daemonsetjob().V1().DaemonSetJobs().Informer().HasSynced,
	}
}

// CollectDaemonSetJobJobs collect the orphaned jobs. First looking in the daemonsetjob informer list
// then retrieve from the API and in case NotFound then remove via DeleteCollection primitive
func (c *DaemonSetJobGarbageCollector) CollectDaemonSetJobJobs() error {
	glog.V(4).Infof("Collecting garbage jobs")
	jobs, err := c.KubeClient.BatchV1().Jobs(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: controller.DaemonSetJobLabelKey,
	})
	if err != nil {
		return fmt.Errorf("unable to list daemonsetjobs jobs to be collected: %v", err)
	}
	errs := []error{}
	collected := make(map[string]struct{})
	for _, job := range jobs.Items {
		daemonSetJobName, found := job.Labels[controller.DaemonSetJobLabelKey]
		if !found || len(daemonSetJobName) == 0 {
			errs = append(errs, fmt.Errorf("unable to find daemonsetjob name for job: %s/%s", job.Namespace, job.Name))
			continue
		}
		if _, done := collected[path.Join(job.Namespace, daemonSetJobName)]; done {
			continue // already collected so skip
		}
		if _, err := c.DaemonSetJobLister.DaemonSetJobs(job.Namespace).Get(daemonSetJobName); err == nil || !apierrors.IsNotFound(err) {
			if err != nil {
				errs = append(errs, fmt.Errorf("unexpected error retrieving daemonsetjob %s/%s cache: %v", job.Namespace, daemonSetJobName, err))
			}
			continue
		}
		// DaemonSetJob couldn't be find in cache. Trying to get it via APIs.
		if _, err := c.DaemonSetJobClient.Daemonsetjob().DaemonSetJobs(job.Namespace).Get(daemonSetJobName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("unexpected error retrieving daemonsetjob %s/%s for job %s/%s: %v", job.Namespace, daemonSetJobName, job.Namespace, job.Name, err))
				continue
			}
			// NotFound error: Hence remove all the jobs.
			if err := c.KubeClient.Batch().Jobs(job.Namespace).DeleteCollection(controller.CascadeDeleteOptions(0), metav1.ListOptions{
				LabelSelector: controller.DaemonSetJobLabelKey + "=" + daemonSetJobName}); err != nil {
				errs = append(errs, fmt.Errorf("unable to delete Collection of jobs for daemonsetjob %s/%s", job.Namespace, daemonSetJobName))
				continue
			}
			collected[path.Join(job.Namespace, daemonSetJobName)] = struct{}{} // inserted in the collected map
			glog.Infof("removed all jobs for daemonsetjob %s/%s", job.Namespace, daemonSetJobName)
		}
	}
	return utilerrors.NewAggregate(errs)
}
