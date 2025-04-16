#!/bin/bash
# Script to deploy Tailpost on Kubernetes

echo -e "\e[32mDeploying Tailpost on Kubernetes...\e[0m"

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo -e "\e[31mError: kubectl is not installed or not in PATH\e[0m"
    echo -e "\e[33mPlease install kubectl first: https://kubernetes.io/docs/tasks/tools/\e[0m"
    exit 1
fi

# Check if Kubernetes cluster is running
if ! kubectl cluster-info &> /dev/null; then
    echo -e "\e[31mError: Kubernetes cluster is not running or not accessible\e[0m"
    echo -e "\e[33mPlease start a Kubernetes cluster (e.g., with Docker Desktop, Minikube, or Kind)\e[0m"
    exit 1
else
    echo -e "\e[32mKubernetes cluster is running\e[0m"
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "\e[33mWarning: Docker is not installed or not in PATH\e[0m"
    echo -e "\e[33mYou may need Docker to build images: https://docs.docker.com/get-docker/\e[0m"
fi

# Check if we need to build the operator image
read -p "Do you want to build the operator image? (y/n): " buildImage
if [ "$buildImage" = "y" ]; then
    echo -e "\e[32mBuilding operator image...\e[0m"
    
    # Build the operator image
    docker build -t tailpost-operator:latest -f deploy/kubernetes/Dockerfile.operator .
    
    # Check if we're using kind and need to load the image into the cluster
    read -p "Are you using kind as your Kubernetes cluster? (y/n): " usingKind
    if [ "$usingKind" = "y" ]; then
        kind load docker-image tailpost-operator:latest
    fi
    
    # If using Minikube we need to use the Minikube Docker daemon
    read -p "Are you using Minikube as your Kubernetes cluster? (y/n): " usingMinikube
    if [ "$usingMinikube" = "y" ]; then
        minikube image load tailpost-operator:latest
    fi
fi

# Apply CRD
echo -e "\e[32mInstalling Custom Resource Definition (CRD)...\e[0m"
kubectl apply -f deploy/kubernetes/tailpost_crd.yaml
echo -e "\e[32mCRD installed successfully\e[0m"

# Apply operator
echo -e "\e[32mDeploying Tailpost Operator...\e[0m"
kubectl apply -f deploy/kubernetes/operator_deployment.yaml
echo -e "\e[32mOperator deployed successfully\e[0m"

# Wait for operator to be ready
echo -e "\e[33mWaiting for operator to be ready...\e[0m"
kubectl wait --for=condition=available deployment/tailpost-operator --timeout=60s

# Deploy mock server
read -p "Do you want to deploy the mock server for testing? (y/n): " deployMock
if [ "$deployMock" = "y" ]; then
    echo -e "\e[32mDeploying Mock Server...\e[0m"
    kubectl apply -f deploy/kubernetes/mock-server.yaml
    echo -e "\e[32mMock Server deployed successfully\e[0m"
    
    # Wait for mock server to be ready
    echo -e "\e[33mWaiting for mock server to be ready...\e[0m"
    kubectl wait --for=condition=available deployment/mock-server --timeout=60s
fi

# Deploy TailpostAgent CR example
read -p "Do you want to deploy the example TailpostAgent CR? (y/n): " deployCR
if [ "$deployCR" = "y" ]; then
    echo -e "\e[32mDeploying TailpostAgent example...\e[0m"
    kubectl apply -f deploy/kubernetes/tailpost_example_cr.yaml
    echo -e "\e[32mTailpostAgent example deployed successfully\e[0m"
fi

# Display status
echo -e "\e[32mDeployment completed. Displaying status...\e[0m"
echo -e "\e[36m\nPods:\e[0m"
kubectl get pods

echo -e "\e[36m\nTailpostAgents:\e[0m"
kubectl get tailpostagents

# Display next steps
echo -e "\e[35m\nNext steps:\e[0m"
echo -e "- Check operator logs:   kubectl logs -l app=tailpost-operator"
echo -e "- Check agent logs:      kubectl logs <tailpost-agent-pod-name>"
echo -e "- Delete all resources:  kubectl delete -f deploy/kubernetes/tailpost_example_cr.yaml -f deploy/kubernetes/mock-server.yaml -f deploy/kubernetes/operator_deployment.yaml -f deploy/kubernetes/tailpost_crd.yaml"

echo -e "\n\e[32mDeployment process completed!\e[0m" 