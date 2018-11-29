package client

import (
	"fmt"
	"reflect"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"

	cronworkflowV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/cronworkflow/v1"
	daemonsetjobV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/daemonsetjob/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/api/workflow"
	workflowV1 "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
	"github.com/amadeusitgroup/workflow-controller/pkg/client/clientset/versioned"
)

// DefineWorklowResource defines a WorkflowResource as a k8s CR
func DefineWorklowResource(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	workflowResourceName := workflowV1.ResourcePlural + "." + workflow.GroupName
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: workflowResourceName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   workflow.GroupName,
			Version: workflowV1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:     workflowV1.ResourcePlural,
				Singular:   workflowV1.ResourceSingular,
				Kind:       reflect.TypeOf(workflowV1.Workflow{}).Name(),
				ShortNames: []string{"wfl"},
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(workflowResourceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					glog.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(workflowResourceName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}

	return crd, nil
}

var cronWorkflowResourceName = cronworkflowV1.ResourcePlural + "." + workflow.GroupName

var cronWorkflowCRD = &apiextensionsv1beta1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: cronWorkflowResourceName,
	},
	Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
		Group:   workflow.GroupName,
		Version: cronworkflowV1.SchemeGroupVersion.Version,
		Scope:   apiextensionsv1beta1.NamespaceScoped,
		Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
			Plural:     cronworkflowV1.ResourcePlural,
			Singular:   cronworkflowV1.ResourceSingular,
			Kind:       reflect.TypeOf(cronworkflowV1.CronWorkflow{}).Name(),
			ShortNames: []string{"cwfl"},
		},
	},
}

// CheckCronWorkflowResourceDefinition checks wether or not CronWorkflow CRD is defined
func CheckCronWorkflowResourceDefinition(clientset apiextensionsclient.Interface) error {
	crd, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(cronWorkflowResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, cond := range crd.Status.Conditions {
		switch cond.Type {
		case apiextensionsv1beta1.Established:
			if cond.Status == apiextensionsv1beta1.ConditionTrue {
				return nil
			}
		case apiextensionsv1beta1.NamesAccepted:
			if cond.Status == apiextensionsv1beta1.ConditionFalse {
				return fmt.Errorf("name conflict registering %s", cronWorkflowResourceName)
			}
		}
	}
	return fmt.Errorf("no info registering %s", cronWorkflowResourceName)
}

// DefineCronWorkflowResource defines a CronWorkflowResource as a k8s CR
func DefineCronWorkflowResource(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	var crd *apiextensionsv1beta1.CustomResourceDefinition
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(cronWorkflowCRD)
	if err != nil {
		return nil, err
	}
	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(cronWorkflowResourceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					glog.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(cronWorkflowResourceName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}

	return crd, nil
}

// NewWorkflowClient builds and initializes a Client and a Scheme for Workflow CR
func NewWorkflowClient(cfg *rest.Config) (versioned.Interface, error) {
	scheme := runtime.NewScheme()
	if err := workflowV1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	config := *cfg
	config.GroupVersion = &workflowV1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	cs, err := versioned.NewForConfig(&config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

// NewCronWorkflowClient builds and initializes a Client and a Scheme for CronWorkflow CR
func NewCronWorkflowClient(cfg *rest.Config) (versioned.Interface, error) {
	scheme := runtime.NewScheme()
	if err := cronworkflowV1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	config := *cfg
	config.GroupVersion = &cronworkflowV1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	cs, err := versioned.NewForConfig(&config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

// NewDaemonSetJobClient builds and initializes a Client and a Scheme for DaemonSetJob CR
func NewDaemonSetJobClient(cfg *rest.Config) (versioned.Interface, error) {
	scheme := runtime.NewScheme()
	if err := daemonsetjobV1.AddToScheme(scheme); err != nil {
		return nil, err
	}

	config := *cfg
	config.GroupVersion = &daemonsetjobV1.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	cs, err := versioned.NewForConfig(&config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}

var daemonsetjobResourceName = daemonsetjobV1.ResourcePlural + "." + workflow.GroupName

var daemonsetjobCRD = &apiextensionsv1beta1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: daemonsetjobResourceName,
	},
	Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
		Group:   workflow.GroupName,
		Version: daemonsetjobV1.SchemeGroupVersion.Version,
		Scope:   apiextensionsv1beta1.NamespaceScoped,
		Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
			Plural:     daemonsetjobV1.ResourcePlural,
			Singular:   daemonsetjobV1.ResourceSingular,
			Kind:       reflect.TypeOf(daemonsetjobV1.DaemonSetJob{}).Name(),
			ShortNames: []string{"dsj"},
		},
	},
}

// DefineDaemonSetJobResource defines a DaemonSetJobResource as a k8s CR
func DefineDaemonSetJobResource(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	var crd *apiextensionsv1beta1.CustomResourceDefinition
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(daemonsetjobCRD)
	if err != nil {
		return nil, err
	}
	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(daemonsetjobResourceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					glog.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(daemonsetjobResourceName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}

	return crd, nil
}

// CheckDaemonSetJobResourceDefinition checks wether or not DaemonSetJob CRD is defined
func CheckDaemonSetJobResourceDefinition(clientset apiextensionsclient.Interface) error {
	crd, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(daemonsetjobResourceName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	for _, cond := range crd.Status.Conditions {
		switch cond.Type {
		case apiextensionsv1beta1.Established:
			if cond.Status == apiextensionsv1beta1.ConditionTrue {
				return nil
			}
		case apiextensionsv1beta1.NamesAccepted:
			if cond.Status == apiextensionsv1beta1.ConditionFalse {
				return fmt.Errorf("name conflict registering %s", daemonsetjobResourceName)
			}
		}
	}
	return fmt.Errorf("no info registering %s", daemonsetjobResourceName)
}
