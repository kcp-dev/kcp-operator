# shellcheck shell=bash
#
# Copyright 2026 The kcp Authors.
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

[[ -n "${KCP_CONTRIB_LIB_SOURCED:-}" ]] && return 0
KCP_CONTRIB_LIB_SOURCED=1

CONTRIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$CONTRIB_DIR/.." && pwd)"
export CONTRIB_DIR REPO_ROOT

export CLUSTER_NAME="${CLUSTER_NAME:-kcp-gateway}"
export BASE_DOMAIN="${BASE_DOMAIN:-kcp.localhost}"
export PORT="${PORT:-6443}"
export GATEWAY_IP="${GATEWAY_IP:-10.96.2.2}"
# One of each. The operator defaults to 2, which needs more CPU than a 4-vCPU
# GitHub runner has: the shards stay Pending on "Insufficient cpu" and the wait
# times out after 15m.
export KCP_REPLICAS="${KCP_REPLICAS:-1}"
# The operator requests 1 full CPU per shard and per virtual workspace, which on
# a 4-vCPU runner does not fit. Requests only, no limits, so components still
# burst as needed.
export KCP_CPU_REQUEST="${KCP_CPU_REQUEST:-100m}"
export KCP_MEMORY_REQUEST="${KCP_MEMORY_REQUEST:-256Mi}"
export KCP_IMAGE_REPOSITORY="${KCP_IMAGE_REPOSITORY:-ghcr.io/kcp-dev/kcp}"
export KCP_IMAGE_TAG="${KCP_IMAGE_TAG:-04fcc9232}"
export KUBECONFIG_OUT="${KUBECONFIG_OUT:-$REPO_ROOT/kcp-admin.kubeconfig}"

export CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.19.2}"
export ENVOY_GATEWAY_VERSION="${ENVOY_GATEWAY_VERSION:-v1.7.0}"

export VIRTUAL_WORKSPACES="${VIRTUAL_WORKSPACES:-access ephemeral-resources mcp}"

# shellcheck disable=SC2034
HOSTNAMES=("$BASE_DOMAIN" "root.$BASE_DOMAIN" "alpha.$BASE_DOMAIN")
export PORT_FORWARD_PID_FILE="${PORT_FORWARD_PID_FILE:-$REPO_ROOT/.gateway-api-port-forward.pid}"

export CLUSTER_KUBECONFIG="${CLUSTER_KUBECONFIG:-$REPO_ROOT/.gateway-api-cluster.kubeconfig}"
export KUBECONFIG="$CLUSTER_KUBECONFIG"

log() {
  echo "[$(date +%H:%M:%S)] $*"
}

die() {
  echo "error: $*" >&2
  exit 1
}

require_tools() {
  local tool
  for tool in "$@"; do
    command -v "$tool" >/dev/null || die "$tool is required but not installed."
  done
}

retry() {
  local attempts="$1" delay="$2" i
  shift 2
  for (( i = 1; i <= attempts; i++ )); do
    if "$@"; then
      return 0
    fi
    log "attempt $i/$attempts failed, retrying in ${delay}s..."
    sleep "$delay"
  done
  return 1
}

# retry_quiet: like retry, but output is shown only if every attempt fails.
retry_quiet() {
  local attempts="$1" delay="$2" i out
  shift 2
  out="$(mktemp)"

  for (( i = 1; i <= attempts; i++ )); do
    if "$@" >"$out" 2>&1; then
      rm -f "$out"
      return 0
    fi
    [[ $i -lt $attempts ]] && sleep "$delay"
  done

  echo "failed after $attempts attempts: $*" >&2
  sed 's/^/    /' "$out" >&2
  rm -f "$out"
  return 1
}

# render: expand ${VAR} against the environment. envsubst, not a heredoc.
render() {
  envsubst < "$1"
}

apply_template() {
  local tmp
  tmp="$(mktemp)"
  render "$1" > "$tmp"
  retry 30 5 kubectl apply -f "$tmp" >/dev/null
  rm -f "$tmp"
}

kcpctl() {
  local workspace="$1"
  shift
  kubectl --kubeconfig "$KUBECONFIG_OUT" \
    --server "https://$BASE_DOMAIN:$PORT/clusters/$workspace" "$@"
}

# ensure_workspace: create a workspace on the root shard, wait for Ready.
# Pinned to one shard; shared because more than one binds into root:consumer.
ensure_workspace() {
  local parent="$1"
  export WORKSPACE_NAME="$2"

  local manifest
  manifest="$(mktemp)"
  render "$CONTRIB_DIR/templates/workspace.yaml" > "$manifest"
  retry_quiet 30 5 kcpctl "$parent" apply -f "$manifest"
  rm -f "$manifest"

  retry 60 5 workspace_ready "$parent" "$WORKSPACE_NAME"
  unset WORKSPACE_NAME
}

workspace_ready() {
  [[ "$(kcpctl "$1" get workspace "$2" -o jsonpath='{.status.phase}' 2>/dev/null)" == "Ready" ]]
}

host_resolves() {
  python3 -c "import socket; socket.gethostbyname('$BASE_DOMAIN')" >/dev/null 2>&1
}

stop_port_forward() {
  if [[ -f "$PORT_FORWARD_PID_FILE" ]]; then
    kill "$(cat "$PORT_FORWARD_PID_FILE")" 2>/dev/null || true
    rm -f "$PORT_FORWARD_PID_FILE"
  fi
}

assert_port_free() {
  command -v lsof >/dev/null 2>&1 || return 0

  local ours="" holders=()
  [[ -f "$PORT_FORWARD_PID_FILE" ]] && ours="$(cat "$PORT_FORWARD_PID_FILE")"

  local pid
  while read -r pid; do
    [[ -z "$pid" || "$pid" == "$ours" ]] && continue
    holders+=("$pid")
  done < <(lsof -nP -iTCP:"$PORT" -sTCP:LISTEN -t 2>/dev/null | sort -u || true)

  [[ ${#holders[@]} -eq 0 ]] && return 0

  local detail="" addr
  for pid in "${holders[@]}"; do
    addr="$(lsof -nP -iTCP:"$PORT" -sTCP:LISTEN -a -p "$pid" 2>/dev/null | awk 'NR>1 {printf "%s ", $9}')"
    detail+="      $pid  $(ps -o comm= -p "$pid" 2>/dev/null || echo unknown)  ${addr:-?}"$'\n'
  done

  die "port $PORT is already in use, so $BASE_DOMAIN:$PORT would reach another
  process rather than this cluster:

$detail
  Stop them and re-run:
      kill ${holders[*]}"
}
