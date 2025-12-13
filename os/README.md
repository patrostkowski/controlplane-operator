# FCOS / bootc Test Environment

This directory contains scripts to build a Fedora bootc-based node image and run a **local end-to-end test environment** using:

- **Docker** (shared L2 network + DHCP)
- **kind** (Kubernetes control plane)
- **libvirt / KVM** (agent nodes as VMs)
- **bootc-image-builder** (qcow2 image)
- **tesseract** (node join agent)

The goal is to spin up a realistic Kubernetes control plane + VM nodes with minimal manual steps.

---

## Prerequisites

Required on the host:

- Docker
- kind
- kubectl
- podman
- libvirt / virsh
- genisoimage (or mkisofs)
- jq
- task (go-task)
- KVM enabled

Your SSH public key must exist at:

```bash
~/.ssh/id_ed25519.pub
```

---

## 1. Build the node image

From `os/fcos`:

```bash
./build.sh
```

This:
- builds the bootc container image
- generates `output/qcow2/disk.qcow2`
- skips qcow2 rebuilds if inputs didn’t change

---

## 2. Set up shared networking and control plane

```bash
./run.sh setup
```

This will:
- create a shared Docker bridge network (`shared-docker-libvirt`)
- start a DHCP container on that network
- create a kind cluster on the same L2 network
- install required CRDs and components (cert-manager, MetalLB, MCP)

At the end, it prints the Linux bridge name (e.g. `br-xxxx`).

---

## 3. Create VM nodes

```bash
./run.sh create node01
./run.sh create node02
```

Each VM:
- boots from the same qcow2 base image
- gets its own disk copy
- receives a unique hostname via cloud-init
- obtains an IP via DHCP
- runs headless (no VNC)

---

## 4. Inspect environment state

```bash
./run.sh info
```

Shows:
- Docker network + DHCP status
- kind clusters and nodes
- libvirt domains
- VM IP addresses (via qemu-guest-agent)

---

## 5. Join nodes to the control plane

Using the printed IPs from `run.sh info`:

```bash
../../bin/tesseractl join   --node <NODE_IP>   --token <BOOTSTRAP_TOKEN>   --endpoint <K8S_API_IP>
```

Example:

```bash
../../bin/tesseractl join   --node 172.30.0.129   --token abc123.abcdefg123456789   --endpoint 172.30.0.250
```

Repeat for each node.

---

## 6. Clean up

To delete VMs:

```bash
./run.sh delete node01 --purge
./run.sh delete node02 --purge
```

To tear down everything (kind + docker network):

```bash
./run.sh cleanup
```

---

## Notes

- VM IP discovery relies on **qemu-guest-agent**, which is enabled in the image.
- All nodes share a single L2 network with the control plane (no NAT).
- The environment is fully reproducible and idempotent.
- No graphical consoles are used; access is via SSH or serial console.
