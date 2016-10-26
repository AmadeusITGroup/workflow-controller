/*
Copyright 2016 The Kubernetes Authors.
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

package client

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	kcache "k8s.io/kubernetes/pkg/client/cache"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	wapitesting "github.com/sdminonne/workflow-controller/pkg/api/testing"
)

func TestStoreToWorkflowLister(t *testing.T) {
	store := kcache.NewStore(kcache.MetaNamespaceKeyFunc)
	group := "example.com"
	version := "v1"
	ids := []string{"foo", "bar", "baz"}
	for _, id := range ids {
		w := wapitesting.NewWorkflow(group, version, id, api.NamespaceDefault, nil)
		store.Add(w)
	}

	swl := StoreToWorkflowLister{store}

	wl, err := swl.List()
	if err != nil {
		t.Errorf("unexpected error %q", err.Error())
	}

	if len(wl.Items) != len(ids) {
		t.Errorf("expected %d items got %d", len(ids), len(wl.Items))
	}
}

func TestGetJobWorkflows(t *testing.T) {
	job := &batch.Job{
		ObjectMeta: api.ObjectMeta{
			Name:      "myobject",
			Namespace: api.NamespaceDefault,
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
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
	testcases := map[string]struct {
		WorkflowName   string
		Selector       map[string]string
		Namespace      string
		WorkflowsFound []string
	}{
		"workflow foo": {
			WorkflowName:   "foo",
			Selector:       map[string]string{"foo": "bar"},
			Namespace:      api.NamespaceDefault,
			WorkflowsFound: []string{"foo"},
		},
		"no workflow due to labels": {
			WorkflowName: "foo",
			Selector:     map[string]string{"bar": "foo"},
			Namespace:    api.NamespaceDefault,
		},
		"no worfklows due to namespace": {
			WorkflowName: "foo",
			Selector:     map[string]string{"foo": "bar"},
			Namespace:    "mynamespace",
		},
	}

	group := "example.com"
	version := "v1"

	for name, tc := range testcases {
		store := kcache.NewStore(kcache.MetaNamespaceKeyFunc)
		w := wapitesting.NewWorkflow(group, version, tc.WorkflowName, tc.Namespace, tc.Selector)
		store.Add(w)
		sw := StoreToWorkflowLister{store}
		workflows, err := sw.GetJobWorkflows(job)
		if err != nil {
			t.Errorf("%s - unexpected error: %v", name, err)
		}
		if len(workflows) != len(tc.WorkflowsFound) {
			t.Errorf("%s - expected %d workflows but got %d", name, len(tc.WorkflowsFound), len(workflows))
		}
	}
}

func TestStoreToWorkflowListerExists(t *testing.T) {
	store := kcache.NewStore(kcache.MetaNamespaceKeyFunc)
	w := &wapi.Workflow{
		ObjectMeta: api.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
	}
	store.Add(w)
	swl := StoreToWorkflowLister{store}
	found, err := swl.Exists(w)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !found {
		t.Errorf("workflow %s should be found", w.Name)
	}
}
