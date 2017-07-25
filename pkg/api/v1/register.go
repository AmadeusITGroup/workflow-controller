package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

const (
	// GroupName is the group name used in this package.
	GroupName = "dag.example.com"
	// ResourcePlural is the id to indentify pluarals
	ResourcePlural = "workflows"
	// ResourceSingular represents the id for identify singular resource
	ResourceSingular = "workflow"
	// ResourceKind
	ResourceKind = "Workflow"
	// ReourceVersion
	ResourceVersion = "v1"
)

// SchemeGroupVersion is the group version used to register these objects.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: ResourceVersion}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Workflow{},
		&WorkflowList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
