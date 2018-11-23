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
	wlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/workflow/v1"

	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
)

// GarbageCollector represents a Workflow Garbage Collector.
// It collects orphaned Jobs
type GarbageCollector struct {
	KubeClient     kclientset.Interface
	WorkflowClient wclientset.Interface
	WorkflowLister wlisters.WorkflowLister
	WorkflowSynced cache.InformerSynced
}

// NewGarbageCollector builds initializes and returns a GarbageCollector
func NewGarbageCollector(workflowClient wclientset.Interface, kubeClient kclientset.Interface, workflowInformerFactory winformers.SharedInformerFactory) *GarbageCollector {
	return &GarbageCollector{
		KubeClient:     kubeClient,
		WorkflowClient: workflowClient,
		WorkflowLister: workflowInformerFactory.Workflow().V1().Workflows().Lister(),
		WorkflowSynced: workflowInformerFactory.Workflow().V1().Workflows().Informer().HasSynced,
	}
}

// CollectWorkflowJobs collect the orphaned jobs. First looking in the workflow informer list
// then retrieve from the API and in case NotFound then remove via DeleteCollection primitive
func (c *GarbageCollector) CollectWorkflowJobs() error {
	glog.V(4).Infof("Collecting garbage jobs")
	jobs, err := c.KubeClient.BatchV1().Jobs(metav1.NamespaceAll).List(metav1.ListOptions{
		LabelSelector: controller.WorkflowLabelKey,
	})
	if err != nil {
		return fmt.Errorf("unable to list workflow jobs to be collected: %v", err)
	}
	errs := []error{}
	collected := make(map[string]struct{})
	for _, job := range jobs.Items {
		workflowName, found := job.Labels[controller.WorkflowLabelKey]
		if !found || len(workflowName) == 0 {
			errs = append(errs, fmt.Errorf("unable to find workflow name for job: %s/%s", job.Namespace, job.Name))
			continue
		}
		if _, done := collected[path.Join(job.Namespace, workflowName)]; done {
			continue // already collected so skip
		}
		if _, err := c.WorkflowLister.Workflows(job.Namespace).Get(workflowName); err == nil || !apierrors.IsNotFound(err) {
			if err != nil {
				errs = append(errs, fmt.Errorf("unexpected error retrieving workflow %s/%s cache: %v", job.Namespace, workflowName, err))
			}
			continue
		}
		// Workflow couldn't be find in cache. Trying to get it via APIs.
		if _, err := c.WorkflowClient.Workflow().Workflows(job.Namespace).Get(workflowName, metav1.GetOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				errs = append(errs, fmt.Errorf("unexpected error retrieving workflow %s/%s for job %s/%s: %v", job.Namespace, workflowName, job.Namespace, job.Name, err))
				continue
			}
			// NotFound error: Hence remove all the jobs.
			if err := c.KubeClient.Batch().Jobs(job.Namespace).DeleteCollection(controller.CascadeDeleteOptions(0), metav1.ListOptions{
				LabelSelector: controller.WorkflowLabelKey + "=" + workflowName}); err != nil {
				errs = append(errs, fmt.Errorf("unable to delete Collection of jobs for workflow %s/%s", job.Namespace, workflowName))
				continue
			}
			collected[path.Join(job.Namespace, workflowName)] = struct{}{} // inserted in the collected map
			glog.Infof("removed all jobs for workflow %s/%s", job.Namespace, workflowName)
		}
	}
	return utilerrors.NewAggregate(errs)
}
