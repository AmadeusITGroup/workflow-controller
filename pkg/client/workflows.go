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
	"fmt"
	"strings"

	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/meta"

	"k8s.io/kubernetes/pkg/api/v1"

	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/extensions"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/typed/dynamic"

	"k8s.io/kubernetes/pkg/registry/thirdpartyresourcedata"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/watch"
)

// WorkflowsNamespacer has methods to work with Workflow resources in a namespace
type WorkflowsNamespacer interface {
	Workflows(namespace string) WorkflowInterface
}

// Interface is just an alias for WorkflowNamespacer
type Interface WorkflowsNamespacer

// WorkflowInterface exposes methods to work on Workflow resources.
type WorkflowInterface interface {
	List(options api.ListOptions) (*runtime.UnstructuredList, error)

	Get(name string) (*runtime.Unstructured, error)
	Update(workflow *runtime.Unstructured) (*runtime.Unstructured, error)
	Delete(name string, options *api.DeleteOptions) error

	Watch(options api.ListOptions) (watch.Interface, error)
}

// Client implements a workflow client
type Client struct {
	*dynamic.Client
	tmpRestClient *restclient.RESTClient // TODO: remove it only needed for DELETE
	restResource  string
}

// Workflows returns a Workflows
func (c Client) Workflows(ns string) WorkflowInterface {
	return newWorkflows(c, ns)
}

// NewForConfigOrDie creates and initializes a Workflow REST client. It panics in case of error
func NewForConfigOrDie(resource *extensions.ThirdPartyResource, config *restclient.Config) Interface {
	kind, group, err := thirdpartyresourcedata.ExtractApiGroupAndKind(resource)
	if err != nil {
		panic(err)
	}
	plural, _ := meta.KindToResource(unversioned.GroupVersionKind{
		Group:   group,
		Version: resource.Versions[0].Name,
		Kind:    kind,
	})
	config.GroupVersion = &unversioned.GroupVersion{
		Group:   group,
		Version: resource.Versions[0].Name,
	}

	config.APIPath = "/apis"
	dynamicClient, err := dynamic.NewClient(config)
	if err != nil {
		panic(err)
	}

	// HACK
	// TODO: remove it when dynamicClient will support DELETE for thirdPartyResource
	config.NegotiatedSerializer = api.Codecs
	tmpRestClient, err := restclient.RESTClientFor(config)
	if err != nil {
		panic(err)
	}

	return Client{dynamicClient, tmpRestClient, plural.Resource}
}

// workflows implements WorkflowsNamespacer interface
type workflows struct {
	c        Client
	ns       string
	resource *unversioned.APIResource
}

// newWorkflows returns a workflows
func newWorkflows(c Client, ns string) *workflows {
	return &workflows{c: c, ns: ns, resource: &unversioned.APIResource{Name: c.restResource, Namespaced: len(ns) != 0}}
}

// Ensure statically that workflows implements WorkflowInterface.
var _ WorkflowInterface = &workflows{}

// List returns a list of workflows that match the label and field selectors.
func (w *workflows) List(options api.ListOptions) (*runtime.UnstructuredList, error) {
	v1Options := v1.ListOptions{}
	v1.Convert_api_ListOptions_To_v1_ListOptions(&options, &v1Options, nil)
	obj, err := w.c.Resource(w.resource, w.ns).List(&v1Options)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*runtime.UnstructuredList)
	if !ok {
		return nil, fmt.Errorf("unable to list workflows as UnstructuredList\n")
	}
	return list, nil
}

// Get returns information about a particular workflow.
func (w *workflows) Get(name string) (*runtime.Unstructured, error) {
	return w.c.Resource(w.resource, w.ns).Get(name)
}

// Update updates an existing workflow. TODO: implement via PATCH
func (w *workflows) Update(workflow *runtime.Unstructured) (*runtime.Unstructured, error) {
	return w.c.Resource(w.resource, w.ns).Update(workflow)
}

// Delete deletes a workflow, returns error if one occurs.
func (w *workflows) Delete(name string, options *api.DeleteOptions) error {
	//return w.c.Resource(w.resource, w.ns).Delete(name, &v1.DeleteOptions{})
	// TODO: using raw request since dynamicClient cannot handle Delete of a ThirdPartyResouce
	return w.c.tmpRestClient.Delete().NamespaceIfScoped(w.ns, len(w.ns) > 0).
		Resource(w.resource.Name).
		Name(name).
		Body(options).
		Do().
		Error()
}

// Watch returns a watch.Interface that watches the requested workflows.
func (w *workflows) Watch(options api.ListOptions) (watch.Interface, error) {
	optionsV1 := v1.ListOptions{}
	v1.Convert_api_ListOptions_To_v1_ListOptions(&options, &optionsV1, nil)
	return w.c.Resource(w.resource, w.ns).Watch(&optionsV1)
}

// RegisterWorkflow registers Workflow resource as k8s ThirdPartyResouce object
func RegisterWorkflow(kubeClient clientset.Interface, resource, domain string, versions []string) (*extensions.ThirdPartyResource, error) {
	glog.V(4).Infof("Trying to create ThirdPartyResource %v.%v version %v", resource, domain, versions)
	APIVersions := []extensions.APIVersion{}
	for _, v := range versions {
		APIVersions = append(APIVersions, extensions.APIVersion{Name: v})
	}
	thirdPartyResource := &extensions.ThirdPartyResource{
		ObjectMeta: api.ObjectMeta{
			Name: strings.Join([]string{resource, domain}, "."),
		},
		Description: "Workflow as thrid party resource. Automatically registered by controller.",
		Versions:    APIVersions,
	}
	_, err := kubeClient.Extensions().ThirdPartyResources().Create(thirdPartyResource)
	return thirdPartyResource, err
}
