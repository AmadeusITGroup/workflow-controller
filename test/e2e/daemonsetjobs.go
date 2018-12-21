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
		client, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myDaemonSetJob := framework.NewDaemonSetJob(workflow.GroupName, "v1", "daemonsetjob1", ns, nil)
		defer func() {
			deleteDaemonSetJob(client, myDaemonSetJob)
			deleteAllJobsFromDaemonSetJob(kubeClient, myDaemonSetJob)
		}()

		Eventually(framework.HOCreateDaemonSetJob(client, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobStarted(client, myDaemonSetJob), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOUpdateDaemonSetJob(client, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobJobsStarted(kubeClient, myDaemonSetJob), "60s", "5s").ShouldNot(HaveOccurred())
	})

	It("should run to finish a daemonsetjob", func() {
		client, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myDaemonSetJob := framework.NewDaemonSetJob(workflow.GroupName, "v1", "daemonsetjob2", ns, nil)
		defer func() {
			deleteDaemonSetJob(client, myDaemonSetJob)
			deleteAllJobsFromDaemonSetJob(kubeClient, myDaemonSetJob)
		}()
		Eventually(framework.HOCreateDaemonSetJob(client, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobFinished(client, myDaemonSetJob), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOCheckAllDaemonSetJobJobsFinished(client, myDaemonSetJob), "60s", "5s").ShouldNot(HaveOccurred())
	})

	It("should not start job since NodeSelector dont match", func() {
		client, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault
		myDaemonSetJob := framework.NewDaemonSetJob(workflow.GroupName, "v1", "daemonsetjob3", ns, map[string]string{"sdfdsfsffsd": "fsdfsdfsdfds"})
		defer func() {
			deleteDaemonSetJob(client, myDaemonSetJob)
			deleteAllJobsFromDaemonSetJob(kubeClient, myDaemonSetJob)
		}()
		Eventually(framework.HOCreateDaemonSetJob(client, myDaemonSetJob), "5s", "1s").ShouldNot(HaveOccurred())

		Eventually(framework.HOIsDaemonSetJobStarted(client, myDaemonSetJob), "40s", "5s").ShouldNot(HaveOccurred())

		Eventually(framework.HOCheckZeroDaemonSetJobJobsWasCreated(client, myDaemonSetJob), "10s", "5s").ShouldNot(HaveOccurred())
	})

})

func deleteDaemonSetJob(client versioned.Interface, daemonsetjob *dapi.DaemonSetJob) {
	client.DaemonsetjobV1().DaemonSetJobs(daemonsetjob.Namespace).Delete(daemonsetjob.Name, nil)
	By("DaemonSetJob deleted")
}

func deleteAllJobsFromDaemonSetJob(kubeClient clientset.Interface, daemonsetjob *dapi.DaemonSetJob) {
	err := kubeClient.Batch().Jobs(daemonsetjob.Namespace).DeleteCollection(controller.CascadeDeleteOptions(0), metav1.ListOptions{
		LabelSelector: controller.InferDaemonSetJobLabelSelectorForJobs(daemonsetjob).String()})
	if err != nil {
		By("Problem deleting DaemonSetJob jobs")
		return
	}
	By("DaemonSetJob jobs deleted")
}
