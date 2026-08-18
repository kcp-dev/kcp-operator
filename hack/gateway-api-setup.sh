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

# One-shot local instance of the Gateway API setup described in
# docs/content/setup/gateway-api.md: a kind cluster running cert-manager,
# Envoy Gateway, kcp-operator, etcd, a root shard, one additional shard and a
# front-proxy — everything exposed through a single TLS-passthrough gateway
# with SNI host routing on kcp.localhost hostnames.
#
#   hack/gateway-api-setup.sh         # create everything
#   hack/gateway-api-setup.sh down    # tear everything down
#
# When it finishes, an admin kubeconfig pointing at https://kcp.localhost:6443
# is written to ./kcp-admin.kubeconfig and a port-forward to the gateway is
# running in the background.
#
# Overridable via environment:
#   CLUSTER_NAME (kcp-gateway), BASE_DOMAIN (kcp.localhost), PORT (6443),
#   GATEWAY_IP (10.96.2.2, must be a free IP in the cluster's service CIDR),
#   KCP_IMAGE_REPOSITORY/KCP_IMAGE_TAG (ghcr.io/kcp-dev/kcp:04fcc9232),
#   KUBECONFIG_OUT (kcp-admin.kubeconfig)

set -euo pipefail

cd "$(dirname "$0")/.."

CLUSTER_NAME="${CLUSTER_NAME:-kcp-gateway}"
BASE_DOMAIN="${BASE_DOMAIN:-kcp.localhost}"
PORT="${PORT:-6443}"
GATEWAY_IP="${GATEWAY_IP:-10.96.2.2}"
KCP_IMAGE_REPOSITORY="${KCP_IMAGE_REPOSITORY:-ghcr.io/kcp-dev/kcp}"
KCP_IMAGE_TAG="${KCP_IMAGE_TAG:-04fcc9232}"
KUBECONFIG_OUT="${KUBECONFIG_OUT:-kcp-admin.kubeconfig}"

CERT_MANAGER_VERSION=v1.19.2
ENVOY_GATEWAY_VERSION=v1.7.0

HOSTNAMES=("$BASE_DOMAIN" "root.$BASE_DOMAIN" "alpha.$BASE_DOMAIN")
PORT_FORWARD_PID_FILE=".gateway-api-port-forward.pid"

# Everything runs against a dedicated kubeconfig so the user's current kubectl
# context is never touched.
CLUSTER_KUBECONFIG="$(pwd)/.gateway-api-cluster.kubeconfig"
export KUBECONFIG="$CLUSTER_KUBECONFIG"

log() {
  echo "[$(date +%H:%M:%S)] $*"
}

stop_port_forward() {
  if [[ -f "$PORT_FORWARD_PID_FILE" ]]; then
    kill "$(cat "$PORT_FORWARD_PID_FILE")" 2>/dev/null || true
    rm -f "$PORT_FORWARD_PID_FILE"
  fi
}

if [[ "${1:-}" == "down" ]]; then
  stop_port_forward
  kind delete cluster --name "$CLUSTER_NAME"
  rm -f "$KUBECONFIG_OUT" "$CLUSTER_KUBECONFIG"
  exit 0
fi

for tool in kind kubectl helm; do
  if ! command -v "$tool" >/dev/null; then
    echo "$tool is required but not installed." >&2
    exit 1
  fi
done

# retry <attempts> <delay> <command...> — for applies racing webhook/CRD readiness.
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

# The operator is built from this checkout rather than installed from a chart
# release: the virtual workspace deployed below relies on VirtualWorkspace
# features (spec.command, initContainers, kubeconfigSecretRef) newer than the
# released operator. Set OPERATOR_IMAGE to a pre-built image to skip the build.
if [[ -z "${OPERATOR_IMAGE:-}" ]]; then
  OPERATOR_IMAGE="kcp-operator:gateway-api-dev"
  log "Building kcp-operator image from this checkout ($OPERATOR_IMAGE)..."
  docker build -t "$OPERATOR_IMAGE" . >/dev/null
  kind load docker-image "$OPERATOR_IMAGE" --name "$CLUSTER_NAME" >/dev/null
fi

log "Installing kcp-operator ($OPERATOR_IMAGE)..."
helm repo add kcp https://kcp-dev.github.io/helm-charts --force-update >/dev/null
helm upgrade --install kcp-operator kcp/kcp-operator \
  --namespace kcp-operator --create-namespace \
  --set image.repository="${OPERATOR_IMAGE%:*}" \
  --set image.tag="${OPERATOR_IMAGE##*:}" >/dev/null
