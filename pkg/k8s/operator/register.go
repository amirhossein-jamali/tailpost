package operator

import (
	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName is the group name used in this package
	GroupName = "tailpost.elastic.co"
	// GroupVersion is the API version
	GroupVersion = "v1alpha1"
)

// SchemeGroupVersion is the group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder initializes a scheme builder
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme is a global function that registers this API group & version to a scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes adds the list of known types to Scheme
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&v1alpha1.TailpostAgent{},
		&v1alpha1.TailpostAgentList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
