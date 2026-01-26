#!/usr/bin/env bash
set -euo pipefail

NAMESPACE=""
CLUSTER_NAME=""
MODE=""
WORKER_NAME=""
VOLUME_NAME=""

IMAGE="kindest/node:v1.34.0"
CNI_VERSION="v1.9.0"

TMPDIR=""
KUBECONFIG_FILE=""
CACHE_DIR="${PWD}/.cache/cni"
DEBUG=0

usage() {
  cat <<EOF
Usage:
  $0 --namespace <namespace> --name <cluster-name> <create|delete> <worker-name>

Examples:
  $0 --namespace mcp --name example-mcp create worker-0
  $0 --namespace mcp --name example-mcp delete worker-0
EOF
}

if [[ -t 2 ]]; then
  COLOR_INFO="\033[1;34m"
  COLOR_WARN="\033[1;33m"
  COLOR_ERR="\033[1;31m"
  COLOR_RESET="\033[0m"
else
  COLOR_INFO=""
  COLOR_WARN=""
  COLOR_ERR=""
  COLOR_RESET=""
fi

log()  { echo -e "${COLOR_INFO}[INFO]${COLOR_RESET} $*" >&2; }
warn() { echo -e "${COLOR_WARN}[WARN]${COLOR_RESET} $*" >&2; }
die()  { echo -e "${COLOR_ERR}[ERROR]${COLOR_RESET} $*" >&2; exit 1; }

require_bin() { command -v "$1" >/dev/null 2>&1 || die "Missing required binary: $1"; }

container_exists() { docker ps -a --format '{{.Names}}' | grep -qx "$1"; }
container_running() { docker ps --format '{{.Names}}' | grep -qx "$1"; }
volume_exists() { docker volume ls --format '{{.Name}}' | grep -qx "$1"; }

cleanup_tmp() {
  if [[ -n "${TMPDIR:-}" && -d "${TMPDIR}" ]]; then
    rm -rf "${TMPDIR}"
  fi
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --namespace) NAMESPACE="${2:-}"; shift 2 ;;
      --name) CLUSTER_NAME="${2:-}"; shift 2 ;;
      --debug) DEBUG=1; shift ;;
      -h|--help) usage; exit 0 ;;
      -*) die "Unknown option: $1" ;;
      *) break ;;
    esac
  done

  [[ $# -eq 2 ]] || { usage; die "Expected: <create|delete> <worker-name>"; }

  MODE="$1"
  WORKER_NAME="$2"
  VOLUME_NAME="${WORKER_NAME}-var"

  [[ -n "$NAMESPACE" ]] || die "--namespace is required"
  [[ -n "$CLUSTER_NAME" ]] || die "--name is required"

  case "$MODE" in
    create|delete) ;;
    *) die "Mode must be 'create' or 'delete' (got: $MODE)" ;;
  esac
}

make_tmp_kubeconfig() {
  TMPDIR="$(mktemp -d)"
  trap cleanup_tmp EXIT
  KUBECONFIG_FILE="${TMPDIR}/kubeconfig"
  log "Generating kubeconfig into ${KUBECONFIG_FILE}"
  go run cmd/tesseractl/main.go mcp kubeconfig -n "${NAMESPACE}" "${CLUSTER_NAME}" > "${KUBECONFIG_FILE}"
}

kube_node_exists() {
  KUBECONFIG="${KUBECONFIG_FILE}" kubectl get node "${WORKER_NAME}" >/dev/null 2>&1
}

kube_node_ready() {
  local cond
  cond="$(KUBECONFIG="${KUBECONFIG_FILE}" kubectl get node "${WORKER_NAME}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || true)"
  [[ "${cond}" == "True" ]]
}

drain_node() {
  log "Draining node ${WORKER_NAME}..."
  KUBECONFIG="${KUBECONFIG_FILE}" kubectl drain "${WORKER_NAME}" \
    --ignore-daemonsets \
    --delete-emptydir-data \
    --force \
    --grace-period=30 \
    --timeout=5m
}

delete_node() {
  log "Deleting node object ${WORKER_NAME}..."
  KUBECONFIG="${KUBECONFIG_FILE}" kubectl delete node "${WORKER_NAME}" --ignore-not-found
}

ensure_volume() {
  if volume_exists "${VOLUME_NAME}"; then
    log "Docker volume already exists: ${VOLUME_NAME} (skipping)"
  else
    log "Creating Docker volume: ${VOLUME_NAME}"
    docker volume create "${VOLUME_NAME}" >/dev/null
  fi
}

ensure_container_running() {
  if container_exists "${WORKER_NAME}"; then
    if container_running "${WORKER_NAME}"; then
      log "Container already running: ${WORKER_NAME} (skipping start)"
    else
      log "Container exists but is stopped: ${WORKER_NAME} (starting)"
      docker start "${WORKER_NAME}" >/dev/null
    fi
    return 0
  fi

  log "Creating & starting worker container: ${WORKER_NAME}"
  docker run -d \
    --name "${WORKER_NAME}" \
    --hostname "${WORKER_NAME}" \
    --privileged \
    --network kind \
    --cgroupns=private \
    -v /lib/modules:/lib/modules:ro \
    -v "${VOLUME_NAME}:/var" \
    --tmpfs /run \
    --tmpfs /tmp \
    --security-opt seccomp=unconfined \
    --security-opt apparmor=unconfined \
    --security-opt label=disable \
    "${IMAGE}" \
    >/dev/null
}

