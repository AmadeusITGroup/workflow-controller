package e2e

import (
	"github.com/golang/glog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	api "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"

	dapi "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
	"github.com/amadeusitgroup/workflow-controller/pkg/controller"
	"github.com/amadeusitgroup/workflow-controller/test/e2e/framework"
)

var _ = Describe("DaemonSetJob CRUD", func() {
	It("check DaemonSetJob registration", func() {
		var (
			f                      *framework.Framework
			err                    error
			apiextensionsclientset *apiextensionsclient.Clientset
		)

		Eventually(func() error {
			f, err = framework.NewFramework()
			if err != nil {
				framework.Logf("Cannot list daemonsetjob:%v", err)
				return err
			}
			apiextensionsclientset, err = apiextensionsclient.NewForConfig(f.KubeConfig)
			if err != nil {
				glog.Fatalf("Unable to init clientset from kubeconfig:%v", err)
				return err
			}
			return nil
		}, "40s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOCheckDaemonSetJobRegistration(*apiextensionsclientset), "10s", "2s").ShouldNot(HaveOccurred())
	})

	It("should create then update a daemonsetjob", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myDaemonSetJob := framework.NewDaemonSetJob(workflow.GroupName, "v1", "daemonsetjob1", ns, nil)
		defer func() {
			deleteDaemonSetJob(workflowClient, myDaemonSetJob)
			deleteAllJobsFromDaemonSetJob(kubeClient, myDaemonSetJob)
		}()

		Eventually(framework.HOCreateDaemonSetJob(workflowClient, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobStarted(workflowClient, myDaemonSetJob), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOUpdateDaemonSetJob(workflowClient, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobJobsStarted(kubeClient, myDaemonSetJob), "60s", "5s").ShouldNot(HaveOccurred())
	})

	It("should run to finish a daemonsetjob", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myDaemonSetJob := framework.NewDaemonSetJob(workflow.GroupName, "v1", "daemonsetjob2", ns, nil)
		defer func() {
			deleteDaemonSetJob(workflowClient, myDaemonSetJob)
			deleteAllJobsFromDaemonSetJob(kubeClient, myDaemonSetJob)
		}()
		Eventually(framework.HOCreateDaemonSetJob(workflowClient, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobFinished(workflowClient, myDaemonSetJob), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOCheckAllDaemonSetJobJobsFinished(workflowClient, myDaemonSetJob), "60s", "5s").ShouldNot(HaveOccurred())
	})

	It("should not start job since NodeSelector dont match", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myDaemonSetJob := framework.NewDaemonSetJob(workflow.GroupName, "v1", "daemonsetjob3", ns, map[string]string{"sdfdsfsffsd": "fsdfsdfsdfds"})
		defer func() {
			deleteDaemonSetJob(workflowClient, myDaemonSetJob)
			deleteAllJobsFromDaemonSetJob(kubeClient, myDaemonSetJob)
		}()
		Eventually(framework.HOCreateDaemonSetJob(workflowClient, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobStarted(workflowClient, myDaemonSetJob), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOCheckZeroDaemonSetJobJobsWasCreated(workflowClient, myDaemonSetJob), "10s", "5s").ShouldNot(HaveOccurred())
	})

})

func deleteDaemonSetJob(workflowClient versioned.Interface, daemonsetjob *dapi.DaemonSetJob) {
	workflowClient.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Delete(daemonsetjob.Name, nil)
	By("DaemonSetJob deleted")
}

func deleteAllJobsFromDaemonSetJob(kubeClient clientset.Interface, daemonsetjob *dapi.DaemonSetJob) {
	jobs, err := kubeClient.Batch().Jobs(daemonsetjob.Namespace).List(metav1.ListOptions{
		LabelSelector: controller.InferDaemonSetJobLabelSelectorForJobs(daemonsetjob).String(),
	})
	if err != nil {
		return
	}
	for i := range jobs.Items {
		kubeClient.Batch().Jobs(daemonsetjob.Namespace).Delete(jobs.Items[i].Name /*cascadeDeleteOptions(0)*/, nil)
	}
	By("Jobs delete")
}
