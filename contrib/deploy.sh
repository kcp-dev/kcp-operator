#!/usr/bin/env bash

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

# See contrib/README.md.

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

usage() {
  cat <<EOF
kcp and its virtual workspaces on a local kind cluster.

  contrib/deploy.sh                 kcp, then every virtual workspace
  contrib/deploy.sh down            tear everything down
  contrib/deploy.sh kcp             only the kcp layer
  contrib/deploy.sh vw              only the virtual workspaces
  contrib/deploy.sh vw access mcp   only these virtual workspaces

Configuration is environment driven; see contrib/lib.sh and contrib/README.md.
EOF
  exit "${1:-0}"
}

report_hostnames() {
  local unresolved=() host
  for host in "${HOSTNAMES[@]}"; do
    if ! python3 -c "import socket; socket.gethostbyname('$host')" >/dev/null 2>&1; then
      unresolved+=("$host")
    fi
  done

  echo
  if [[ ${#unresolved[@]} -gt 0 ]]; then
    echo "The following hostnames do not resolve on this machine yet:"
    echo
    echo "  ${unresolved[*]}"
    echo
    echo "Add this line to /etc/hosts:"
    echo
    echo "  127.0.0.1 ${HOSTNAMES[*]}"
    echo
  else
    log "Verifying connectivity through the gateway..."
    retry 30 2 kubectl --kubeconfig "$KUBECONFIG_OUT" version >/dev/null 2>&1
    log "kcp answers at https://$BASE_DOMAIN:$PORT."
  fi
}

report_done() {
  echo "kcp is up. Use it with:"
  echo
  echo "  export KUBECONFIG=$KUBECONFIG_OUT"
  echo "  kubectl get workspaces"
  echo
  if [[ -f "$PORT_FORWARD_PID_FILE" ]]; then
    echo "The gateway port-forward runs in the background (PID $(cat "$PORT_FORWARD_PID_FILE"))."
  fi
  echo "Tear everything down with: contrib/deploy.sh down"
}

case "${1:-all}" in
  -h|--help|help)
    usage
    ;;
  down)
    source "$CONTRIB_DIR/kcp/deploy.sh"
    kcp_down
    ;;
  kcp)
    source "$CONTRIB_DIR/kcp/deploy.sh"
    kcp_deploy
    report_hostnames
    report_done
    ;;
  vw|virtualworkspaces)
    shift
    source "$CONTRIB_DIR/virtualworkspaces/deploy.sh"
    vw_deploy_all "$@"
    ;;
  all)
    source "$CONTRIB_DIR/kcp/deploy.sh"
    kcp_deploy
    report_hostnames
    source "$CONTRIB_DIR/virtualworkspaces/deploy.sh"
    vw_deploy_all
    report_done
    ;;
  *)
    echo "unknown command: $1" >&2
    usage 1
    ;;
esac
