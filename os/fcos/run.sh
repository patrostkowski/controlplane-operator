#!/bin/bash
# Copyright 2025 Patryk Rostkowski
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#!/usr/bin/env bash
set -euo pipefail

# Defaults
DEFAULT_DOCKER_NET="shared-docker-libvirt"
DEFAULT_SUBNET="172.30.0.0/24"

DEFAULT_DHCP_CONTAINER="dhcp-shared-net"
DEFAULT_DHCP_IMAGE="andyshinn/dnsmasq"
DEFAULT_DHCP_RANGE="172.30.0.100,172.30.0.150,12h"

DEFAULT_KIND_CLUSTER="kind"          # default kind cluster name
DEFAULT_BRIDGE=""                    # if empty, auto-detect from docker network

DEFAULT_BASE_IMAGE="./output/qcow2/disk.qcow2"
LIBVIRT_IMG_DIR="/var/lib/libvirt/images"
DEFAULT_SSH_KEY="${HOME}/.ssh/id_ed25519.pub"
DEFAULT_MEM_MB="4096"
DEFAULT_VCPUS="2"
DEFAULT_OS_VARIANT="fedora-eln"

usage() {
  cat <<'EOF'
Usage:
  ./run setup [--net NAME] [--subnet CIDR] [--cluster NAME]
  ./run cleanup [--net NAME] [--cluster NAME] [--dhcp NAME]

  ./run create <name> [--bridge BR] [--image PATH] [--ssh-key PATH] [--mem MB] [--vcpus N] [--os-variant VAR]
  ./run delete <name> [--purge]

Examples:
  ./run setup
  ./run info
  ./run create node01 --bridge br-ce6671d13ec5
  ./run delete node01 --purge
  ./run cleanup
EOF
}

die() { echo "ERROR: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"; }

# pick ISO tool
pick_iso_tool() {
  if command -v genisoimage >/dev/null 2>&1; then echo "genisoimage"; return; fi
  if command -v mkisofs >/dev/null 2>&1; then echo "mkisofs"; return; fi
  die "missing genisoimage (or mkisofs)"
}

# Detect linux bridge name for a docker bridge network (e.g. br-<idprefix>)
docker_bridge_name() {
  local net="${1}"
  need docker
  need jq

  local id
  id="$(docker network inspect "${net}" | jq -r '.[0].Id' | cut -c1-12)"
  [[ -n "${id}" && "${id}" != "null" ]] || die "cannot inspect docker network '${net}'"
  # docker usually names it br-<first 12 chars of network id>
  echo "br-${id}"
}

subcmd="${1:-}"
shift || true
[[ -n "${subcmd}" ]] || { usage; exit 1; }

