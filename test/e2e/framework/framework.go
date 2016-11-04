package framework

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// Framework stores necessary info to run e2e
type Framework struct {
	KubeMasterURL  string
	KubeConfigPath string

	ResourceName    string
	ResourceGroup   string
	ResourceVersion string
}

type frameworkContextType struct {
	Host            string
	KubeConfigPath  string
	ResourceName    string
	ResourceGroup   string
	ResourceVersion string
}

var FrameworkContext frameworkContextType

func init() {
	fmt.Printf("framework.Init\n")
	flag.StringVar(&FrameworkContext.KubeConfigPath, clientcmd.RecommendedConfigPathFlag, os.Getenv(clientcmd.RecommendedConfigPathEnvVar), "Path to kubeconfig")
	flag.StringVar(&FrameworkContext.Host, "host", "http://127.0.0.1:8080", "The host, or apiserver, to connect to")
	flag.StringVar(&FrameworkContext.ResourceName, "name", "workflow", "The name of the resource")
	flag.StringVar(&FrameworkContext.ResourceGroup, "group", "example.com", "The API group of the resource")
	flag.StringVar(&FrameworkContext.ResourceVersion, "version", "v1", "The resource versions")
}

// NewFramework creates and initializes the a Framework struct
func NewFramework() *Framework {
	Logf("Host-> %q", FrameworkContext.Host)
	Logf("KubeconfigPath-> %q", FrameworkContext.KubeConfigPath)
	Logf("ResourceName-> %q", FrameworkContext.ResourceName)
	Logf("ResourceGroup-> %q", FrameworkContext.ResourceGroup)
	Logf("ResourceVersion-> %q", FrameworkContext.ResourceVersion)
	return &Framework{
		KubeMasterURL:   FrameworkContext.Host,
		KubeConfigPath:  FrameworkContext.KubeConfigPath,
		ResourceName:    FrameworkContext.ResourceName,
		ResourceGroup:   FrameworkContext.ResourceGroup,
		ResourceVersion: FrameworkContext.ResourceVersion,
	}
}
