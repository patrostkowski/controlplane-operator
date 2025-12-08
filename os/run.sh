#!/bin/bash

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
