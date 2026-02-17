# Getting started

This guide walks you through installing **controlplane-operator** on an existing Kubernetes cluster.

The installation assumes you already have a working Kubernetes cluster where the control planes will be hosted.

---

## Prerequisites

Before installing `controlplane-operator`, make sure your cluster meets the following requirements:

### Kubernetes cluster

- Kubernetes **v1.34+** (recommended)
- Access to the cluster as a **cluster-admin**

### LoadBalancer support
The operator provisions Kubernetes API servers that must be reachable via a `LoadBalancer` Service.

You need **one of the following**:

- A cloud provider with native `LoadBalancer` support, **or**
- A LoadBalancer implementation such as:
  - MetalLB (bare metal)
  - Kube-VIP

### cert-manager
`controlplane-operator` relies on `cert-manager` to issue and manage TLS certificates for control plane components.

If `cert-manager` is not already installed, install it first.

---

## Install cert-manager

Install cert-manager using the official Helm chart:

```bash
helm install \
  cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true
```

---

## Install controlplane-operator

### Install Custom Resource Definitions (CRDs)

```bash
kubectl apply -f https://raw.githubusercontent.com/patrostkowski/controlplane-operator/main/config/crd/controlplane.patrostkowski.dev_managedcontrolplanes.yaml
```

### Deploy the operator

```bash
kubectl apply -f https://raw.githubusercontent.com/patrostkowski/controlplane-operator/main/config/deploy/manifests.yaml
```

Verify the operator is running:

```bash
kubectl get pods -n controlplane-system
```

