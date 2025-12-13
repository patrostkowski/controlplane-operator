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

set -o errexit
set -o nounset
set -o pipefail

# TODO: make scripts runnable via Docker, without installing deps locally

# Repo + module info
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
MODULE="$(cd "${REPO_ROOT}" && go list -m)"

APIS_DIR="${REPO_ROOT}/pkg/apis"                # <-- LOCAL PATH (important!)
OUT_DIR="${REPO_ROOT}/pkg/client"
OUT_PKG="${MODULE}/pkg/client"

# Use the code-generator pinned in go.mod
CODEGEN_PKG="$(go list -m -f '{{.Dir}}' k8s.io/code-generator)"
BOILERPLATE="${REPO_ROOT}/hack/boilerplate.go.txt"

# Verbosity (export to let kube_codegen pick it up)
export KUBE_VERBOSE="${KUBE_VERBOSE:-3}"

# Kubernetes

# Source the helpers and call functions directly
source "${CODEGEN_PKG}/kube_codegen.sh"

# 1) Deepcopy (looks for // +k8s:deepcopy-gen=package under APIS_DIR)
kube::codegen::gen_helpers --boilerplate "${BOILERPLATE}" "${APIS_DIR}"

# 2) Clients + listers + informers (looks for // +genclient under APIS_DIR)
kube::codegen::gen_client \
  --output-dir "${OUT_DIR}" \
  --output-pkg "${OUT_PKG}" \
  --clientset-name clientset \
  --versioned-name versioned \
  --with-watch \
  --boilerplate "${BOILERPLATE}" \
  "${APIS_DIR}"

${GOPATH}/bin/controller-gen rbac:roleName=manager-role paths=./... output:rbac:dir=./config/rbac

# gRPC

PROTO_ROOT="${REPO_ROOT}/proto"
GRPC_OUT="${PROTO_ROOT}"

echo "Generating gRPC code from protos under ${PROTO_ROOT}"

protoc \
  -I "${PROTO_ROOT}" \
  --go_out="${GRPC_OUT}" --go_opt=paths=source_relative \
  --go-grpc_out="${GRPC_OUT}" --go-grpc_opt=paths=source_relative \
  $(find "${PROTO_ROOT}" -type f -name '*.proto' | sort)

echo "code-generation done"
