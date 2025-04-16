# Script to deploy Tailpost on Kubernetes
Write-Host "Deploying Tailpost on Kubernetes..." -ForegroundColor Green

# Check if kubectl is installed
try {
    kubectl version --client
} catch {
    Write-Host "Error: kubectl is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install kubectl first: https://kubernetes.io/docs/tasks/tools/install-kubectl-windows/" -ForegroundColor Yellow
    exit 1
}

# Check if Kubernetes cluster is running
try {
    $result = kubectl cluster-info
    Write-Host "Kubernetes cluster is running" -ForegroundColor Green
} catch {
    Write-Host "Error: Kubernetes cluster is not running or not accessible" -ForegroundColor Red
    Write-Host "Please start a Kubernetes cluster (e.g., with Docker Desktop, Minikube, or Kind)" -ForegroundColor Yellow
    exit 1
}

# Check if Docker is installed
try {
    docker --version
} catch {
    Write-Host "Warning: Docker is not installed or not in PATH" -ForegroundColor Yellow
    Write-Host "You may need Docker to build images: https://docs.docker.com/desktop/install/windows-install/" -ForegroundColor Yellow
}

# Check if we need to build the operator image
$buildImage = Read-Host "Do you want to build the operator image? (y/n)"
if ($buildImage -eq "y") {
    Write-Host "Building operator image..." -ForegroundColor Green
    
    # Build the operator image
    docker build -t tailpost-operator:latest -f deploy/kubernetes/Dockerfile.operator .
    
    # Check if we're using kind and need to load the image into the cluster
    $usingKind = Read-Host "Are you using kind as your Kubernetes cluster? (y/n)"
    if ($usingKind -eq "y") {
        kind load docker-image tailpost-operator:latest
    }
    
    # If using Minikube we need to use the Minikube Docker daemon
    $usingMinikube = Read-Host "Are you using Minikube as your Kubernetes cluster? (y/n)"
    if ($usingMinikube -eq "y") {
        minikube image load tailpost-operator:latest
    }
}

# Apply CRD
Write-Host "Installing Custom Resource Definition (CRD)..." -ForegroundColor Green
kubectl apply -f deploy/kubernetes/tailpost_crd.yaml
Write-Host "CRD installed successfully" -ForegroundColor Green

# Apply operator
Write-Host "Deploying Tailpost Operator..." -ForegroundColor Green
kubectl apply -f deploy/kubernetes/operator_deployment.yaml
Write-Host "Operator deployed successfully" -ForegroundColor Green

# Wait for operator to be ready
Write-Host "Waiting for operator to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=available deployment/tailpost-operator --timeout=60s

# Deploy mock server
$deployMock = Read-Host "Do you want to deploy the mock server for testing? (y/n)"
if ($deployMock -eq "y") {
    Write-Host "Deploying Mock Server..." -ForegroundColor Green
    kubectl apply -f deploy/kubernetes/mock-server.yaml
    Write-Host "Mock Server deployed successfully" -ForegroundColor Green
    
    # Wait for mock server to be ready
    Write-Host "Waiting for mock server to be ready..." -ForegroundColor Yellow
    kubectl wait --for=condition=available deployment/mock-server --timeout=60s
}

# Deploy TailpostAgent CR example
$deployCR = Read-Host "Do you want to deploy the example TailpostAgent CR? (y/n)"
if ($deployCR -eq "y") {
    Write-Host "Deploying TailpostAgent example..." -ForegroundColor Green
    kubectl apply -f deploy/kubernetes/tailpost_example_cr.yaml
    Write-Host "TailpostAgent example deployed successfully" -ForegroundColor Green
}

# Display status
Write-Host "Deployment completed. Displaying status..." -ForegroundColor Green
Write-Host "`nPods:" -ForegroundColor Cyan
kubectl get pods

Write-Host "`nTailpostAgents:" -ForegroundColor Cyan
kubectl get tailpostagents

# Display next steps
Write-Host "`nNext steps:" -ForegroundColor Magenta
Write-Host "- Check operator logs:   kubectl logs -l app=tailpost-operator" -ForegroundColor White
Write-Host "- Check agent logs:      kubectl logs <tailpost-agent-pod-name>" -ForegroundColor White
Write-Host "- Delete all resources:  kubectl delete -f deploy/kubernetes/tailpost_example_cr.yaml -f deploy/kubernetes/mock-server.yaml -f deploy/kubernetes/operator_deployment.yaml -f deploy/kubernetes/tailpost_crd.yaml" -ForegroundColor White

Write-Host "`nDeployment process completed!" -ForegroundColor Green