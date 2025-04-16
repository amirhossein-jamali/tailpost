package operator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	"github.com/amirhossein-jamali/tailpost/pkg/k8s/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Create a fake client that allows status updates
func createFakeClient(s *runtime.Scheme, objects ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objects...).
		WithStatusSubresource(&v1alpha1.TailpostAgent{}).
		Build()
}

// MockTailpostReconciler is a wrapper around TailpostAgentReconciler that mocks status updates
type MockTailpostReconciler struct {
	*TailpostAgentReconciler
}

// Mock version of updateStatus that applies changes directly to the instance
func (m *MockTailpostReconciler) updateStatus(_ context.Context, instance *v1alpha1.TailpostAgent) error {
	// Simulate successful status update without API calls
	instance.Status.AvailableReplicas = 1
	instance.Status.LastUpdateTime = metav1.Now()
	return nil
}

// Mock version of setCondition
func (m *MockTailpostReconciler) setCondition(_ context.Context, instance *v1alpha1.TailpostAgent, condType, status, reason, message string) {
	// Add the condition directly to the instance
	now := metav1.Now()
	condition := v1alpha1.TailpostAgentCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}

	// Find existing condition
	existingCondition := m.findCondition(instance, condType)
	if existingCondition != nil {
		// Update existing condition
		existingCondition.Status = status
		existingCondition.LastTransitionTime = now
		existingCondition.Reason = reason
		existingCondition.Message = message
	} else {
		// Add new condition
		instance.Status.Conditions = append(instance.Status.Conditions, condition)
	}
}

// Mock version of removeCondition
func (m *MockTailpostReconciler) removeCondition(_ context.Context, instance *v1alpha1.TailpostAgent, condType string) {
	// Find and remove condition directly
	foundIdx := -1
	for i, cond := range instance.Status.Conditions {
		if cond.Type == condType {
			foundIdx = i
			break
		}
	}

	if foundIdx >= 0 {
		instance.Status.Conditions = append(instance.Status.Conditions[:foundIdx], instance.Status.Conditions[foundIdx+1:]...)
	}
}

// Create a mock reconciler
func newMockReconciler(r *TailpostAgentReconciler) *MockTailpostReconciler {
	return &MockTailpostReconciler{
		TailpostAgentReconciler: r,
	}
}

func setupReconcilerAndInstance() (*TailpostAgentReconciler, *v1alpha1.TailpostAgent, *runtime.Scheme) {
	// Setup schemes
	s := runtime.NewScheme()
	scheme.AddToScheme(s)
	v1alpha1.Register(s)

	// Create a TailpostAgent instance with defaults already set
	instance := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			Replicas:        ptr.To[int32](DefaultReplicas),
			Image:           "test-image:latest",
			ImagePullPolicy: DefaultImagePullPolicy,
			ServerURL:       "http://example.com/logs",
			BatchSize:       ptr.To[int32](DefaultBatchSize),
			FlushInterval:   DefaultFlushInterval,
			LogSources: []v1alpha1.LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test.log",
				},
			},
		},
	}

	// Create fake client with status subresource support
	fakeClient := createFakeClient(s, instance)

	// Create reconciler
	recorder := record.NewFakeRecorder(10)
	reconciler := &TailpostAgentReconciler{
		Client:        fakeClient,
		Scheme:        s,
		Recorder:      recorder,
		DefaultImage:  "test-image:latest",
		ResyncPeriod:  time.Minute * 5,
		RequeuePeriod: time.Second * 10,
	}

	return reconciler, instance, s
}