# Sync the CRDs to this checkout, in case the chart release lags behind. The
# deploy.operator.kcp.io group lives in its own kustomization.
kubectl apply --server-side --force-conflicts -k config/crd >/dev/null
kubectl apply --server-side --force-conflicts -k config/crd/deploy >/dev/null
# The chart's RBAC matches its release, not this checkout — grant the operator
# admin so a newer operator's API groups work too. Local development only.
kubectl apply -f - <<EOF >/dev/null
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kcp-operator-local-dev-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: kcp-operator
    namespace: kcp-operator
EOF
kubectl --namespace kcp-operator rollout restart deployment >/dev/null 2>&1 || true
kubectl --namespace kcp-operator wait deployment --all --for=condition=Available --timeout=5m >/dev/null

log "Installing etcd (single instance, no TLS — local development only)..."
helm upgrade --install etcd ./hack/ci/testdata/etcd >/dev/null

log "Creating Issuer, Gateway and TLSRoutes..."
# Retried because cert-manager's and Envoy Gateway's webhooks can take a moment
# to actually serve after their deployments report Available.
retry 10 5 kubectl apply -f - <<EOF >/dev/null
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
---
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: eg
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: eg
  namespace: envoy-gateway-system
spec:
  gatewayClassName: eg
  # Pinned so the gateway becomes Programmed without a load balancer
  # implementation; Envoy Gateway copies it into the Envoy Service's
  # externalIPs, which kube-proxy routes in-cluster.
  addresses:
    - type: IPAddress
      value: $GATEWAY_IP
  listeners:
    - name: passthrough
      protocol: TLS
      port: $PORT
      tls:
        mode: Passthrough
      allowedRoutes:
        namespaces:
          from: All
EOF

log "Creating CacheServer, RootShard, Shard, FrontProxy and their TLSRoutes..."
kubectl apply -f - <<EOF >/dev/null
# A dedicated cache server (with embedded etcd), shared by all shards. The
# embedded per-shard cache is not reachable across shards, so any multi-shard
# setup needs this.
apiVersion: operator.kcp.io/v1alpha1
kind: CacheServer
metadata:
  name: cache
spec:
  # Same image as the shards. All kcp components must run the same build —
  # a cache server on another version bootstraps a different API set and the
  # shards never sync their informers against it.
  image:
    repository: $KCP_IMAGE_REPOSITORY
    tag: "$KCP_IMAGE_TAG"
  certificates:
    issuerRef:
      group: cert-manager.io
      kind: Issuer
      name: selfsigned
---
apiVersion: operator.kcp.io/v1alpha1
kind: RootShard
metadata:
  name: root
spec:
  image:
    repository: $KCP_IMAGE_REPOSITORY
    tag: "$KCP_IMAGE_TAG"
  shardBaseURL: https://root.$BASE_DOMAIN:$PORT
  external:
    hostname: $BASE_DOMAIN
    port: $PORT
  extraArgs:
    # both shards share the etcd installed above
    - --etcd-prefix=/shard/root
  # The operator also deploys an internal proxy (root-proxy) through which
  # other shards reach the root shard. It dials shardBaseURL, so it needs the
  # same hostname mapping as everything else — and the same image version.
  proxy:
    image:
      repository: $KCP_IMAGE_REPOSITORY
      tag: "$KCP_IMAGE_TAG"
    deploymentTemplate:
      spec:
        template:
          spec:
            hostAliases:
              - ip: $GATEWAY_IP
                hostnames:
                  - $BASE_DOMAIN
                  - root.$BASE_DOMAIN
                  - alpha.$BASE_DOMAIN
  certificates:
    issuerRef:
      group: cert-manager.io
      kind: Issuer
      name: selfsigned
  certificateTemplates:
    server:
      spec:
        dnsNames:
          - root.$BASE_DOMAIN
  cache:
    ref:
      name: cache
  etcd:
    endpoints:
      - http://etcd.default.svc.cluster.local:2379
  deploymentTemplate:
    spec:
      template:
        spec:
          hostAliases:
            - ip: $GATEWAY_IP
              hostnames:
                - $BASE_DOMAIN
                - root.$BASE_DOMAIN
                - alpha.$BASE_DOMAIN
---
apiVersion: gateway.networking.k8s.io/v1alpha3
kind: TLSRoute
metadata:
  name: root
  namespace: default
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - root.$BASE_DOMAIN
  rules:
    - backendRefs:
        - name: root-kcp
          port: 6443
          namespace: default
---
apiVersion: operator.kcp.io/v1alpha1
kind: Shard
metadata:
  name: alpha
