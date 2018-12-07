package controller

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	batch "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"

	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	batchv1listers "k8s.io/client-go/listers/batch/v1"
	corev1Listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob"
	dapi "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	wclient "github.com/amadeusitgroup/workflow-controller/pkg/client"
	dclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	winformers "github.com/amadeusitgroup/workflow-controller/pkg/client/informers/externalversions"
	dlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/daemonsetjob/v1"
	utiltesting "github.com/amadeusitgroup/workflow-controller/pkg/util/testing"
)

func TestDaemonSetJobController_Run(t *testing.T) {
	restConfig := &rest.Config{Host: "localhost"}

	type fields struct {
		DaemonSetJobClient dclientset.Interface
		KubeClient         clientset.Interface
		DaemonSetJobLister dlisters.DaemonSetJobLister
		DaemonSetJobSynced cache.InformerSynced
		JobLister          batchv1listers.JobLister
		JobSynced          cache.InformerSynced
		JobControl         JobControlInterface
		NodeLister         corev1Listers.NodeLister
		NodeSynced         cache.InformerSynced
		updateHandler      func(*dapi.DaemonSetJob) error
		queue              workqueue.RateLimitingInterface
		Recorder           record.EventRecorder
		config             DaemonSetJobControllerConfig
	}
	tests := []struct {
		name          string
		fields        fields
		keys          []string
		wantErr       bool
		WantErrString string
	}{
		{
			name: "nominal case",
			fields: fields{
				JobSynced:          func() bool { return true },
				DaemonSetJobSynced: func() bool { return true },
				NodeSynced:         func() bool { return true },
			},
			keys:    []string{"default/dsj1", "default/dsj2", "default/dsj3", "default/last"},
			wantErr: false,
		},
		{
			name: "no sync for Job",
			fields: fields{
				JobSynced:          func() bool { return false },
				DaemonSetJobSynced: func() bool { return true },
				NodeSynced:         func() bool { return true },
			},
			keys:          []string{"default/dsj1", "default/dsj2", "default/dsj3", "default/last"},
			wantErr:       true,
			WantErrString: "Timed out waiting for caches to sync",
		},
		{
			name: "no sync for DaemonSetJob",
			fields: fields{
				JobSynced:          func() bool { return true },
				DaemonSetJobSynced: func() bool { return false },
				NodeSynced:         func() bool { return true },
			},
			keys:          []string{"default/dsj1", "default/dsj2", "default/dsj3", "default/last"},
			wantErr:       true,
			WantErrString: "Timed out waiting for caches to sync",
		},
		{
			name: "no sync for Node",
			fields: fields{
				JobSynced:          func() bool { return true },
				DaemonSetJobSynced: func() bool { return true },
				NodeSynced:         func() bool { return false },
			},
			keys:          []string{"default/dsj1", "default/dsj2", "default/dsj3", "default/last"},
			wantErr:       true,
			WantErrString: "Timed out waiting for caches to sync",
		},
		{
			name: "error during sync",
			fields: fields{
				JobSynced:          func() bool { return true },
				DaemonSetJobSynced: func() bool { return true },
				NodeSynced:         func() bool { return true },
			},
			keys:          []string{"default/dsj1", "default/dsj2", "default/error", "default/last"},
			wantErr:       true,
			WantErrString: "DaemonSetJobController.sync - DaemonSetJob default/error not valid",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := wclient.NewDaemonSetJobClient(restConfig)
			if err != nil {
				t.Fatalf("%s:%v", tt.name, err)
			}
			kubeClient := clientset.NewForConfigOrDie(restConfig)

			d, _, _ := newDaemonSetJobControllerFromClients(client, kubeClient)
			d.JobSynced = tt.fields.JobSynced
			d.DaemonSetJobSynced = tt.fields.DaemonSetJobSynced
			d.NodeSynced = tt.fields.NodeSynced
			d.syncHandler = func(key string) error {
				if key == "default/last" {
					d.queue.ShutDown()
				} else if key == "default/error" {
					return fmt.Errorf(tt.WantErrString)
				}
				return nil
			}
			for _, k := range tt.keys {
				d.queue.Add(k)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = d.Run(ctx)
			if (err != nil) != tt.wantErr && err != context.DeadlineExceeded {
				t.Errorf("DaemonSetJobController.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != context.DeadlineExceeded && err.Error() != tt.WantErrString {
				t.Errorf("DaemonSetJobController.Run() error msg: '%s', want err msg: '%s'", err.Error(), tt.WantErrString)
			}
		})
	}
}

func newDaemonSetJobControllerFromClients(daemonsetjobClient dclientset.Interface, kubeClient clientset.Interface) (*DaemonSetJobController, kubeinformers.SharedInformerFactory, winformers.SharedInformerFactory) {
	kubeInformers := kubeinformers.NewSharedInformerFactory(kubeClient, 0)
	workflowInformers := winformers.NewSharedInformerFactory(daemonsetjobClient, 0)
	wc := NewDaemonSetJobController(daemonsetjobClient, kubeClient, kubeInformers, workflowInformers)
	wc.JobControl = &FakeJobControl{}
	wc.JobSynced = func() bool { return true }
	wc.NodeSynced = func() bool { return true }

	return wc, kubeInformers, workflowInformers
}

func Test_updateDaemonSetJobStatusConditions(t *testing.T) {
	now := metav1.Now()
	old := metav1.Now()
	old.Add(-time.Minute)
	type args struct {
		status    *dapi.DaemonSetJobStatus
		now       metav1.Time
		completed bool
		failed    bool
	}
	tests := []struct {
		name       string
		args       args
		wantStatus *dapi.DaemonSetJobStatus
	}{

		{
			name: "new completed condition",
			args: args{
				status: &dapi.DaemonSetJobStatus{
					Conditions: nil,
				},
				now:       now,
				completed: true,
			},
			wantStatus: &dapi.DaemonSetJobStatus{
				Conditions: []dapi.DaemonSetJobCondition{
					newDaemonSetJobStatusCondition(dapi.DaemonSetJobComplete, now, "", ""),
				},
				CompletionTime: &now,
			},
		},
		{
			name: "update completed condition",
			args: args{
				status: &dapi.DaemonSetJobStatus{
					Conditions: []dapi.DaemonSetJobCondition{
						newDaemonSetJobStatusCondition(dapi.DaemonSetJobComplete, old, "", ""),
					},
					CompletionTime: &old,
				},
				now:       now,
				completed: true,
			},
			wantStatus: &dapi.DaemonSetJobStatus{
				Conditions: []dapi.DaemonSetJobCondition{
					{
						Type:               dapi.DaemonSetJobComplete,
						Status:             apiv1.ConditionTrue,
						LastProbeTime:      now,
						LastTransitionTime: old,
					},
				},
				CompletionTime: &old,
			},
		},
		{
			name: "update uncompleted condition",
			args: args{
				status: &dapi.DaemonSetJobStatus{
					Conditions: []dapi.DaemonSetJobCondition{
						{
							Type:               dapi.DaemonSetJobComplete,
							Status:             apiv1.ConditionFalse,
							LastProbeTime:      old,
							LastTransitionTime: old,
						},
					},
				},
				now:       now,
				completed: true,
			},
			wantStatus: &dapi.DaemonSetJobStatus{
				Conditions: []dapi.DaemonSetJobCondition{
					{
						Type:               dapi.DaemonSetJobComplete,
						Status:             apiv1.ConditionTrue,
						LastProbeTime:      now,
						LastTransitionTime: now,
					},
				},
				CompletionTime: &now,
			},
		},
		{
			name: "completed but failed condition",
			args: args{
				status: &dapi.DaemonSetJobStatus{
					Conditions: []dapi.DaemonSetJobCondition{},
				},
				now:       now,
				completed: true,
				failed:    true,
			},
			wantStatus: &dapi.DaemonSetJobStatus{
				Conditions: []dapi.DaemonSetJobCondition{
					{
						Type:               dapi.DaemonSetJobComplete,
						Status:             apiv1.ConditionTrue,
						LastProbeTime:      now,
						LastTransitionTime: now,
					},
					{
						Type:               dapi.DaemonSetJobFailed,
						Status:             apiv1.ConditionTrue,
						LastProbeTime:      now,
						LastTransitionTime: now,
					},
				},
				CompletionTime: &now,
			},
		},
		{
			name: "to not completed but failed condition",
			args: args{
				status: &dapi.DaemonSetJobStatus{
					Conditions: []dapi.DaemonSetJobCondition{
						{
							Type:               dapi.DaemonSetJobComplete,
							Status:             apiv1.ConditionTrue,
							LastProbeTime:      old,
							LastTransitionTime: old,
						},
					},
				},
				now:       now,
				completed: false,
				failed:    true,
			},
			wantStatus: &dapi.DaemonSetJobStatus{
				Conditions: []dapi.DaemonSetJobCondition{
					{
						Type:               dapi.DaemonSetJobComplete,
						Status:             apiv1.ConditionFalse,
						LastProbeTime:      now,
						LastTransitionTime: now,
					},
					{
						Type:               dapi.DaemonSetJobFailed,
						Status:             apiv1.ConditionTrue,
						LastProbeTime:      now,
						LastTransitionTime: now,
					},
				},
				CompletionTime: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateDaemonSetJobStatusConditions(tt.args.status, tt.args.now, tt.args.completed, tt.args.failed)
			if !reflect.DeepEqual(tt.wantStatus, tt.args.status) {
				t.Errorf("status wanted: %#v but got %#v", tt.wantStatus, tt.args.status)
			}
		})
	}
}

func TestDaemonSetJobController_sync(t *testing.T) {
	restConfig := &rest.Config{Host: "localhost"}
	now := &metav1.Time{time.Now()}

	tests := []struct {
		name string
		// test setup
		startTime           *metav1.Time
		deleting            bool
		tweakDaemonSetJob   func(*dapi.DaemonSetJob) *dapi.DaemonSetJob
		customUpdateHandler func(*dapi.DaemonSetJob) error // custom update func
		// jobs setup
		activeJobs    int32
		succeededJobs int32
		failedJobs    int32
		deleteJobs    int32
		hashDeleteJob string
		nbNodes       int32
		// expectations
		wantErr            bool
		wantActiveJobs     int32
		wantSucceededJobs  int32
		wantFailedJobs     int32
		wantCompletionTime bool
	}{

		{
			name:       "daemonsetjob defaulted",
			startTime:  nil,
			deleting:   false,
			activeJobs: 0, succeededJobs: 0, failedJobs: 0,
			customUpdateHandler: func(d *dapi.DaemonSetJob) error {
				if !dapi.IsDaemonSetJobDefaulted(d) {
					return fmt.Errorf("daemonsetjob %q not defaulted", d.Name)
				}
				return nil
			},
		},
		{
			name:       "daemonsetjob validated",
			startTime:  nil,
			deleting:   false,
			activeJobs: 0, succeededJobs: 0, failedJobs: 0,
			customUpdateHandler: func(d *dapi.DaemonSetJob) error {
				errs := dapi.ValidateDaemonSetJob(d)
				if len(errs) > 0 {
					return fmt.Errorf("daemonsetjob %q not valid", d.Name)
				}
				return nil
			},
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
		},

		{
			name:       "daemonsetjob jobs running",
			startTime:  now,
			deleting:   false,
			activeJobs: 3, succeededJobs: 0, failedJobs: 0,
			nbNodes: 3,
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
			wantActiveJobs: 3,
		},
		{
			name:       "daemonsetjob jobs succeed",
			startTime:  now,
			deleting:   false,
			activeJobs: 0, succeededJobs: 3, failedJobs: 0,
			nbNodes: 3,
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
			wantSucceededJobs:  3,
			wantCompletionTime: true,
		},
		{
			name:       "daemonsetjob jobs failed",
			startTime:  now,
			deleting:   false,
			activeJobs: 0, succeededJobs: 0, failedJobs: 3,
			nbNodes: 3,
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
			wantFailedJobs:     3,
			wantCompletionTime: true,
		},
		{
			name:       "daemonsetjob jobs not completly done",
			startTime:  now,
			deleting:   false,
			activeJobs: 2, succeededJobs: 1, failedJobs: 0,
			nbNodes: 3,
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
			wantActiveJobs:     2,
			wantSucceededJobs:  1,
			wantCompletionTime: false,
		},
		{
			name:       "daemonsetjob missing job",
			startTime:  now,
			deleting:   false,
			activeJobs: 2, succeededJobs: 1, failedJobs: 0,
			nbNodes: 4,
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
			wantActiveJobs:     2,
			wantSucceededJobs:  1,
			wantCompletionTime: false,
		},
		{
			name:       "daemonsetjob job to be deleted",
			startTime:  now,
			deleting:   false,
			activeJobs: 2, succeededJobs: 1, failedJobs: 0,
			deleteJobs: 1, hashDeleteJob: "oldHash",
			nbNodes: 3,
			tweakDaemonSetJob: func(d *dapi.DaemonSetJob) *dapi.DaemonSetJob {
				return dapi.DefaultDaemonSetJob(d)
			},
			wantActiveJobs:     2,
			wantSucceededJobs:  1,
			wantCompletionTime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			client, err := wclient.NewDaemonSetJobClient(restConfig)
			if err != nil {
				t.Fatalf("%s:%v", tt.name, err)
			}
			kubeClient := clientset.NewForConfigOrDie(restConfig)

			d, kubeInformerFactory, workflowInformerFactory := newDaemonSetJobControllerFromClients(client, kubeClient)

			daemonsetjob := newDaemonSetJob(tt.startTime, nil, 0, 0, 0)
			key, err := cache.MetaNamespaceKeyFunc(daemonsetjob)
			if err != nil {
				t.Fatalf("%s - unable to get key from daemonsetjob:%v", tt.name, err)
			}
			if tt.deleting {
				now := metav1.Now()
				daemonsetjob.DeletionTimestamp = &now
			}
			if tt.tweakDaemonSetJob != nil {
				daemonsetjob = tt.tweakDaemonSetJob(daemonsetjob)
			}
			workflowInformerFactory.Daemonsetjob().V1().DaemonSetJobs().Informer().GetStore().Add(daemonsetjob)

			nodeIndexer := kubeInformerFactory.Core().V1().Nodes().Informer().GetIndexer()
			for _, node := range newNodeList(tt.nbNodes) {
				nodeIndexer.Add(node.DeepCopy())
			}

			nodeSelector := labels.Set{}

			nodes, _ := d.NodeLister.List(nodeSelector.AsSelector())
			t.Logf("Node: %d", len(nodes))

			jobIndexer := kubeInformerFactory.Batch().V1().Jobs().Informer().GetIndexer()
			for _, job := range newDSJJobList(tt.activeJobs, 0, 0, "", daemonsetjob) {
				jobIndexer.Add(job.DeepCopy())
			}
			for _, job := range newDSJJobList(tt.succeededJobs, tt.activeJobs, tt.activeJobs, batch.JobComplete, daemonsetjob) {
				jobIndexer.Add(job.DeepCopy())
			}
			for _, job := range newDSJJobList(tt.failedJobs, tt.activeJobs+tt.succeededJobs, tt.activeJobs+tt.succeededJobs, batch.JobFailed, daemonsetjob) {
				jobIndexer.Add(job.DeepCopy())
			}
			for _, job := range newDSJJobList(tt.deleteJobs, tt.activeJobs+tt.succeededJobs+tt.failedJobs, 1, batch.JobComplete, daemonsetjob) {
				job.Annotations = map[string]string{DaemonSetJobMD5AnnotationKey: tt.hashDeleteJob}
				jobIndexer.Add(job.DeepCopy())
			}

			if tt.customUpdateHandler != nil {
				d.updateHandler = tt.customUpdateHandler
			} else {
				d.updateHandler = func(dsj *dapi.DaemonSetJob) error {
					// update workflow in store only
					if err := workflowInformerFactory.Daemonsetjob().V1().DaemonSetJobs().Informer().GetStore().Update(dsj); err != nil {
						t.Errorf("%s - %v", tt.name, err)
					}
					return nil
				}
			}

			// run sync twice: first time to update workflow steps, second to update status
			if err := d.sync(key); (err != nil) != tt.wantErr {
				t.Errorf("DaemonSetJobController.sync() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := d.sync(key); (err != nil) != tt.wantErr {
				t.Errorf("DaemonSetJobController.sync() error = %v, wantErr %v", err, tt.wantErr)
			}

			dsj, err := workflowInformerFactory.Daemonsetjob().V1().DaemonSetJobs().Lister().DaemonSetJobs(daemonsetjob.Namespace).Get(daemonsetjob.Name)
			if err != nil {
				t.Errorf("unable to retrieve the daemonsetjob, err:%v", err)
			}
			status := dsj.Status
			t.Logf("status:%v", status)
			if status.Active != tt.wantActiveJobs {
				t.Errorf("Status.Active:%d but want %d", status.Active, tt.wantActiveJobs)
			}
			if status.Failed != tt.wantFailedJobs {
				t.Errorf("Status.Failed:%d but want %d", status.Failed, tt.wantFailedJobs)
			}
			if status.Succeeded != tt.wantSucceededJobs {
				t.Errorf("Status.Succeeded:%d but want %d", status.Succeeded, tt.wantSucceededJobs)
			}
			if tt.wantCompletionTime && status.CompletionTime == nil {
				t.Errorf("Status.CompletionTime == nil, but want != nil")
			}
		})
	}
}

// create count jobs with the given state (Active, Complete, Failed) for the given daemonsetjob
func newDSJJobList(count int32, fromIndex int32, nodeIndex int32, status batch.JobConditionType, daemonsetjob *dapi.DaemonSetJob) []batch.Job {
	var succeededPods, failedPods, activePods int32
	var condition batch.JobCondition
	switch status {
	case batch.JobComplete:
		succeededPods = 1
		condition.Type = batch.JobComplete
		condition.Status = apiv1.ConditionTrue
	case batch.JobFailed:
		failedPods = 1
		condition.Type = batch.JobFailed
		condition.Status = apiv1.ConditionTrue
	default:
		activePods = 1
	}
	jobs := []batch.Job{}

	hash, _ := generateMD5JobSpec(&daemonsetjob.Spec.JobTemplate.Spec)

	for i := fromIndex; i < count+fromIndex; i++ {
		// set step name
		labelset, _ := getJobLabelsSetFromDaemonSetJob(daemonsetjob, daemonsetjob.Spec.JobTemplate)
		labels := map[string]string{}
		for k, v := range labelset {
			labels[k] = v
		}
		// create Job
		newJob := batch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%d", daemonsetjob.Name, i),
				Labels:    labels,
				Namespace: daemonsetjob.Namespace,
				SelfLink:  "/apiv1s/extensions/v1beta1/namespaces/default/jobs/job",
			},
			Spec: batch.JobSpec{
				Template: apiv1.PodTemplateSpec{
					Spec: apiv1.PodSpec{
						NodeName: fmt.Sprintf("node-%d", i),
					},
				},
			},
			Status: batch.JobStatus{
				Conditions: []batch.JobCondition{condition},
				Active:     activePods,
				Failed:     failedPods,
				Succeeded:  succeededPods,
			},
		}

		newJob.Annotations = map[string]string{DaemonSetJobMD5AnnotationKey: hash}
		jobs = append(jobs, newJob)
		nodeIndex++
	}
	return jobs
}

func newNodeList(count int32) []apiv1.Node {
	nodes := []apiv1.Node{}
	for i := int32(0); i < count; i++ {
		newNode := apiv1.Node{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "",
				Kind:       "Node",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("node-%d", i),
				Namespace: apiv1.NamespaceDefault,
			},
		}

		nodes = append(nodes, newNode)
	}
	return nodes
}

// utility function to create a basic Workflow with steps
func newDaemonSetJob(startTime *metav1.Time, nodeSelector *metav1.LabelSelector, successed, failed, active int32) *dapi.DaemonSetJob {
	return &dapi.DaemonSetJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: daemonsetjob.GroupName + "/v1",
			Kind:       "DaemonSetJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mydsj",
			Namespace: apiv1.NamespaceDefault,
		},
		Spec: dapi.DaemonSetJobSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"daemonsetjob": "example-selector",
				},
			},
			NodeSelector: nodeSelector,
			JobTemplate:  utiltesting.ValidFakeTemplateSpec(),
		},
		Status: dapi.DaemonSetJobStatus{
			StartTime: startTime,
			Succeeded: successed,
			Failed:    failed,
			Active:    active,
		},
	}
}
