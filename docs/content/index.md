---
hide:
  - navigation
  - toc
---

# controlplane-operator

controlplane-operator provides a simple way to run and manage Kubernetes control planes on top of an existing Kubernetes cluster.

## How it works?

controlplane-operator runs as a Kubernetes operator and watches `ManagedControlPlane` custom resources. For each resource, it provisions and manages a fully functional Kubernetes API server according to the declared specification.

The operator handles the full lifecycle of the control plane, including:

- Bootstrapping the Kubernetes API
- Configuration and reconciliation
- Upgrades and version changes

Unlike traditional approaches, controlplane-operator abstracts the control plane away from the underlying hosts. Only worker nodes are managed as cluster hosts, making deployment and operations simpler and more flexible.

