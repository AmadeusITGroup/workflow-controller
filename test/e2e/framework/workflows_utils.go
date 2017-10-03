package framework

import (
	"fmt"

	"github.com/golang/glog"
	. "github.com/onsi/gomega"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	wapi "github.com/sdminonne/workflow-controller/pkg/api/v1"
	"github.com/sdminonne/workflow-controller/pkg/controller"
)

// IsWorkflowFailedDueDeadline check whether a workflow failed due a deadline
func IsWorkflowFailedDueDeadline(w *wapi.Workflow) bool {
	for _, c := range w.Status.Conditions {
		if c.Status == api.ConditionTrue && c.Type == wapi.WorkflowFailed && c.Reason == "DeadlineExceeded" {
			return true
		}
	}
	return false
}

// BuildAndSetClients builds and initilize workflow and kube client
func BuildAndSetClients() (*rest.RESTClient, *clientset.Clientset) {
	f, err := NewFramework()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(f).ShouldNot(BeNil())

	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())
	Logf("Check wether Workflow resource is registered...")
	/*
		TODO: check whether CRD is registered
		Eventually(func() bool {
			r, err := client.IsWorkflowRegistered(kubeClient, f.ResourceName, f.ResourceGroup, f.ResourceVersion)
			if err != nil {
				Logf("Error: %v", err)
			}
			return (r && err == nil)
		}, "5s", "1s").Should(BeTrue())
		Logf("It is!")
		resourceName := strings.Join([]string{f.ResourceName, f.ResourceGroup}, ".")
		thirdPartyResource, err := kubeClient.Extensions().ThirdPartyResources().Get(resourceName)
	*/
	workflowClient, err := f.workflowClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(workflowClient).ShouldNot(BeNil())
	return workflowClient, kubeClient
}

func NewWorkflowStep(name string, dependencies []string) *wapi.WorkflowStep {
	s := &wapi.WorkflowStep{
		Name: name,
		JobTemplate: &batchv2.JobTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{},
			Spec: batch.JobSpec{
				Template: api.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec: api.PodSpec{
						Containers: []api.Container{
							{
								Name:            "step-wait-and-exit",
								Image:           "gcr.io/google_containers/busybox",
								Command:         []string{"sh", "-c", "echo Starting on: $(date); sleep 5; echo Goodbye cruel world at: $(date)"},
								ImagePullPolicy: "IfNotPresent",
							},
						},
						RestartPolicy: "Never",
						DNSPolicy:     "Default",
					},
				},
			},
		},
	}
	s.Dependencies = append([]string(nil), dependencies...)
	return s
}

// NewWorkflowWithLoop it creates an invalid (due to a loop in graph depencies) Workflow
func NewWorkflowWithLoop(group, version, name, namespace string) *wapi.Workflow {
	w := NewWorkflow(group, version, name, namespace, nil)
	w.Spec.Steps[0].Dependencies = append(w.Spec.Steps[0].Dependencies, "two")
	step2 := NewWorkflowStep("two", []string{"one"})
	w.Spec.Steps = append(w.Spec.Steps, *step2)
	return w
}

// NewWorkflow creates a workflow
func NewWorkflow(group, version, name, namespace string, activeDeadlineSeconds *int64) *wapi.Workflow {
	return &wapi.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Workflow",
			APIVersion: group + "/" + version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: wapi.WorkflowSpec{
			ActiveDeadlineSeconds: activeDeadlineSeconds,
			Steps: []wapi.WorkflowStep{
				{
					Name: "one",
					JobTemplate: &batchv2.JobTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"workflow": "step_one",
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
									Containers: []api.Container{
										{
											Name:            "step-one-wait-and-exit",
											Image:           "gcr.io/google_containers/busybox",
											Command:         []string{"sh", "-c", "echo Starting on: $(date); sleep 5; echo Goodbye cruel world at: $(date)"},
											ImagePullPolicy: "IfNotPresent",
										},
									},
									RestartPolicy: "Never",
									DNSPolicy:     "Default",
								},
							},
						},
					},
				},
			},

			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"workflow": "hello",
				},
			},
		},
	}
}

// NewWorkflowWithLoop it creates an invalid (due to a loop in graph depencies) Workflow
func NewWorkflowWithThreeSteps(group, version, name, namespace string) *wapi.Workflow {
	w := NewWorkflow(group, version, name, namespace, nil)
	step2 := NewWorkflowStep("two", []string{"one"})
	w.Spec.Steps = append(w.Spec.Steps, *step2)
	step3 := NewWorkflowStep("three", []string{"two"})
	w.Spec.Steps = append(w.Spec.Steps, *step3)
	return w
}

// HOCreateWorkflow is an higher order func that returns the func to create a Workflow
func HOCreateWorkflow(workflowClient *rest.RESTClient, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		result := wapi.Workflow{}
		err := workflowClient.Post().Resource(wapi.ResourcePlural).Namespace(namespace).Body(workflow).Do().Into(&result)
		if err != nil {
			glog.Warningf("cannot create Workflow %s/%s: %v", namespace, workflow.Name, err)
			return err
		}
		Logf("Workflow created")
		return nil
	}
}

// HOIsWorkflowStarted is an higher order func that returns the func that checks whether Workflow is started
func HOIsWorkflowStarted(workflowClient *rest.RESTClient, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		workflows := wapi.WorkflowList{}
		err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(namespace).Do().Into(&workflows)
		if err != nil {
			Logf("Cannot list workflows:%v", err)
			return err
		}
		if len(workflows.Items) != 1 {
			return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
		}
		if workflows.Items[0].Status.StartTime != nil {
			Logf("Workflow started")
			return nil
		}
		Logf("Workflow %s not updated", workflow.Name)
		return fmt.Errorf("workflow %s not updated", workflow.Name)
	}
}

// HOIsWorkflowFinished is an higher order func that returns the func that checks whether a Workflow is finished
func HOIsWorkflowFinished(workflowClient *rest.RESTClient, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		workflows := wapi.WorkflowList{}
		err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(namespace).Do().Into(&workflows)
		if err != nil {
			Logf("Cannot list workflows:%v", err)
			return err
		}
		if len(workflows.Items) != 1 {
			return fmt.Errorf("Expected only 1 workflows got %d", len(workflows.Items))
		}
		if controller.IsWorkflowFinished(&workflows.Items[0]) {
			Logf("Workflow %s finished", workflow.Name)
			return nil
		}
		Logf("Workflow %s not finished", workflow.Name)
		return fmt.Errorf("workflow %s not finished", workflow.Name)
	}
}

// HONoWorkflowsShouldRemains is an higher order func that returns the func that checks whether some workflows are still present
func HONoWorkflowsShouldRemains(workflowClient *rest.RESTClient, namespace string) func() error {
	return func() error {
		workflows := wapi.WorkflowList{}
		err := workflowClient.Get().Resource(wapi.ResourcePlural).Namespace(namespace).Do().Into(&workflows)
		if err != nil {
			Logf("Cannot list workflows:%v", err)
			return err
		}
		if len(workflows.Items) == 0 {
			Logf("no workflows found")
			return nil // OK workflows removed or never created
		}
		return fmt.Errorf("still some workflows found")
	}
}
