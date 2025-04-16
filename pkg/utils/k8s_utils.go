package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubernetesClient returns a Kubernetes clientset
func GetKubernetesClient() (*kubernetes.Clientset, error) {
	// Try to get in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		// If not in cluster, use kubeconfig
		kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		if envVar := os.Getenv("KUBECONFIG"); envVar != "" {
			kubeconfig = envVar
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

// IsRunningInKubernetes checks if the application is running inside a Kubernetes cluster
// Declare osStatFunc variable that can be replaced in tests
var osStatFunc = os.Stat

func IsRunningInKubernetes() bool {
	// Check for Kubernetes service account token
	_, err := osStatFunc("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return err == nil
}

// GetPodLogs retrieves logs from a pod container
func GetPodLogs(client kubernetes.Interface, namespace, podName, containerName string, tailLines int64) (string, error) {
	// Validate parameters
	if namespace == "" || podName == "" {
		return "", fmt.Errorf("namespace and pod name must be specified")
	}

	// Get pod logs
	podLogOptions := corev1.PodLogOptions{
		Container: containerName,
	}

	if tailLines > 0 {
		podLogOptions.TailLines = &tailLines
	}

	req := client.CoreV1().Pods(namespace).GetLogs(podName, &podLogOptions)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	// Read logs
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ListNamespacedPods returns a list of pods in a namespace with optional label selector
func ListNamespacedPods(client kubernetes.Interface, namespace, labelSelector string) (*corev1.PodList, error) {
	// Validate parameters
	if namespace == "" {
		return nil, fmt.Errorf("namespace must be specified")
	}

	return client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

// GetPodContainers returns the list of container names in a pod
func GetPodContainers(pod *corev1.Pod) []string {
	if pod == nil {
		return []string{}
	}

	containers := make([]string, len(pod.Spec.Containers))
	for i, container := range pod.Spec.Containers {
		containers[i] = container.Name
	}
	return containers
}
