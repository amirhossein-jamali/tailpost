package operator

import (
	"testing"

	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRegister(t *testing.T) {
	// Create a new scheme
	s := runtime.NewScheme()

	// Register our types with the scheme
	if err := AddToScheme(s); err != nil {
		t.Fatalf("Failed to add types to scheme: %v", err)
	}

	// Verify SchemeGroupVersion is correct
	expectedGV := schema.GroupVersion{Group: "tailpost.elastic.co", Version: "v1alpha1"}
	if SchemeGroupVersion != expectedGV {
		t.Errorf("Expected GroupVersion to be %v, got %v", expectedGV, SchemeGroupVersion)
	}

	// Test that we can create a TailpostAgent from the scheme
	gvk := SchemeGroupVersion.WithKind("TailpostAgent")
	obj, err := s.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create TailpostAgent from scheme: %v", err)
	}

	// Verify the created object is a TailpostAgent
	if _, ok := obj.(*v1alpha1.TailpostAgent); !ok {
		t.Fatalf("Created object is not a TailpostAgent")
	}

	// Test that we can create a TailpostAgentList from the scheme
	gvk = SchemeGroupVersion.WithKind("TailpostAgentList")
	obj, err = s.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create TailpostAgentList from scheme: %v", err)
	}

	// Verify the created object is a TailpostAgentList
	if _, ok := obj.(*v1alpha1.TailpostAgentList); !ok {
		t.Fatalf("Created object is not a TailpostAgentList")
	}

	// Test Resource function
	resource := Resource("tailpostagents")
	expectedResource := schema.GroupResource{Group: "tailpost.elastic.co", Resource: "tailpostagents"}
	if resource != expectedResource {
		t.Errorf("Expected resource to be %v, got %v", expectedResource, resource)
	}
}
