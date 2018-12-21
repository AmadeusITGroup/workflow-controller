package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	"github.com/heptiolabs/healthcheck"

	batch "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	corev1Listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	dapi "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	dclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	dinformers "github.com/amadeusitgroup/workflow-controller/pkg/client/informers/externalversions"
	dlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/daemonsetjob/v1"
)

// DaemonSetJobControllerConfig contains info to customize DaemonSetJob controller behaviour
type DaemonSetJobControllerConfig struct {
	RemoveInvalidDaemonSetJob bool
	NumberOfWorkers           int
}

// DaemonSetJobLabelKey defines the key of label to be injected by DaemonSetJob controller
const (
	DaemonSetJobLabelKey         = "kubernetes.io/DaemonSetJob"
	DaemonSetJobMD5AnnotationKey = "workflow.k8s.io/jobspec-md5"
)

// DaemonSetJobController represents the DaemonSetJob controller
type DaemonSetJobController struct {
	Client     dclientset.Interface
	KubeClient clientset.Interface

	DaemonSetJobLister dlisters.DaemonSetJobLister
	DaemonSetJobSynced cache.InformerSynced

	JobLister batchv1listers.JobLister
	JobSynced cache.InformerSynced // returns true if job has been synced. Added as member for testing

	JobControl JobControlInterface

	NodeLister corev1Listers.NodeLister
	NodeSynced cache.InformerSynced

	updateHandler func(*dapi.DaemonSetJob) error // callback to upate DaemonSetJob. Added as member for testing
	syncHandler   func(string) error

	queue workqueue.RateLimitingInterface // DaemonSetJobs to be synced

	Recorder record.EventRecorder

	config DaemonSetJobControllerConfig
}

// NewDaemonSetJobController creates and initializes the DaemonSetJobController instance
func NewDaemonSetJobController(
	client dclientset.Interface,
	kubeClient clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	deamonsetJobInformerFactory dinformers.SharedInformerFactory) *DaemonSetJobController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	nodeInformer := kubeInformerFactory.Core().V1().Nodes()
	daemonsetjobInformer := deamonsetJobInformerFactory.Daemonsetjob().V1().DaemonSetJobs()

	dj := &DaemonSetJobController{
		Client:             client,
		KubeClient:         kubeClient,
		DaemonSetJobLister: daemonsetjobInformer.Lister(),
		DaemonSetJobSynced: daemonsetjobInformer.Informer().HasSynced,
		JobLister:          jobInformer.Lister(),
		JobSynced:          jobInformer.Informer().HasSynced,
		NodeLister:         nodeInformer.Lister(),
		NodeSynced:         nodeInformer.Informer().HasSynced,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "daemonsetjob"),
		config: DaemonSetJobControllerConfig{
			RemoveInvalidDaemonSetJob: true,
			NumberOfWorkers:           1,
		},

		Recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "daemonsetjob-controller"}),
	}
	daemonsetjobInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    dj.onAddDaemonSetJob,
			UpdateFunc: dj.onUpdateDaemonSetJob,
			DeleteFunc: dj.onDeleteDaemonSetJob,
		},
	)

	jobInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    dj.onAddJob,
			UpdateFunc: dj.onUpdateJob,
			DeleteFunc: dj.onDeleteJob,
		},
	)

	nodeInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    dj.onAddNode,
			UpdateFunc: dj.onUpdateNode,
			DeleteFunc: dj.onDeleteNode,
		},
	)
	dj.JobControl = &WorkflowJobControl{kubeClient, dj.Recorder}
	dj.updateHandler = dj.updateDaemonSetJob
	dj.syncHandler = dj.sync

	return dj
}