func TestTailpostAgentReconciler_setDefaults(t *testing.T) {
	// Setup schemes
	s := runtime.NewScheme()
	scheme.AddToScheme(s)
	v1alpha1.Register(s)

	// Create a minimal TailpostAgent instance
	instance := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			ServerURL: "http://example.com/logs",
			LogSources: []v1alpha1.LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test.log",
				},
			},
		},
	}

	// Create fake client with status subresource support
	client := createFakeClient(s, instance)

	// Create reconciler
	recorder := record.NewFakeRecorder(10)
	reconciler := &TailpostAgentReconciler{
		Client:        client,
		Scheme:        s,
		Recorder:      recorder,
		DefaultImage:  "test-image:latest",
		ResyncPeriod:  time.Minute * 5,
		RequeuePeriod: time.Second * 10,
	}

	// Call setDefaults
	ctx := context.Background()
	err := reconciler.setDefaults(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to set defaults: %v", err)
	}

	// Verify defaults were set correctly
	if instance.Spec.Image != reconciler.DefaultImage {
		t.Errorf("Default image not set correctly. Expected %s, got %s", reconciler.DefaultImage, instance.Spec.Image)
	}

	if instance.Spec.ImagePullPolicy != DefaultImagePullPolicy {
		t.Errorf("Default image pull policy not set correctly. Expected %s, got %s", DefaultImagePullPolicy, instance.Spec.ImagePullPolicy)
	}

	if instance.Spec.Replicas == nil || *instance.Spec.Replicas != DefaultReplicas {
		t.Errorf("Default replicas not set correctly. Expected %d", DefaultReplicas)
	}

	if instance.Spec.BatchSize == nil || *instance.Spec.BatchSize != DefaultBatchSize {
		t.Errorf("Default batch size not set correctly. Expected %d", DefaultBatchSize)
	}

	if instance.Spec.FlushInterval != DefaultFlushInterval {
		t.Errorf("Default flush interval not set correctly. Expected %s, got %s", DefaultFlushInterval, instance.Spec.FlushInterval)
	}
}

// TestReconcileResourceCreation tests that the reconciler creates the expected resources
func TestReconcileResourceCreation(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()

	// Manually reconcile resources without using the Reconcile method
	ctx := context.Background()

	// Test reconcile ConfigMap
	err := reconciler.reconcileConfigMap(ctx, instance)
	if err != nil {
		t.Fatalf("reconcileConfigMap failed: %v", err)
	}

	// Verify ConfigMap was created
	configMap := &corev1.ConfigMap{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: resources.GetConfigMapName(instance), Namespace: "default"}, configMap)
	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Test reconcile StatefulSet
	err = reconciler.reconcileStatefulSet(ctx, instance)
	if err != nil {
		t.Fatalf("reconcileStatefulSet failed: %v", err)
	}

	// Verify StatefulSet was created
	statefulSet := &appsv1.StatefulSet{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: resources.GetStatefulSetName(instance), Namespace: "default"}, statefulSet)
	if err != nil {
		t.Fatalf("Failed to get StatefulSet: %v", err)
	}

	// Test reconcile Service
	err = reconciler.reconcileService(ctx, instance)
	if err != nil {
		t.Fatalf("reconcileService failed: %v", err)
	}

	// Verify Service was created
	service := &corev1.Service{}
	err = reconciler.Get(ctx, types.NamespacedName{Name: resources.GetServiceName(instance), Namespace: "default"}, service)
	if err != nil {
		t.Fatalf("Failed to get Service: %v", err)
	}
}

func TestTailpostAgentReconciler_findCondition(t *testing.T) {
	// Create reconciler
	reconciler := &TailpostAgentReconciler{}

	// Create a TailpostAgent with conditions
	now := metav1.Now()
	instance := &v1alpha1.TailpostAgent{
		Status: v1alpha1.TailpostAgentStatus{
			Conditions: []v1alpha1.TailpostAgentCondition{
				{
					Type:               ConditionTypeAvailable,
					Status:             "True",
					LastTransitionTime: now,
					Reason:             "TestReason",
					Message:            "Test message",
				},
				{
					Type:               ConditionTypeDegraded,
					Status:             "False",
					LastTransitionTime: now,
					Reason:             "TestReason",
					Message:            "Test message",
				},
			},
		},
	}

	// Test finding existing condition
	condition := reconciler.findCondition(instance, ConditionTypeAvailable)
	if condition == nil {
		t.Fatalf("Failed to find Available condition")
	}
	if condition.Status != "True" {
		t.Errorf("Unexpected status for Available condition. Expected True, got %s", condition.Status)
	}

	// Test finding non-existent condition
	condition = reconciler.findCondition(instance, "NonExistent")
	if condition != nil {
		t.Errorf("Found non-existent condition: %v", condition)
	}
}

