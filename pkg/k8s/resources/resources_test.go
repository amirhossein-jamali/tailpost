package resources

import (
	"reflect"
	"strings"
	"testing"

	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetLabels(t *testing.T) {
	// Create a TailpostAgent
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-agent",
		},
	}

	// Get labels
	labels := GetLabels(agent)

	// Verify labels
	expected := map[string]string{
		"app.kubernetes.io/name":       Component,
		"app.kubernetes.io/instance":   agent.Name,
		"app.kubernetes.io/managed-by": "tailpost-operator",
	}

	if !reflect.DeepEqual(labels, expected) {
		t.Errorf("GetLabels() = %v, want %v", labels, expected)
	}
}

func TestGetConfigMapName(t *testing.T) {
	// Create a TailpostAgent
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-agent",
		},
	}

	// Get configmap name
	name := GetConfigMapName(agent)

	// Verify name
	expected := "test-agent-config"
	if name != expected {
		t.Errorf("GetConfigMapName() = %v, want %v", name, expected)
	}
}

func TestGetStatefulSetName(t *testing.T) {
	// Create a TailpostAgent
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-agent",
		},
	}

	// Get statefulset name
	name := GetStatefulSetName(agent)

	// Verify name
	expected := "test-agent"
	if name != expected {
		t.Errorf("GetStatefulSetName() = %v, want %v", name, expected)
	}
}

func TestGetServiceName(t *testing.T) {
	// Create a TailpostAgent
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-agent",
		},
	}

	// Get service name
	name := GetServiceName(agent)

	// Verify name
	expected := "test-agent"
	if name != expected {
		t.Errorf("GetServiceName() = %v, want %v", name, expected)
	}
}

func TestCreateConfigMap(t *testing.T) {
	// Create a TailpostAgent
	batchSize := int32(10)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			ServerURL:     "http://example.com/logs",
			BatchSize:     &batchSize,
			FlushInterval: "5s",
			LogSources: []v1alpha1.LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test.log",
				},
			},
		},
	}

	// Create configmap
	configMap, err := CreateConfigMap(agent)
	if err != nil {
		t.Fatalf("CreateConfigMap() error = %v", err)
	}

	// Verify configmap
	if configMap.Name != GetConfigMapName(agent) {
		t.Errorf("ConfigMap name = %v, want %v", configMap.Name, GetConfigMapName(agent))
	}

	if configMap.Namespace != agent.Namespace {
		t.Errorf("ConfigMap namespace = %v, want %v", configMap.Namespace, agent.Namespace)
	}

	if !reflect.DeepEqual(configMap.Labels, GetLabels(agent)) {
		t.Errorf("ConfigMap labels = %v, want %v", configMap.Labels, GetLabels(agent))
	}

	// Verify data contains our config
	configData, ok := configMap.Data[ConfigFileName]
	if !ok {
		t.Errorf("ConfigMap data missing %s", ConfigFileName)
	}

	// Check if config data contains the expected values
	if !contains(configData, "server_url: http://example.com/logs") {
		t.Errorf("ConfigMap data missing server_url")
	}

	if !contains(configData, "batch_size: 10") {
		t.Errorf("ConfigMap data missing batch_size")
	}

	if !contains(configData, "flush_interval: 5s") {
		t.Errorf("ConfigMap data missing flush_interval")
	}

	if !contains(configData, "log_path: /var/log/test.log") {
		t.Errorf("ConfigMap data missing log_path")
	}
}

