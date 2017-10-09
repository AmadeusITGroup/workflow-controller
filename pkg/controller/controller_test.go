package controller

import (
	"fmt"
	"testing"
	"time"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	wapi "github.com/sdminonne/workflow-controller/pkg/api/workflow/v1"
	wclient "github.com/sdminonne/workflow-controller/pkg/client"
	winformers "github.com/sdminonne/workflow-controller/pkg/client/informers/externalversions"
)

// utility function to create a basic Workflow
func newWorkflow() *wapi.Workflow {
	return &wapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "example.com/v1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mydag",
			Namespace: api.NamespaceDefault,
		},
		Spec: wapi.WorkflowSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"workflow": "example-selector",
				},
			},
			Steps: []wapi.WorkflowStep{
				{
					Name:        "myJob",
					JobTemplate: newJobTemplateSpec(),
				},
			},
		},
	}
}

// utility function to create a JobTemplateSpec
func newJobTemplateSpec() *batchv2.JobTemplateSpec {
	return &batchv2.JobTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
		},
		Spec: batch.JobSpec{
			Template: api.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: api.PodSpec{
					RestartPolicy: "Never",
					Containers: []api.Container{
						{Image: "foo/bar"},
					},
				},
			},
		},
	}
}

func TestControllerSyncWorkflow(t *testing.T) {
	testCases := map[string]struct {
		workflow            *wapi.Workflow
		jobs                []batch.Job
		workflowTweak       func(*wapi.Workflow) *wapi.Workflow
		customUpdateHandler func(*wapi.Workflow) error // custom update func
	}{
		"workflow default": { // it tests if the workflow is defaulted
			workflow: newWorkflow(),
			jobs:     []batch.Job{},
			customUpdateHandler: func(w *wapi.Workflow) error {

				if !wapi.IsWorkflowDefaulted(w) {
					return fmt.Errorf("workflow %q not defaulted", w.Name)
				}
				return nil
			},
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return w
			},
		},
		"workflow validated": {
			workflow: newWorkflow(),
			jobs:     []batch.Job{},
			customUpdateHandler: func(w *wapi.Workflow) error {
				errs := wapi.ValidateWorkflow(w)
				if len(errs) > 0 {
					return fmt.Errorf("workflow %q not valid", w.Name)
				}
				return nil
			},
			workflowTweak: func(w *wapi.Workflow) *wapi.Workflow {
				return wapi.DefaultWorkflow(w) // workflow must be defaulted to be validated
			},
		},
	}
	for name, tc := range testCases {
		restConfig := &rest.Config{Host: "localhost"}
		workflowClient, err := wclient.NewClient(restConfig)

		kubeClient, err := clientset.NewForConfig(restConfig)
		if err != nil {
			t.Fatalf("%s:%v", name, err)
		}
		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
		workflowInformerFactory := winformers.NewSharedInformerFactory(workflowClient, time.Second*30)

		controller := NewWorkflowController(workflowClient, kubeClient, kubeInformerFactory, workflowInformerFactory)
		controller.JobControl = &FakeJobControl{}
		controller.JobSynced = func() bool { return true }
		key, err := cache.MetaNamespaceKeyFunc(tc.workflow)
		if err != nil {
			t.Fatalf("%s - unable to get key from workflow:%v", name, err)
		}
		tweakedWorkflow := tc.workflowTweak(tc.workflow) // modify basic workflow
		workflowInformerFactory.Workflow().V1().Workflows().Informer().GetStore().Add(tweakedWorkflow)
		for i := range tc.jobs {
			kubeInformerFactory.Batch().V1().Jobs().Informer().GetStore().Add(tc.jobs[i])
		}
		controller.updateHandler = tc.customUpdateHandler
		if err := controller.sync(key); err != nil {
			t.Errorf("%s - %v", name, err)
		}
	}
}
