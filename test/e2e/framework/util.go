package framework

import (
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	clientset "k8s.io/client-go/kubernetes"

	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

func nowStamp() string {
	return time.Now().Format(time.StampMilli)
}

func log(level string, format string, args ...interface{}) {
	fmt.Fprintf(ginkgo.GinkgoWriter, nowStamp()+": "+level+": "+format+"\n", args...)
}

// Logf logs in e2e framework
func Logf(format string, args ...interface{}) {
	log("INFO", format, args...)
}

// Failf reports a failure in the current e2e
func Failf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log("INFO", msg)
	ginkgo.Fail(nowStamp()+": "+msg, 1)
}

// BuildAndSetClients builds and initilize workflow and kube client
func BuildAndSetClients() (versioned.Interface, versioned.Interface, *clientset.Clientset) {
	f, err := NewFramework()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(f).ShouldNot(BeNil())

	kubeClient, err := f.kubeClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(kubeClient).ShouldNot(BeNil())

	workflowClient, err := f.workflowClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(workflowClient).ShouldNot(BeNil())

	cronWorkflowClient, err := f.cronWorkflowClient()
	Ω(err).ShouldNot(HaveOccurred())
	Ω(cronWorkflowClient).ShouldNot(BeNil())

	return workflowClient, cronWorkflowClient, kubeClient
}