// AddHealthCheck add Readiness and Liveness Checks to the handler
func (d *DaemonSetJobController) AddHealthCheck(h healthcheck.Handler) {
	h.AddReadinessCheck("DaemonSetJob_cache_sync", func() error {
		if d.DaemonSetJobSynced() {
			return nil
		}
		return fmt.Errorf("DaemonSetJob cache not sync")
	})

	h.AddReadinessCheck("Job_cache_sync", func() error {
		if d.JobSynced() {
			return nil
		}
		return fmt.Errorf("Job cache not sync")
	})

	h.AddReadinessCheck("Node_cache_sync", func() error {
		if d.NodeSynced() {
			return nil
		}
		return fmt.Errorf("Node cache not sync")
	})
}

// Run simply runs the controller. Assuming Informers are already running: via factories
func (d *DaemonSetJobController) Run(ctx context.Context) error {
	glog.Infof("Starting daemonsetjob controller")

	if !cache.WaitForCacheSync(ctx.Done(), d.JobSynced, d.DaemonSetJobSynced, d.NodeSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := 0; i < d.config.NumberOfWorkers; i++ {
		go wait.Until(d.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	return ctx.Err()
}

func (d *DaemonSetJobController) runWorker() {
	for d.processNextItem() {
	}
}

func (d *DaemonSetJobController) processNextItem() bool {
	key, quit := d.queue.Get()
	if quit {
		return false
	}
	defer d.queue.Done(key)
	err := d.syncHandler(key.(string))
	if err == nil {
		d.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("Error syncing daemonsejob: %v", err))
	d.queue.AddRateLimited(key)

	return true
}

// enqueue adds key in the controller queue
func (d *DaemonSetJobController) enqueue(daemonsetjob *dapi.DaemonSetJob) {
	key, err := cache.MetaNamespaceKeyFunc(daemonsetjob)
	if err != nil {
		glog.Errorf("DaemonSetJobController:enqueue: couldn't get key for DaemonSetJob   %s/%s", daemonsetjob.Namespace, daemonsetjob.Name)
		return
	}
	d.queue.Add(key)
}

// daemonsetjob sync method
func (d *DaemonSetJobController) sync(key string) error {
	startTime := time.Now()
	defer func() {
		glog.V(6).Infof("Finished syncing DaemonSetJob %q (%v", key, time.Now().Sub(startTime))
	}()

	var sharedDaemonSetJob *dapi.DaemonSetJob
	var err error
	if sharedDaemonSetJob, err = d.getDaemonSetJobByKey(key); err != nil {
		glog.V(4).Infof("unable to get Workflow %s: %v. Maybe deleted", key, err)
		return nil
	}
	namespace := sharedDaemonSetJob.Namespace
	name := sharedDaemonSetJob.Name

	if !dapi.IsDaemonSetJobDefaulted(sharedDaemonSetJob) {
		defaultedDaemonSetJob := dapi.DefaultDaemonSetJob(sharedDaemonSetJob)
		if err := d.updateHandler(defaultedDaemonSetJob); err != nil {
			glog.Errorf("DaemonSetJobController.sync unable to default DaemonSetJob %s/%s: %v", namespace, name, err)
			return fmt.Errorf("unable to default DaemonSetJob %s/%s: %v", namespace, name, err)
		}
		glog.V(6).Infof("DaemonSetJobController.sync Defaulted %s/%s", namespace, name)
		return nil
	}

	// Validation. We always revalidate DaemonSetJob since any update must be re-checked.
	if errs := dapi.ValidateDaemonSetJob(sharedDaemonSetJob); errs != nil && len(errs) > 0 {
		glog.Errorf("DaemonSetJobController.sync DaemonSetJob %s/%s not valid: %v", namespace, name, errs)
		if d.config.RemoveInvalidDaemonSetJob {
			glog.Errorf("Invalid DaemonSetJob %s/%s is going to be removed", namespace, name)
			if err := d.deleteDaemonSetJob(namespace, name); err != nil {
				glog.Errorf("unable to delete invalid DaemonSetJob %s/%s: %v", namespace, name, err)
				return fmt.Errorf("unable to delete invalid DaemonSetJob %s/%s: %v", namespace, name, err)
			}
		}
		return nil
	}

	// TODO: add test the case of graceful deletion
	if sharedDaemonSetJob.DeletionTimestamp != nil {
		return nil
	}

	daemonsetjob := sharedDaemonSetJob.DeepCopy()
	now := metav1.Now()

	if daemonsetjob.Status.StartTime == nil {
		// init daemonsetjob.Status
		daemonsetjob.Status.StartTime = &now
		if err := d.updateHandler(daemonsetjob); err != nil {
			glog.Errorf("daemonsetjob %s/%s: unable to init startTime: %v", namespace, name, err)
			return err
		}
		glog.V(4).Infof("daemonsetjob %s/%s: startTime updated", namespace, name)
		return nil
	}

	if d.pastActiveDeadline(daemonsetjob, startTime) {
		daemonsetjob.Status.CompletionTime = &now
		daemonsetjob.Status.Conditions = append(daemonsetjob.Status.Conditions, d.newDeadlineExceededCondition())
		if err := d.updateHandler(daemonsetjob); err != nil {
			glog.Errorf("daemonsetjob %s/%s: unable to set DeadlineExceeded: %v", namespace, name, err)
			return fmt.Errorf("unable to set DeadlineExceeded for DaemonSetJob %s/%s: %v", namespace, name, err)
		}
		if err := d.deleteDaemonSetJobJobs(daemonsetjob); err != nil {
			glog.Errorf("daemonsetjob %s/%s: unable to cleanup jobs: %v", namespace, name, err)
			return fmt.Errorf("daemonsetjob %s/%s: unable to cleanup jobs: %v", namespace, name, err)
		}
		return nil
	}

	return d.manageDaemonSetJob(daemonsetjob)
}

func (d *DaemonSetJobController) manageDaemonSetJob(daemonsetjob *dapi.DaemonSetJob) error {
	daemonsetjobToBeUpdated := false
	daemonsetjobComplete := false
	daemonsetjobFailed := false

	md5JobSpec, err := generateMD5JobSpec(&daemonsetjob.Spec.JobTemplate.Spec)
	if err != nil {
		return fmt.Errorf("unable to generates the JobSpec MD5, %v", err)
	}

	errs := []error{}
	jobSelector := labels.Set{DaemonSetJobLabelKey: daemonsetjob.Name}
	jobs, err := d.JobLister.List(jobSelector.AsSelectorPreValidated())
	jobsByNodeName := make(map[string]*batch.Job)
	jobsToBeDeleted := []*batch.Job{}
	for _, job := range jobs {
		if _, ok := jobsByNodeName[job.Spec.Template.Spec.NodeName]; ok {
			// a job is already associated with this node, we should delete the other one.
			jobsToBeDeleted = append(jobsToBeDeleted, job)
		} else {
			jobsByNodeName[job.Spec.Template.Spec.NodeName] = job
		}
	}

	for _, job := range jobsToBeDeleted {
		err = d.JobControl.DeleteJob(job.Namespace, job.Name, job)
		if err != nil {
			errs = append(errs, err)
		}
	}

	nodeSelector := labels.Set{}
	if daemonsetjob.Spec.NodeSelector != nil {
		nodeSelector = labels.Set(daemonsetjob.Spec.NodeSelector.MatchLabels)
	}

	nodes, err := d.NodeLister.List(nodeSelector.AsSelectorPreValidated())
	if err != nil {
		return err
	}

	allJobs := []*batch.Job{}
	for _, node := range nodes {
		job, ok := jobsByNodeName[node.Name]
		if !ok {
			glog.V(6).Infof("Job for the node %s not found %s/%s", node.Name, daemonsetjob.Namespace, daemonsetjob.Name)
			job, err = d.JobControl.CreateJobFromDaemonSetJob(daemonsetjob.Namespace, daemonsetjob.Spec.JobTemplate, daemonsetjob, node.Name)
			if err != nil {
				errs = append(errs, err)
			}
			daemonsetjobToBeUpdated = true
		} else if job != nil {
			if !compareJobSpecMD5Hash(md5JobSpec, job) {
				glog.V(6).Infof("JobTemplateSpec has changed, %s/%s, previousMD5:%s, current:%s", job.Namespace, job.Name, md5JobSpec, job.GetAnnotations()[DaemonSetJobMD5AnnotationKey])
				switch getJobStatus(job) {
				case failedJobStatus:
				case succeededJobStatus:
					err = d.JobControl.DeleteJob(job.Namespace, job.Name, job)
					if err != nil {
						errs = append(errs, err)
					}
				case activeJobStatus:
					// wait that the job finished or failed before deleting it.
					continue
				}
				daemonsetjobToBeUpdated = true
			} else {
				allJobs = append(allJobs, job)
			}
		}
		delete(jobsByNodeName, node.Name)
	}

	// if the map still contains Jobs it means Nodes have been removed or the node selector was updated
	// so delete those jobs
	for node, job := range jobsByNodeName {
		glog.V(6).Infof("Node %s doesn't match anymore the nodeSelector for daemonsetjob %s/%s", node, daemonsetjob.Namespace, daemonsetjob.Name)
		err = d.JobControl.DeleteJob(job.Namespace, job.Name, job)
		if err != nil {
			errs = append(errs, err)
		}
	}

	// build status
	now := metav1.Now()
	activeJobs, succeededJobs, failedJobs := getJobsStatus(allJobs)
	if failedJobs > 0 {
		daemonsetjobFailed = true
	}
	if activeJobs == 0 {
		daemonsetjobComplete = true
	}
	if activeJobs != daemonsetjob.Status.Active || succeededJobs != daemonsetjob.Status.Succeeded || failedJobs != daemonsetjob.Status.Failed {
		daemonsetjob.Status.Active = activeJobs
		daemonsetjob.Status.Succeeded = succeededJobs
		daemonsetjob.Status.Failed = failedJobs
		daemonsetjobToBeUpdated = true
	}
	updateDaemonSetJobStatusConditions(&daemonsetjob.Status, now, daemonsetjobComplete, daemonsetjobFailed)
	if daemonsetjobComplete {
		glog.Infof("Workflow %s/%s complete.", daemonsetjob.Namespace, daemonsetjob.Name)
		daemonsetjobToBeUpdated = true
	}

	if daemonsetjobToBeUpdated {
		err = d.updateHandler(daemonsetjob)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return utilerrors.NewAggregate(errs)
}

func getJobsStatus(jobs []*batch.Job) (activeJobs, succeededJobs, failedJobs int32) {
	for _, job := range jobs {
		switch getJobStatus(job) {
		case failedJobStatus:
			failedJobs++
		case succeededJobStatus:
			succeededJobs++
		case activeJobStatus:
			activeJobs++
		}
	}
	return
}

type jobStatus string

const (
	activeJobStatus    jobStatus = "active"
	succeededJobStatus jobStatus = "successed"
	failedJobStatus    jobStatus = "failed"
)

func getJobStatus(job *batch.Job) jobStatus {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batch.JobFailed && condition.Status == apiv1.ConditionTrue {
			return failedJobStatus
		}
		if condition.Type == batch.JobComplete && condition.Status == apiv1.ConditionTrue {
			return succeededJobStatus
		}
	}
	return activeJobStatus
}

func (d *DaemonSetJobController) onAddDaemonSetJob(obj interface{}) {
	daemonsetjob, ok := obj.(*dapi.DaemonSetJob)
	if !ok {
		glog.Errorf("adding daemonsetjob, expected daemonsetjob object. Got: %+v", obj)
		return
	}

	if !reflect.DeepEqual(daemonsetjob.Status, dapi.DaemonSetJobStatus{}) {
		glog.Errorf("daemonsetjob %s/%s created with non empty status. Going to be removed", daemonsetjob.Namespace, daemonsetjob.Name)
		if _, err := cache.MetaNamespaceKeyFunc(daemonsetjob); err != nil {
			glog.Errorf("couldn't get key for DaemonSetJob (to be deleted) %s/%s: %v", daemonsetjob.Namespace, daemonsetjob.Name, err)
			return
		}
		// TODO: how to remove a DaemonSetJob created with an invalid or even with a valid status. What in case of error for this delete?
		if err := d.deleteDaemonSetJob(daemonsetjob.Namespace, daemonsetjob.Name); err != nil {
			glog.Errorf("unable to delete non empty status DaemonSetJob %s/%s: %v. No retry will be performed.", daemonsetjob.Namespace, daemonsetjob.Name, err)
		}
		return
	}
	d.enqueue(daemonsetjob)
}

func (d *DaemonSetJobController) onUpdateDaemonSetJob(oldObj, newObj interface{}) {
	daemonsetjob, ok := newObj.(*dapi.DaemonSetJob)
	if !ok {
		glog.Errorf("Expected DaemonSetJob object. Got: %+v", newObj)
		return
	}
	if IsDaemonSetJobFinished(daemonsetjob) {
		glog.Warningf("Update event received on complete DaemonSetJob: %s/%s", daemonsetjob.Namespace, daemonsetjob.Name)
		return
	}
	d.enqueue(daemonsetjob)
}

func (d *DaemonSetJobController) onDeleteDaemonSetJob(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		glog.Errorf("Unable to get key for %#v: %v", obj, err)
		return
	}
	d.queue.Add(key)
}

func (d *DaemonSetJobController) updateDaemonSetJob(dj *dapi.DaemonSetJob) error {
	if _, err := d.Client.DaemonsetjobV1().DaemonSetJobs(dj.Namespace).Update(dj); err != nil {
		glog.V(6).Infof("DaemonSetJob %s/%s updated", dj.Namespace, dj.Name)
	}
	return nil
}

func (d *DaemonSetJobController) deleteDaemonSetJob(namespace, name string) error {
	if err := d.Client.DaemonsetjobV1().DaemonSetJobs(namespace).Delete(name, nil); err != nil {
		return fmt.Errorf("unable to delete DaemonSetJob %s/%s: %v", namespace, name, err)
	}

	glog.V(6).Infof("DaemonSetJob %s/%s deleted", namespace, name)
	return nil
}

func (d *DaemonSetJobController) onAddJob(obj interface{}) {
	job := obj.(*batch.Job)
	daemonsetJobs, err := d.getDaemonSetJobFromJob(job)
	if err != nil {
		glog.Errorf("unable to get DaemonSetJobs from job %s/%s: %v", job.Namespace, job.Name, err)
		return
	}
	for i := range daemonsetJobs {
		d.enqueue(daemonsetJobs[i])
	}
}

func (d *DaemonSetJobController) onUpdateJob(oldObj, newObj interface{}) {
	oldJob := oldObj.(*batch.Job)
	newJob := newObj.(*batch.Job)
	if oldJob.ResourceVersion == newJob.ResourceVersion { // Since periodic resync will send update events for all known jobs.
		return
	}
	d.onAddJob(newObj)
}

func (d *DaemonSetJobController) onDeleteJob(obj interface{}) {
	job, ok := obj.(*batch.Job)
	glog.V(6).Infof("onDeleteJob old=%v", job.Name)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %+v", obj)
			return
		}
		job, ok = tombstone.Obj.(*batch.Job)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a job %+v", obj)
			return
		}
	}
	daemonsetJobs, err := d.getDaemonSetJobFromJob(job)
	if err != nil {
		glog.Errorf("DaemonSetJob.onDeleteJob: %v", err)
		return
	}
	for i := range daemonsetJobs {
		d.enqueue(daemonsetJobs[i])
	}
}