func TestCreateStatefulSet(t *testing.T) {
	// Create a TailpostAgent
	replicas := int32(2)
	batchSize := int32(10)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			Replicas:        &replicas,
			Image:           "tailpost:v1",
			ImagePullPolicy: "IfNotPresent",
			ServiceAccount:  "tailpost-sa",
			ServerURL:       "http://example.com/logs",
			BatchSize:       &batchSize,
			FlushInterval:   "5s",
			LogSources: []v1alpha1.LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test.log",
				},
			},
			Resources: v1alpha1.ResourceRequirementsSpec{
				Limits: v1alpha1.ResourceListSpec{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Requests: v1alpha1.ResourceListSpec{
					CPU:    "50m",
					Memory: "64Mi",
				},
			},
		},
	}

	// Create statefulset
	statefulSet, err := CreateStatefulSet(agent)
	if err != nil {
		t.Fatalf("CreateStatefulSet() error = %v", err)
	}

	// Verify statefulset
	if statefulSet.Name != GetStatefulSetName(agent) {
		t.Errorf("StatefulSet name = %v, want %v", statefulSet.Name, GetStatefulSetName(agent))
	}

	if statefulSet.Namespace != agent.Namespace {
		t.Errorf("StatefulSet namespace = %v, want %v", statefulSet.Namespace, agent.Namespace)
	}

	if !reflect.DeepEqual(statefulSet.Labels, GetLabels(agent)) {
		t.Errorf("StatefulSet labels = %v, want %v", statefulSet.Labels, GetLabels(agent))
	}

	// Verify replicas
	if *statefulSet.Spec.Replicas != *agent.Spec.Replicas {
		t.Errorf("StatefulSet replicas = %v, want %v", *statefulSet.Spec.Replicas, *agent.Spec.Replicas)
	}

	// Verify container
	container := statefulSet.Spec.Template.Spec.Containers[0]
	if container.Image != agent.Spec.Image {
		t.Errorf("Container image = %v, want %v", container.Image, agent.Spec.Image)
	}

	if string(container.ImagePullPolicy) != agent.Spec.ImagePullPolicy {
		t.Errorf("Container image pull policy = %v, want %v", container.ImagePullPolicy, agent.Spec.ImagePullPolicy)
	}

	// Verify resources
	cpu := container.Resources.Limits[corev1.ResourceCPU]
	expectedCPU := resource.MustParse(agent.Spec.Resources.Limits.CPU)
	if !cpu.Equal(expectedCPU) {
		t.Errorf("Container CPU limit = %v, want %v", cpu, expectedCPU)
	}

	memory := container.Resources.Limits[corev1.ResourceMemory]
	expectedMemory := resource.MustParse(agent.Spec.Resources.Limits.Memory)
	if !memory.Equal(expectedMemory) {
		t.Errorf("Container memory limit = %v, want %v", memory, expectedMemory)
	}

	// Verify service account
	if statefulSet.Spec.Template.Spec.ServiceAccountName != agent.Spec.ServiceAccount {
		t.Errorf("StatefulSet service account = %v, want %v", statefulSet.Spec.Template.Spec.ServiceAccountName, agent.Spec.ServiceAccount)
	}
}

func TestCreateService(t *testing.T) {
	// Create a TailpostAgent
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
	}

	// Create service
	service := CreateService(agent)

	// Verify service
	if service.Name != GetServiceName(agent) {
		t.Errorf("Service name = %v, want %v", service.Name, GetServiceName(agent))
	}

	if service.Namespace != agent.Namespace {
		t.Errorf("Service namespace = %v, want %v", service.Namespace, agent.Namespace)
	}

	if !reflect.DeepEqual(service.Labels, GetLabels(agent)) {
		t.Errorf("Service labels = %v, want %v", service.Labels, GetLabels(agent))
	}

	// Verify selector
	if !reflect.DeepEqual(service.Spec.Selector, GetLabels(agent)) {
		t.Errorf("Service selector = %v, want %v", service.Spec.Selector, GetLabels(agent))
	}

	// Verify port
	if len(service.Spec.Ports) != 1 {
		t.Errorf("Service ports count = %v, want 1", len(service.Spec.Ports))
	} else {
		port := service.Spec.Ports[0]
		if port.Port != MetricsPort {
			t.Errorf("Service port = %v, want %v", port.Port, MetricsPort)
		}
	}
}

