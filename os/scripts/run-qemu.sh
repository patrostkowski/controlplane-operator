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

SCRIPT_NAME="run-qemu"

log() {
  local ts
  ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "${ts} [MCPos][${SCRIPT_NAME}] $*"
}

# -----------------------------------------------------------------------------
# Paths & defaults
# -----------------------------------------------------------------------------
ROOT_DIR="${ROOT_DIR:-$(cd "$(dirname "$0")/.." && pwd)}"
BUILD_DIR="${BUILD_DIR:-${ROOT_DIR}/build}"
TARGET_ARCH="${TARGET_ARCH:-arm64}"

MCP_API="${MCP_API:-}"
MCP_CA="${MCP_CA:-}"
MCP_TOKEN="${MCP_TOKEN:-}"

# Allow overrides from env if you want:
#   KERNEL_IMAGE=... INITRAMFS=... ./scripts/run-qemu.sh
KERNEL_IMAGE="${KERNEL_IMAGE:-${BUILD_DIR}/vmlinuz-mcpos}"
INITRAMFS="${INITRAMFS:-${BUILD_DIR}/initramfs-${TARGET_ARCH}.cpio.gz}"

# QEMU binary (override via QEMU_BIN if needed)
QEMU_BIN="${QEMU_BIN:-qemu-system-aarch64}"

log "Using ROOT_DIR=${ROOT_DIR}"
log "Using BUILD_DIR=${BUILD_DIR}"
log "Using TARGET_ARCH=${TARGET_ARCH}"
log "Using KERNEL_IMAGE=${KERNEL_IMAGE}"
log "Using INITRAMFS=${INITRAMFS}"
log "Using QEMU_BIN=${QEMU_BIN}"

# -----------------------------------------------------------------------------
# Sanity checks
# -----------------------------------------------------------------------------
if ! command -v "${QEMU_BIN}" >/dev/null 2>&1; then
  log "ERROR: ${QEMU_BIN} not found in PATH"
  exit 1
fi

if [ ! -f "${KERNEL_IMAGE}" ]; then
  log "ERROR: Kernel image not found at ${KERNEL_IMAGE}"
  exit 1
fi

if [ ! -f "${INITRAMFS}" ]; then
  log "ERROR: Initramfs not found at ${INITRAMFS}"
  exit 1
fi

log "Starting QEMU (Ctrl+a x to exit)"

CMDLINE="console=ttyAMA0 earlyprintk=serial init=/init"

# Only append MCP_* if they’re non-empty
[ -n "${MCP_API}" ]   && CMDLINE="${CMDLINE} MCP_API=${MCP_API}"
[ -n "${MCP_CA}" ]    && CMDLINE="${CMDLINE} MCP_CA=${MCP_CA}"
[ -n "${MCP_TOKEN}" ] && CMDLINE="${CMDLINE} MCP_TOKEN=${MCP_TOKEN}"

# -----------------------------------------------------------------------------
# QEMU run
# -----------------------------------------------------------------------------
exec "${QEMU_BIN}" \
  -machine virt,gic-version=3 \
  -cpu cortex-a57 \
  -m 2048 \
  -smp 2 \
  -kernel "${KERNEL_IMAGE}" \
  -initrd "${INITRAMFS}" \
  -append "${CMDLINE}" \
  -nographic