func (d *DaemonSetJobController) onAddNode(obj interface{}) {
	node := obj.(*apiv1.Node)
	daemonsetjobList, err := d.DaemonSetJobLister.List(labels.Everything())
	if err != nil {
		glog.Errorf("unable to get DaemonSetJobs from node %s/%s: %v", node.Namespace, node.Name, err)
		return
	}
	for i := range daemonsetjobList {
		d.enqueue(daemonsetjobList[i])
	}
}

func (d *DaemonSetJobController) onUpdateNode(oldObj, newObj interface{}) {
	oldNode := oldObj.(*apiv1.Node)
	newNode := newObj.(*apiv1.Node)
	if oldNode.ResourceVersion == newNode.ResourceVersion { // Since periodic resync will send update events for all known jobs.
		return
	}
	d.onAddNode(newObj)
}

func (d *DaemonSetJobController) onDeleteNode(obj interface{}) {
	node, ok := obj.(*apiv1.Node)
	glog.V(6).Infof("onDeleteNode old=%v", node.Name)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			glog.Errorf("Couldn't get object from tombstone %+v", obj)
			return
		}
		node, ok = tombstone.Obj.(*apiv1.Node)
		if !ok {
			glog.Errorf("Tombstone contained object that is not a node %+v", obj)
			return
		}
	}
	daemonsetjobList, err := d.DaemonSetJobLister.List(labels.Everything())
	if err != nil {
		glog.Errorf("unable to get DaemonSetJobs from node %s/%s: %v", node.Namespace, node.Name, err)
		return
	}
	for i := range daemonsetjobList {
		d.enqueue(daemonsetjobList[i])
	}
}

