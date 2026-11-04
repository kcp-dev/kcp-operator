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

ACCESS_DIR="$CONTRIB_DIR/virtualworkspaces/access"

vw_access_deploy() {
  log "Deploying the access virtual workspace..."
  apply_template "$ACCESS_DIR/templates/virtualworkspace.yaml"

  if ! host_resolves; then
    echo "NOTE: $BASE_DOMAIN does not resolve here, so the workspace-access grant the"
    echo "server needs could not be applied and its Deployment will crash-loop until"
    echo "you apply it manually once the hostname resolves:"
    echo "  kubectl --kubeconfig $KUBECONFIG_OUT --server https://$BASE_DOMAIN:$PORT/clusters/root:access:controllers apply -f $ACCESS_DIR/templates/workspace-access-rbac.yaml"
    return 0
  fi

  log "Waiting for the init container to bootstrap root:access:controllers..."
  retry 60 10 vw_access_controllers_ready

  log "Granting the server's identity workspace access in root:access:controllers..."
  retry_quiet 30 5 kcpctl root:access:controllers apply -f "$ACCESS_DIR/templates/workspace-access-rbac.yaml"

  log "Waiting for the access virtual workspace..."
  retry 60 10 kubectl get deployment access-virtual-workspace >/dev/null 2>&1
  kubectl wait deployment/access-virtual-workspace --for=condition=Available --timeout=10m >/dev/null
  log "  access-virtual-workspace is ready."

  log "Verifying the access virtual workspace answers a SelfClusterAccessReview..."
  retry 30 5 vw_access_answers_scar
  log "  /services/access answers SelfClusterAccessReview."

  log "Binding the access APIExport in root:consumer so it appears in the graph..."
  # Created here, not assumed: ephemeral-resources uses it too, either may run first.
  ensure_workspace root consumer
  retry_quiet 30 5 kcpctl root:consumer apply -f "$ACCESS_DIR/templates/apibinding-consumer.yaml"
  log "  root:consumer binds access.contrib.kcp.io."
}

vw_access_controllers_ready() {
  [[ "$(kcpctl root:access get workspace controllers -o jsonpath='{.status.phase}' 2>/dev/null)" == "Ready" ]]
}

# SCAR is create-only, served directly under /services/access (no /clusters/<ws>).
vw_access_answers_scar() {
  local body
  body="$(mktemp)"
  printf '{"apiVersion":"access.contrib.kcp.io/v1alpha1","kind":"SelfClusterAccessReview"}' > "$body"
  local ok=1
  kubectl --kubeconfig "$KUBECONFIG_OUT" \
    create --raw '/services/access/apis/access.contrib.kcp.io/v1alpha1/selfclusteraccessreviews' \
    -f "$body" 2>/dev/null | grep -q SelfClusterAccessReview && ok=0
  rm -f "$body"
  return $ok
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  vw_access_deploy
fi
