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
	"k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	wfake "github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned/fake"
	wlister "github.com/amadeusitgroup/workflow-controller/pkg/client/listers/workflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
)

type FakeWorkflowLister struct {
	workflows []*v1.Workflow
}

func (f FakeWorkflowLister) List(labels.Selector) (ret []*v1.Workflow, err error) {
	return f.workflows, nil
}

func (f FakeWorkflowLister) Workflows(namespace string) wlister.WorkflowNamespaceLister {
	return &FakeWorkflowNamespaceLister{
		namespace: namespace,
		Lister:    f,
	}
}

type FakeWorkflowNamespaceLister struct {
	namespace string
	Lister    FakeWorkflowLister
}

func (f FakeWorkflowNamespaceLister) List(labels.Selector) (ret []*v1.Workflow, err error) {
	return []*v1.Workflow{}, nil
}

func (f FakeWorkflowNamespaceLister) Get(name string) (*v1.Workflow, error) {
	for i := range f.Lister.workflows {
		if f.Lister.workflows[i].Name == name {
			return f.Lister.workflows[i], nil
		}
	}
	return nil, newWorkflowNotFoundError(name)
}

func TestGarbageCollector_CollectWorkflowJobs(t *testing.T) {
	tests := map[string]struct {
		TweakGarbageCollector func(*GarbageCollector) *GarbageCollector
		errorMessage          string
	}{
		"nominal case": {
			// getting 4 jobs:
			//   2 to be removed 2 (no workflow)
			//   2 to be preserved (found workflow)
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					jobs := &batchv1.JobList{
						Items: []batchv1.Job{
							createJobWithLabels("testJob1", map[string]string{
								controller.WorkflowLabelKey: "workflow1",
							}),
							createJobWithLabels("testJob2", map[string]string{
								controller.WorkflowLabelKey: "workflow1",
							}),
							createJobWithLabels("testJob3", map[string]string{
								controller.WorkflowLabelKey: "workflow2",
							}),
							createJobWithLabels("testJob4", map[string]string{
								controller.WorkflowLabelKey: "workflow2",
							}),
						},
					}
					return true, jobs, nil
				})
				gc.KubeClient = fakeClient
				gc.WorkflowLister = &FakeWorkflowLister{
					workflows: []*v1.Workflow{
						createWorkflow("workflow2"),
					},
				}
				return gc
			},
			errorMessage: "",
		},
		"no jobs": {
			errorMessage: "",
		},
		"error getting jobs": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), action.GetResource().Resource)
				})
				gc.KubeClient = fakeClient
				return gc
			},
			errorMessage: "unable to list workflow jobs to be collected: jobs.batch \"jobs\" not found",
		},
		"error getting job with label without value": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					j := createJobWithLabels("testjob", map[string]string{
						controller.WorkflowLabelKey: "",
					})
					var jobs runtime.Object
					jobs = &batchv1.JobList{
						Items: []batchv1.Job{
							j,
						}}
					return true, jobs, nil
				})
				gc.KubeClient = fakeClient
				return gc
			},
			errorMessage: "unable to find workflow name for job: /testjob",
		},
		"no workflow found for job": {
			TweakGarbageCollector: func(gc *GarbageCollector) *GarbageCollector {
				fakeClient := &fake.Clientset{}
				fakeClient.AddReactor("list", "jobs", func(action core.Action) (bool, runtime.Object, error) {
					j := createJobWithLabels("testjob", map[string]string{
						controller.WorkflowLabelKey: "foo",
					})
					var jobs runtime.Object
					jobs = &batchv1.JobList{
						Items: []batchv1.Job{
							j,
						}}
					return true, jobs, nil
				})
				gc.KubeClient = fakeClient
				return gc
			},
			errorMessage: "",
		},
		/*
			"error getting workflow from lister (via Get)":             {},
			"NotFound getting workflow from lister so getting via API": {},
		*/
	}
	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			GC := &GarbageCollector{
				KubeClient:     fake.NewSimpleClientset(),
				WorkflowClient: wfake.NewSimpleClientset(),
				WorkflowLister: FakeWorkflowLister{},
				WorkflowSynced: func() bool { return true },
			}
			if tt.TweakGarbageCollector != nil {
				GC = tt.TweakGarbageCollector(GC)
			}
			err := GC.CollectWorkflowJobs()
			errorMessage := ""
			if err != nil {
				errorMessage = err.Error()
			}
			if tt.errorMessage != errorMessage {
				t.Errorf("%q\nExpected error: `%s`\nBut got       : `%s`\n", tn, tt.errorMessage, errorMessage)
			}
		})
	}
}

func createWorkflow(name string) *v1.Workflow {
	return &v1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newWorkflowNotFoundError(name string) *apierrors.StatusError {
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