// IsDaemonSetJobFinished checks wether a daemonsetjob is finished. A daemonsetjob is finished if one of its condition is Complete or Failed.
func IsDaemonSetJobFinished(d *dapi.DaemonSetJob) bool {
	for _, c := range d.Status.Conditions {
		if c.Status == apiv1.ConditionTrue && (c.Type == dapi.DaemonSetJobComplete || c.Type == dapi.DaemonSetJobFailed) {
			return true
		}
	}
	return false
}

// InferDaemonSetJobLabelSelectorForJobs returns labels.Selector corresponding to the associated Jobs
func InferDaemonSetJobLabelSelectorForJobs(daemonsetjob *dapi.DaemonSetJob) labels.Selector {
	set := fetchLabelsSetFromLabelSelector(daemonsetjob.Spec.Selector)
	set[DaemonSetJobLabelKey] = daemonsetjob.Name
	return labels.SelectorFromSet(set)
}

// get daemonsetjob by key method
func (d *DaemonSetJobController) getDaemonSetJobByKey(key string) (*dapi.DaemonSetJob, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil, err
	}
	glog.V(6).Infof("Syncing %s/%s", namespace, name)
	daemonsetJob, err := d.DaemonSetJobLister.DaemonSetJobs(namespace).Get(name)
	if err != nil {
		glog.V(4).Infof("unable to get DaemonSetJob %s/%s: %v. Maybe deleted", namespace, name, err)
		return nil, err
	}
	return daemonsetJob, nil
}

