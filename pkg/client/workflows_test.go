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

package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"

	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/runtime"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	wcodec "github.com/sdminonne/workflow-controller/pkg/api/codec"
	wapitesting "github.com/sdminonne/workflow-controller/pkg/api/testing"
)

func TestEncodeDecode(t *testing.T) {
	w1 := wapitesting.NewWorkflow("mydomain.com", "v1", "mydag", api.NamespaceDefault, nil)
	gvk := &unversioned.GroupVersionKind{
		Group:   "mydomain.com",
		Version: "v1",
		Kind:    "Workflow",
	}
	unstruct, err := wcodec.WorkflowToUnstructured(w1, gvk)
	if err != nil {
		t.Fatal(err)
	}

	w2, err := wcodec.UnstructuredToWorkflow(unstruct)
	if err != nil {
		t.Fatal(err)
	}
	if !api.Semantic.DeepEqual(w1, w2) {
		t.Fatalf("Fail two objects look different...")
	}
}

func TestListWorkflows(t *testing.T) {
	bytes, _ := json.Marshal(wapitesting.NewWorkflowList("example.com", "v1"))
	ns := api.NamespaceAll

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET got '%s'\n", r.Method)
		}
		if r.RequestURI != "/apis/acme.org/v1/workflows" {
			t.Errorf("Expected /apis/acme.org/v1/workflows got %s\n", r.RequestURI)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != "" {
			t.Errorf("Expected '' got '%s'\n", body)
		}

		w.Header().Set("Content-Type", runtime.ContentTypeJSON)
		w.Write(bytes)
	}))
	defer ts.Close()

	domain := "acme.org"
	resource := "workflow"
	versions := []string{"v1", "v2"}
	fakeClient := fake.NewSimpleClientset()
	workflowResource, err := RegisterWorkflow(fakeClient, resource, domain, versions)
	fmt.Printf("\n%#v\n", workflowResource)
	if err != nil {
		t.Fatalf("Unexpected error : %v", err)
	}
	config := &restclient.Config{
		Host:          ts.URL,
		ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Group: "gtest", Version: "vtest"}},
	}
	c := NewForConfigOrDie(workflowResource, config)
	workflowList, err := c.Workflows(ns).List(api.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(workflowList.Items) != 3 {
		t.Errorf("Unexpected error found %d errors wanted 1", len(workflowList.Items))
	}
}

func TestGetWorkflow(t *testing.T) {
	bytes, _ := json.Marshal(wapitesting.NewWorkflow("example.com", "v1", "mydag", "mynamespace", nil))
	ns := api.NamespaceDefault
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET got '%s'\n", r.Method)
		}
		expectedURI := "/apis/example.com/v1/namespaces/default/workflows/mydag"
		if r.RequestURI != expectedURI {
			t.Errorf("Expected %q got %q\n", expectedURI, r.RequestURI)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != "" {
			t.Errorf("Expected '' got '%s'\n", body)
		}

		w.Header().Set("Content-Type", runtime.ContentTypeJSON)
		w.Write(bytes)

	}))
	defer ts.Close()

	domain := "example.com"
	resource := "workflow"
	versions := []string{"v1", "v2"}
	fakeClient := fake.NewSimpleClientset()
	workflowResource, err := RegisterWorkflow(fakeClient, resource, domain, versions)
	if err != nil {
		t.Fatalf("Unexpected error : %v", err)
	}
	config := &restclient.Config{
		Host:          ts.URL,
		ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Group: "gtest", Version: "vtest"}},
	}
	c := NewForConfigOrDie(workflowResource, config)
	workflow, err := c.Workflows(ns).Get("mydag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if workflow == nil {
		t.Errorf("Unexpected error coulnd't GET workflow")
	}
}

func TestUpdateWorkflow(t *testing.T) {
	bytes, _ := json.Marshal(wapitesting.NewWorkflow("example.com", "v1", "mydag", "yournamespace", nil))
	ns := api.NamespaceDefault
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/apis/example.com/v1/namespaces/default/workflows/mydag"
		switch r.Method {
		case "PUT":
			if r.RequestURI != expectedURI {
				t.Errorf("Expected %q", expectedURI)
			}
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", runtime.ContentTypeJSON)
			w.Write(body)
		case "GET":
			if r.RequestURI != expectedURI {
				t.Errorf("Expected %q got %q\n", expectedURI, r.RequestURI)
			}
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if string(body) != "" {
				t.Errorf("Expected '' got '%s'\n", body)
			}

			w.Header().Set("Content-Type", runtime.ContentTypeJSON)
			w.Write(bytes)
		}

	}))
	defer ts.Close()

	domain := "example.com"
	resource := "workflow"
	versions := []string{"v1", "v2"}
	fakeClient := fake.NewSimpleClientset()
	workflowResource, err := RegisterWorkflow(fakeClient, resource, domain, versions)
	if err != nil {
		t.Fatalf("Unexpected error : %v", err)
	}
	config := &restclient.Config{
		Host:          ts.URL,
		ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Group: "gtest", Version: "vtest"}},
	}
	c := NewForConfigOrDie(workflowResource, config)

	workflow, err := c.Workflows(ns).Get("mydag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workflow == nil {
		t.Errorf("Unexpected error coulnd't GET workflow")
	}

	expectedNumberOfSteps := len(workflow.Spec.Steps)
	if len(workflow.Spec.Steps) != expectedNumberOfSteps {
		t.Errorf("Bad number of steps expetectd %d got %d", expectedNumberOfSteps, len(workflow.Spec.Steps))
	}

	workflow.Spec.Steps = append(workflow.Spec.Steps, wapi.WorkflowStep{Name: "three"})
	expectedNumberOfSteps++

	newWorkflow, err := c.Workflows(ns).Update(workflow)
	if err != nil {
		t.Errorf("Unexpected error. Couldn't UPDATE workflow: %v", err)
	}
	if len(newWorkflow.Spec.Steps) != expectedNumberOfSteps {
		t.Errorf("Bad number of steps expected %d got %d", expectedNumberOfSteps, len(newWorkflow.Spec.Steps))
	}

}

func TestDeleteWorkflow(t *testing.T) {
	statusOK := &unversioned.Status{
		TypeMeta: unversioned.TypeMeta{Kind: "Status"},
		Status:   unversioned.StatusSuccess,
	}
	ns := api.NamespaceDefault

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedURI := "/apis/example.com/v1/namespaces/default/workflows/two-steps-workflow"
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE got '%s'\n", r.Method)
		}
		if r.RequestURI != expectedURI {
			t.Errorf("Expected %q got %q\n", expectedURI, r.RequestURI)
		}
		w.Header().Set("Content-Type", runtime.ContentTypeJSON)
		runtime.UnstructuredJSONScheme.Encode(statusOK, w)
	}))
	defer ts.Close()

	domain := "example.com"
	resource := "workflow"
	versions := []string{"v1", "v2"}
	fakeClient := fake.NewSimpleClientset()
	workflowResource, err := RegisterWorkflow(fakeClient, resource, domain, versions)
	if err != nil {
		t.Fatalf("Unexpected error : %v", err)
	}
	config := &restclient.Config{
		Host:          ts.URL,
		ContentConfig: restclient.ContentConfig{GroupVersion: &unversioned.GroupVersion{Group: "gtest", Version: "vtest"}},
	}
	c := NewForConfigOrDie(workflowResource, config)
	err = c.Workflows(ns).Delete("two-steps-workflow", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
