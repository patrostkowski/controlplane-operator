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
FROM debian:12

ARG K8S_VERSION=1.33.1
ARG CONTAINERD_VERSION=1.7.23
ARG CRICTL_VERSION=1.32.0

ENV DEBIAN_FRONTEND=noninteractive

# basic deps
RUN apt-get update && apt-get install -y \
    ca-certificates curl gnupg lsb-release iproute2 iptables ebtables ethtool socat conntrack \
    systemd systemd-sysv \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update && apt-get install -y \
    # core runtime deps
    ca-certificates curl wget gnupg lsb-release \
    systemd systemd-sysv systemd-container login \
    # networking + LB tools
    iproute2 iptables nftables ebtables ethtool socat conntrack \
    iputils-ping net-tools dnsutils tcpdump traceroute \
    # process monitoring
    procps htop psmisc sysstat lsof \
    # debugging + scripting
    strace jq file \
    # filesystem + storage
    util-linux mount \
    # shells, editors, CLI utilities
    less vim nano tree \
    coreutils findutils grep sed gawk tar unzip bsdutils \
    # optional but useful
    rsync \
    && rm -rf /var/lib/apt/lists/*

# install containerd
RUN curl -L https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz \
    | tar -C /usr/local -xz

# install runc
RUN curl -L https://github.com/opencontainers/runc/releases/download/v1.2.0/runc.amd64 \
    -o /usr/local/sbin/runc && chmod +x /usr/local/sbin/runc

# install crictl
RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
      amd64) CRICTL_ARCH=amd64 ;; \
      arm64) CRICTL_ARCH=arm64 ;; \
      armhf) CRICTL_ARCH=armv7 ;; \
      *) echo "unsupported arch: $arch" >&2; exit 1 ;; \
    esac; \
    curl -L "https://github.com/kubernetes-sigs/cri-tools/releases/download/v${CRICTL_VERSION}/crictl-v${CRICTL_VERSION}-linux-${CRICTL_ARCH}.tar.gz" \
      | tar -C /usr/local/bin -xz; \
    chmod +x /usr/local/bin/crictl

# crictl config pointing at containerd
RUN cat >/etc/crictl.yaml <<'EOF'
runtime-endpoint: unix:///run/containerd/containerd.sock
image-endpoint: unix:///run/containerd/containerd.sock
timeout: 10
debug: false
pull-image-on-create: false
EOF

# install kubelet + kubectl from Kubernetes apt repo
RUN mkdir -p /etc/apt/keyrings \
 && curl -fsSL https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION%.*}/deb/Release.key \
    | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
 && echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v${K8S_VERSION%.*}/deb/ /" \
    > /etc/apt/sources.list.d/kubernetes.list \
 && apt-get update \
 && apt-get install -y kubelet=${K8S_VERSION}-* kubectl=${K8S_VERSION}-* \
 && apt-mark hold kubelet kubectl \
 && rm -rf /var/lib/apt/lists/*

# containerd + kubelet config dirs
RUN mkdir -p /etc/containerd /var/lib/containerd /var/lib/kubelet

# systemd services
COPY systemd/containerd.service /etc/systemd/system/containerd.service
COPY systemd/kubelet.service /etc/systemd/system/kubelet.service

RUN systemctl enable containerd kubelet

RUN echo "root:root" | chpasswd

RUN cat >/root/.bash_logout <<'EOF'
systemctl poweroff
EOF

# kind-style entrypoint
ENTRYPOINT ["/sbin/init"]
