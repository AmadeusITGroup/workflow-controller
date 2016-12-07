/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workflow

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/extensions"

	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/controller"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	wclient "github.com/sdminonne/workflow-controller/pkg/client"
)

var alwaysReady = func() bool { return true }

// utility function to create a JobTemplateSpec
func newJobTemplateSpec() *batch.JobTemplateSpec {
	return &batch.JobTemplateSpec{
		ObjectMeta: api.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{Image: "foo/bar"},
					},
				},
			},
		},
	}
}

func newJobTemplateStatus() wapi.WorkflowStepStatus {
	return wapi.WorkflowStepStatus{
		Name:     "myJob",
		Complete: false,
		Reference: api.ObjectReference{
			Kind:      "Job",
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
	}
}

func getKey(w *wapi.Workflow, t *testing.T) string {
	key, err := controller.KeyFunc(w)
	if err != nil {
		t.Errorf("Unexpected error getting key for workflow: %v", err)
		return ""
	}
	return key
}

func TestControllerSyncWorkflow(t *testing.T) {
	now := unversioned.Now()
	testCases := map[string]struct {
		workflow           *wapi.Workflow
		jobs               []batch.Job
		checkWorkflow      func(testName string, workflow *wapi.Workflow, t *testing.T)
		expectedStartedJob int
	}{
		"workflow start": {
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "myJob",
							JobTemplate: newJobTemplateSpec(),
						},
					},
				},
			},
			jobs:               []batch.Job{},
			expectedStartedJob: 1,
		},
		"workflow status update": {
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "myJob",
							JobTemplate: newJobTemplateSpec(),
						},
					},
				},
				Status: wapi.WorkflowStatus{
					Conditions: []wapi.WorkflowCondition{},
					Statuses: []wapi.WorkflowStepStatus{
						newJobTemplateStatus(),
					},
				},
			},
			jobs: []batch.Job{
				{
					ObjectMeta: api.ObjectMeta{
						Name:      "foo",
						Namespace: api.NamespaceDefault,
						Labels: map[string]string{
							"foo":                "bar",
							WorkflowStepLabelKey: "myJob",
						},
						SelfLink: "/apis/v1/jobs/foo",
					},
					Spec:   batch.JobSpec{},
					Status: batch.JobStatus{},
				},
			},
			checkWorkflow: func(testName string, workflow *wapi.Workflow, t *testing.T) {
				stepStatus := workflow.GetStepStatusByName("myJob")
				if stepStatus == nil {
					t.Errorf("%s, Workflow step not updated", testName)
					return
				}
				if stepStatus.Complete {
					t.Errorf("%s, Workflow wrongly updated", testName)
				}
			},
			expectedStartedJob: 0,
		},

		"workflow step status update to complete": {
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "myJob",
							JobTemplate: newJobTemplateSpec(),
						},
					},
				},
				Status: wapi.WorkflowStatus{
					StartTime:  &now,
					Conditions: []wapi.WorkflowCondition{},
					Statuses: []wapi.WorkflowStepStatus{
						newJobTemplateStatus(),
					},
				},
			},
			jobs: []batch.Job{
				{
					ObjectMeta: api.ObjectMeta{
						Name:      "foo",
						Namespace: api.NamespaceDefault,
						Labels: map[string]string{
							"foo":                "bar",
							WorkflowStepLabelKey: "myJob",
						},
						SelfLink: "/apis/v1/jobs/foo",
					},
					Spec: batch.JobSpec{},
					Status: batch.JobStatus{
						Conditions: []batch.JobCondition{
							{
								Type:   batch.JobComplete,
								Status: api.ConditionTrue,
							},
						},
					},
				},
			},
			checkWorkflow: func(testName string, workflow *wapi.Workflow, t *testing.T) {
				stepStatus := workflow.GetStepStatusByName("myJob")
				if stepStatus == nil {
					t.Errorf("%s, Workflow step not updated", testName)
					return
				}
				if !stepStatus.Complete {
					t.Errorf("%s, Workflow wrongly updated", testName)
				}
			},
			expectedStartedJob: 0,
		},

		"workflow status update to complete": {
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "myJob",
							JobTemplate: newJobTemplateSpec(),
						},
					},
				},
				Status: wapi.WorkflowStatus{
					StartTime:  &now,
					Conditions: []wapi.WorkflowCondition{},
					Statuses: []wapi.WorkflowStepStatus{
						{
							Name:     "myJob",
							Complete: true,
							Reference: api.ObjectReference{
								Kind:      "Job",
								Name:      "foo",
								Namespace: api.NamespaceDefault,
							},
						},
					},
				},
			},
			jobs: []batch.Job{}, // jobs no retrieved step only
			checkWorkflow: func(testName string, workflow *wapi.Workflow, t *testing.T) {
				if !IsWorkflowFinished(workflow) {
					t.Errorf("%s, Workflow should be finished:\n %#v", testName, workflow)
				}
				if workflow.Status.CompletionTime == nil {
					t.Errorf("%s, CompletionTime not set", testName)
				}
			},
			expectedStartedJob: 0,
		},
		"workflow step dependency complete 3": {
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "one",
							JobTemplate: newJobTemplateSpec(),
						},
						{
							Name:         "two",
							JobTemplate:  newJobTemplateSpec(),
							Dependencies: []string{"one"},
						},
						{
							Name:         "three",
							JobTemplate:  newJobTemplateSpec(),
							Dependencies: []string{"one"},
						},
						{
							Name:         "four",
							JobTemplate:  newJobTemplateSpec(),
							Dependencies: []string{"one"},
						},
						{
							Name:         "five",
							JobTemplate:  newJobTemplateSpec(),
							Dependencies: []string{"two", "three", "four"},
						},
					},
				},
				Status: wapi.WorkflowStatus{
					StartTime:  &now,
					Conditions: []wapi.WorkflowCondition{},
					Statuses: []wapi.WorkflowStepStatus{
						{
							Name:     "one",
							Complete: true,
							Reference: api.ObjectReference{
								Kind:      "Job",
								Name:      "foo",
								Namespace: api.NamespaceDefault,
							},
						},
					},
				},
			},
			jobs: []batch.Job{
				{
					ObjectMeta: api.ObjectMeta{
						Name:      "foo",
						Namespace: api.NamespaceDefault,
						Labels: map[string]string{
							"foo":                "bar",
							WorkflowStepLabelKey: "one",
						},
						SelfLink: "/apis/v1/jobs/foo",
					},
					Spec:   batch.JobSpec{},
					Status: batch.JobStatus{},
				},
			},
			checkWorkflow:      func(testName string, workflow *wapi.Workflow, t *testing.T) {},
			expectedStartedJob: 3,
		},
		"workflow step dependency not complete": {
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "one",
							JobTemplate: newJobTemplateSpec(),
						},
						{
							Name:         "two",
							JobTemplate:  newJobTemplateSpec(),
							Dependencies: []string{"one"},
						},
					},
				},
				Status: wapi.WorkflowStatus{
					Conditions: []wapi.WorkflowCondition{},
					Statuses: []wapi.WorkflowStepStatus{
						{
							Name:     "one",
							Complete: false,
							Reference: api.ObjectReference{
								Kind:      "Job",
								Name:      "foo",
								Namespace: api.NamespaceDefault,
							},
						},
					},
				},
			},
			jobs: []batch.Job{
				{
					ObjectMeta: api.ObjectMeta{
						Name:      "foo",
						Namespace: api.NamespaceDefault,
						Labels: map[string]string{
							"foo":                "bar",
							WorkflowStepLabelKey: "one",
						},
						SelfLink: "/apis/v1/jobs/foo",
					},
					Spec:   batch.JobSpec{},
					Status: batch.JobStatus{},
				},
			},
			checkWorkflow:      func(testName string, workflow *wapi.Workflow, t *testing.T) {},
			expectedStartedJob: 0,
		},
	}
	for name, tc := range testCases {
		kubeconfig, _ := clientcmd.BuildConfigFromFlags("localhost", "")
		workflowResource := &extensions.ThirdPartyResource{
			ObjectMeta: api.ObjectMeta{
				Name: "workflow.mydomain.com",
			},
			Versions: []extensions.APIVersion{
				extensions.APIVersion{Name: "v1"},
			},
		}

		wfClient := wclient.NewForConfigOrDie(workflowResource, kubeconfig)
		kubeClient := fake.NewSimpleClientset()
		controller := NewController(kubeClient, wfClient, workflowResource, controller.NoResyncPeriodFunc)
		fakeJobControl := FakeJobControl{}
		controller.jobControl = &fakeJobControl
		controller.jobStoreSynced = alwaysReady
		var actual *wapi.Workflow
		controller.updateHandler = func(workflow *wapi.Workflow) error {
			actual = workflow
			return nil
		}
		// setup workflow, jobs
		controller.workflowStore.Store.Add(tc.workflow)
		for _, job := range tc.jobs {
			controller.jobStore.Store.Add(&job)
		}
		if err := controller.syncWorkflow(getKey(tc.workflow, t)); err != nil {
			t.Errorf("%s: unexpected error syncing workflow %v", name, err)
			continue
		}
		if len(fakeJobControl.CreatedJobTemplates) != tc.expectedStartedJob {
			t.Errorf("%s: unexpected # of created jobs: expected %d got %d", name, tc.expectedStartedJob, len(fakeJobControl.CreatedJobTemplates))
			continue

		}
		if tc.checkWorkflow != nil {
			tc.checkWorkflow(name, actual, t)
		}
	}
}

