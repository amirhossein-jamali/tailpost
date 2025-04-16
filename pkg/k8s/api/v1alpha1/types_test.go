package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRegister(t *testing.T) {
	// Create a new scheme
	scheme := runtime.NewScheme()

	// Register our types with the scheme
	if err := Register(scheme); err != nil {
		t.Fatalf("Failed to register types with scheme: %v", err)
	}

	// Test creating TailpostAgent from gvk
	gvk := SchemeGroupVersion.WithKind("TailpostAgent")
	obj, err := scheme.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create TailpostAgent from gvk: %v", err)
	}
	if _, ok := obj.(*TailpostAgent); !ok {
		t.Errorf("Expected obj to be TailpostAgent, got %T", obj)
	}

	// Test creating TailpostAgentList from gvk
	gvk = SchemeGroupVersion.WithKind("TailpostAgentList")
	obj, err = scheme.New(gvk)
	if err != nil {
		t.Fatalf("Failed to create TailpostAgentList from gvk: %v", err)
	}
	if _, ok := obj.(*TailpostAgentList); !ok {
		t.Errorf("Expected obj to be TailpostAgentList, got %T", obj)
	}
}

func TestKind(t *testing.T) {
	kind := Kind("TailpostAgent")
	expected := schema.GroupKind{Group: "tailpost.elastic.co", Kind: "TailpostAgent"}
	if kind != expected {
		t.Errorf("Expected Kind('TailpostAgent') to be %v, got %v", expected, kind)
	}
}

func TestResource(t *testing.T) {
	resource := Resource("tailpostagents")
	expected := schema.GroupResource{Group: "tailpost.elastic.co", Resource: "tailpostagents"}
	if resource != expected {
		t.Errorf("Expected Resource('tailpostagents') to be %v, got %v", expected, resource)
	}
}

func TestDeepCopy(t *testing.T) {
	// Create a TailpostAgent instance with non-nil fields
	original := &TailpostAgent{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TailpostAgent",
			APIVersion: "tailpost.elastic.co/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: TailpostAgentSpec{
			Replicas: func() *int32 { i := int32(2); return &i }(),
			Image:    "tailpost:latest",
			LogSources: []LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test.log",
				},
			},
			ServerURL: "http://example.com",
		},
		Status: TailpostAgentStatus{
			AvailableReplicas: 1,
			Conditions: []TailpostAgentCondition{
				{
					Type:    "Available",
					Status:  "True",
					Reason:  "TestReason",
					Message: "Test message",
				},
			},
		},
	}

	// Test DeepCopy
	copy := original.DeepCopy()
	if copy == original {
		t.Error("DeepCopy returned the same pointer")
	}

	// Verify all fields are equal but separate
	if copy.Name != original.Name {
		t.Errorf("Expected Name to be %v, got %v", original.Name, copy.Name)
	}
	if *copy.Spec.Replicas != *original.Spec.Replicas {
		t.Errorf("Expected Replicas to be %v, got %v", *original.Spec.Replicas, *copy.Spec.Replicas)
	}
	if len(copy.Spec.LogSources) != len(original.Spec.LogSources) {
		t.Errorf("Expected LogSources length to be %v, got %v", len(original.Spec.LogSources), len(copy.Spec.LogSources))
	}
	if len(copy.Status.Conditions) != len(original.Status.Conditions) {
		t.Errorf("Expected Conditions length to be %v, got %v", len(original.Status.Conditions), len(copy.Status.Conditions))
	}

	// Verify that changing copy doesn't affect original
	*copy.Spec.Replicas = 3
	if *copy.Spec.Replicas == *original.Spec.Replicas {
		t.Error("Changing copy Replicas affected original")
	}

	// Test DeepCopyObject
	objCopy := original.DeepCopyObject()
	if objCopy == original {
		t.Error("DeepCopyObject returned the same pointer")
	}
	typedCopy, ok := objCopy.(*TailpostAgent)
	if !ok {
		t.Fatalf("DeepCopyObject didn't return a *TailpostAgent")
	}
	if typedCopy.Name != original.Name {
		t.Errorf("Expected Name to be %v, got %v", original.Name, typedCopy.Name)
	}
}

func TestTailpostAgentListDeepCopy(t *testing.T) {
	// Create a TailpostAgentList
	original := &TailpostAgentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TailpostAgentList",
			APIVersion: "tailpost.elastic.co/v1alpha1",
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: []TailpostAgent{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "agent1",
				},
				Spec: TailpostAgentSpec{
					Image: "tailpost:latest",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "agent2",
				},
				Spec: TailpostAgentSpec{
					Image: "tailpost:dev",
				},
			},
		},
	}

	// Test DeepCopy
	copy := original.DeepCopy()
	if copy == original {
		t.Error("DeepCopy returned the same pointer")
	}

	// Verify fields
	if len(copy.Items) != len(original.Items) {
		t.Errorf("Expected Items length to be %v, got %v", len(original.Items), len(copy.Items))
	}
	if copy.Items[0].Name != original.Items[0].Name {
		t.Errorf("Expected first item Name to be %v, got %v", original.Items[0].Name, copy.Items[0].Name)
	}

	// Verify that changing copy doesn't affect original
	copy.Items[0].Name = "changed"
	if copy.Items[0].Name == original.Items[0].Name {
		t.Error("Changing copy Items affected original")
	}

	// Test DeepCopyObject
	objCopy := original.DeepCopyObject()
	if objCopy == original {
		t.Error("DeepCopyObject returned the same pointer")
	}
	typedCopy, ok := objCopy.(*TailpostAgentList)
	if !ok {
		t.Fatalf("DeepCopyObject didn't return a *TailpostAgentList")
	}
	if len(typedCopy.Items) != len(original.Items) {
		t.Errorf("Expected Items length to be %v, got %v", len(original.Items), len(typedCopy.Items))
	}
}