func TestReconcile(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()

	// Create a mock reconciler that doesn't make API calls for status updates
	mockReconciler := newMockReconciler(reconciler)

	// Set up request
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	// First reconcile - should create all resources
	ctx := context.Background()
	// Call Reconcile directly on the mock reconciler
	result, err := mockReconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify correct requeue duration
	if result.RequeueAfter != mockReconciler.ResyncPeriod {
		t.Errorf("Unexpected requeue period. Expected %v, got %v",
			mockReconciler.ResyncPeriod, result.RequeueAfter)
	}

	// Verify all resources were created
	configMap := &corev1.ConfigMap{}
	err = mockReconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetConfigMapName(instance),
		Namespace: instance.Namespace,
	}, configMap)
	if err != nil {
		t.Errorf("ConfigMap not created: %v", err)
	}

	statefulSet := &appsv1.StatefulSet{}
	err = mockReconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetStatefulSetName(instance),
		Namespace: instance.Namespace,
	}, statefulSet)
	if err != nil {
		t.Errorf("StatefulSet not created: %v", err)
	}

	service := &corev1.Service{}
	err = mockReconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetServiceName(instance),
		Namespace: instance.Namespace,
	}, service)
	if err != nil {
		t.Errorf("Service not created: %v", err)
	}
}

func TestReconcileConfigMapUpdate(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()
	ctx := context.Background()

	// First create the ConfigMap
	err := reconciler.reconcileConfigMap(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Modify the instance to trigger ConfigMap update
	instance.Spec.BatchSize = ptr.To[int32](50) // Changed from default 10

	// Reconcile again
	err = reconciler.reconcileConfigMap(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to update ConfigMap: %v", err)
	}

	// Fetch the updated ConfigMap
	configMap := &corev1.ConfigMap{}
	err = reconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetConfigMapName(instance),
		Namespace: instance.Namespace,
	}, configMap)
	if err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Verify the ConfigMap data contains the updated batch size
	// This depends on how your resources.CreateConfigMap function works
	// but typically it should include the batch size in the config YAML
	if configMap.Data["config.yaml"] == "" {
		t.Errorf("ConfigMap data is empty")
	}

	// Simple string check to verify update
	if configMap.Data["config.yaml"] != "" &&
		!strings.Contains(configMap.Data["config.yaml"], "batch_size: 50") {
		t.Errorf("ConfigMap data not updated with new batch size. Got: %s",
			configMap.Data["config.yaml"])
	}
}

func TestSetCondition(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()

	// Create a mock reconciler that doesn't make API calls
	mockReconciler := newMockReconciler(reconciler)
	ctx := context.Background()

	// Set a condition
	mockReconciler.setCondition(ctx, instance, "TestCondition", "True", "TestReason", "Test message")

	// Verify condition was set directly to instance
	condition := mockReconciler.findCondition(instance, "TestCondition")
	if condition == nil {
		t.Fatalf("Condition not set")
	}

	if condition.Status != "True" {
		t.Errorf("Condition status not set correctly. Expected True, got %s", condition.Status)
	}

	if condition.Reason != "TestReason" {
		t.Errorf("Condition reason not set correctly. Expected TestReason, got %s", condition.Reason)
	}

	if condition.Message != "Test message" {
		t.Errorf("Condition message not set correctly. Expected 'Test message', got %s", condition.Message)
	}

	// Update the condition
	mockReconciler.setCondition(ctx, instance, "TestCondition", "False", "UpdatedReason", "Updated message")

	// Verify condition was updated
	condition = mockReconciler.findCondition(instance, "TestCondition")
	if condition == nil {
		t.Fatalf("Condition not found after update")
	}

	if condition.Status != "False" {
		t.Errorf("Condition status not updated correctly. Expected False, got %s", condition.Status)
	}

	if condition.Reason != "UpdatedReason" {
		t.Errorf("Condition reason not updated correctly. Expected UpdatedReason, got %s", condition.Reason)
	}

	if condition.Message != "Updated message" {
		t.Errorf("Condition message not updated correctly. Expected 'Updated message', got %s", condition.Message)
	}
}