func TestConfigMapNeedsUpdate(t *testing.T) {
	// Create ConfigMaps
	current := &corev1.ConfigMap{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	same := &corev1.ConfigMap{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	different := &corev1.ConfigMap{
		Data: map[string]string{
			"key1": "value1",
			"key2": "changed",
		},
	}

	// Test with same data
	if ConfigMapNeedsUpdate(current, same) {
		t.Errorf("ConfigMapNeedsUpdate() = true, want false for same ConfigMaps")
	}

	// Test with different data
	if !ConfigMapNeedsUpdate(current, different) {
		t.Errorf("ConfigMapNeedsUpdate() = false, want true for different ConfigMaps")
	}
}

func TestStatefulSetNeedsUpdate(t *testing.T) {
	// Create StatefulSets
	replicas1 := int32(1)
	replicas2 := int32(2)

	current := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas1,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "image:v1",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	same := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas1,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "image:v1",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	differentReplicas := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas2,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "image:v1",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	differentImage := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas1,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "image:v2",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("100m"),
								},
							},
						},
					},
				},
			},
		},
	}

	differentResources := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas1,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "image:v1",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("200m"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Test with same StatefulSets
	if StatefulSetNeedsUpdate(current, same) {
		t.Errorf("StatefulSetNeedsUpdate() = true, want false for same StatefulSets")
	}

	// Test with different replicas
	if !StatefulSetNeedsUpdate(current, differentReplicas) {
		t.Errorf("StatefulSetNeedsUpdate() = false, want true for different replicas")
	}

	// Test with different image
	if !StatefulSetNeedsUpdate(current, differentImage) {
		t.Errorf("StatefulSetNeedsUpdate() = false, want true for different image")
	}

	// Test with different resources
	if !StatefulSetNeedsUpdate(current, differentResources) {
		t.Errorf("StatefulSetNeedsUpdate() = false, want true for different resources")
	}
}

func TestServiceNeedsUpdate(t *testing.T) {
	// Create Services
	current := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "tailpost",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	same := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "tailpost",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	differentSelector := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "changed",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}

	differentPorts := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "tailpost",
			},
			Ports: []corev1.ServicePort{
				{
					Port: 9090,
				},
			},
		},
	}

	// Test with same Services
	if ServiceNeedsUpdate(current, same) {
		t.Errorf("ServiceNeedsUpdate() = true, want false for same Services")
	}

	// Test with different selector
	if !ServiceNeedsUpdate(current, differentSelector) {
		t.Errorf("ServiceNeedsUpdate() = false, want true for different selector")
	}

	// Test with different ports
	if !ServiceNeedsUpdate(current, differentPorts) {
		t.Errorf("ServiceNeedsUpdate() = false, want true for different ports")
	}
}

func TestYaml(t *testing.T) {
	// Create data
	data := map[string]interface{}{
		"string":  "value",
		"int":     10,
		"float":   3.14,
		"boolean": true,
	}

	// Convert to YAML
	yamlStr, err := yaml(data)
	if err != nil {
		t.Fatalf("yaml() error = %v", err)
	}

	// Verify YAML
	if !contains(yamlStr, "string: value") {
		t.Errorf("YAML missing string value")
	}

	if !contains(yamlStr, "int: 10") {
		t.Errorf("YAML missing int value")
	}

	if !contains(yamlStr, "float: 3.14") {
		t.Errorf("YAML missing float value")
	}

	if !contains(yamlStr, "boolean: true") {
		t.Errorf("YAML missing boolean value")
	}
}

func TestParseQuantity(t *testing.T) {
	// Parse CPU quantity
	cpuValue := "100m"
	cpuQuantity := parseQuantity(cpuValue)
	expectedCPUQuantity := resource.MustParse(cpuValue)
	if !cpuQuantity.Equal(expectedCPUQuantity) {
		t.Errorf("parseQuantity() for CPU = %v, want %v", cpuQuantity, expectedCPUQuantity)
	}

	// Parse memory quantity
	memoryValue := "128Mi"
	memoryQuantity := parseQuantity(memoryValue)
	expectedMemoryQuantity := resource.MustParse(memoryValue)
	if !memoryQuantity.Equal(expectedMemoryQuantity) {
		t.Errorf("parseQuantity() for memory = %v, want %v", memoryQuantity, expectedMemoryQuantity)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestCreateConfigMapWithEmptyValues tests creating a ConfigMap with empty values
func TestCreateConfigMapWithEmptyValues(t *testing.T) {
	// Create a TailpostAgent with minimal values
	batchSize := int32(0)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			ServerURL:     "",
			BatchSize:     &batchSize,
			FlushInterval: "",
		},
	}

	// Create configmap
	configMap, err := CreateConfigMap(agent)
	if err != nil {
		t.Fatalf("CreateConfigMap() error = %v", err)
	}

	// Verify configmap was created despite empty values
	if configMap.Name != GetConfigMapName(agent) {
		t.Errorf("ConfigMap name = %v, want %v", configMap.Name, GetConfigMapName(agent))
	}

	// Verify config data reflects empty values
	configData, ok := configMap.Data[ConfigFileName]
	if !ok {
		t.Errorf("ConfigMap data missing %s", ConfigFileName)
	}

	if !contains(configData, "batch_size: 0") {
		t.Errorf("ConfigMap data missing batch_size")
	}
}