case "${subcmd}" in
  setup)
    net="${DEFAULT_DOCKER_NET}"
    subnet="${DEFAULT_SUBNET}"
    cluster="${DEFAULT_KIND_CLUSTER}"
    dhcp_name="${DEFAULT_DHCP_CONTAINER}"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --net) net="${2:-}"; shift 2 ;;
        --subnet) subnet="${2:-}"; shift 2 ;;
        --cluster) cluster="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option: $1" ;;
      esac
    done

    need docker
    need kind
    need jq

    # 1) docker network
    if docker network inspect "${net}" >/dev/null 2>&1; then
      echo "Docker network '${net}' already exists"
    else
      echo "Creating docker network '${net}' (${subnet})"
      docker network create --driver bridge --subnet "${subnet}" "${net}" >/dev/null
    fi

    # 2) kind cluster (on docker network)
    # kind doesn't have a great "exists" check; `kind get clusters` is easiest
    if kind get clusters | grep -qx "${cluster}"; then
      echo "Kind cluster '${cluster}' already exists"
    else
      echo "Creating kind cluster '${cluster}' on docker network '${net}'"
      KIND_EXPERIMENTAL_DOCKER_NETWORK="${net}" kind create cluster --name "${cluster}"
    fi

    # Assume that cluster was created successfully/already exists
    # Kind creates cluster with prefix "kind"
    echo "Switching to the kind cluster context"
    kubectl config use-context kind-${cluster}

    # 3) DHCP container on that network
    if docker ps -a --format '{{.Names}}' | grep -qx "${dhcp_name}"; then
      # start if stopped
      if ! docker ps --format '{{.Names}}' | grep -qx "${dhcp_name}"; then
        echo "Starting DHCP container '${dhcp_name}'"
        docker start "${dhcp_name}" >/dev/null
      else
        echo "DHCP container '${dhcp_name}' already running"
      fi
    else
      echo "Starting DHCP container '${dhcp_name}' on network '${net}'"
      docker run -d --name "${dhcp_name}" \
        --network "${net}" \
        --cap-add NET_ADMIN \
        --cap-add NET_RAW \
        --restart unless-stopped \
        "${DEFAULT_DHCP_IMAGE}" \
        --dhcp-range="${DEFAULT_DHCP_RANGE}" >/dev/null
    fi
    task dev:install
    kubectl apply -f ../kubeadm/mcp.yaml
    br="$(docker_bridge_name "${net}")"
    echo
    echo "Bridge for docker network '${net}': ${br}"
    echo "Use it like: ./run create node01 --bridge ${br}"
    ;;

  cleanup)
    net="${DEFAULT_DOCKER_NET}"
    cluster="${DEFAULT_KIND_CLUSTER}"
    dhcp_name="${DEFAULT_DHCP_CONTAINER}"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --net) net="${2:-}"; shift 2 ;;
        --cluster) cluster="${2:-}"; shift 2 ;;
        --dhcp) dhcp_name="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option: $1" ;;
      esac
    done

    need docker
    need kind

    # delete kind cluster
    if kind get clusters | grep -qx "${cluster}"; then
      echo "Deleting kind cluster '${cluster}'"
      kind delete cluster --name "${cluster}"
    else
      echo "Kind cluster '${cluster}' not found (skipping)"
    fi

    # remove dhcp container
    if docker ps -a --format '{{.Names}}' | grep -qx "${dhcp_name}"; then
      echo "Removing DHCP container '${dhcp_name}'"
      docker rm -f "${dhcp_name}" >/dev/null || true
    else
      echo "DHCP container '${dhcp_name}' not found (skipping)"
    fi

    # remove docker network
    if docker network inspect "${net}" >/dev/null 2>&1; then
      echo "Removing docker network '${net}'"
      docker network rm "${net}" >/dev/null || true
    else
      echo "Docker network '${net}' not found (skipping)"
    fi

    echo "Done."
    ;;

  create)
    name="${1:-}"; shift || true
    [[ -n "${name}" ]] || die "create requires <name>"

    # If bridge is not given, we will auto-detect from DEFAULT_DOCKER_NET
    bridge="${DEFAULT_BRIDGE}"
    base_image="${DEFAULT_BASE_IMAGE}"
    ssh_key_path="${DEFAULT_SSH_KEY}"
    mem_mb="${DEFAULT_MEM_MB}"
    vcpus="${DEFAULT_VCPUS}"
    os_variant="${DEFAULT_OS_VARIANT}"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --bridge) bridge="${2:-}"; shift 2 ;;
        --image) base_image="${2:-}"; shift 2 ;;
        --ssh-key) ssh_key_path="${2:-}"; shift 2 ;;
        --mem) mem_mb="${2:-}"; shift 2 ;;
        --vcpus) vcpus="${2:-}"; shift 2 ;;
        --os-variant) os_variant="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option: $1" ;;
      esac
    done

    need sudo
    need virt-install
    need virsh

    # auto-detect bridge if not provided
    if [[ -z "${bridge}" ]]; then
      bridge="$(docker_bridge_name "${DEFAULT_DOCKER_NET}")"
    fi

    iso_tool="$(pick_iso_tool)"

    [[ -f "${base_image}" ]] || die "base image not found: ${base_image}"
    [[ -f "${ssh_key_path}" ]] || die "ssh key not found: ${ssh_key_path}"

    ssh_pubkey="$(cat "${ssh_key_path}")"

    vm_disk="${LIBVIRT_IMG_DIR}/${name}.qcow2"
    seed_iso="${LIBVIRT_IMG_DIR}/${name}-seed.iso"

    sudo install -d -m 0755 "${LIBVIRT_IMG_DIR}"

    if sudo virsh dominfo "${name}" >/dev/null 2>&1; then
      die "VM '${name}' already exists. Run: ./run delete ${name}"
    fi

    echo "Copying base image -> ${vm_disk}"
    sudo cp -f "${base_image}" "${vm_disk}"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "${tmpdir}"' EXIT

    cat > "${tmpdir}/meta-data" <<EOF
instance-id: ${name}
local-hostname: ${name}
EOF

    cat > "${tmpdir}/user-data" <<EOF
#cloud-config
preserve_hostname: false
ssh_pwauth: false
users:
  - name: root
    ssh_authorized_keys:
      - ${ssh_pubkey}
    lock_passwd: true