func (d *DaemonSetJobController) getDaemonSetJobFromJob(job *batch.Job) ([]*dapi.DaemonSetJob, error) {
	deamonsetjobs := []*dapi.DaemonSetJob{}
	if len(job.Labels) == 0 {
		return deamonsetjobs, fmt.Errorf("no deamonsetjobs found for job. Job %s/%s has no labels", job.Namespace, job.Name)
	}
	deamonsetjobList, err := d.DaemonSetJobLister.List(labels.Everything())
	if err != nil {
		return deamonsetjobs, fmt.Errorf("no deamonsetjobs found for job. Job %s/%s. Cannot list deamonsetjobs: %v", job.Namespace, job.Name, err)
	}
	for i := range deamonsetjobList {
		deamonsetjob := deamonsetjobList[i]
		if deamonsetjob.Namespace != job.Namespace {
			continue
		}
		if labels.SelectorFromSet(deamonsetjob.Spec.Selector.MatchLabels).Matches(labels.Set(job.Labels)) {
			deamonsetjobs = append(deamonsetjobs, deamonsetjob)
		}
	}
	return deamonsetjobs, nil
}

func (d *DaemonSetJobController) pastActiveDeadline(daemonsetjob *dapi.DaemonSetJob, now time.Time) bool {
	if daemonsetjob.Spec.ActiveDeadlineSeconds == nil || daemonsetjob.Status.StartTime == nil {
		return false
	}
	start := daemonsetjob.Status.StartTime.Time
	duration := now.Sub(start)
	allowedDuration := time.Duration(*daemonsetjob.Spec.ActiveDeadlineSeconds) * time.Second
	return duration >= allowedDuration
}

