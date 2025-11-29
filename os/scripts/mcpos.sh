#!/bin/sh
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

# mcpos.sh - tiny init / supervisor for containerd + kubelet on MCPos
set -eu

SCRIPT_NAME="mcpos"

log() {
  # UTC timestamp for consistent logs
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ || echo unknown-time)"
  echo "${ts} [MCPos][${SCRIPT_NAME}] $*" >&2
}

# ---------------------------------------------------------------------------
# Defaults / env overrides
# ---------------------------------------------------------------------------

CONTAINERD_BIN="${CONTAINERD_BIN:-/usr/bin/containerd}"
KUBELET_BIN="${KUBELET_BIN:-/usr/bin/kubelet}"

CONTAINERD_CONFIG="${CONTAINERD_CONFIG:-/etc/containerd/config.toml}"

CONTAINERD_ROOT="${CONTAINERD_ROOT:-/var/lib/containerd}"
CONTAINERD_STATE="${CONTAINERD_STATE:-/run/containerd}"

KUBELET_DIR="${KUBELET_DIR:-/var/lib/kubelet}"
KUBELET_RUN_DIR="${KUBELET_RUN_DIR:-/run/kubelet}"
KUBELET_LOG_DIR="${KUBELET_LOG_DIR:-/var/log}"

# Allow additional flags via env if you like:
#   KUBELET_OPTS=...
CONTAINERD_OPTS="${CONTAINERD_OPTS:-}"
KUBELET_OPTS="${KUBELET_OPTS:-\
  --config=/etc/kubernetes/kubelet-config.yaml \
  --container-runtime=remote \
  --container-runtime-endpoint=unix://${CONTAINERD_STATE}/containerd.sock \
  --root-dir=${KUBELET_DIR} \
  --kubeconfig=/etc/kubernetes/kubelet.conf \
}"

# ---------------------------------------------------------------------------
# Basic filesystem & config setup
# ---------------------------------------------------------------------------

setup_fs() {
  log "Setting up filesystem layout"

  mkdir -p \
    /etc/containerd \
    /etc/cni/net.d \
    /etc/kubernetes \
    "${CONTAINERD_ROOT}" \
    "${KUBELET_DIR}" \
    "${KUBELET_LOG_DIR}" \
    "${CONTAINERD_STATE}" \
    "${KUBELET_RUN_DIR}"

  # Ensure PATH is sane inside initramfs/rootfs
  export PATH="/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:${PATH:-}"

  if [ ! -f "${CONTAINERD_CONFIG}" ]; then
    log "Creating default containerd config at ${CONTAINERD_CONFIG}"
    cat >"${CONTAINERD_CONFIG}" <<EOF
version = 2

root = "${CONTAINERD_ROOT}"
state = "${CONTAINERD_STATE}"

[grpc]
  address = "${CONTAINERD_STATE}/containerd.sock"
  uid = 0
  gid = 0

[debug]
  level = "info"

[plugins]
  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"

  [plugins."io.containerd.metadata.v1.bolt"]
    content_sharing_policy = "shared"

  [plugins."io.containerd.grpc.v1.cri"]
    sandbox_image = "registry.k8s.io/pause:3.9"

    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "overlayfs"
      default_runtime_name = "runc"

      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
        runtime_type = "io.containerd.runc.v2"
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
          SystemdCgroup = false

    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
EOF
  else
    log "Using existing containerd config at ${CONTAINERD_CONFIG}"
  fi
}

# ---------------------------------------------------------------------------
# Start / supervise processes
# ---------------------------------------------------------------------------

start_containerd() {
  if [ ! -x "${CONTAINERD_BIN}" ]; then
    log "ERROR: containerd binary not found or not executable at ${CONTAINERD_BIN}"
    return 1
  fi

  log "Starting containerd"
  "${CONTAINERD_BIN}" \
    --config "${CONTAINERD_CONFIG}" \
    ${CONTAINERD_OPTS} &
  echo $!
}

start_kubelet() {
  if [ ! -x "${KUBELET_BIN}" ]; then
    log "ERROR: kubelet binary not found or not executable at ${KUBELET_BIN}"
    return 1
  fi

  log "Starting kubelet"
  "${KUBELET_BIN}" ${KUBELET_OPTS} &
  echo $!
}

shutdown_children() {
  log "Shutting down containerd and kubelet"
  # Try to kill children nicely
  kill "${CONTAINERD_PID:-}" 2>/dev/null || true
  kill "${KUBELET_PID:-}" 2>/dev/null || true
  # Give them a moment
  sleep 2
  kill -9 "${CONTAINERD_PID:-}" 2>/dev/null || true
  kill -9 "${KUBELET_PID:-}" 2>/dev/null || true
}

# ---------------------------------------------------------------------------
# Main supervisor loop (suitable for PID 1)
# ---------------------------------------------------------------------------

main() {
  log "mcpos init starting"
  setup_fs

  # Handle signals (useful if QEMU sends a poweroff)
  trap 'shutdown_children; log "Exiting by signal"; exit 0' INT TERM

  while :; do
    CONTAINERD_PID="$(start_containerd || echo "")"
    KUBELET_PID="$(start_kubelet || echo "")"

    if [ -z "${CONTAINERD_PID}" ] || [ -z "${KUBELET_PID}" ]; then
      log "One or both services failed to start; retrying in 5 seconds"
      sleep 5
      continue
    fi

    log "containerd PID=${CONTAINERD_PID}, kubelet PID=${KUBELET_PID}"

    # Wait for either to exit; since POSIX sh doesn't have wait -n,
    # we wait for both; if either fails, loop restarts them.
    wait "${CONTAINERD_PID}" || log "containerd exited with status $?"
    wait "${KUBELET_PID}" || log "kubelet exited with status $?"

    log "containerd or kubelet exited; restarting in 5 seconds"
    sleep 5
  done
}

main "$@"