// TestCreateConfigMapWithMultipleLogSources tests creating a ConfigMap with multiple log sources
func TestCreateConfigMapWithMultipleLogSources(t *testing.T) {
	// Create a TailpostAgent with multiple log sources
	batchSize := int32(10)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			ServerURL:     "http://example.com/logs",
			BatchSize:     &batchSize,
			FlushInterval: "5s",
			LogSources: []v1alpha1.LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test1.log",
				},
				{
					Type: "file",
					Path: "/var/log/test2.log",
				},
				{
					Type: "syslog",
				},
			},
		},
	}

	// Create configmap
	configMap, err := CreateConfigMap(agent)
	if err != nil {
		t.Fatalf("CreateConfigMap() error = %v", err)
	}

	// Verify config data contains the first file log source
	configData := configMap.Data[ConfigFileName]
	if !contains(configData, "log_path: /var/log/test1.log") {
		t.Errorf("ConfigMap data should contain the first file log path")
	}
}

// TestCreateStatefulSetWithoutResources tests creating a StatefulSet without resource requirements
func TestCreateStatefulSetWithoutResources(t *testing.T) {
	// Create a TailpostAgent without resource requirements
	replicas := int32(2)
	batchSize := int32(10)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			Replicas:        &replicas,
			Image:           "tailpost:v1",
			ImagePullPolicy: "IfNotPresent",
			ServiceAccount:  "tailpost-sa",
			ServerURL:       "http://example.com/logs",
			BatchSize:       &batchSize,
			FlushInterval:   "5s",
			LogSources: []v1alpha1.LogSourceSpec{
				{
					Type: "file",
					Path: "/var/log/test.log",
				},
			},
			// No Resources specified
		},
	}

	// Create statefulset
	statefulSet, err := CreateStatefulSet(agent)
	if err != nil {
		t.Fatalf("CreateStatefulSet() error = %v", err)
	}

	// Verify container has empty resource requirements
	container := statefulSet.Spec.Template.Spec.Containers[0]
	if len(container.Resources.Limits) > 0 {
		t.Errorf("Container should have no resource limits, got %v", container.Resources.Limits)
	}

	if len(container.Resources.Requests) > 0 {
		t.Errorf("Container should have no resource requests, got %v", container.Resources.Requests)
	}
}

// TestCreateStatefulSetWithPartialResources tests creating a StatefulSet with only some resource fields
func TestCreateStatefulSetWithPartialResources(t *testing.T) {
	// Create a TailpostAgent with partial resource requirements
	replicas := int32(2)
	batchSize := int32(10)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			Replicas:        &replicas,
			Image:           "tailpost:v1",
			ImagePullPolicy: "IfNotPresent",
			ServerURL:       "http://example.com/logs",
			BatchSize:       &batchSize,
			FlushInterval:   "5s",
			Resources: v1alpha1.ResourceRequirementsSpec{
				Limits: v1alpha1.ResourceListSpec{
					CPU: "100m",
					// No memory limit
				},
				// No requests
			},
		},
	}

	// Create statefulset
	statefulSet, err := CreateStatefulSet(agent)
	if err != nil {
		t.Fatalf("CreateStatefulSet() error = %v", err)
	}

	// Verify container has only CPU limit set
	container := statefulSet.Spec.Template.Spec.Containers[0]
	if cpu := container.Resources.Limits[corev1.ResourceCPU]; cpu.String() != "100m" {
		t.Errorf("Container CPU limit = %v, want %v", cpu.String(), "100m")
	}

	if _, exists := container.Resources.Limits[corev1.ResourceMemory]; exists {
		t.Errorf("Container should not have memory limit set")
	}

	if len(container.Resources.Requests) > 0 {
		t.Errorf("Container should have no resource requests, got %v", container.Resources.Requests)
	}
}

