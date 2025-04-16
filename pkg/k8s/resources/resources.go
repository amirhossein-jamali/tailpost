package resources

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/amirhossein-jamali/tailpost/pkg/k8s/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// Component is the component name used in labels
	Component = "tailpost-agent"
	// ConfigFileName is the name of the config file in the ConfigMap
	ConfigFileName = "config.yaml"
	// MetricsPort is the port for exposing metrics
	MetricsPort = 8080
)

// GetLabels returns the labels for the TailpostAgent
func GetLabels(cr *v1alpha1.TailpostAgent) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       Component,
		"app.kubernetes.io/instance":   cr.Name,
		"app.kubernetes.io/managed-by": "tailpost-operator",
	}
}

// GetConfigMapName returns the name of the ConfigMap
func GetConfigMapName(cr *v1alpha1.TailpostAgent) string {
	return cr.Name + "-config"
}

// GetStatefulSetName returns the name of the StatefulSet
func GetStatefulSetName(cr *v1alpha1.TailpostAgent) string {
	return cr.Name
}

// GetServiceName returns the name of the Service
func GetServiceName(cr *v1alpha1.TailpostAgent) string {
	return cr.Name
}

// CreateConfigMap creates a ConfigMap for the TailpostAgent
func CreateConfigMap(cr *v1alpha1.TailpostAgent) (*corev1.ConfigMap, error) {
	configData := map[string]interface{}{
		"server_url":     cr.Spec.ServerURL,
		"batch_size":     *cr.Spec.BatchSize,
		"flush_interval": cr.Spec.FlushInterval,
	}

	// Add log source configurations
	// For file type sources, add the log_path
	for _, source := range cr.Spec.LogSources {
		if source.Type == "file" && source.Path != "" {
			configData["log_path"] = source.Path
			break
		}
	}

	// Convert to YAML format
	yamlData, err := yaml(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to convert config to YAML: %w", err)
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetConfigMapName(cr),
			Namespace: cr.Namespace,
			Labels:    GetLabels(cr),
		},
		Data: map[string]string{
			ConfigFileName: yamlData,
		},
	}, nil
}

// CreateStatefulSet creates a StatefulSet for the TailpostAgent
func CreateStatefulSet(cr *v1alpha1.TailpostAgent) (*appsv1.StatefulSet, error) {
	labels := GetLabels(cr)
	configMapName := GetConfigMapName(cr)

	// Configure container ports
	containerPorts := []corev1.ContainerPort{
		{
			Name:          "metrics",
			ContainerPort: MetricsPort,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	// Configure volumes
	volumes := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		},
		{
			Name: "log-volume",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
	}

	// Configure volume mounts
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/app/config",
		},
		{
			Name:      "log-volume",
			MountPath: "/host/var/log",
			ReadOnly:  true,
		},
	}

	// Configure resource requirements
	resourceRequirements := corev1.ResourceRequirements{}
	if cr.Spec.Resources.Limits.CPU != "" || cr.Spec.Resources.Limits.Memory != "" {
		resourceRequirements.Limits = corev1.ResourceList{}
		if cr.Spec.Resources.Limits.CPU != "" {
			resourceRequirements.Limits[corev1.ResourceCPU] = parseQuantity(cr.Spec.Resources.Limits.CPU)
		}
		if cr.Spec.Resources.Limits.Memory != "" {
			resourceRequirements.Limits[corev1.ResourceMemory] = parseQuantity(cr.Spec.Resources.Limits.Memory)
		}
	}
	if cr.Spec.Resources.Requests.CPU != "" || cr.Spec.Resources.Requests.Memory != "" {
		resourceRequirements.Requests = corev1.ResourceList{}
		if cr.Spec.Resources.Requests.CPU != "" {
			resourceRequirements.Requests[corev1.ResourceCPU] = parseQuantity(cr.Spec.Resources.Requests.CPU)
		}
		if cr.Spec.Resources.Requests.Memory != "" {
			resourceRequirements.Requests[corev1.ResourceMemory] = parseQuantity(cr.Spec.Resources.Requests.Memory)
		}
	}

	// Create container
	container := corev1.Container{
		Name:            "tailpost-agent",
		Image:           cr.Spec.Image,
		ImagePullPolicy: corev1.PullPolicy(cr.Spec.ImagePullPolicy),
		Command:         []string{"/app/tailpost"},
		Args:            []string{"-config", "/app/config/" + ConfigFileName},
		Ports:           containerPorts,
		VolumeMounts:    volumeMounts,
		Resources:       resourceRequirements,
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/health",
					Port:   intstr.FromInt(MetricsPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      5,
			PeriodSeconds:       10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path:   "/ready",
					Port:   intstr.FromInt(MetricsPort),
					Scheme: corev1.URISchemeHTTP,
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      5,
			PeriodSeconds:       10,
		},
	}

	// Create StatefulSet
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetStatefulSetName(cr),
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: cr.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: GetServiceName(cr),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: cr.Spec.ServiceAccount,
					Containers:         []corev1.Container{container},
					Volumes:            volumes,
				},
			},
		},
	}

	return statefulSet, nil
}

// CreateService creates a Service for the TailpostAgent
func CreateService(cr *v1alpha1.TailpostAgent) *corev1.Service {
	labels := GetLabels(cr)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetServiceName(cr),
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "metrics",
					Port:       MetricsPort,
					TargetPort: intstr.FromInt(MetricsPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// ConfigMapNeedsUpdate compares two ConfigMaps to see if an update is needed
func ConfigMapNeedsUpdate(current, desired *corev1.ConfigMap) bool {
	return !reflect.DeepEqual(current.Data, desired.Data)
}

// StatefulSetNeedsUpdate compares two StatefulSets to see if an update is needed
func StatefulSetNeedsUpdate(current, desired *appsv1.StatefulSet) bool {
	return !reflect.DeepEqual(current.Spec.Replicas, desired.Spec.Replicas) ||
		!reflect.DeepEqual(current.Spec.Template.Spec.Containers[0].Image, desired.Spec.Template.Spec.Containers[0].Image) ||
		!reflect.DeepEqual(current.Spec.Template.Spec.Containers[0].Resources, desired.Spec.Template.Spec.Containers[0].Resources)
}

// ServiceNeedsUpdate compares two Services to see if an update is needed
func ServiceNeedsUpdate(current, desired *corev1.Service) bool {
	return !reflect.DeepEqual(current.Spec.Selector, desired.Spec.Selector) ||
		!reflect.DeepEqual(current.Spec.Ports, desired.Spec.Ports)
}

// yaml converts a map to a YAML string
func yaml(data map[string]interface{}) (string, error) {
	// Simple YAML formatter
	var builder strings.Builder
	for k, v := range data {
		valueStr := ""
		switch val := v.(type) {
		case string:
			valueStr = val
		case int, int32, int64:
			valueStr = fmt.Sprintf("%d", val)
		case float32, float64:
			valueStr = strconv.FormatFloat(val.(float64), 'f', -1, 64)
		default:
			jsonBytes, err := json.Marshal(val)
			if err != nil {
				return "", err
			}
			valueStr = string(jsonBytes)
		}
		builder.WriteString(fmt.Sprintf("%s: %s\n", k, valueStr))
	}
	return builder.String(), nil
}

// parseQuantity is a helper function to parse resource quantities
func parseQuantity(value string) resource.Quantity {
	return resource.MustParse(value)
}
