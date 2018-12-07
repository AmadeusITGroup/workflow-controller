package framework

import (
	"fmt"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/amadeusitgroup/workflow-controller/pkg/client"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

// Framework stores necessary info to run e2e
type Framework struct {
	KubeConfig *rest.Config
}

type frameworkContextType struct {
	KubeConfigPath string
}

// FrameworkContext stores globally the framework context
var FrameworkContext frameworkContextType

// NewFramework creates and initializes the a Framework struct
func NewFramework() (*Framework, error) {
	Logf("KubeconfigPath: %q", FrameworkContext.KubeConfigPath)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", FrameworkContext.KubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve kubeConfig:%v", err)
	}
	return &Framework{
		KubeConfig: kubeConfig,
	}, nil
}

func (f *Framework) kubeClient() (*clientset.Clientset, error) {
	return clientset.NewForConfig(f.KubeConfig)
}

func (f *Framework) client() (versioned.Interface, error) {
	c, err := client.NewWorkflowClient(f.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create workflow client: %v", err)
	}
	return c, err
}

func (f *Framework) cronWorkflowClient() (versioned.Interface, error) {
	c, err := client.NewCronWorkflowClient(f.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create cronWorkflow client: %v", err)
	}
	return c, err
}