// TestYamlWithComplexStructure tests the yaml function with nested structures
func TestYamlWithComplexStructure(t *testing.T) {
	// Create complex data with nested maps and arrays
	data := map[string]interface{}{
		"string": "value",
		"nested": map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
		"array": []string{"item1", "item2"},
	}

	// Convert to YAML
	yamlStr, err := yaml(data)
	if err != nil {
		t.Fatalf("yaml() error = %v", err)
	}

	// Verify YAML contains the top-level string
	if !contains(yamlStr, "string: value") {
		t.Errorf("YAML missing string value")
	}

	// Verify YAML contains the nested structure (in JSON format)
	if !contains(yamlStr, "nested:") {
		t.Errorf("YAML missing nested structure")
	}

	// Verify YAML contains the array (in JSON format)
	if !contains(yamlStr, "array:") {
		t.Errorf("YAML missing array value")
	}
}

// TestParseQuantityInvalidValue tests parseQuantity with invalid values
func TestParseQuantityInvalidValue(t *testing.T) {
	// This test relies on the fact that resource.MustParse will panic for invalid values
	// So we use a recover to catch the panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("parseQuantity with invalid value should panic")
		}
	}()

	// Try to parse an invalid quantity
	_ = parseQuantity("invalid")
}

// TestServiceNeedsUpdateWithEmptyFields tests ServiceNeedsUpdate with empty fields
func TestServiceNeedsUpdateWithEmptyFields(t *testing.T) {
	// Create Services with empty fields
	current := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{},
			Ports:    []corev1.ServicePort{},
		},
	}

	same := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{},
			Ports:    []corev1.ServicePort{},
		},
	}

	different := &corev1.Service{
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "tailpost"},
			Ports:    []corev1.ServicePort{},
		},
	}

	// Test with same empty services
	if ServiceNeedsUpdate(current, same) {
		t.Errorf("ServiceNeedsUpdate() = true, want false for same empty Services")
	}

	// Test with different services (one empty, one with selector)
	if !ServiceNeedsUpdate(current, different) {
		t.Errorf("ServiceNeedsUpdate() = false, want true for different Services")
	}
}

// TestConfigMapNeedsUpdateWithEmptyFields tests ConfigMapNeedsUpdate with empty fields
func TestConfigMapNeedsUpdateWithEmptyFields(t *testing.T) {
	// Create ConfigMaps with empty fields
	current := &corev1.ConfigMap{
		Data: map[string]string{},
	}

	same := &corev1.ConfigMap{
		Data: map[string]string{},
	}

	different := &corev1.ConfigMap{
		Data: map[string]string{
			"key": "value",
		},
	}

	// Test with same empty ConfigMaps
	if ConfigMapNeedsUpdate(current, same) {
		t.Errorf("ConfigMapNeedsUpdate() = true, want false for same empty ConfigMaps")
	}

	// Test with different ConfigMaps (one empty, one with data)
	if !ConfigMapNeedsUpdate(current, different) {
		t.Errorf("ConfigMapNeedsUpdate() = false, want true for different ConfigMaps")
	}
}

// TestStatefulSetNeedsUpdateWithEmptyFields tests StatefulSetNeedsUpdate with empty fields
func TestStatefulSetNeedsUpdateWithEmptyFields(t *testing.T) {
	// Create StatefulSets with nil replicas and empty containers
	current := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: nil,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image:     "",
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			},
		},
	}

	same := &appsv1.StatefulSet{
		Spec: appsv1.StatefulSetSpec{
			Replicas: nil,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image:     "",
							Resources: corev1.ResourceRequirements{},
						},
					},
				},
			},
		},
	}

	// Test with same empty StatefulSets
	if StatefulSetNeedsUpdate(current, same) {
		t.Errorf("StatefulSetNeedsUpdate() = true, want false for same empty StatefulSets")
	}
}

