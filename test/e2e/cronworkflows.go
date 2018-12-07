package e2e

import (
	"path"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	api "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	cwapiV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
	"github.com/amadeusitgroup/workflow-controller/test/e2e/framework"
)

func deleteWorkflowsFromCronWorkflow(wclient versioned.Interface, cronWorkflow *cwapiV1.CronWorkflow) {
	set := labels.Set{}
	set[controller.CronWorkflowLabelKey] = cronWorkflow.Name
	wclient.WorkflowV1().Workflows(cronWorkflow.Namespace).DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(set).String()})
}

func deleteCronWorkflow(client versioned.Interface, kubeClient clientset.Interface, cronWorkflow *cwapiV1.CronWorkflow) {
	nameToLog := path.Join([]string{cronWorkflow.Namespace, cronWorkflow.Name}...)
	By("CronWorkflow " + nameToLog + " deleted")
	client.CronworkflowV1().CronWorkflows(cronWorkflow.Namespace).Delete(cronWorkflow.Name, nil)
}

var _ = Describe("CronWorkflow CRUD", func() {
	It("check CronWorkflow registration", func() {
		var (
			f                      *framework.Framework
			err                    error
			apiextensionsclientset *apiextensionsclient.Clientset
		)

		Eventually(func() error {
			f, err = framework.NewFramework()
			if err != nil {
				framework.Logf("Cannot list workflows:%v", err)
				return err
			}
			apiextensionsclientset, err = apiextensionsclient.NewForConfig(f.KubeConfig)
			if err != nil {
				glog.Fatalf("Unable to init clientset from kubeconfig:%v", err)
				return err
			}
			return nil
		}, "40s", "1s").ShouldNot(HaveOccurred())

		ns := api.NamespaceDefault
		Eventually(framework.HOCheckCronWorkflowRegistration(*apiextensionsclientset, ns), "10s", "2s").ShouldNot(HaveOccurred())
	})

	It("should create a workflow every minute", func() {
		client, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myCw := framework.NewOneStepCronWorkflow(cwapiV1.ResourceVersion, "cron-workflow1", ns)
		defer func() {
			deleteCronWorkflow(client, kubeClient, myCw) // cascading workflows/jobs/pods
			deleteWorkflowsFromCronWorkflow(client, myCw)
		}()
		Eventually(framework.HOCreateCronWorkflow(client, myCw, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowCreatedFromCronWorkflow(client, myCw, ns), "2m", "10s").ShouldNot(HaveOccurred())
	})

	It("should create and delete cronWorkflow  without cascading", func() {
		client, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myCw := framework.NewOneStepCronWorkflow(cwapiV1.ResourceVersion, "cron-workflow2", ns)
		defer func() {
			deleteCronWorkflow(client, kubeClient, myCw) // cascading workflows/jobs/pods
			deleteWorkflowsFromCronWorkflow(client, myCw)
		}()
		Eventually(framework.HOCreateCronWorkflow(client, myCw, ns), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsWorkflowCreatedFromCronWorkflow(client, myCw, ns), "2m", "10s").ShouldNot(HaveOccurred())
	})
})
