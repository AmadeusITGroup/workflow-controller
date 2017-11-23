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
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	wclient "github.com/amadeusitgroup/workflow-controller/pkg/client"
	winformers "github.com/amadeusitgroup/workflow-controller/pkg/client/informers/externalversions"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
	"github.com/amadeusitgroup/workflow-controller/pkg/garbagecollector"
)

// WorkflowController contains all info to run the worklow controller app
type WorkflowController struct {
	kubeInformerFactory     kubeinformers.SharedInformerFactory
	workflowInformerFactory winformers.SharedInformerFactory
	controller              *controller.WorkflowController
	GC                      *garbagecollector.GarbageCollector
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
	workflowClient, err := wclient.NewClient(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize a WorkflowClient:%v", err)
	}

	kubeClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	workflowInformerFactory := winformers.NewSharedInformerFactory(workflowClient, time.Second*30)

	return &WorkflowController{
		kubeInformerFactory:     kubeInformerFactory,
		workflowInformerFactory: workflowInformerFactory,
		controller:              controller.NewWorkflowController(workflowClient, kubeClient, kubeInformerFactory, workflowInformerFactory),
		GC:                      garbagecollector.NewGarbageCollector(workflowClient, kubeClient, workflowInformerFactory),
	}
}

// Run executes the WorkflowController
func (c *WorkflowController) Run() {
	if c.controller != nil {
		ctx, cancelFunc := context.WithCancel(context.Background())
		go handleSignal(cancelFunc)
		c.kubeInformerFactory.Start(ctx.Done())
		c.workflowInformerFactory.Start(ctx.Done())
		c.runGC(ctx)
		c.controller.Run(ctx)

	}
}

func (c *WorkflowController) runGC(ctx context.Context) {
	go func() {
		if !cache.WaitForCacheSync(ctx.Done(), c.GC.WorkflowSynced) {
			glog.Errorf("Timed out waiting for caches to sync")
		}
		wait.Until(func() {
			err := c.GC.CollectWorkflowJobs()
			if err != nil {
				glog.Errorf("collecting workflow jobs: %v", err)
			}
		}, garbagecollector.Interval, ctx.Done())
	}()
}

func handleSignal(cancelFunc context.CancelFunc) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-sigc
	glog.Infof("Signal received: %s, stop the process", sig.String())
	cancelFunc()
	close(sigc)
}
