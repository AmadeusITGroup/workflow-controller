package e2e

import (
	"github.com/golang/glog"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	api "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/amadeusitgroup/workflow-controller/test/e2e/framework"
)

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
})
