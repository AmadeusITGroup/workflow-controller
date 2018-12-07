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

	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
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
func BuildAndSetClients() (versioned.Interface, *clientset.Clientset) {
	f, err := NewFramework()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(f).ShouldNot(BeNil())

	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())

	client, err := f.client()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(client).ShouldNot(BeNil())

	return client, kubeClient
}

// NewWorkflowStep creates a step for a workflow
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

// NewWorkflowWithThreeSteps it creates a valid Workflow with three steps
func NewWorkflowWithThreeSteps(group, version, name, namespace string) *wapi.Workflow {
	w := NewWorkflow(group, version, name, namespace, nil)
	step2 := NewWorkflowStep("two", []string{"one"})
	w.Spec.Steps = append(w.Spec.Steps, *step2)
	step3 := NewWorkflowStep("three", []string{"two"})
	w.Spec.Steps = append(w.Spec.Steps, *step3)
	return w
}

// HOCreateWorkflow is an higher order func that returns the func to create a Workflow
func HOCreateWorkflow(workflowClient versioned.Interface, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		if _, err := workflowClient.WorkflowV1().Workflows(namespace).Create(workflow); err != nil {
			glog.Warningf("cannot create Workflow %s/%s: %v", namespace, workflow.Name, err)
			return err
		}
		Logf("Workflow created")
		return nil
	}
}

// HOIsWorkflowStarted is an higher order func that returns the func that checks whether Workflow is started
func HOIsWorkflowStarted(workflowClient versioned.Interface, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		w, err := workflowClient.WorkflowV1().Workflows(workflow.Namespace).Get(workflow.Name, metav1.GetOptions{})
		if err != nil {
			Logf("Cannot find workflow %s:%v", workflow.Name, err)
			return err
		}
		if w.Status.StartTime != nil {
			Logf("Workflow started")
			return nil
		}
		Logf("Workflow %s/%s not updated", w.Namespace, w.Name)
		return fmt.Errorf("workflow %s/%s not updated", w.Namespace, w.Name)
	}
}

// HOIsWorkflowFinished is an higher order func that returns the func that checks whether a Workflow is finished
func HOIsWorkflowFinished(workflowClient versioned.Interface, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		w, err := workflowClient.WorkflowV1().Workflows(workflow.Namespace).Get(workflow.Name, metav1.GetOptions{})
		if err != nil {
			Logf("Cannot find workflow %s:%v", workflow.Name, err)
			return err
		}
		if controller.IsWorkflowFinished(w) {
			Logf("Workflow %s/%s finished", w.Namespace, w.Name)
			return nil
		}
		Logf("Workflow %s not finished", w.Name)
		return fmt.Errorf("workflow %s/%s not finished", w.Namespace, w.Name)
	}
}

// HONoWorkflowsShouldRemains is an higher order func that returns the func that checks whether some workflows are still present
func HONoWorkflowsShouldRemains(workflowClient versioned.Interface, namespace string) func() error {
	return func() error {
		workflows, err := workflowClient.WorkflowV1().Workflows(namespace).List(metav1.ListOptions{})
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

// HONoJobsShouldRemains is an higher order func that returns the func that checks whether some jobs are still present
func HONoJobsShouldRemains(kubeClient clientset.Interface, labelSelector, namespace string) func() error {
	return func() error {
		jobs, err := kubeClient.Batch().Jobs(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			Logf("Cannot list jobs: %v", err)
			return err
		}
		if len(jobs.Items) == 0 {
			Logf("no jobs found")
			return nil // OK jobs removed or never created
		}
		return fmt.Errorf("still some jobs found")
	}
}

// HODeleteWorkflow is an higher order func that returns the func to remove the Workflow
func HODeleteWorkflow(workflowClient versioned.Interface, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		return workflowClient.WorkflowV1().Workflows(namespace).Delete(workflow.Name, controller.CascadeDeleteOptions(0))
	}
}

// HOCheckAllStepsFinished is an higher order func that returns func that checks
// whether all step are finished
func HOCheckAllStepsFinished(workflowClient versioned.Interface, workflow *wapi.Workflow, namespace string) func() error {
	return func() error {
		w, err := workflowClient.WorkflowV1().Workflows(namespace).Get(workflow.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to get %s:%v", workflow.Name, err)
		}
		if len(w.Status.Statuses) == 0 {
			return fmt.Errorf("workflow %s/%s not fully completed", w.Namespace, w.Name)
		}
		for _, s := range w.Status.Statuses {
			if !s.Complete {
				return fmt.Errorf("workflow %s/%s not fully completed", w.Namespace, w.Name)
			}
		}
		return nil
	}
}

// HOCheckStepFinished is an higher order func that returns the func that check if the job
// at the specified step finished
func HOCheckStepFinished(workflowClient versioned.Interface, workflow *wapi.Workflow, step, namespace string) func() error {
	return func() error {
		w, err := workflowClient.WorkflowV1().Workflows(namespace).Get(workflow.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		stepStatus := controller.GetStepStatusByName(w, step)
		if stepStatus == nil {
			return fmt.Errorf("unable to find step %q in %s/%s", step, w.Namespace, w.Name)
		}
		if !stepStatus.Complete {
			return fmt.Errorf("step %q in %s/%s not (yet) complete", step, w.Namespace, w.Name)
		}
		return nil
	}
}