ensure_cni_tarball() {
  local cni_arch="$1"
  local tarball="cni-plugins-linux-${cni_arch}-${CNI_VERSION}.tgz"
  local url="https://github.com/containernetworking/plugins/releases/download/${CNI_VERSION}/${tarball}"
  local path="${CACHE_DIR}/${tarball}"

  mkdir -p "${CACHE_DIR}"

  if [[ -f "${path}" ]]; then
    log "Using cached CNI tarball: ${path}"
    return 0
  fi

  log "Downloading CNI plugins to cache: ${url}"
  curl -fSL "${url}" -o "${path}"
  tar -tzf "${path}" >/dev/null
  log "Cached CNI tarball at ${path}"
}

install_cni_plugins() {
  log "Installing full CNI plugins into ${WORKER_NAME} (CNI ${CNI_VERSION})"

  local arch cni_arch tarball_name tarball_path
  arch="$(docker exec "${WORKER_NAME}" uname -m)"

  case "$arch" in
    x86_64) cni_arch="amd64" ;;
    aarch64|arm64) cni_arch="arm64" ;;
    *) die "Unsupported container architecture: ${arch}" ;;
  esac

  tarball_name="cni-plugins-linux-${cni_arch}-${CNI_VERSION}.tgz"
  tarball_path="${CACHE_DIR}/${tarball_name}"

  ensure_cni_tarball "${cni_arch}"

  docker exec "${WORKER_NAME}" mkdir -p /tmp/cni /opt/cni/bin
  docker exec -i "${WORKER_NAME}" sh -c 'cat > /tmp/cni/cni-plugins.tgz' < "${tarball_path}"
  docker exec "${WORKER_NAME}" tar -xzf /tmp/cni/cni-plugins.tgz -C /opt/cni/bin
  docker exec "${WORKER_NAME}" chmod -R a+x /opt/cni/bin
  docker exec "${WORKER_NAME}" test -x /opt/cni/bin/bridge

  log "CNI install OK; bridge present"
}

generate_join_command() {
  log "Generating kubeadm join command..."
  go run cmd/tesseractl/main.go mcp kubeadm-join -n "${NAMESPACE}" "${CLUSTER_NAME}"
}

join_cluster() {
  local join_cmd="$1"
  log "Executing join command inside container ${WORKER_NAME}:"
  echo "----------------------------------------"
  echo "${join_cmd}"
  echo "----------------------------------------"
  docker exec "${WORKER_NAME}" bash -c "${join_cmd}"
}

do_create() {
  require_bin docker
  require_bin curl
  require_bin go

  if command -v kubectl >/dev/null 2>&1; then
    make_tmp_kubeconfig
    if kube_node_exists && kube_node_ready; then
      log "Node ${WORKER_NAME} already exists and is Ready in cluster (skipping join)"
      ensure_volume
      ensure_container_running
      install_cni_plugins
      log "${WORKER_NAME} already in desired state"
      return 0
    fi
  else
    warn "kubectl not found; cannot check if node is already joined"
  fi

  ensure_volume
  ensure_container_running
  install_cni_plugins

  local join_cmd
  join_cmd="$(generate_join_command)"

  if [[ -n "${KUBECONFIG_FILE:-}" ]] && kube_node_exists; then
    warn "Node ${WORKER_NAME} exists in cluster but is not Ready; attempting join anyway"
  fi

  if ! join_cluster "${join_cmd}"; then
    warn "Join command failed; node may already be joined"
  fi

  log "Create finished for ${WORKER_NAME}"
}

delete_worker_container() {
  log "Deleting worker container: ${WORKER_NAME}"
  if container_exists "${WORKER_NAME}"; then
    docker rm -f "${WORKER_NAME}" >/dev/null
    log "Removed container ${WORKER_NAME}"
  else
    warn "Container ${WORKER_NAME} not found"
  fi
}

delete_volume() {
  log "Deleting Docker volume: ${VOLUME_NAME}"
  if volume_exists "${VOLUME_NAME}"; then
    docker volume rm "${VOLUME_NAME}" >/dev/null
    log "Removed volume ${VOLUME_NAME}"
  else
    warn "Volume ${VOLUME_NAME} not found"
  fi
}

do_delete() {
  require_bin docker
  require_bin go
  require_bin kubectl

  make_tmp_kubeconfig

  if kube_node_exists; then
    drain_node || warn "Drain failed for ${WORKER_NAME}"
    delete_node
  else
    warn "Node ${WORKER_NAME} not found in cluster"
  fi

  delete_worker_container
  delete_volume
  log "Delete finished for ${WORKER_NAME}"
}

main() {
  parse_args "$@"

  if [[ "${DEBUG}" -eq 1 ]]; then
    set -x
    log "Debug mode enabled (set -x)"
  fi

  case "$MODE" in
    create) do_create ;;
    delete) do_delete ;;
  esac
}

main "$@"
