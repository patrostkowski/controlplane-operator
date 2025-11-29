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

###############################################################################
# MCPos fetch-binaries.sh (idempotent)
#
# Downloads:
#   - kubelet, kubectl
#   - containerd (static)
#   - runc
#   - CNI plugins
#   - crictl
#
# Layout under $ROOT_DIR/rootfs:
#   rootfs/usr/local/bin/   -> kubelet, kubectl, crictl
#   rootfs/usr/local/sbin/  -> runc (and containerd/bin is under usr/local/bin)
#   rootfs/opt/cni/bin/     -> CNI plugins
#
# Idempotency:
#   - If a binary already exists and `--version` (or similar) works, we skip.
#
# Config via env:
#   ROOT_DIR           (default: /workspace/os)
#   KUBERNETES_VERSION (default: 1.34.0)
#   CONTAINERD_VERSION (default: 1.7.23)
#   RUNC_VERSION       (default: 1.2.0)
#   CNI_VERSION        (default: 1.5.1)
#   CRICTL_VERSION     (default: =KUBERNETES_VERSION)
#   TARGET_ARCH        (default: auto from uname -m)
###############################################################################

log() {
  echo "$(date -u +"%Y-%m-%dT%H:%M:%SZ") [MCPos][fetch-binaries] $*"
}

###############################################################################
# Config
###############################################################################

ROOT_DIR="${ROOT_DIR:-/workspace/os}"
ROOTFS_DIR="${ROOTFS_DIR:-${ROOT_DIR}/rootfs}"

KUBERNETES_VERSION="${KUBERNETES_VERSION:-1.34.0}"
CONTAINERD_VERSION="${CONTAINERD_VERSION:-1.7.23}"
RUNC_VERSION="${RUNC_VERSION:-1.2.0}"
CNI_VERSION="${CNI_VERSION:-1.5.1}"
CRICTL_VERSION="${CRICTL_VERSION:-${KUBERNETES_VERSION}}"

detect_arch() {
  local uname_arch
  uname_arch="$(uname -m)"

  case "${uname_arch}" in
    x86_64)
      echo "amd64"
      ;;
    aarch64 | arm64)
      echo "arm64"
      ;;
    *)
      log "FATAL: unsupported architecture: ${uname_arch}"
      exit 1
      ;;
  esac
}

TARGET_ARCH="${TARGET_ARCH:-$(detect_arch)}"

log "Using ROOT_DIR=${ROOT_DIR}"
log "Using ROOTFS_DIR=${ROOTFS_DIR}"
log "Using TARGET_ARCH=${TARGET_ARCH}"
log "Versions: kube=${KUBERNETES_VERSION}, containerd=${CONTAINERD_VERSION}, runc=${RUNC_VERSION}, cni=${CNI_VERSION}, crictl=${CRICTL_VERSION}"

BIN_DIR="${ROOTFS_DIR}/usr/local/bin"
SBIN_DIR="${ROOTFS_DIR}/usr/local/sbin"
CNI_BIN_DIR="${ROOTFS_DIR}/opt/cni/bin"

mkdir -p "${BIN_DIR}" "${SBIN_DIR}" "${CNI_BIN_DIR}"

###############################################################################
# Helpers
###############################################################################

# Check if a binary exists, is executable, and returns 0 for the given args.
binary_ok() {
  local bin="$1"
  shift || true

  if [[ ! -x "${bin}" ]]; then
    return 1
  fi

  if "${bin}" "$@" >/dev/null 2>&1; then
    return 0
  fi

  return 1
}

download_file() {
  local url="$1"
  local dest="$2"

  if [[ -f "${dest}" ]]; then
    log "File already exists, not clobbering: ${dest}"
    return 0
  fi

  log "Downloading ${url} -> ${dest}"
  curl -L --fail --retry 5 --retry-delay 3 -o "${dest}" "${url}"
}

