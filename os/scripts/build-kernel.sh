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

log() {
  echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") [MCPos][build-kernel] $*"
}

ROOT_DIR="${ROOT_DIR:-/workspace/os}"
BUILD_DIR="${BUILD_DIR:-"$ROOT_DIR/build"}"
KERNEL_VERSION="${KERNEL_VERSION:-6.11.5}"
KERNEL_URL="https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${KERNEL_VERSION}.tar.xz"
OUT_KERNEL="${BUILD_DIR}/vmlinuz-mcpos"

log "Using ROOT_DIR=$ROOT_DIR"
log "Using BUILD_DIR=$BUILD_DIR"

mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

TARBALL="linux-${KERNEL_VERSION}.tar.xz"

if [[ -f "$OUT_KERNEL" ]]; then
  log "Kernel $OUT_KERNEL already exists, skipping build"
  exit 0
fi

if [[ ! -f "$TARBALL" ]]; then
  log "Downloading kernel $KERNEL_VERSION from $KERNEL_URL"
  curl -L "$KERNEL_URL" -o "$TARBALL"
else
  log "Kernel tarball already exists, skipping download"
fi

log "Extracting kernel sources (via /tmp inside container)"

# 1. Extract into a "real" local FS
TMP_EXTRACT_DIR="$(mktemp -d /tmp/linux-src.XXXXXX)"
tar --delay-directory-restore -xJf "$TARBALL" -C "$TMP_EXTRACT_DIR"

# 2. Move into BUILD_DIR atomically
rm -rf "linux-${KERNEL_VERSION}" || true
mv "$TMP_EXTRACT_DIR/linux-${KERNEL_VERSION}" "linux-${KERNEL_VERSION}"
rmdir "$TMP_EXTRACT_DIR"

log "Kernel sources ready in $BUILD_DIR/linux-${KERNEL_VERSION}"

cd "${BUILD_DIR}/linux-${KERNEL_VERSION}"

log "Preparing defconfig for arm64 (qemu virt)"
# On an arm64 builder, ARCH=arm64 is normally implied, but we keep it explicit.
make ARCH=arm64 defconfig

log "Adjusting kernel config for initramfs boot"

# Use kernel's 'scripts/config' helper
./scripts/config --enable BLK_DEV_INITRD
./scripts/config --enable BLK_DEV_RAM
./scripts/config --set-val BLK_DEV_RAM_SIZE 65536

./scripts/config --enable DEVTMPFS
./scripts/config --enable DEVTMPFS_MOUNT

# Optional but nice
./scripts/config --disable MODULE_SIG
./scripts/config --disable SYSTEM_TRUSTED_KEYS
./scripts/config --disable SYSTEM_REVOCATION_KEYS

./scripts/config --enable NET          # CONFIG_NET=y
./scripts/config --enable INET         # CONFIG_INET=y (IPv4)
./scripts/config --enable NETDEVICES   # CONFIG_NETDEVICES=y
./scripts/config --enable NET_CORE     # CONFIG_NET_CORE=y

# Recommended extras for kube/containerd world
./scripts/config --enable UNIX         # Unix domain sockets (often needed)
./scripts/config --enable IPV6         # (optional) IPv6 support

# ⚠️ Optional: here you could drop a tiny config fragment and do merge_config.sh
# if you need specific options. For now we rely on defconfig being enough for
# qemu-system-aarch64 'virt'.

log "Building kernel Image (this may take a while)"
make -j"$(nproc)" ARCH=arm64 Image

# For qemu-system-aarch64 'virt', we can boot arch/arm64/boot/Image directly.
cp "arch/arm64/boot/Image" "${OUT_KERNEL}"

log "Kernel build done → ${OUT_KERNEL}"
