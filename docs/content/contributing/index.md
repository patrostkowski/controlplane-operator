# Contributing Guide

Thanks for your interest in contributing to **controlplane-operator** 🎉  
This document describes the local development workflow and tools used when working on the project.

The focus here is on **developing and testing the operator itself**.  

---

## Project Overview

**controlplane-operator** is a Kubernetes operator that manages fully functional Kubernetes control planes as workloads running on top of an existing Kubernetes cluster.

It watches `ManagedControlPlane` custom resources and handles the full lifecycle of a control plane, including:

- Bootstrapping the Kubernetes API
- Configuration and reconciliation
- Version upgrades and changes

The operator abstracts control plane management away from underlying hosts — only worker nodes act as cluster hosts.

---

## Prerequisites

The following tools are required for local development:

- **Docker**
- **Go**
- [**Task**](https://taskfile.dev)

Make sure your Go environment is properly configured (`GOPATH`, `GOBIN`, modules enabled).

---

## Getting Started

For and clone the repository:

```bash
export GITHUB_USERNAME=<your-username>
git clone https://github.com/${GITHUB_USERNAME}/controlplane-operator.git
cd controlplane-operator
```

---

## Development Workflow

The project uses a `Taskfile.yaml` to standardize common development tasks.

### Create a local kind cluster

```bash
task kind:create
```

This creates a local Kubernetes cluster suitable for running the operator during development.

To delete the cluster:

```bash
task kind:delete
```

---

### Install dependencies and CRDs

```bash
task dev:install
```

This will:

- Install required dependencies (e.g. cert-manager, if applicable)
- Apply all CustomResourceDefinitions needed by the operator

---

### Run the operator locally

For the fastest feedback loop, run the operator directly on your machine:

```bash
task dev:run
```

This runs the operator against your current Kubernetes context and immediately reflects code changes.

Make sure CRDs are installed before running.

---

## Testing the Operator

Create a test `ManagedControlPlane` resource:

```bash
kubectl apply -f example/mcp.yaml
```

Then watch the operator logs to confirm reconciliation is happening as expected.

---

## Roadmap & Contributions

The project roadmap is tracked in the main README.  
Contributions in the form of issues, discussions, and pull requests are very welcome.

If you’re unsure where to start, improving documentation or tests is always appreciated.

