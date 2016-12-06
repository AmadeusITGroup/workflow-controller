package framework

import (
	"strings"

	. "github.com/onsi/gomega"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/extensions"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

	wapi "github.com/sdminonne/workflow-controller/pkg/api"
	"github.com/sdminonne/workflow-controller/pkg/client"
)

// WorkflowClient creates a Workflow client for e2e framework
func (f *Framework) workflowClient(resource *extensions.ThirdPartyResource) (client.Interface, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(f.KubeMasterURL, f.KubeConfigPath)
	if err != nil {
		return nil, err
	}
	return client.NewForConfigOrDie(resource, kubeconfig), nil
}

// WorkflowClient defines client for e2e tests
type WorkflowClient struct {
	f *Framework
	client.Interface
}

func (f *Framework) kubeClient() (clientset.Interface, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(f.KubeMasterURL, f.KubeConfigPath)
	if err != nil {
		Failf("Unable to initialize KubeClient: %v", err)
		return nil, err
	}
	kubeClient := clientset.NewForConfigOrDie(restclient.AddUserAgent(kubeconfig, "e2e-workflow"))
	return kubeClient, err
}

// BuildAndSetClients builds and initilize workflow and kube client
func BuildAndSetClients() (client.Interface, clientset.Interface) {
	f := NewFramework()
	Ω(f).ShouldNot(BeNil())
	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())
	Logf("Check wether Workflow resource is registered...")
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
	workflowClient, err := f.workflowClient(thirdPartyResource)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(workflowClient).ShouldNot(BeNil())
	return workflowClient, kubeClient
}

// NewWorkflow creates a workflow
func NewWorkflow(group, version, name, namespace string) *wapi.Workflow {
	return &wapi.Workflow{
		TypeMeta: unversioned.TypeMeta{
			Kind:       "Workflow",
			APIVersion: group + "/" + version,
		},

		ObjectMeta: api.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: wapi.WorkflowSpec{
			Steps: []wapi.WorkflowStep{
				{
					Name: "one",
					JobTemplate: &batch.JobTemplateSpec{
						ObjectMeta: api.ObjectMeta{
							Labels: map[string]string{
								"workflow": "step_one",
							},
						},
						Spec: batch.JobSpec{
							Template: api.PodTemplateSpec{
								ObjectMeta: api.ObjectMeta{
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

			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"workflow": "hello",
				},
			},
		},
		Status: wapi.WorkflowStatus{
			Conditions: []wapi.WorkflowCondition{},
			Statuses:   []wapi.WorkflowStepStatus{},
		},
	}
}
