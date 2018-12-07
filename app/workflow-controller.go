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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/golang/glog"
	"github.com/heptiolabs/healthcheck"

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
	workflowController      *controller.WorkflowController
	workflowGC              *garbagecollector.GarbageCollector

	cronWorkflowInfomerFactory winformers.SharedInformerFactory
	cronWorkflowController     *controller.CronWorkflowController

	daemonSetJobController     *controller.DaemonSetJobController
	daemonSetJobInfomerFactory winformers.SharedInformerFactory
	daemonSetJobGC             *garbagecollector.DaemonSetJobGarbageCollector

	httpServer *http.Server // Used for Probes and later prometheus
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags("", c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}

// NewWorkflowControllerApp initializes and returns a ready to run WorkflowController and CronWorkflowController
func NewWorkflowControllerApp(c *Config) *WorkflowController {
	kubeConfig, err := initKubeConfig(c)
	if err != nil {
		glog.Fatalf("Unable to init workflow controller: %v", err)
	}

	kubeClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	if c.InstallCRDs {
		apiextensionsclientset, err := apiextensionsclient.NewForConfig(kubeConfig)
		if err != nil {
			glog.Fatalf("Unable to init clientset from kubeconfig:%v", err)
		}

		_, err = wclient.DefineWorklowResource(apiextensionsclientset)
		if err != nil && !apierrors.IsAlreadyExists(err) { // TODO:
			glog.Fatalf("Unable to define Workflow resource:%v", err)
		}

		_, err = wclient.DefineCronWorkflowResource(apiextensionsclientset)
		if err != nil && !apierrors.IsAlreadyExists(err) { // TODO:
			glog.Fatalf("Unable to define CronWorkflow resource:%v", err)
		}

		_, err = wclient.DefineDaemonSetJobResource(apiextensionsclientset)
		if err != nil && !apierrors.IsAlreadyExists(err) { // TODO:
			glog.Fatalf("Unable to define DaemonSetJob resource:%v", err)
		}
	}

	client, err := wclient.NewWorkflowClient(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to initialize a client:%v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)

	workflowInformerFactory := winformers.NewSharedInformerFactory(client, time.Second*30)
	workflowCtrl := controller.NewWorkflowController(client, kubeClient, kubeInformerFactory, workflowInformerFactory)

	cronWorkflowInformerFactory := winformers.NewSharedInformerFactory(client, time.Second*30)
	cronWorkflowCtrl := controller.NewCronWorkflowController(client, kubeClient)

	daemonSetJobInformerFactory := winformers.NewSharedInformerFactory(client, time.Second*30)
	daemonSetJobCtrl := controller.NewDaemonSetJobController(client, kubeClient, kubeInformerFactory, daemonSetJobInformerFactory)

	// configure readiness and liveness probes
	health := healthcheck.NewHandler()
	workflowCtrl.AddHealthCheck(health)
	//cronWorkflowCtrl.AddHealthCheck(health)
	daemonSetJobCtrl.AddHealthCheck(health)

	return &WorkflowController{
		kubeInformerFactory:     kubeInformerFactory,
		workflowInformerFactory: workflowInformerFactory,
		workflowController:      workflowCtrl,
		workflowGC:              garbagecollector.NewGarbageCollector(client, kubeClient, workflowInformerFactory),

		cronWorkflowInfomerFactory: cronWorkflowInformerFactory,
		cronWorkflowController:     cronWorkflowCtrl,

		daemonSetJobInfomerFactory: daemonSetJobInformerFactory,
		daemonSetJobController:     daemonSetJobCtrl,
		daemonSetJobGC:             garbagecollector.NewDaemonSetJobGarbageCollector(client, kubeClient, daemonSetJobInformerFactory),

		httpServer: &http.Server{Addr: c.ListenHTTPAddr, Handler: health},
	}
}

// Run executes the WorkflowController
func (c *WorkflowController) Run() {
	if c.workflowController != nil {
		ctx, cancelFunc := context.WithCancel(context.Background())
		go handleSignal(cancelFunc)
		c.kubeInformerFactory.Start(ctx.Done())
		c.workflowInformerFactory.Start(ctx.Done())
		c.runGCs(ctx)
		go c.runHTTPServer(ctx)
		c.workflowInformerFactory.Start(ctx.Done())
		c.cronWorkflowInfomerFactory.Start(ctx.Done())
		c.daemonSetJobInfomerFactory.Start(ctx.Done())
		go c.daemonSetJobController.Run(ctx)
		go c.workflowController.Run(ctx)
		go c.cronWorkflowController.Run(ctx)
		<-ctx.Done()
	}
}

func (c *WorkflowController) runGCs(ctx context.Context) {
	go func() {
		if !cache.WaitForCacheSync(ctx.Done(), c.workflowGC.WorkflowSynced) {
			glog.Errorf("Timed out waiting for caches to sync")
		}
		wait.Until(func() {
			err := c.workflowGC.CollectWorkflowJobs()
			if err != nil {
				glog.Errorf("collecting workflow jobs: %v", err)
			}
		}, garbagecollector.Interval, ctx.Done())
	}()
	go func() {
		if !cache.WaitForCacheSync(ctx.Done(), c.daemonSetJobGC.DaemonSetJobSynced) {
			glog.Errorf("Timed out waiting for caches to sync")
		}
		wait.Until(func() {
			err := c.daemonSetJobGC.CollectDaemonSetJobJobs()
			if err != nil {
				glog.Errorf("collecting daemonsetjob jobs: %v", err)
			}
		}, garbagecollector.Interval, ctx.Done())
	}()
}

func (c *WorkflowController) runHTTPServer(ctx context.Context) error {
	go func() {
		glog.Infof("Listening on http://%s\n", c.httpServer.Addr)

		if err := c.httpServer.ListenAndServe(); err != nil {
			glog.Error("Http server error: ", err)
		}
	}()
	<-ctx.Done()
	glog.Info("Shutting down the http server...")
	return c.httpServer.Shutdown(context.Background())
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