func TestSyncWorkflowPastDeadline(t *testing.T) {
	now := unversioned.Now()
	testCases := map[string]struct {
		startTime             int64
		activeDeadlineSeconds int64
		workflow              *wapi.Workflow
		jobs                  []batch.Job
		checkWorkflow         func(testName string, workflow *wapi.Workflow, t *testing.T)
	}{
		"activeDeadllineSeconds one": {
			startTime:             10,
			activeDeadlineSeconds: 5,
			workflow: &wapi.Workflow{
				TypeMeta: unversioned.TypeMeta{
					APIVersion: "example.com/v1",
					Kind:       "Workflow",
				},
				ObjectMeta: api.ObjectMeta{
					Name:      "mydag",
					Namespace: api.NamespaceDefault,
					SelfLink:  "/apis/v1/workflows/mydag",
				},
				Spec: wapi.WorkflowSpec{
					Steps: []wapi.WorkflowStep{
						{
							Name:        "myJob",
							JobTemplate: newJobTemplateSpec(),
						},
					},
				},
				Status: wapi.WorkflowStatus{
					StartTime:  &now,
					Conditions: []wapi.WorkflowCondition{},
					Statuses: []wapi.WorkflowStepStatus{
						newJobTemplateStatus(),
					},
				},
			},
			jobs: []batch.Job{
				{
					ObjectMeta: api.ObjectMeta{
						Name:      "foo",
						Namespace: api.NamespaceDefault,
						Labels: map[string]string{
							"foo":                "bar",
							WorkflowStepLabelKey: "myJob",
						},
						SelfLink: "/apis/v1/jobs/foo",
					},
					Spec:   batch.JobSpec{},
					Status: batch.JobStatus{},
				},
			},
			checkWorkflow: func(testName string, workflow *wapi.Workflow, t *testing.T) {
				if !IsWorkflowFinished(workflow) {
					t.Errorf("%s, Workflow should be finished:\n %#v", testName, workflow)
				}
				if workflow.Status.CompletionTime == nil {
					t.Errorf("%s, CompletionTime not set", testName)
				}
			},
		},
	}
	for name, tc := range testCases {

		kubeconfig, _ := clientcmd.BuildConfigFromFlags("localhost", "")
		workflowResource := &extensions.ThirdPartyResource{
			ObjectMeta: api.ObjectMeta{
				Name: "workflow.mydomain.com",
			},
			Versions: []extensions.APIVersion{
				extensions.APIVersion{Name: "v1"},
			},
		}
		wfClient := wclient.NewForConfigOrDie(workflowResource, kubeconfig)
		kubeClient := fake.NewSimpleClientset()
		manager := NewController(kubeClient, wfClient, workflowResource, controller.NoResyncPeriodFunc)
		fakeJobControl := FakeJobControl{}
		manager.jobControl = &fakeJobControl
		manager.jobStoreSynced = alwaysReady

		var actual *wapi.Workflow
		manager.updateHandler = func(workflow *wapi.Workflow) error {
			actual = workflow
			return nil
		}
		startTime := unversioned.Unix(unversioned.Now().Time.Unix()-tc.startTime, 0)
		tc.workflow.Status.StartTime = &startTime
		tc.workflow.Spec.ActiveDeadlineSeconds = &tc.activeDeadlineSeconds
		manager.workflowStore.Store.Add(tc.workflow)
		for _, job := range tc.jobs {
			manager.jobStore.Store.Add(&job)
		}
		if err := manager.syncWorkflow(getKey(tc.workflow, t)); err != nil {
			t.Errorf("%s: unexpected error syncing workflow %v", name, err)
			continue
		}
	}
}

