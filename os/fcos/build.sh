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
[[ "${TRACE:-0}" == "1" ]] && set -x

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

IMAGE="${IMAGE:-localhost/my-fedora-kube-bootc:latest}"
BUILDER_IMAGE="${BUILDER_IMAGE:-quay.io/centos-bootc/bootc-image-builder:latest}"
OUT_QCOW2="${OUT_QCOW2:-${SCRIPT_DIR}/output/qcow2/disk.qcow2}"
STAMP="${SCRIPT_DIR}/output/.build.sha256"

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing $1" >&2; exit 1; }; }
need sha256sum
need podman
need task

mkdir -p "${SCRIPT_DIR}/bin" "${SCRIPT_DIR}/output"

# 1) build tesseractd (always, or you can also cache this separately)
task dev:build-tesseractd
cp "${SCRIPT_DIR}/../../bin/tesseractd" "${SCRIPT_DIR}/bin/"

# 2) compute fingerprint of inputs that define the image
#    - include anything you COPY into the image or feed into bootc-image-builder
fingerprint="$(
  {
    echo "IMAGE=${IMAGE}"
    echo "BUILDER_IMAGE=${BUILDER_IMAGE}"
    sha256sum Containerfile config.toml 2>/dev/null || true
    # hash all config files (stable order)
    find config -type f -print0 2>/dev/null | sort -z | xargs -0 sha256sum 2>/dev/null || true
    sha256sum bin/tesseractd 2>/dev/null || true
  } | sha256sum | awk '{print $1}'
)"

old=""
[[ -f "${STAMP}" ]] && old="$(cat "${STAMP}" || true)"

if [[ -n "${old}" && "${old}" == "${fingerprint}" && -f "${OUT_QCOW2}" ]]; then
  echo "No changes in inputs. Skipping qcow2 rebuild."
  echo "Existing: ${OUT_QCOW2}"
  exit 0
fi

echo "Inputs changed (or missing output). Building qcow2..."
echo "${fingerprint}" > "${STAMP}.tmp"

podman --connection podman-machine-default-root build --pull=newer -t "${IMAGE}" .

sudo chmod -R a+rwx "${SCRIPT_DIR}/output"

podman --connection podman-machine-default-root run \
  --rm -it \
  --privileged \
  --pull=newer \
  --security-opt label=type:unconfined_t \
  -v "${SCRIPT_DIR}/config.toml:/config.toml:ro" \
  -v "${SCRIPT_DIR}/output:/output" \
  -v /var/lib/containers/storage:/var/lib/containers/storage \
  "${BUILDER_IMAGE}" \
  --type qcow2 --rootfs ext4 --progress verbose -v \
  --use-librepo=True \
  "${IMAGE}"

# only commit stamp if build succeeded and output exists
[[ -f "${OUT_QCOW2}" ]] || { echo "qcow2 not found at ${OUT_QCOW2}" >&2; exit 1; }
mv -f "${STAMP}.tmp" "${STAMP}"
echo "Built: ${OUT_QCOW2}"
echo "Stamp: ${STAMP}"
