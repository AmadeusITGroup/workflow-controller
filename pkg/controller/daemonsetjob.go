package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
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
	DaemonSetJobLabelKey = "kubernetes.io/DaemonSetJob"
)

// DaemonSetJobController represents the DaemonSetJob controller
type DaemonSetJobController struct {
	DaemonSetJobClient dclientset.Interface
	KubeClient         clientset.Interface

	DaemonSetJobLister dlisters.DaemonSetJobLister
	DaemonSetJobSynced cache.InformerSynced

	JobLister batchv1listers.JobLister
	JobSynced cache.InformerSynced // returns true if job has been synced. Added as member for testing

	JobControl JobControlInterface

	updateHandler func(*dapi.DaemonSetJob) error // callback to upate DaemonSetJob. Added as member for testing
	syncHandler   func(string) error

	queue workqueue.RateLimitingInterface // DaemonSetJobs to be synced

	Recorder record.EventRecorder

	config DaemonSetJobControllerConfig
}

// NewDaemonSetJobController creates and initializes the DaemonSetJobController instance
func NewDaemonSetJobController(
	daemonsetjobClient dclientset.Interface,
	kubeClient clientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	deamonsetJobInformerFactory dinformers.SharedInformerFactory) *DaemonSetJobController {

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	daemonsetjobInformer := deamonsetJobInformerFactory.Daemonsetjob().V1().DaemonSetJobs()

	dj := &DaemonSetJobController{
		DaemonSetJobClient: daemonsetjobClient,
		KubeClient:         kubeClient,
		DaemonSetJobLister: daemonsetjobInformer.Lister(),
		DaemonSetJobSynced: daemonsetjobInformer.Informer().HasSynced,
		JobLister:          jobInformer.Lister(),
		JobSynced:          jobInformer.Informer().HasSynced,

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
			//	AddFunc:    dj.onAddJob,
			//	UpdateFunc: dj.onUpdateJob,
			//	DeleteFunc: dj.onDeleteJob,
		},
	)
	//	dj.JobControl = &DaemonSetJobJobControl{kubeClient, wc.Recorder}
	dj.updateHandler = dj.updateDaemonSetJob
	dj.syncHandler = dj.sync

	return dj
}

// Run simply runs the controller. Assuming Informers are already running: via factories
func (d *DaemonSetJobController) Run(ctx context.Context) error {
	glog.Infof("Starting daemonsetjob controller")

	if !cache.WaitForCacheSync(ctx.Done(), d.JobSynced, d.DaemonSetJobSynced) {
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

	// TODO implement this function

	return nil
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
	if _, err := d.DaemonSetJobClient.DaemonsetjobV1().DaemonSetJobs(dj.Namespace).Update(dj); err != nil {
		glog.V(6).Infof("DaemonSetJob %s/%s updated", dj.Namespace, dj.Name)
	}
	return nil
}

func (d *DaemonSetJobController) deleteDaemonSetJob(namespace, name string) error {
	if err := d.DaemonSetJobClient.DaemonsetjobV1().DaemonSetJobs(namespace).Delete(name, nil); err != nil {
		return fmt.Errorf("unable to delete DaemonSetJob %s/%s: %v", namespace, name, err)
	}

	glog.V(6).Infof("DaemonSetJob %s/%s deleted", namespace, name)
	return nil
}

func (d *DaemonSetJobController) onAddJob(obj interface{}) {
	// TODO implement this method
}

func (d *DaemonSetJobController) onUpdateJob(oldObj, newObj interface{}) {
	// TODO implement this method
}

func (d *DaemonSetJobController) onDeleteJob(obj interface{}) {
	// TODO implement this method
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