// TestCreateStatefulSetProbesConfiguration tests that the StatefulSet has properly configured probes
func TestCreateStatefulSetProbesConfiguration(t *testing.T) {
	// Create a minimal TailpostAgent
	replicas := int32(1)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			Replicas: &replicas,
			Image:    "tailpost:v1",
		},
	}

	// Create statefulset
	statefulSet, err := CreateStatefulSet(agent)
	if err != nil {
		t.Fatalf("CreateStatefulSet() error = %v", err)
	}

	// Verify liveness probe
	container := statefulSet.Spec.Template.Spec.Containers[0]
	if container.LivenessProbe == nil {
		t.Fatal("LivenessProbe not configured")
	}

	if container.LivenessProbe.HTTPGet == nil {
		t.Errorf("LivenessProbe HTTPGet not configured")
	} else {
		if container.LivenessProbe.HTTPGet.Path != "/health" {
			t.Errorf("LivenessProbe path = %v, want %v", container.LivenessProbe.HTTPGet.Path, "/health")
		}
		if container.LivenessProbe.HTTPGet.Port.IntValue() != MetricsPort {
			t.Errorf("LivenessProbe port = %v, want %v", container.LivenessProbe.HTTPGet.Port.IntValue(), MetricsPort)
		}
	}

	// Verify readiness probe
	if container.ReadinessProbe == nil {
		t.Fatal("ReadinessProbe not configured")
	}

	if container.ReadinessProbe.HTTPGet == nil {
		t.Errorf("ReadinessProbe HTTPGet not configured")
	} else {
		if container.ReadinessProbe.HTTPGet.Path != "/ready" {
			t.Errorf("ReadinessProbe path = %v, want %v", container.ReadinessProbe.HTTPGet.Path, "/ready")
		}
		if container.ReadinessProbe.HTTPGet.Port.IntValue() != MetricsPort {
			t.Errorf("ReadinessProbe port = %v, want %v", container.ReadinessProbe.HTTPGet.Port.IntValue(), MetricsPort)
		}
	}
}

// TestCreateStatefulSetVolumeMounts tests that the StatefulSet has properly configured volume mounts
func TestCreateStatefulSetVolumeMounts(t *testing.T) {
	// Create a minimal TailpostAgent
	replicas := int32(1)
	agent := &v1alpha1.TailpostAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "default",
		},
		Spec: v1alpha1.TailpostAgentSpec{
			Replicas: &replicas,
			Image:    "tailpost:v1",
		},
	}

	// Create statefulset
	statefulSet, err := CreateStatefulSet(agent)
	if err != nil {
		t.Fatalf("CreateStatefulSet() error = %v", err)
	}

	// Verify volumes
	if len(statefulSet.Spec.Template.Spec.Volumes) < 2 {
		t.Fatalf("Expected at least 2 volumes, got %d", len(statefulSet.Spec.Template.Spec.Volumes))
	}

	// Check for config volume
	configVolumeFound := false
	for _, volume := range statefulSet.Spec.Template.Spec.Volumes {
		if volume.Name == "config" {
			configVolumeFound = true
			if volume.ConfigMap == nil {
				t.Errorf("Expected config volume to have ConfigMap source")
			} else if volume.ConfigMap.Name != GetConfigMapName(agent) {
				t.Errorf("Expected config volume ConfigMap name to be %s, got %s",
					GetConfigMapName(agent), volume.ConfigMap.Name)
			}
			break
		}
	}
	if !configVolumeFound {
		t.Errorf("Config volume not found")
	}

	// Check for log volume
	logVolumeFound := false
	for _, volume := range statefulSet.Spec.Template.Spec.Volumes {
		if volume.Name == "log-volume" {
			logVolumeFound = true
			if volume.HostPath == nil {
				t.Errorf("Expected log volume to have HostPath source")
			} else if volume.HostPath.Path != "/var/log" {
				t.Errorf("Expected log volume HostPath path to be /var/log, got %s", volume.HostPath.Path)
			}
			break
		}
	}
	if !logVolumeFound {
		t.Errorf("Log volume not found")
	}

	// Verify volume mounts
	container := statefulSet.Spec.Template.Spec.Containers[0]
	if len(container.VolumeMounts) < 2 {
		t.Fatalf("Expected at least 2 volume mounts, got %d", len(container.VolumeMounts))
	}

	// Check for config volume mount
	configMountFound := false
	for _, mount := range container.VolumeMounts {
		if mount.Name == "config" {
			configMountFound = true
			if mount.MountPath != "/app/config" {
				t.Errorf("Expected config mount path to be /app/config, got %s", mount.MountPath)
			}
			break
		}
	}
	if !configMountFound {
		t.Errorf("Config volume mount not found")
	}

	// Check for log volume mount
	logMountFound := false
	for _, mount := range container.VolumeMounts {
		if mount.Name == "log-volume" {
			logMountFound = true
			if mount.MountPath != "/host/var/log" {
				t.Errorf("Expected log mount path to be /host/var/log, got %s", mount.MountPath)
			}
			if !mount.ReadOnly {
				t.Errorf("Expected log mount to be read-only")
			}
			break
		}
	}
	if !logMountFound {
		t.Errorf("Log volume mount not found")
	}
}