EOF

    echo "Generating cloud-init ISO -> ${seed_iso}"
    sudo "${iso_tool}" -output "${seed_iso}" -volid cidata -joliet -rock \
      "${tmpdir}/user-data" "${tmpdir}/meta-data" >/dev/null

    echo "Creating VM '${name}' (headless, bridge=${bridge})"
    sudo virt-install \
      --name "${name}" \
      --cpu host-model \
      --vcpus "${vcpus}" \
      --memory "${mem_mb}" \
      --import \
      --disk "path=${vm_disk},format=qcow2,bus=virtio" \
      --disk "path=${seed_iso},device=cdrom" \
      --network "bridge=${bridge},model=virtio" \
      --graphics none \
      --console pty,target.type=serial \
      --noautoconsole \
      --os-variant "${os_variant}"

    echo
    echo "Done."
    echo "  VM name:      ${name}"
    echo "  Hostname:     ${name} (via cloud-init, if NoCloud is enabled in image)"
    echo "  Disk:         ${vm_disk}"
    echo "  Cloud-init:   ${seed_iso}"
    echo "Console (optional): sudo virsh console ${name}"
    ;;

  delete)
    name="${1:-}"; shift || true
    [[ -n "${name}" ]] || die "delete requires <name>"

    purge="false"
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --purge) purge="true"; shift ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option: $1" ;;
      esac
    done

    need sudo
    need virsh

    vm_disk="${LIBVIRT_IMG_DIR}/${name}.qcow2"
    seed_iso="${LIBVIRT_IMG_DIR}/${name}-seed.iso"

    if sudo virsh dominfo "${name}" >/dev/null 2>&1; then
      state="$(sudo virsh domstate "${name}" 2>/dev/null || true)"
      if [[ "${state}" == *running* ]]; then
        echo "Stopping VM '${name}'"
        sudo virsh destroy "${name}" >/dev/null || true
      fi

      echo "Undefining VM '${name}'"
      sudo virsh undefine "${name}" --nvram >/dev/null 2>&1 || sudo virsh undefine "${name}" >/dev/null
    else
      echo "VM '${name}' not defined (continuing)"
    fi

    if [[ "${purge}" == "true" ]]; then
      echo "Purging disk + seed ISO"
      sudo rm -f "${vm_disk}" "${seed_iso}"
    else
      echo "Not removing disk/seed (use --purge to delete files):"
      echo "  ${vm_disk}"
      echo "  ${seed_iso}"
    fi

    echo "Done."
    ;;

  -h|--help|"")
    usage
    exit 0
    ;;

    info)
    net="${DEFAULT_DOCKER_NET}"
    cluster="${DEFAULT_KIND_CLUSTER}"
    dhcp_name="${DEFAULT_DHCP_CONTAINER}"

    while [[ $# -gt 0 ]]; do
      case "$1" in
        --net) net="${2:-}"; shift 2 ;;
        --cluster) cluster="${2:-}"; shift 2 ;;
        --dhcp) dhcp_name="${2:-}"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) die "unknown option: $1" ;;
      esac
    done

    echo "=== docker ==="
    if command -v docker >/dev/null 2>&1; then
      docker version --format 'Client: {{.Client.Version}}  Server: {{.Server.Version}}' 2>/dev/null || docker version || true
      echo "- network '${net}':"
      if docker network inspect "${net}" >/dev/null 2>&1; then
        br="$(docker_bridge_name "${net}")"
        echo "  present (bridge: ${br})"
      else
        echo "  missing"
      fi
      echo "- dhcp container '${dhcp_name}':"
      if docker ps --format '{{.Names}}' | grep -qx "${dhcp_name}"; then
        echo "  running"
      elif docker ps -a --format '{{.Names}}' | grep -qx "${dhcp_name}"; then
        echo "  exists (stopped)"
      else
        echo "  missing"
      fi
    else
      echo "docker: not installed"
    fi

    echo
    echo "=== kind ==="
    if command -v kind >/dev/null 2>&1; then
      echo "- clusters:"
      kind get clusters || true
      if kind get clusters 2>/dev/null | grep -qx "${cluster}"; then
        echo
        echo "- nodes in '${cluster}':"
        kind get nodes --name "${cluster}" || true
        echo
        echo "- kubeconfig context(s):"
        kubectl config get-contexts 2>/dev/null | grep -E "(NAME|kind-${cluster})" || true
      else
        echo "(cluster '${cluster}' not found)"
      fi
    else
      echo "kind: not installed"
    fi

    echo
    echo "=== libvirt ==="
    if command -v virsh >/dev/null 2>&1; then
      sudo virsh list --all || true

      echo
      echo "=== domifaddr (running domains) ==="
      # get running domain names
      mapfile -t running < <(sudo virsh list --name 2>/dev/null | sed '/^$/d' || true)
      if [[ "${#running[@]}" -eq 0 ]]; then
        echo "(no running domains)"
      else
        for d in "${running[@]}"; do
          echo
          echo "--- ${d} ---"
          echo "[agent]"
          sudo virsh domifaddr "${d}" --source agent 2>/dev/null || echo "(no agent data)"
          echo "[any]"
          sudo virsh domifaddr "${d}" 2>/dev/null || echo "(no domifaddr data)"
        done
      fi
    else
      echo "virsh: not installed"
    fi
    ;;

  *)
    die "unknown subcommand: ${subcmd}"
    ;;
esac
