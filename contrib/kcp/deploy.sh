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

source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/lib.sh"

KCP_DIR="$CONTRIB_DIR/kcp"

kcp_down() {
  stop_port_forward
  kind delete cluster --name "$CLUSTER_NAME"
  rm -f "$KUBECONFIG_OUT" "$CLUSTER_KUBECONFIG"
}

kcp_deploy() {
  require_tools kind kubectl helm docker envsubst python3
  assert_port_free

  if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER_NAME"; then
    log "Creating kind cluster $CLUSTER_NAME..."
    kind create cluster --name "$CLUSTER_NAME" --kubeconfig "$CLUSTER_KUBECONFIG"
  else
    log "Reusing existing kind cluster $CLUSTER_NAME."
    kind export kubeconfig --name "$CLUSTER_NAME" --kubeconfig "$CLUSTER_KUBECONFIG"
  fi

  log "Installing cert-manager $CERT_MANAGER_VERSION..."
  helm repo add jetstack https://charts.jetstack.io --force-update >/dev/null
  helm upgrade --install cert-manager jetstack/cert-manager \
    --namespace cert-manager --create-namespace \
    --version "$CERT_MANAGER_VERSION" \
    --set crds.enabled=true >/dev/null
  kubectl --namespace cert-manager wait deployment --all --for=condition=Available --timeout=5m >/dev/null

  log "Installing Envoy Gateway $ENVOY_GATEWAY_VERSION..."
  helm upgrade --install envoy oci://docker.io/envoyproxy/gateway-helm \
    --version "$ENVOY_GATEWAY_VERSION" \
    --namespace envoy-gateway-system --create-namespace >/dev/null
  kubectl --namespace envoy-gateway-system wait deployment/envoy-gateway --for=condition=Available --timeout=5m >/dev/null

  if [[ -z "${OPERATOR_IMAGE:-}" ]]; then
    OPERATOR_IMAGE="kcp-operator:gateway-api-dev"
    log "Building kcp-operator image from this checkout ($OPERATOR_IMAGE)..."
    docker build -t "$OPERATOR_IMAGE" "$REPO_ROOT" >/dev/null
    kind load docker-image "$OPERATOR_IMAGE" --name "$CLUSTER_NAME" >/dev/null
  fi

  log "Installing kcp-operator ($OPERATOR_IMAGE)..."
  helm repo add kcp https://kcp-dev.github.io/helm-charts --force-update >/dev/null
  helm upgrade --install kcp-operator kcp/kcp-operator \
    --namespace kcp-operator --create-namespace \
    --set image.repository="${OPERATOR_IMAGE%:*}" \
    --set image.tag="${OPERATOR_IMAGE##*:}" >/dev/null
  # Sync CRDs to this checkout, in case the chart release lags behind.
  kubectl apply --server-side --force-conflicts -k "$REPO_ROOT/config/crd" >/dev/null
  kubectl apply --server-side --force-conflicts -k "$REPO_ROOT/config/crd/deploy" >/dev/null
  # Chart RBAC matches its release, not this checkout. Local dev only.
  retry 10 5 kubectl apply -f "$KCP_DIR/templates/operator-rbac.yaml" >/dev/null
  kubectl --namespace kcp-operator rollout restart deployment >/dev/null 2>&1 || true
  kubectl --namespace kcp-operator wait deployment --all --for=condition=Available --timeout=5m >/dev/null

  log "Installing etcd (single instance, no TLS — local development only)..."
  helm upgrade --install etcd "$REPO_ROOT/hack/ci/testdata/etcd" >/dev/null

  log "Creating Issuer, Gateway and TLSRoutes..."
  apply_template "$KCP_DIR/templates/gateway.yaml"

  log "Creating CacheServer, RootShard, Shard, FrontProxy and their TLSRoutes..."
  apply_template "$KCP_DIR/templates/kcp.yaml"

  log "Waiting for the gateway to be programmed..."
  kubectl --namespace envoy-gateway-system wait gateway/eg --for=condition=Programmed --timeout=5m >/dev/null

  kcp_trust_shard_identity

  log "Waiting for kcp to come up (first start pulls images and issues the whole PKI, this can take a few minutes)..."
  local deployment
  for deployment in root-kcp alpha-shard-kcp frontproxy-front-proxy; do
    retry 60 10 kubectl get deployment "$deployment" >/dev/null 2>&1
    # Not `rollout status`: it fails on any past ProgressDeadlineExceeded.
    kubectl wait "deployment/$deployment" --for=condition=Available --timeout=15m >/dev/null
    log "  $deployment is ready."
  done

  log "Waiting for the admin kubeconfig..."
  retry 60 5 kubectl get secret admin-kubeconfig >/dev/null 2>&1
  kubectl get secret admin-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d > "$KUBECONFIG_OUT"

  kcp_start_port_forward
}

kcp_trust_shard_identity() {
  log "Trusting the shards' forwarded identity at the front-proxy..."
  retry 60 5 kubectl get secret root-requestheader-client-ca >/dev/null 2>&1
  retry 60 5 kubectl get secret root-client-ca >/dev/null 2>&1

  local bundle
  bundle="$(mktemp)"
  kubectl get secret root-requestheader-client-ca -o jsonpath='{.data.tls\.crt}' | base64 -d >  "$bundle"
  kubectl get secret root-client-ca              -o jsonpath='{.data.tls\.crt}' | base64 -d >> "$bundle"
  kubectl create secret generic frontproxy-shard-requestheader-ca \
    --from-file=tls.crt="$bundle" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  rm -f "$bundle"

  local patch
  patch="$(mktemp)"
  render "$KCP_DIR/templates/frontproxy-requestheader-patch.json" > "$patch"
  kubectl patch frontproxy frontproxy --type=merge --patch-file "$patch" >/dev/null
  rm -f "$patch"
}

kcp_start_port_forward() {
  log "Starting port-forward to the gateway on :$PORT..."
  stop_port_forward
  # The Envoy Service name is generated, hence the label selector.
  nohup bash -c '
    while true; do
      svc_name="$(kubectl --namespace envoy-gateway-system \
        get svc -l gateway.envoyproxy.io/owning-gateway-name=eg \
        -o jsonpath="{.items[0].metadata.name}" 2>/dev/null || true)"
      if [[ -n "$svc_name" ]]; then
        kubectl --namespace envoy-gateway-system port-forward "svc/$svc_name" '"$PORT:$PORT"' || true
      fi
      sleep 2
    done
  ' >/dev/null 2>&1 &
  echo $! > "$PORT_FORWARD_PID_FILE"
}

# Only act when executed; when sourced, the caller picks the function.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  if [[ "${1:-}" == "down" ]]; then
    kcp_down
  else
    kcp_deploy
  fi
fi