spec:
  rootShard:
    ref:
      name: root
  image:
    repository: $KCP_IMAGE_REPOSITORY
    tag: "$KCP_IMAGE_TAG"
  shardBaseURL: https://alpha.$BASE_DOMAIN:$PORT
  extraArgs:
    - --etcd-prefix=/shard/alpha
  # Unlike the RootShard, a Shard has no certificates/issuer or external
  # config of its own: its PKI and client-facing address come from the
  # referenced root shard. Only the serving certificate's SAN is shard-local.
  certificateTemplates:
    server:
      spec:
        dnsNames:
          - alpha.$BASE_DOMAIN
  etcd:
    endpoints:
      - http://etcd.default.svc.cluster.local:2379
  deploymentTemplate:
    spec:
      template:
        spec:
          hostAliases:
            - ip: $GATEWAY_IP
              hostnames:
                - $BASE_DOMAIN
                - root.$BASE_DOMAIN
                - alpha.$BASE_DOMAIN
---
apiVersion: gateway.networking.k8s.io/v1alpha3
kind: TLSRoute
metadata:
  name: alpha
  namespace: default
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - alpha.$BASE_DOMAIN
  rules:
    - backendRefs:
        - name: alpha-shard-kcp
          port: 6443
          namespace: default
---
apiVersion: operator.kcp.io/v1alpha1
kind: FrontProxy
metadata:
  name: frontproxy
spec:
  image:
    repository: $KCP_IMAGE_REPOSITORY
    tag: "$KCP_IMAGE_TAG"
  rootShard:
    ref:
      name: root
  external:
    hostname: $BASE_DOMAIN
    port: $PORT
  # Virtual workspaces are routed by path, not hostname: /services/access rides
  # the ordinary front-proxy hostname and is forwarded to the virtual
  # workspace's in-cluster Service.
  additionalPathMappings:
    - path: /services/access
      backend: https://access-virtual-workspace.default.svc.cluster.local:6443
      backend_server_ca: /etc/kcp-front-proxy/tls/ca/tls.crt
      proxy_client_cert: /etc/kcp-front-proxy/requestheader-client/tls.crt
      proxy_client_key: /etc/kcp-front-proxy/requestheader-client/tls.key
  deploymentTemplate:
    spec:
      template:
        spec:
          hostAliases:
            - ip: $GATEWAY_IP
              hostnames:
                - $BASE_DOMAIN
                - root.$BASE_DOMAIN
                - alpha.$BASE_DOMAIN
---
apiVersion: gateway.networking.k8s.io/v1alpha3
kind: TLSRoute
metadata:
  name: front-proxy
  namespace: default
spec:
  parentRefs:
    - name: eg
      namespace: envoy-gateway-system
  hostnames:
    - $BASE_DOMAIN
  rules:
    - backendRefs:
        - name: frontproxy-front-proxy
          port: $PORT
          namespace: default
---
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: kubeconfig-kcp-admin
spec:
  username: kcp-admin
  groups:
    - system:kcp:admin
  validity: 8766h
  secretRef:
    name: admin-kubeconfig
  target:
    frontProxyRef:
      name: frontproxy
EOF

log "Waiting for the gateway to be programmed..."
kubectl --namespace envoy-gateway-system wait gateway/eg --for=condition=Programmed --timeout=5m >/dev/null

log "Waiting for kcp to come up (first start pulls images and issues the whole PKI, this can take a few minutes)..."
for deployment in root-kcp alpha-shard-kcp frontproxy-front-proxy; do
  retry 60 10 kubectl get deployment "$deployment" >/dev/null 2>&1
  # Not `rollout status`: it fails immediately on a deployment that was ever
  # ProgressDeadlineExceeded, e.g. while kcp waits for its PKI on first start.
  kubectl wait "deployment/$deployment" --for=condition=Available --timeout=15m >/dev/null
  log "  $deployment is ready."
done

log "Waiting for the admin kubeconfig..."
retry 60 5 kubectl get secret admin-kubeconfig >/dev/null 2>&1
kubectl get secret admin-kubeconfig -o jsonpath='{.data.kubeconfig}' | base64 -d > "$KUBECONFIG_OUT"

log "Starting port-forward to the gateway on :$PORT..."
stop_port_forward
# The Envoy Service name is generated, hence the label selector. The loop keeps
# the forward alive across Envoy pod churn.
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

# --- Virtual workspaces --------------------------------------------------
#
# Of the three virtual workspace samples in config/samples/virtual-workspaces/,
# only the access virtual workspace has a published container image, so it is
# the one deployed here. The MCP server (which depends on it) and the
# ephemeral-resources server have no image on ghcr.io yet — build your own and
# adapt the samples to deploy them.

log "Deploying the access virtual workspace (see config/samples/virtual-workspaces/access.yaml)..."
kubectl apply -f - <<EOF >/dev/null
# The identity the init container bootstraps with: walks the workspace tree
# through the front-proxy and installs the APIExport, so it runs as kcp-admin.
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: access-vw-bootstrap
spec:
  target:
    frontProxyRef:
      name: frontproxy
  username: kcp-admin
  groups:
    - system:kcp:admin
  validity: 8766h
  secretRef:
    name: access-vw-bootstrap-kubeconfig
