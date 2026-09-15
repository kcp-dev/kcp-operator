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

set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/lib.sh"

MCP_DIR="$CONTRIB_DIR/virtualworkspaces/mcp"

# Where the server may impersonate. root is required; the rest are what the
# resource tools reach into.
export MCP_IMPERSONATION_WORKSPACES="${MCP_IMPERSONATION_WORKSPACES:-root root:access root:consumer}"

vw_mcp_deploy() {
  log "Deploying the MCP virtual workspace..."

  if host_resolves; then
    log "Granting impersonation in the workspaces the server reaches..."
    local ws
    for ws in $MCP_IMPERSONATION_WORKSPACES; do
      retry_quiet 30 5 kcpctl "$ws" apply -f "$MCP_DIR/templates/impersonator-rbac.yaml"
    done
  else
    echo "NOTE: $BASE_DOMAIN does not resolve here, so the impersonation grants could not"
    echo "be applied. Apply them once the hostname resolves, or MCP tools return errors:"
    echo "  for ws in $MCP_IMPERSONATION_WORKSPACES; do"
    echo "    kubectl --kubeconfig $KUBECONFIG_OUT --server https://$BASE_DOMAIN:$PORT/clusters/\$ws apply -f $MCP_DIR/templates/impersonator-rbac.yaml"
    echo "  done"
  fi

  apply_template "$MCP_DIR/templates/virtualworkspace.yaml"

  log "Waiting for the MCP virtual workspace..."
  retry 60 5 kubectl get deployment mcp-virtual-workspace >/dev/null 2>&1
  kubectl wait deployment/mcp-virtual-workspace --for=condition=Available --timeout=10m >/dev/null
  log "  mcp-virtual-workspace is ready."

  if host_resolves; then
    log "Verifying /services/mcp routes and authenticates..."
    retry 30 5 vw_mcp_rejects_anonymous
    log "  /services/mcp is routed and rejects anonymous callers."
  fi
}

vw_mcp_rejects_anonymous() {
  local code
  code="$(curl -sk -o /dev/null -w '%{http_code}' --max-time 10 \
    -X POST "https://$BASE_DOMAIN:$PORT/services/mcp" \
    -H 'Content-Type: application/json' \
    -H 'Accept: application/json, text/event-stream' \
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' 2>/dev/null)"
  [[ "$code" == "401" || "$code" == "403" ]]
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  vw_mcp_deploy
fi