download_and_extract_tar() {
  local url="$1"
  local dest_dir="$2"
  local tmp_tar

  mkdir -p "${dest_dir}"
  tmp_tar="$(mktemp)"

  log "Downloading and extracting ${url} into ${dest_dir}"
  curl -L --fail --retry 5 --retry-delay 3 -o "${tmp_tar}" "${url}"
  tar -xzf "${tmp_tar}" -C "${dest_dir}"
  rm -f "${tmp_tar}"
}

###############################################################################
# 1. Kubernetes: kubelet + kubectl
###############################################################################

fetch_kubernetes_binaries() {
  local base="https://dl.k8s.io/release/v${KUBERNETES_VERSION}/bin/linux/${TARGET_ARCH}"

  # kubelet
  local kubelet_dest="${BIN_DIR}/kubelet"
  if binary_ok "${kubelet_dest}" --version; then
    log "kubelet already present and working, skipping"
  else
    local kubelet_url="${base}/kubelet"
    download_file "${kubelet_url}" "${kubelet_dest}"
    chmod +x "${kubelet_dest}"
    log "kubelet installed at ${kubelet_dest}"
  fi

  # kubectl
  local kubectl_dest="${BIN_DIR}/kubectl"
  if binary_ok "${kubectl_dest}" version --client; then
    log "kubectl already present and working, skipping"
  else
    local kubectl_url="${base}/kubectl"
    download_file "${kubectl_url}" "${kubectl_dest}"
    chmod +x "${kubectl_dest}"
    log "kubectl installed at ${kubectl_dest}"
  fi
}

###############################################################################
# 2. containerd (static tarball)
###############################################################################

fetch_containerd() {
  local url="https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-${TARGET_ARCH}.tar.gz"
  local dest_prefix="${ROOTFS_DIR}/usr/local"

  # If containerd already works, skip
  if binary_ok "${dest_prefix}/bin/containerd" --version; then
    log "containerd already present and working, skipping"
    return 0
  fi

  log "Fetching containerd v${CONTAINERD_VERSION} for ${TARGET_ARCH}"
  download_and_extract_tar "${url}" "${dest_prefix}"

  # Ensure main binaries are executable
  if [[ -f "${dest_prefix}/bin/containerd" ]]; then
    chmod +x "${dest_prefix}/bin/containerd"
  fi
  if [[ -f "${dest_prefix}/bin/containerd-shim-runc-v2" ]]; then
    chmod +x "${dest_prefix}/bin/containerd-shim-runc-v2"
  fi

  log "containerd installed in ${dest_prefix}/bin"
}

###############################################################################
# 3. runc
###############################################################################

fetch_runc() {
  local dest="${SBIN_DIR}/runc"

  if binary_ok "${dest}" --version; then
    log "runc already present and working, skipping"
    return 0
  fi

  local url="https://github.com/opencontainers/runc/releases/download/v${RUNC_VERSION}/runc.${TARGET_ARCH}"

  download_file "${url}" "${dest}"
  chmod +x "${dest}"
  log "runc installed at ${dest}"
}

###############################################################################
# 4. CNI plugins
###############################################################################

