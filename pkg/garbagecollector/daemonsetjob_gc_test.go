package garbagecollector

import (
	"fmt"
	"net/http"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	kclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	wclientset "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	wfake "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/fake"
	wlisters "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/daemonsetjob/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
)

func TestDaemonSetJobGarbageCollector_CollectDaemonSetJobJobs(t *testing.T) {
	type fields struct {
		KubeClient         kclientset.Interface
		DaemonSetJobClient wclientset.Interface
		DaemonSetJobLister wlisters.DaemonSetJobLister
		DaemonSetJobSynced cache.InformerSynced
	}
	tests := []struct {
		name                  string
		fields                fields
		TweakGarbageCollector func(*DaemonSetJobGarbageCollector)
		wantErr               bool
		WantErrString         string
	}{
		{
			name: "no jobs",
			fields: fields{
				KubeClient:         fake.NewSimpleClientset(),
				DaemonSetJobClient: wfake.NewSimpleClientset(),
				DaemonSetJobLister: FakeDaemonSetJobLister{},
				DaemonSetJobSynced: func() bool { return true },
			},
			wantErr: false,
		},
		{
			name: "nominal case",
			fields: fields{
				DaemonSetJobClient: wfake.NewSimpleClientset(),
				DaemonSetJobSynced: func() bool { return true },
			},
			TweakGarbageCollector: func(gc *DaemonSetJobGarbageCollector) {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					jobs := &batchv1.JobList{
						Items: []batchv1.Job{
							createJobWithLabels("testJob1", map[string]string{
								controller.DaemonSetJobLabelKey: "daemonsetjob1",
							}),
							createJobWithLabels("testJob2", map[string]string{
								controller.DaemonSetJobLabelKey: "daemonsetjob1",
							}),
							createJobWithLabels("testJob3", map[string]string{
								controller.DaemonSetJobLabelKey: "daemonsetjob2",
							}),
							createJobWithLabels("testJob4", map[string]string{
								controller.DaemonSetJobLabelKey: "daemonsetjob2",
							}),
						},
					}
					return true, jobs, nil
				})
				gc.KubeClient = fakeClient
				gc.DaemonSetJobLister = &FakeDaemonSetJobLister{
					daemonsetjobs: []*v1.DaemonSetJob{
						createDaemonSetJob("daemonsetjob2"),
					},
				}

			},
			wantErr: false,
		},
		{
			name: "error getting jobs",
			fields: fields{
				DaemonSetJobClient: wfake.NewSimpleClientset(),
				DaemonSetJobSynced: func() bool { return true },
			},
			TweakGarbageCollector: func(gc *DaemonSetJobGarbageCollector) {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
				})
				gc.KubeClient = fakeClient
			},
			wantErr:       true,
			WantErrString: "unable to list daemonsetjobs jobs to be collected: jobs.batch \"jobs\" not found",
		},
		{
			name: "no daemonsetjob found for job",
			fields: fields{
				DaemonSetJobClient: wfake.NewSimpleClientset(),
				DaemonSetJobSynced: func() bool { return true },
				DaemonSetJobLister: FakeDaemonSetJobLister{},
			},
			TweakGarbageCollector: func(gc *DaemonSetJobGarbageCollector) {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					j := createJobWithLabels("testjob", map[string]string{
						controller.DaemonSetJobLabelKey: "foo",
					})
					var jobs runtime.Object
					jobs = &batchv1.JobList{
						Items: []batchv1.Job{
							j,
						}}
					return true, jobs, nil
				})
				gc.KubeClient = fakeClient
			},
			wantErr: false,
		},
		{
			name: "error getting job with label without value",
			fields: fields{
				DaemonSetJobClient: wfake.NewSimpleClientset(),
				DaemonSetJobSynced: func() bool { return true },
				DaemonSetJobLister: FakeDaemonSetJobLister{},
			},
			TweakGarbageCollector: func(gc *DaemonSetJobGarbageCollector) {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					j := createJobWithLabels("testjob", map[string]string{
						controller.DaemonSetJobLabelKey: "",
					})
					var jobs runtime.Object
					jobs = &batchv1.JobList{
						Items: []batchv1.Job{
							j,
						}}
					return true, jobs, nil
				})
				gc.KubeClient = fakeClient
			},
			wantErr:       true,
			WantErrString: "unable to find daemonsetjob name for job: /testjob",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &DaemonSetJobGarbageCollector{
				KubeClient:         tt.fields.KubeClient,
				DaemonSetJobClient: tt.fields.DaemonSetJobClient,
				DaemonSetJobLister: tt.fields.DaemonSetJobLister,
				DaemonSetJobSynced: tt.fields.DaemonSetJobSynced,
			}
			if tt.TweakGarbageCollector != nil {
				tt.TweakGarbageCollector(c)
			}

			err := c.CollectDaemonSetJobJobs()
			if (err != nil) != tt.wantErr {
				t.Errorf("DaemonSetJobGarbageCollector.CollectDaemonSetJobJobs() error = %v, wantErr %v", err, tt.wantErr)
				if err.Error() != tt.WantErrString {
					t.Errorf("DaemonSetJobGarbageCollector.CollectDaemonSetJobJobs() error msg: '%s', want err msg: '%s'", err.Error(), tt.WantErrString)
				}
			}

		})
	}
}

type FakeDaemonSetJobLister struct {
	daemonsetjobs []*v1.DaemonSetJob
}

func (f FakeDaemonSetJobLister) List(labels.Selector) (ret []*v1.DaemonSetJob, err error) {
	return f.daemonsetjobs, nil
}

func (f FakeDaemonSetJobLister) DaemonSetJobs(namespace string) wlisters.DaemonSetJobNamespaceLister {
	return &FakeDaemonSetJobNamespaceLister{
		namespace: namespace,
		Lister:    f,
	}
}

type FakeDaemonSetJobNamespaceLister struct {
	namespace string
	Lister    FakeDaemonSetJobLister
}

func (f FakeDaemonSetJobNamespaceLister) List(labels.Selector) (ret []*v1.DaemonSetJob, err error) {
	return []*v1.DaemonSetJob{}, nil
}

func (f FakeDaemonSetJobNamespaceLister) Get(name string) (*v1.DaemonSetJob, error) {
	for i := range f.Lister.daemonsetjobs {
		if f.Lister.daemonsetjobs[i].Name == name {
			return f.Lister.daemonsetjobs[i], nil
		}
	}
	return nil, newDaemonSetJobNotFoundError(name)
}

func createDaemonSetJob(name string) *v1.DaemonSetJob {
	return &v1.DaemonSetJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newDaemonSetJobNotFoundError(name string) *apierrors.StatusError {
	return &apierrors.StatusError{
		ErrStatus: metav1.Status{
			Status: metav1.StatusFailure,
			Code:   http.StatusNotFound,
			Reason: metav1.StatusReasonNotFound,
			Details: &metav1.StatusDetails{
				Group: workflow.GroupName,
				Kind:  v1.ResourcePlural + "." + workflow.GroupName,
				Name:  name,
			},
			Message: fmt.Sprintf("%s %q not found", v1.ResourcePlural+"."+workflow.GroupName, name),
		}}
}
