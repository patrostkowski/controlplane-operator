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


set -xeu

CONTAINER_NAME="${1:-worker-0}"

docker build . -t worker:latest

VOLUME_NAME="${CONTAINER_NAME}-var"

# Create if missing
docker volume inspect "$VOLUME_NAME" >/dev/null 2>&1 || \
    docker volume create "$VOLUME_NAME"

docker run -it --privileged \
  --hostname "${CONTAINER_NAME}" \
  --network kind \
  -v /lib/modules:/lib/modules:ro \
  -v /dev:/dev \
  -v "${PWD}/config/kubelet/config.yaml:/etc/kubernetes/config.yaml:ro" \
  -v "${PWD}/config/kubelet/kubeconfig:/etc/kubernetes/kubeconfig:ro" \
  -v "${PWD}/ca.crt:/etc/kubernetes/pki/ca.crt:ro" \
  -v "${PWD}/config/containerd/config.toml:/etc/containerd/config.toml:ro" \
  -v "${PWD}/config/containerd/cri-base.json:/etc/containerd/cri-base.json:ro" \
  -v "${VOLUME_NAME}:/var" \
  --rm \
  --name "${CONTAINER_NAME}" \
  --add-host kubernetes:172.18.0.3 \
  worker:latest
