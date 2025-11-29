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

ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
ROOTFS_DIR="${ROOTFS_DIR:-${ROOT_DIR}/rootfs}"
BUILD_DIR="${BUILD_DIR:-${ROOT_DIR}/build}"

log() {
  TS="$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "0000-00-00T00:00:00Z")"
  echo " ${TS} [MCPos][fetch-glibc] $*" >&2
}

mkdir -p "${BUILD_DIR}"

# Pick one – I’d go with the bookworm one for stability:
# GLIBC_DEB_URL="http://ftp.debian.org/debian/pool/main/g/glibc/libc6_2.42-2_arm64.deb"
GLIBC_DEB_URL="http://ftp.debian.org/debian/pool/main/g/glibc/libc6_2.36-9+deb12u13_arm64.deb"
GLIBC_DEB="${BUILD_DIR}/libc6-arm64.deb"

# Idempotency: if libc is already there, skip
if [ -f "${ROOTFS_DIR}/lib/aarch64-linux-gnu/libc.so.6" ]; then
  log "glibc already present in rootfs, skipping"
  exit 0
fi

log "Downloading glibc from ${GLIBC_DEB_URL}"
curl -fsSL -o "${GLIBC_DEB}.tmp" "${GLIBC_DEB_URL}"
mv "${GLIBC_DEB}.tmp" "${GLIBC_DEB}"

log "Extracting glibc into rootfs (${ROOTFS_DIR})"
dpkg-deb -x "${GLIBC_DEB}" "${ROOTFS_DIR}"

# Make sure the dynamic loader path that kubelet/containerd expect exists:
# file(1) showed:  /lib/ld-linux-aarch64.so.1
if [ ! -e "${ROOTFS_DIR}/lib/ld-linux-aarch64.so.1" ]; then
  LOADER="$(find "${ROOTFS_DIR}/lib" -maxdepth 1 -name 'ld-linux-aarch64.so.*' | head -n1 || true)"
  if [ -n "${LOADER}" ]; then
    log "Creating ld-linux-aarch64.so.1 symlink -> $(basename "${LOADER}")"
    ( cd "${ROOTFS_DIR}/lib" && ln -sf "$(basename "${LOADER}")" ld-linux-aarch64.so.1 )
  else
    log "WARN: could not find ld-linux-aarch64.so.* in ${ROOTFS_DIR}/lib"
  fi
fi

log "glibc installed into rootfs"