/*
func TestSyncWorkflowDelete(t *testing.T) {
	// @sdminonne: TODO
}
*/
/*
type FakeWatcher struct {
	w *watch.FakeWatcher
	*testclient.Fake
}

func TestWatchWorkflows(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()
	//kubeClient.PrependWatchReactor("workflows", core.DefaultWatchReactor(fakeWatch, nil))
	kubeconfig, _ := clientcmd.BuildConfigFromFlags("localhost", "")
	workflowResource := &extensions.ThirdPartyResource{
		ObjectMeta: api.ObjectMeta{
			Name: "workflow.mydomain.com",
		},
		Versions: []extensions.APIVersion{
			extensions.APIVersion{Name: "v1"},
		},
	}
	//wfClient := wclient.NewForConfigOrDie(workflowResource, kubeconfig)
	wfClient := wclient.NewFakeClient()
	manager := NewController(kubeClient, wfclient, workflowResource, controller.NoResyncPeriodFunc)
	fakeJobControl := FakeJobControl{}
	manager.jobControl = &fakeJobControl
	manager.jobStoreSynced = alwaysReady
	var actual *wapi.Workflow
	manager.updateHandler = func(workflow *wapi.Workflow) error {
		actual = workflow
		return nil
	}

	var u wapi.Workflow
	received := make(chan struct{})
	manager.syncHandler = func(key string) error {
		obj, exists, err := manager.workflowStore.GetByKey(key)
		if !exists || err != nil {
			t.Errorf("Expected to find workflow under key: %v", key)
		}
		unstruct, ok := obj.(*wapi.Workflow)
		if !ok {
			t.Fatalf("unexpected type: %v %#v", reflect.TypeOf(obj), obj)
		}
		if !api.Semantic.DeepDerivative(u, unstruct) {
			t.Errorf("Expected %#v, but got %#v", u, *unstruct)
		}
		close(received)
		return nil
	}

	stopCh := make(chan struct{})
	defer close(stopCh)
	go manager.Run(1, stopCh)

	// now sends data
	u.SetName("foo")
	fakeWatch.Add(&u)
	t.Log("waiting for workflow to reach synchHandler")
	<-received
}
*/