func (d *DaemonSetJobController) deleteDaemonSetJobJobs(daemonsetjob *dapi.DaemonSetJob) error {
	glog.V(6).Infof("deleting all jobs for daemonsetjob %s/%s", daemonsetjob.Namespace, daemonsetjob.Name)

	jobsSelector := InferDaemonSetJobLabelSelectorForJobs(daemonsetjob)
	return deleteJobsFromLabelSelector(daemonsetjob.Namespace, jobsSelector, d.JobLister, d.JobControl)
}

func (d *DaemonSetJobController) newDeadlineExceededCondition() dapi.DaemonSetJobCondition {
	return newDaemonSetJobStatusCondition(dapi.DaemonSetJobFailed, metav1.Now(), "DeadlineExceeded", "DaemonSetJob was active longer than specified deadline")
}

func (d *DaemonSetJobController) newCompletedCondition() dapi.DaemonSetJobCondition {
	return newDaemonSetJobStatusCondition(dapi.DaemonSetJobComplete, metav1.Now(), "", "")
}

func searchDaemonSetJobStatusConditionType(status *dapi.DaemonSetJobStatus, t dapi.DaemonSetJobConditionType) int {
	idCondition := -1
	for i, condition := range status.Conditions {
		if condition.Type == t {
			idCondition = i
			break
		}
	}
	return idCondition
}

