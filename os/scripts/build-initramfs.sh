#!/usr/bin/env bash
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

set -euo pipefail

SCRIPT_NAME="build-initramfs"

log() {
  local ts
  ts="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo "${ts} [MCPos][${SCRIPT_NAME}] $*" >&2
}

# Root of your os/ repo (inside the builder container this should be /workspace/os)
ROOT_DIR="${ROOT_DIR:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)}"
ROOTFS_DIR="${ROOTFS_DIR:-"${ROOT_DIR}/rootfs"}"
BUILD_DIR="${BUILD_DIR:-"${ROOT_DIR}/build"}"

ARCH="${TARGET_ARCH:-arm64}"
INITRAMFS_NAME="initramfs-${ARCH}.cpio.gz"
INITRAMFS_PATH="${BUILD_DIR}/${INITRAMFS_NAME}"

log "Using ROOT_DIR=${ROOT_DIR}"
log "Using ROOTFS_DIR=${ROOTFS_DIR}"
log "Using BUILD_DIR=${BUILD_DIR}"
log "Target ARCH=${ARCH}"
log "Will output initramfs: ${INITRAMFS_PATH}"

# Ensure directories exist
mkdir -p "${ROOTFS_DIR}" "${BUILD_DIR}"

if [[ -f "$INITRAMFS_PATH" ]]; then
  log "$INITRAMFS_PATH already exists, skipping build"
  exit 0
fi

# -----------------------------------------------------------------------------
# Copy /init from os/scripts into rootfs
# -----------------------------------------------------------------------------
SCRIPT_INIT="${ROOT_DIR}/scripts/init"
ROOTFS_INIT="${ROOTFS_DIR}/init"
SCRIPT_MCPOS="${ROOT_DIR}/scripts/mcpos.sh"
ROOTFS_MCPOS="${ROOTFS_DIR}/mcpos.sh"

if [[ ! -f "${SCRIPT_INIT}" ]]; then
  log "ERROR: ${SCRIPT_INIT} not found – expected init script at os/scripts/init"
  exit 1
fi

log "Copying ${SCRIPT_INIT} -> ${ROOTFS_INIT}"
cp "${SCRIPT_INIT}" "${ROOTFS_INIT}"
chmod +x "${ROOTFS_INIT}"

log "Copying ${SCRIPT_MCPOS} -> ${ROOTFS_MCPOS}"
cp "${SCRIPT_MCPOS}" "${ROOTFS_MCPOS}"
chmod +x "${ROOTFS_MCPOS}"

CONTAINERD_CONFIG_MCPOS="${ROOT_DIR}/config/containerd"
KUBELET_CONFIG_MCPOS="${ROOT_DIR}/config/kubelet"
ROOTFS_CONFIG_MCPOS="${ROOTFS_DIR}/etc"

log "Copying configs to /etc"
cp -r "${KUBELET_CONFIG_MCPOS}" "${ROOTFS_CONFIG_MCPOS}"
cp -r "${CONTAINERD_CONFIG_MCPOS}" "${ROOTFS_CONFIG_MCPOS}"

# Optional warning if you still care about /sbin/init:
if [[ ! -x "${ROOTFS_DIR}/sbin/init" ]]; then
  log "Note: using /init from rootfs; /sbin/init is not present (this is fine)."
fi

log "Building initramfs from ${ROOTFS_DIR}"

tmpfile="$(mktemp "${BUILD_DIR}/initramfs.XXXXXX")"
trap 'rm -f "${tmpfile}"' EXIT

(
  cd "${ROOTFS_DIR}"
  find . -print0 \
    | cpio --null -ov --format=newc 2>/dev/null \
    | gzip -9 > "${tmpfile}"
)

mv "${tmpfile}" "${INITRAMFS_PATH}"

log "Initramfs built at ${INITRAMFS_PATH}"
log "Done."