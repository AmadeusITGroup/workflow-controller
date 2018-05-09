package e2e

import (
	"fmt"

	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"

	"github.com/amadeusitgroup/workflow-controller/test/e2e/framework"
)

var _ = Describe("Workflow CRUD", func() {
	It("check CronWorkflow registration", func() {
		workflowClient, kubeClient := framework.BuildAndSetClients()
		ns := api.NamespaceDefault

		Eventually(framework.HOCheckCronWorkflowRegistration(workflowClient, ns), "5s", "1s").ShouldNot(HaveOccurred())
	})
}