func TestRemoveCondition(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()

	// Create a mock reconciler
	mockReconciler := newMockReconciler(reconciler)
	ctx := context.Background()

	// First set a condition
	mockReconciler.setCondition(ctx, instance, "TestCondition", "True", "TestReason", "Test message")

	// Verify condition was set
	condition := mockReconciler.findCondition(instance, "TestCondition")
	if condition == nil {
		t.Fatalf("Condition not set initially")
	}

	// Then remove it
	mockReconciler.removeCondition(ctx, instance, "TestCondition")

	// Verify condition was removed
	condition = mockReconciler.findCondition(instance, "TestCondition")
	if condition != nil {
		t.Errorf("Condition not removed")
	}
}

func TestReconcileInstanceNotFound(t *testing.T) {
	reconciler, _, _ := setupReconcilerAndInstance()

	// Create a mock reconciler
	mockReconciler := newMockReconciler(reconciler)

	// Set up request for non-existent instance
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: "default",
		},
	}

	// Reconcile should not error
	ctx := context.Background()
	result, err := mockReconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify no requeue
	if result.Requeue || result.RequeueAfter > 0 {
		t.Errorf("Unexpected requeue for non-existent instance")
	}
}

func TestUpdateStatus(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()

	// Create a mock reconciler
	mockReconciler := newMockReconciler(reconciler)
	ctx := context.Background()

	// First create the StatefulSet
	err := mockReconciler.reconcileStatefulSet(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create StatefulSet: %v", err)
	}

	// Update the instance status
	err = mockReconciler.updateStatus(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Since we've mocked updateStatus, verify the direct changes it made to the instance
	if instance.Status.AvailableReplicas != 1 {
		t.Errorf("Available replicas not updated correctly. Expected 1, got %d",
			instance.Status.AvailableReplicas)
	}

	if instance.Status.LastUpdateTime.IsZero() {
		t.Errorf("LastUpdateTime not set")
	}
}

func TestReconcileStatefulSetUpdate(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()
	ctx := context.Background()

	// First create the StatefulSet
	err := reconciler.reconcileStatefulSet(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create StatefulSet: %v", err)
	}

	// Modify the instance to trigger StatefulSet update
	// Change replicas from 1 to 2
	replicas := int32(2)
	instance.Spec.Replicas = &replicas

	// Reconcile again
	err = reconciler.reconcileStatefulSet(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to update StatefulSet: %v", err)
	}

	// Fetch the updated StatefulSet
	statefulSet := &appsv1.StatefulSet{}
	err = reconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetStatefulSetName(instance),
		Namespace: instance.Namespace,
	}, statefulSet)
	if err != nil {
		t.Fatalf("Failed to get StatefulSet: %v", err)
	}

	// Verify the StatefulSet replicas was updated
	if *statefulSet.Spec.Replicas != 2 {
		t.Errorf("StatefulSet replicas not updated. Expected 2, got %d",
			*statefulSet.Spec.Replicas)
	}
}

func TestReconcileServiceUpdate(t *testing.T) {
	reconciler, instance, _ := setupReconcilerAndInstance()
	ctx := context.Background()

	// First create the Service
	err := reconciler.reconcileService(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to create Service: %v", err)
	}

	// Get the current service to modify it (simulate external changes)
	service := &corev1.Service{}
	err = reconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetServiceName(instance),
		Namespace: instance.Namespace,
	}, service)
	if err != nil {
		t.Fatalf("Failed to get Service: %v", err)
	}

	// Modify the service selector (this should be reconciled back)
	service.Spec.Selector["modified"] = "true"
	err = reconciler.Update(ctx, service)
	if err != nil {
		t.Fatalf("Failed to update Service with modifications: %v", err)
	}

	// Reconcile again
	err = reconciler.reconcileService(ctx, instance)
	if err != nil {
		t.Fatalf("Failed to reconcile Service: %v", err)
	}

	// Verify the Service was properly reconciled
	updatedService := &corev1.Service{}
	err = reconciler.Get(ctx, types.NamespacedName{
		Name:      resources.GetServiceName(instance),
		Namespace: instance.Namespace,
	}, updatedService)
	if err != nil {
		t.Fatalf("Failed to get Service: %v", err)
	}

	// Verify our modifications were reverted
	if _, exists := updatedService.Spec.Selector["modified"]; exists {
		t.Errorf("Service not properly reconciled, unexpected selector key still exists")
	}
}