func TestIsWorkflowFinished(t *testing.T) {
	cases := []struct {
		name     string
		finished bool
		workflow *wapi.Workflow
	}{
		{
			name:     "Complete and True",
			finished: true,
			workflow: &wapi.Workflow{
				Status: wapi.WorkflowStatus{
					Conditions: []wapi.WorkflowCondition{
						{
							Type:   wapi.WorkflowComplete,
							Status: api.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name:     "Failed and True",
			finished: true,
			workflow: &wapi.Workflow{
				Status: wapi.WorkflowStatus{
					Conditions: []wapi.WorkflowCondition{
						{
							Type:   wapi.WorkflowFailed,
							Status: api.ConditionTrue,
						},
					},
				},
			},
		},
		{
			name:     "Complete and False",
			finished: false,
			workflow: &wapi.Workflow{
				Status: wapi.WorkflowStatus{
					Conditions: []wapi.WorkflowCondition{
						{
							Type:   wapi.WorkflowComplete,
							Status: api.ConditionFalse,
						},
					},
				},
			},
		},
		{
			name:     "Failed and False",
			finished: false,
			workflow: &wapi.Workflow{
				Status: wapi.WorkflowStatus{
					Conditions: []wapi.WorkflowCondition{
						{
							Type:   wapi.WorkflowComplete,
							Status: api.ConditionFalse,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		if IsWorkflowFinished(tc.workflow) != tc.finished {
			t.Errorf("%s - Expected %v got %v", tc.name, tc.finished, IsWorkflowFinished(tc.workflow))
		}
	}
}
