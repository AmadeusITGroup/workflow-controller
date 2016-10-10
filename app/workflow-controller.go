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
	"math/rand"
	"net/http"
	"time"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/apis/extensions"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/wait"

	wclient "github.com/sdminonne/workflow-controller/pkg/client"
	"github.com/sdminonne/workflow-controller/pkg/workflow"
)

// WorkflowController contains all info to run the worklow controller app
type WorkflowController struct {
	controller *workflow.Controller
}

// NewWorkflowController  initializes and returns a ready to run WorkflowController
func NewWorkflowController(c *Config) *WorkflowController {

	kubeconfig, err := clientcmd.BuildConfigFromFlags(c.KubeMasterURL, c.KubeConfigFile)
	if err != nil {
		glog.Fatalf("Unable to start workflow controller %v", err)
	}

	kubeClient := clientset.NewForConfigOrDie(restclient.AddUserAgent(kubeconfig, "workflow-controller"))
	thirdPartyResource := &extensions.ThirdPartyResource{}
	stopChannel := make(chan struct{})
	wait.Until(func() {
		thirdPartyResource, err = wclient.RegisterWorkflow(kubeClient, c.ResourceName, c.ResourceDomain, c.ResourceVersions)
		if err == nil {
			glog.V(2).Infof("ThirdPartyResource %v.%v versions %v created", c.ResourceName, c.ResourceDomain, c.ResourceVersions)
			close(stopChannel)
		} else {
			status, ok := err.(*errors.StatusError)
			switch {
			case ok && status.Status().Code == http.StatusConflict: // TODO @sdminonne: handle errors in case resource is there and there's a version conflict
				glog.Warningf("ThirdPartyResource  %v.%v versions %v already registered", c.ResourceName, c.ResourceDomain, c.ResourceVersions)
				close(stopChannel)
			case !ok:
				glog.Errorf("Unable to fetch information from error: %v", err)
			default:
				glog.Errorf("Unable to register ThirdPartyResource %v.%v version %v: %v", c.ResourceName, c.ResourceDomain, c.ResourceVersions, err.Error())
			}
		}
	}, 1*time.Second, stopChannel)

	wfClient := wclient.NewForConfigOrDie(thirdPartyResource, kubeconfig)
	resyncPeriod := func() time.Duration {
		factor := rand.Float64() + 1
		return time.Duration(float64(time.Hour) * 12.0 * factor)
	}

	glog.V(4).Infof("Creating workflow controller...")
	controller := workflow.NewController(kubeClient, wfClient, thirdPartyResource, resyncPeriod)
	return &WorkflowController{controller: controller}
}

// Run executes the WorkflowController
func (c *WorkflowController) Run() {
	glog.Infof("Starting workflow controller")
	c.controller.Run(1, wait.NeverStop)
}
