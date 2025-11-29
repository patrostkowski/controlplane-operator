# Contributing Guide

This document provides a short overview of the development workflow for the Controlplane Operator project. It explains how to use the Taskfile and how to run the operator locally for development.  
OS-building tasks (MCPos) are intentionally **not** covered here.

---

## Prerequisites

Before contributing, ensure you have the following installed:

- **Go ≥ 1.25**
- **Docker**
- **kubectl**
- **helm**
- [**Task**](https://taskfile.dev)

Your shell should have `GOPATH`, `GOBIN` and `GO111MODULE=on` set up. For codegen, contributors must have `controller-gen` in place:

```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
```

Clone the repository:

```bash
git clone https://github.com/patrostkowski/controlplane-operator.git
cd controlplane-operator
```

---

## Using the Taskfile

This project uses a `Taskfile.yaml` to simplify common development actions.  
Below are the most useful tasks for contributors.

### Serve the documentation

Runs MkDocs:

```bash
task docs
```

Documentation is available at:  
**http://localhost:8000**

---

### Generate code (clientsets + CRDs)

If you change API types under `pkg/apis/...`, regenerate code with:

```bash
task dev:codegen
```

This runs:

- Kubernetes code-generator
- CRD generation scripts

---

### Install project CRDs into your cluster

Recommended for development using **kind**, **k3d**, or **minikube**.

```bash
task dev:install
```

This will:

- Install **cert-manager** (required for PKI management)
- Apply all CRDs from `config/crd/`

---

## Running the Operator Locally

Run the operator directly on your machine for fastest development feedback:

```bash
task dev:run
```

This:

- Enters `cmd/controlplane-operator/`
- Runs the operator with `go run main.go`
- Reconciles resources in your current Kubernetes context

Make sure CRDs are installed before running.

---

## Working with kind for Development

A kind cluster setup is included for convenience.

### Create a kind cluster

```bash
task kind:create
```

### Delete the kind cluster

```bash
task kind:delete
```

### Build & load the operator image into kind

```bash
task dev:build
task kind:load-image
```

This workflow is helpful for testing in-cluster deployments.

---

## Creating a Test ManagedControlPlane

Apply the example ManagedControlPlane resource:

```bash
kubectl apply -f example/mcp.yaml
```

Check your operator logs to confirm reconciliation is happening.

---

## Developing Controllers

Controller logic resides in:

```
pkg/controller/
```

Each CRD has a dedicated controller:

- `managedcontrolplane_controller.go`
- `managedapiserver_controller.go`
- `managedetcd_controller.go`
- `managedpki_controller.go`
- `managedscheduler_controller.go`

Changes take effect immediately when using:

```bash
task dev:run
```

---

## Where to Go Next

- API definitions: `pkg/apis/controlplane.patrostkowski.dev/v1alpha1/`
- Reconciliation logic: `pkg/controller/`
- Controlplane component logic: `pkg/controlplane/`

Contributions, issues, and pull requests are always welcome!

