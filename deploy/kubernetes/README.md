# Running Tailpost on Kubernetes

This directory contains the necessary files for deploying Tailpost on Kubernetes. Tailpost includes an operator and an agent that collects logs from containers.

## Prerequisites

- An active Kubernetes cluster
- kubectl installed and configured
- Docker (for building images)

## File Structure

- `Dockerfile.operator`: Docker file for building the operator image
- `operator_deployment.yaml`: Operator deployment file
- `tailpost_crd.yaml`: Custom Resource Definition for TailpostAgent
- `tailpost_example_cr.yaml`: Example Custom Resource for deploying an agent
- `mock-server.yaml`: A mock server for testing
- `tailpost-agent.yaml`: Direct agent deployment file (without using the operator)
- `tailpost_statefulset_example.yaml`: Example deployment as a StatefulSet
- `run-on-kubernetes.ps1`: PowerShell script for automated deployment
- `run-on-kubernetes.sh`: Bash script for automated deployment

## Deployment with Scripts

### Windows

On Windows, you can use the PowerShell script:

```powershell
.\deploy\kubernetes\run-on-kubernetes.ps1
```

### Linux/Mac

On Linux or Mac, you can use the Bash script:

```bash
./deploy/kubernetes/run-on-kubernetes.sh
```

## Manual Deployment

### 1. Build the Operator Image

```bash
docker build -t tailpost-operator:latest -f deploy/kubernetes/Dockerfile.operator .
```

### 2. Install the CRD

```bash
kubectl apply -f deploy/kubernetes/tailpost_crd.yaml
```

### 3. Deploy the Operator

```bash
kubectl apply -f deploy/kubernetes/operator_deployment.yaml
```

### 4. Deploy the Mock Server (Optional, for Testing)

```bash
kubectl apply -f deploy/kubernetes/mock-server.yaml
```

### 5. Create a TailpostAgent

```bash
kubectl apply -f deploy/kubernetes/tailpost_example_cr.yaml
```

## Checking Status

### View Pods

```bash
kubectl get pods
```

### View TailpostAgents

```bash
kubectl get tailpostagents
```

### View Operator Logs

```bash
kubectl logs -l app=tailpost-operator
```

### View Agent Logs

```bash
kubectl logs <tailpost-agent-pod-name>
```

## Removing Resources

```bash
kubectl delete -f deploy/kubernetes/tailpost_example_cr.yaml
kubectl delete -f deploy/kubernetes/mock-server.yaml
kubectl delete -f deploy/kubernetes/operator_deployment.yaml
kubectl delete -f deploy/kubernetes/tailpost_crd.yaml
``` 