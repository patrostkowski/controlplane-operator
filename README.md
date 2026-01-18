# controlplane-operator

controlplane-operator provides a simple way to run and manage Kubernetes control planes on top of an existing Kubernetes cluster.

## How it works?

controlplane-operator runs as a Kubernetes operator and watches `ManagedControlPlane` custom resources.
For each resource, it provisions and manages a fully functional Kubernetes API server according to the declared specification.

The operator handles the full lifecycle of the control plane, including:
- Bootstrapping the Kubernetes API
- Configuration and reconciliation
- Upgrades and version changes

Unlike traditional approaches, controlplane-operator abstracts the control plane away from the underlying hosts.
Only worker nodes are managed as cluster hosts, making deployment and operations simpler and more flexible.

## Getting started

```bash
# crds
kubectl apply -f https://raw.githubusercontent.com/patrostkowski/controlplane-operator/main/config/crd/controlplane.patrostkowski.dev_managedcontrolplanes.yaml

# deploy
kubectl apply -f https://raw.githubusercontent.com/patrostkowski/controlplane-operator/main/config/deploy/manifests.yaml
```

## Development

### Prerequisites

The following tools are required for local development:

- Docker
- kind
- Taskfile

### Local workflow

```bash
# create Kind cluster
task kind:create

# install dependencies
task dev:install

# run locally for quick feedback loop
task dev:run
```

## Roadmap

- [x] Deploy control plane as Kubernetes workloads
- [ ] kubeadm-compatible control plane
- [ ] Highly available control plane
- [ ] Pluggable etcd-compatible backend
- [ ] Control plane upgrade orchestration
- [ ] Metrics, health checks, and observability
- [ ] Helm chart
- [ ] Documentation and usage examples
- [ ] CI/CD automation

## Contributing 

See [CONTRIBUTING](CONTRIBUTING) for details on submitting patches and contribution workflow.

## Licensing 

controlplane-operator is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
