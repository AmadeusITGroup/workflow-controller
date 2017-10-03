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

package app

import (
	"context"

	"github.com/golang/glog"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	wclient "github.com/sdminonne/workflow-controller/pkg/client"
	"github.com/sdminonne/workflow-controller/pkg/controller"
)

// WorkflowController contains all info to run the worklow controller app
type WorkflowController struct {
	controller *controller.WorkflowController
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags("", c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}

// NewWorkflowController  initializes and returns a ready to run WorkflowController
func NewWorkflowController(c *Config) *WorkflowController {
	kubeConfig, err := initKubeConfig(c)
	if err != nil {
		glog.Fatalf("Unable to init workflow controller: %v", err)
	}

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to init clientset from kubeconfig:%v", err)
	}
	_, err = wclient.DefineWorklowResource(apiextensionsclientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		glog.Fatalf("Unable to define Workflow resource:%v", err)
	}
	workflowClient, workflowScheme, err := wclient.NewClient(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize a WorkflowClient:%v", err)
	}

	kubeclient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	return &WorkflowController{
		controller: controller.NewWorkflowController(workflowClient, workflowScheme, kubeclient),
	}
}

// Run executes the WorkflowController
func (c *WorkflowController) Run() {
	if c.controller != nil {
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		c.controller.Run(ctx)
	}
}