func updateDaemonSetJobStatusCondition(status *dapi.DaemonSetJobStatus, now metav1.Time, t dapi.DaemonSetJobConditionType, conditionStatus apiv1.ConditionStatus) {
	idConditionComplete := searchDaemonSetJobStatusConditionType(status, t)
	if idConditionComplete >= 0 {
		if status.Conditions[idConditionComplete].Status != conditionStatus {
			status.Conditions[idConditionComplete].LastTransitionTime = now
			status.Conditions[idConditionComplete].Status = conditionStatus
		}
		status.Conditions[idConditionComplete].LastProbeTime = now
	} else if conditionStatus == apiv1.ConditionTrue {
		// Only add if the condition is True
		status.Conditions = append(status.Conditions, newDaemonSetJobStatusCondition(t, now, "", ""))
	}
}

func updateDaemonSetJobStatusConditions(status *dapi.DaemonSetJobStatus, now metav1.Time, completed bool, failed bool) {
	if completed {
		updateDaemonSetJobStatusCondition(status, now, dapi.DaemonSetJobComplete, apiv1.ConditionTrue)
		if status.CompletionTime == nil {
			status.CompletionTime = &now
		}
	} else {
		// daemonsetjob is currently running
		updateDaemonSetJobStatusCondition(status, now, dapi.DaemonSetJobComplete, apiv1.ConditionFalse)
		status.CompletionTime = nil
	}

	if failed {
		updateDaemonSetJobStatusCondition(status, now, dapi.DaemonSetJobFailed, apiv1.ConditionTrue)
	} else {
		updateDaemonSetJobStatusCondition(status, now, dapi.DaemonSetJobFailed, apiv1.ConditionFalse)
	}
}

func newDaemonSetJobStatusCondition(conditionType dapi.DaemonSetJobConditionType, now metav1.Time, reason, message string) dapi.DaemonSetJobCondition {
	return dapi.DaemonSetJobCondition{
		Type:               conditionType,
		Status:             apiv1.ConditionTrue,
		LastProbeTime:      now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}