fetch_cni_plugins() {
  # Heuristic: if "bridge" plugin exists & executable, assume all CNI OK
  if [[ -x "${CNI_BIN_DIR}/bridge" ]]; then
    log "CNI plugins already present (bridge found), skipping"
    return 0
  fi

  local url="https://github.com/containernetworking/plugins/releases/download/v${CNI_VERSION}/cni-plugins-linux-${TARGET_ARCH}-v${CNI_VERSION}.tgz"

  log "Fetching CNI plugins v${CNI_VERSION} for ${TARGET_ARCH}"
  download_and_extract_tar "${url}" "${CNI_BIN_DIR}"

  chmod +x "${CNI_BIN_DIR}"/*
  log "CNI plugins installed in ${CNI_BIN_DIR}"
}

###############################################################################
# 5. crictl
###############################################################################

fetch_crictl() {
  local dest="${BIN_DIR}/crictl"

  if binary_ok "${dest}" --version; then
    log "crictl already present and working, skipping"
    return 0
  fi

  local url="https://github.com/kubernetes-sigs/cri-tools/releases/download/v${CRICTL_VERSION}/crictl-v${CRICTL_VERSION}-linux-${TARGET_ARCH}.tar.gz"

  log "Fetching crictl v${CRICTL_VERSION} for ${TARGET_ARCH}"
  download_and_extract_tar "${url}" "${BIN_DIR}"

  if [[ -f "${dest}" ]]; then
    chmod +x "${dest}"
  fi

  log "crictl installed at ${dest}"
}

fetch_toybox() {
  local dest_dir="${ROOTFS_DIR}/bin"
  mkdir -p "${dest_dir}"

  local toybox_bin="${dest_dir}/toybox"
  local url=""

  case "${TARGET_ARCH}" in
    arm64)
      url="https://landley.net/bin/toybox/0.8.9/toybox-aarch64"
      ;;
    amd64)
      url="https://landley.net/bin/toybox/0.8.9/toybox-x86_64"
      ;;
    *)
      log "WARNING: no prebuilt toybox for TARGET_ARCH=${TARGET_ARCH}, skipping toybox"
      return 0
      ;;
  esac

  if [[ ! -x "${toybox_bin}" ]]; then
    log "Downloading toybox from ${url}"
    curl -L --fail --retry 5 --retry-delay 3 -o "${toybox_bin}" "${url}"
    chmod +x "${toybox_bin}"
  else
    log "toybox already present, skipping download"
  fi

  # Essential symlinks so common commands work
  # You can expand this list over time.
  local app
  for app in acpi arch ascii base32 base64 basename bash blkdiscard blkid blockdev \
    bunzip2 bzcat cal cat chattr chgrp chmod chown chroot chrt chvt cksum \
    clear cmp comm count cp cpio crc32 cut date deallocvt devmem df dirname \
    dmesg dnsdomainname dos2unix du echo egrep eject env expand factor \
    fallocate false fgrep file find flock fmt free freeramdisk fsfreeze \
    fstype fsync ftpget ftpput getconf gpiodetect gpiofind gpioget gpioinfo \
    gpioset grep groups gunzip halt head help hexedit host hostname httpd \
    hwclock i2cdetect i2cdump i2cget i2cset iconv id ifconfig inotifyd \
    insmod install ionice iorenice iotop kill killall killall5 link linux32 \
    ln logger login logname losetup ls lsattr lsmod lspci lsusb makedevs \
    mcookie md5sum microcom mix mkdir mkfifo mknod mkpasswd mkswap mktemp \
    modinfo mount mountpoint mv nbd-client nbd-server nc netcat netstat \
    nice nl nohup nproc nsenter od oneit openvt partprobe passwd paste \
    patch pgrep pidof ping ping6 pivot_root pkill pmap poweroff printenv \
    printf prlimit ps pwd pwdx pwgen readahead readelf readlink realpath \
    reboot renice reset rev rfkill rm rmdir rmmod route rtcwake sed seq \
    setfattr setsid sh sha1sum sha224sum sha256sum sha384sum sha3sum sha512sum \
    shred sleep sntp sort split stat strings su swapoff swapon switch_root \
    sync sysctl tac tail tar taskset tee test time timeout top touch toysh \
    true truncate tty tunctl uclampset ulimit umount uname unicode uniq \
    unix2dos unlink unshare uptime usleep uudecode uuencode uuidgen vconfig \
    vmstat w watch watchdog wc wget which who whoami xargs xxd yes zcat; do
    ln -sf toybox "${dest_dir}/${app}"
  done

  log "Toybox installed at ${toybox_bin} with basic applet symlinks in ${dest_dir}"
}

###############################################################################
# Main
###############################################################################

main() {
  log "Starting fetch-binaries (idempotent)"

  fetch_toybox
  fetch_kubernetes_binaries
  fetch_containerd
  fetch_runc
  fetch_cni_plugins
  fetch_crictl

  log "All binaries fetched and verified in ${ROOTFS_DIR}"
}

main "$@"