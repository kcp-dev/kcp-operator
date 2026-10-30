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

EPHEMERAL_DIR="$CONTRIB_DIR/virtualworkspaces/ephemeral-resources"
export EPHEMERAL_VW_DIR="${EPHEMERAL_VW_DIR:-$REPO_ROOT/../contrib-virtual-ephemeral-resources-virtual-workspace}"
export EPHEMERAL_IMAGE="${EPHEMERAL_IMAGE:-ghcr.io/kcp-dev/contrib-virtual-ephemeral-resources-virtual-workspace:latest}"
# Split out: envsubst has no ${VAR%:*}.
export EPHEMERAL_IMAGE_REPOSITORY="${EPHEMERAL_IMAGE%:*}"
export EPHEMERAL_IMAGE_TAG="${EPHEMERAL_IMAGE##*:}"

vw_ephemeral_resources_deploy() {
  if [[ ! -d "$EPHEMERAL_VW_DIR/docs/example" ]]; then
    echo "NOTE: no contrib-virtual-ephemeral-resources-virtual-workspace checkout at"
    echo "$EPHEMERAL_VW_DIR (set EPHEMERAL_VW_DIR); skipping the ephemeral-resources virtual workspace."
    return 0
  fi
  if ! host_resolves; then
    echo "NOTE: $BASE_DOMAIN does not resolve on this machine; skipping the"
    echo "ephemeral-resources virtual workspace (its kcp-side bootstrap runs through the gateway)."
    return 0
  fi

  log "Using the ephemeral-resources image $EPHEMERAL_IMAGE."

  log "Bootstrapping the provider workspace root:providers:s3..."
  ensure_workspace root providers
  ensure_workspace root:providers s3

  vw_ephemeral_apply_provider_objects

  log "Issuing the example webhook's serving certificate..."
  apply_template "$EPHEMERAL_DIR/templates/webhook-certs.yaml"

  log "Deploying the example webhook..."
  apply_template "$EPHEMERAL_DIR/templates/webhook.yaml"

  log "Deploying the ephemeral-resources virtual workspace and endpoint slice controller..."
  apply_template "$EPHEMERAL_DIR/templates/virtualworkspace.yaml"

  log "Waiting for the webhook, server and controller..."
  local deployment
  for deployment in ephemeral-webhook ephemeral-virtual-workspace ephemeral-endpointslice-controller; do
    retry 60 5 kubectl get deployment "$deployment" >/dev/null 2>&1
    kubectl wait "deployment/$deployment" --for=condition=Available --timeout=10m >/dev/null
    log "  $deployment is ready."
  done

  log "Waiting for the endpoint slice to carry this server's URL..."
  retry 60 5 vw_ephemeral_slice_has_url

  log "Binding the export in a consumer workspace..."
  ensure_workspace root consumer
  retry_quiet 30 5 kcpctl root:consumer apply -f "$EPHEMERAL_VW_DIR/docs/example/04-apibinding.yaml"

  log "Verifying a BucketInfo create is answered by the webhook (nothing is stored)..."
  retry 60 10 vw_ephemeral_create_answered
  log "  BucketInfo answered by the webhook through /services/ephemeral-buckets."
}

vw_ephemeral_apply_provider_objects() {
  local asset
  asset="$(mktemp)"

  sed '/identityHash:/d' "$EPHEMERAL_VW_DIR/docs/example/00-endpointslice-crd.yaml" > "$asset"
  retry_quiet 30 5 kcpctl root:providers:s3 apply -f "$asset"

  log "Waiting for the EphemeralResourceEndpointSlice CRD to be served..."
  retry 60 2 vw_ephemeral_slice_crd_served

  local name
  for name in 01-apiresourceschema 02-apiexport 03-endpointslice; do
    sed '/identityHash:/d' "$EPHEMERAL_VW_DIR/docs/example/$name.yaml" > "$asset"
    retry_quiet 30 5 kcpctl root:providers:s3 apply -f "$asset"
  done

  rm -f "$asset"
}

vw_ephemeral_slice_crd_served() {
  kcpctl root:providers:s3 api-resources --api-group=ephemeral.contrib.kcp.io 2>/dev/null \
    | grep -q '^ephemeralresourceendpointslices[[:space:]]'
}

vw_ephemeral_slice_has_url() {
  kcpctl root:providers:s3 get ephemeralresourceendpointslices s3.example.com \
    -o jsonpath='{.status.endpoints[0].url}' 2>/dev/null | grep -q '^https://'
}

vw_ephemeral_create_answered() {
  kcpctl root:consumer create -f "$EPHEMERAL_VW_DIR/docs/example/bucketinfo.yaml" -o yaml 2>/dev/null \
    | grep -q 'kind: BucketInfo'
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  vw_ephemeral_resources_deploy
fi