---
# The identity the server runs as: scoped to the workspace holding the
# APIExport, bound to the access-vw-controller ClusterRole the init container
# installs there.
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: access-vw-server
spec:
  target:
    frontProxyRef:
      name: frontproxy
  targetWorkspace: root:access:controllers
  username: access-vw
  validity: 8766h
  secretRef:
    name: access-vw-server-kubeconfig
  authorization:
    clusterRoleBindings:
      clusterRoles:
        - access-vw-controller
---
apiVersion: operator.kcp.io/v1alpha1
kind: VirtualWorkspace
metadata:
  name: access
spec:
  target:
    rootShardRef:
      name: root
  external:
    hostname: $BASE_DOMAIN
    port: $PORT
  image:
    repository: ghcr.io/kcp-dev/contrib-access-virtual-workspace
    tag: latest
  command:
    - /access-vw
  replicas: 1
  kubeconfigSecretRef:
    name: access-vw-server-kubeconfig
  initContainers:
    - name: init
      kubeconfigSecretRef:
        name: access-vw-bootstrap-kubeconfig
      command:
        - /access-vw-init
      args:
        - --kubeconfig=/etc/kcp/init-kubeconfig/kubeconfig
        - --workspace-prefix=root:access
        - --controllers-workspace=controllers
  extraArgs:
    - --apiexport-endpointslice=access.contrib.kcp.io
    - --endpoint-base=https://$BASE_DOMAIN:$PORT/clusters/
  deploymentTemplate:
    spec:
      template:
        spec:
          hostAliases:
            - ip: $GATEWAY_IP
              hostnames:
                - $BASE_DOMAIN
                - root.$BASE_DOMAIN
                - alpha.$BASE_DOMAIN
EOF

log "Waiting for the access virtual workspace (the init container bootstraps root:access in kcp first)..."
retry 60 10 kubectl get deployment access-virtual-workspace >/dev/null 2>&1
kubectl wait deployment/access-virtual-workspace --for=condition=Available --timeout=10m >/dev/null
log "  access-virtual-workspace is ready."

# The access-vw-controller ClusterRole the init container installs was written
# for a ServiceAccount, which kcp exempts from the workspace `access` check.
# The certificate identity minted above is not exempt, so grant it explicitly —
# see the comments in config/samples/virtual-workspaces/access.yaml.
ACCESS_RBAC_MANIFEST="$(mktemp)"
cat > "$ACCESS_RBAC_MANIFEST" <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: access-vw-workspace-access
rules:
  - verbs: ["access"]
    nonResourceURLs: ["/"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: access-vw-workspace-access
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: access-vw-workspace-access
subjects:
  - apiGroup: rbac.authorization.k8s.io
    kind: User
    name: access-vw
EOF

if python3 -c "import socket; socket.gethostbyname('$BASE_DOMAIN')" >/dev/null 2>&1; then
  log "Granting the server's identity workspace access in root:access:controllers..."
  retry 30 5 kubectl --kubeconfig "$KUBECONFIG_OUT" \
    --server "https://$BASE_DOMAIN:$PORT/clusters/root:access:controllers" \
    apply -f "$ACCESS_RBAC_MANIFEST" >/dev/null

  log "Verifying the access virtual workspace answers through the front-proxy..."
  retry 30 5 sh -c "kubectl --kubeconfig '$KUBECONFIG_OUT' get --raw '/services/access/clusters/root/apis/access.contrib.kcp.io/v1alpha1' 2>/dev/null | grep -q access.contrib.kcp.io"
  log "  /services/access serves the access.contrib.kcp.io API."
else
  echo "NOTE: $BASE_DOMAIN does not resolve here yet, so one manual step remains once it does:"
  echo "  kubectl --kubeconfig $KUBECONFIG_OUT --server https://$BASE_DOMAIN:$PORT/clusters/root:access:controllers apply -f $ACCESS_RBAC_MANIFEST"
fi

log "Skipping the MCP and ephemeral-resources virtual workspaces: no published images yet (see config/samples/virtual-workspaces/README.md)."

# --- Client-side wrap-up -------------------------------------------------

# The .localhost TLD is not auto-resolved on macOS (and not everywhere on
# Linux), so check and tell the user rather than editing /etc/hosts unasked.
unresolved=()
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

echo "kcp is up. Use it with:"
echo
echo "  export KUBECONFIG=\$(pwd)/$KUBECONFIG_OUT"
echo "  kubectl get workspaces"
echo
echo "The gateway port-forward runs in the background (PID $(cat "$PORT_FORWARD_PID_FILE"))."
echo "Tear everything down with: hack/gateway-api-setup.sh down"
