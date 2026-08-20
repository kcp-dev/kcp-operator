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
    # for the ephemeral-resources virtual workspace: reference-driven
    # replication is alpha and gated, and the shard has to trust the serving
    # chain of the endpoint URLs it dials (they go through the front-proxy,
    # whose certificate chains up to the root CA mounted in every shard pod).
    - --feature-gates=CacheAPIs=true
    - --shard-virtual-workspace-ca-file=/etc/kcp/tls/ca/root/tls.crt
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
    # see the RootShard above
    - --feature-gates=CacheAPIs=true
    - --shard-virtual-workspace-ca-file=/etc/kcp/tls/ca/root/tls.crt
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
      backend_server_ca: /etc/kcp/tls/ca/tls.crt
      proxy_client_cert: /etc/kcp-front-proxy/requestheader-client/tls.crt
      proxy_client_key: /etc/kcp-front-proxy/requestheader-client/tls.key
    - path: /services/ephemeral-buckets
      backend: https://ephemeral-virtual-workspace.default.svc.cluster.local:6443
      backend_server_ca: /etc/kcp/tls/ca/tls.crt
      proxy_client_cert: /etc/kcp-front-proxy/requestheader-client/tls.crt
      proxy_client_key: /etc/kcp-front-proxy/requestheader-client/tls.key
    - path: /services/mcp
      backend: https://mcp-virtual-workspace.default.svc.cluster.local:6443
      backend_server_ca: /etc/kcp/tls/ca/tls.crt
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

# Let the front-proxy carry a shard's forwarded identity through to a virtual
# workspace.
#
# A shard proxying a request to a virtual workspace stamps the caller into
# X-Remote-User and dials with --shard-client-cert-file. When that dial goes
# through the front-proxy (which it does here: the endpoint slice publishes
# $BASE_DOMAIN:$PORT), the proxy does what an edge proxy must — strips the
# inbound X-Remote-* and re-stamps whoever it authenticated. The shard's
# certificate is signed by the client CA and named after the shard, so it
# authenticates as an ordinary user and the caller is lost. Every ephemeral
# create then fails with
#
#   User "root" cannot create resource "bucketinfos" ... : access denied
#
# where "root" is the root shard's certificate common name, not an account.
#
# The front-proxy preserves a forwarded identity only for callers that pass
# request-header authentication, so the shards have to satisfy it: their CA in
# the request-header bundle, their common names in the allowed list.
#
# WHAT THIS COSTS: the client CA becomes an impersonation CA for those two
# names. Anyone who can get a certificate issued from it with CN=root can assert
# any identity at the edge — and that CA also signs every ordinary user
# certificate, kcp-admin included. Acceptable in a throwaway dev stack, which is
# the only thing this script builds. The clean fix is a shard certificate issued
# from the request-header CA specifically for dialling virtual workspaces; kcp
# reuses --shard-client-cert-file for that hop today, so there is nothing to
# point at yet.
log "Trusting the shards' forwarded identity at the front-proxy..."
retry 60 5 kubectl get secret root-requestheader-client-ca >/dev/null 2>&1
retry 60 5 kubectl get secret root-client-ca >/dev/null 2>&1

RH_BUNDLE="$(mktemp)"
kubectl get secret root-requestheader-client-ca -o jsonpath='{.data.tls\.crt}' | base64 -d >  "$RH_BUNDLE"
kubectl get secret root-client-ca              -o jsonpath='{.data.tls\.crt}' | base64 -d >> "$RH_BUNDLE"
kubectl create secret generic frontproxy-shard-requestheader-ca \
  --from-file=tls.crt="$RH_BUNDLE" --dry-run=client -o yaml | kubectl apply -f - >/dev/null
rm -f "$RH_BUNDLE"

# Shard client certificate common names: the root shard uses its own name, an
# additional shard is prefixed. Both are listed so this keeps working whichever
# shard a consumer workspace lands on.
#
# The allowed-names list is passed in full rather than as an addition: pflag
# appends on repeated string slices, so naming only the extras would depend on
# that staying true. Duplicates are harmless.
kubectl patch frontproxy frontproxy --type=merge -p "$(cat <<EOF
{
  "spec": {
    "extraArgs": [
      "--requestheader-client-ca-file=/etc/kcp-front-proxy/shard-requestheader-ca/tls.crt",
      "--requestheader-allowed-names=kcp-front-proxy,kcp-mounts-proxy,root,shard-alpha"
    ],
    "extraVolumes": [
      {"name": "shard-requestheader-ca", "secret": {"secretName": "frontproxy-shard-requestheader-ca"}}
    ],
    "extraVolumeMounts": [
      {"name": "shard-requestheader-ca", "mountPath": "/etc/kcp-front-proxy/shard-requestheader-ca", "readOnly": true}
    ]
  }
}
EOF
)" >/dev/null

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
# All three virtual workspace samples in config/samples/virtual-workspaces/ are
# deployed here, in dependency order: access first, then ephemeral-resources,
# then MCP — which calls the access virtual workspace on every request and so
# needs it serving before it starts.
#
# access and MCP have published images on ghcr.io. The ephemeral-resources
# server does not, so it is built from a local checkout further down.

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

# The access-vw-controller ClusterRole the init container installs was written
# for a ServiceAccount, which kcp exempts from the workspace `access` check.
# The certificate identity minted above is not exempt, so grant it explicitly —
# see the comments in config/samples/virtual-workspaces/access.yaml.
#
# Ordering matters: the server refuses to start until this grant exists (its
# startup discovery is answered with "access denied" otherwise), so the grant
# has to land BEFORE waiting for the Deployment — right after the init
# container has bootstrapped the workspace it applies to.
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
  log "Waiting for the init container to bootstrap root:access:controllers..."
  retry 60 10 sh -c "[ \"\$(kubectl --kubeconfig '$KUBECONFIG_OUT' --server 'https://$BASE_DOMAIN:$PORT/clusters/root:access' get workspace controllers -o jsonpath='{.status.phase}' 2>/dev/null)\" = Ready ]"

  log "Granting the server's identity workspace access in root:access:controllers..."
  retry 30 5 kubectl --kubeconfig "$KUBECONFIG_OUT" \
    --server "https://$BASE_DOMAIN:$PORT/clusters/root:access:controllers" \
    apply -f "$ACCESS_RBAC_MANIFEST" >/dev/null

  log "Waiting for the access virtual workspace..."
  retry 60 10 kubectl get deployment access-virtual-workspace >/dev/null 2>&1
  kubectl wait deployment/access-virtual-workspace --for=condition=Available --timeout=10m >/dev/null
  log "  access-virtual-workspace is ready."

  log "Verifying the access virtual workspace answers a SelfClusterAccessReview..."
  # SCAR is a create-only REST resource served directly under /services/access —
  # no /clusters/<ws> segment, see the repository's test/e2e.
  SCAR_FILE="$(mktemp)"
  printf '{"apiVersion":"access.contrib.kcp.io/v1alpha1","kind":"SelfClusterAccessReview"}' > "$SCAR_FILE"
  retry 30 5 sh -c "kubectl --kubeconfig '$KUBECONFIG_OUT' create --raw '/services/access/apis/access.contrib.kcp.io/v1alpha1/selfclusteraccessreviews' -f '$SCAR_FILE' 2>/dev/null | grep -q SelfClusterAccessReview"
  log "  /services/access answers SelfClusterAccessReview."

  # A workspace only appears in the access graph once it binds the export: the
  # server builds the graph from APIBindings, and the permission claims are what
  # let it read that workspace's RBAC. Without at least one binding everything
  # below still works and answers with an empty list, which is a tedious thing
  # to debug — /services/access returns 200, MCP's list_workspaces returns
  # {"workspaces":[]}, and nothing anywhere is in error.
  # Mirrors config/examples/apibinding-consumer.yaml in the access virtual
  # workspace repository. Inlined rather than read from a checkout, because
  # nothing else in this section needs one. The permission claims are what let
  # the server read this workspace's RBAC; without them the binding succeeds and
  # the graph stays empty.
  log "Binding the access APIExport in root:consumer so it appears in the graph..."
  ACCESS_BINDING_MANIFEST="$(mktemp)"
  cat > "$ACCESS_BINDING_MANIFEST" <<'EOF'
apiVersion: apis.kcp.io/v1alpha2
kind: APIBinding
metadata:
  name: access.contrib.kcp.io
spec:
  reference:
    export:
      path: root:access:controllers
      name: access.contrib.kcp.io
  permissionClaims:
    - group: rbac.authorization.k8s.io
      resource: clusterrolebindings
      verbs: [get, list, watch]
      state: Accepted
      selector:
        matchAll: true
    - group: rbac.authorization.k8s.io
      resource: rolebindings
      verbs: [get, list, watch]
      state: Accepted
      selector:
        matchAll: true
    - group: rbac.authorization.k8s.io
      resource: clusterroles
      verbs: [get, list, watch]
      state: Accepted
      selector:
        matchAll: true
    - group: rbac.authorization.k8s.io
      resource: roles
      verbs: [get, list, watch]
      state: Accepted
      selector:
        matchAll: true
EOF
  retry 30 5 kubectl --kubeconfig "$KUBECONFIG_OUT" \
    --server "https://$BASE_DOMAIN:$PORT/clusters/root:consumer" \
    apply -f "$ACCESS_BINDING_MANIFEST" >/dev/null
  log "  root:consumer binds access.contrib.kcp.io."
else
  echo "NOTE: $BASE_DOMAIN does not resolve here, so the workspace-access grant the"
  echo "server needs could not be applied and its Deployment will crash-loop until"
  echo "you apply it manually once the hostname resolves:"
  echo "  kubectl --kubeconfig $KUBECONFIG_OUT --server https://$BASE_DOMAIN:$PORT/clusters/root:access:controllers apply -f $ACCESS_RBAC_MANIFEST"
fi

# --- Ephemeral resources virtual workspace -------------------------------
#
# Webhook-backed, never-persisted resources (see
# config/samples/virtual-workspaces/ephemeral-resources.yaml). There is no
# published container image, so this builds one from a local checkout of
# https://github.com/kcp-dev/contrib-virtual-ephemeral-resources-virtual-workspace
# (override the location with EPHEMERAL_VW_DIR) covering all three binaries:
# the server, the endpointslice-controller and the example webhook.
#
# What gets stood up, following the repository's docs/example:
#   * provider workspace root:providers:s3 with the endpoint slice CRD,
#     APIResourceSchemas, the APIExport (bucketinfos with virtual storage) and
#     the EphemeralResourceEndpointSlice;
#   * the example webhook as a Deployment, with a cert-manager-issued serving
#     certificate;
#   * the virtual workspace itself through the operator, routed via the
#     front-proxy at /services/ephemeral-buckets;
#   * the endpointslice-controller as a plain Deployment, publishing
#     https://$BASE_DOMAIN:$PORT into the slice;
#   * a consumer workspace binding the export, verified by creating a
#     BucketInfo and getting the webhook's answer back.
#
# Provider-side RBAC for the shard identity is deliberately absent: the
# operator issues shard client certificates in system:masters, which
# short-circuits the server's authorizer. Production deployments issue a
# non-privileged shard identity and grant it `access` on / plus create on
# apiexports/content in the provider workspace instead.

EPHEMERAL_VW_DIR="${EPHEMERAL_VW_DIR:-$(pwd)/../contrib-virtual-ephemeral-resources-virtual-workspace}"

fpctl() {
  local workspace="$1"
  shift
  kubectl --kubeconfig "$KUBECONFIG_OUT" --server "https://$BASE_DOMAIN:$PORT/clusters/$workspace" "$@"
}

if [[ ! -d "$EPHEMERAL_VW_DIR/docs/example" ]]; then
  echo "NOTE: no contrib-virtual-ephemeral-resources-virtual-workspace checkout at"
  echo "$EPHEMERAL_VW_DIR (set EPHEMERAL_VW_DIR); skipping the ephemeral-resources virtual workspace."
elif ! python3 -c "import socket; socket.gethostbyname('$BASE_DOMAIN')" >/dev/null 2>&1; then
  echo "NOTE: $BASE_DOMAIN does not resolve on this machine; skipping the"
  echo "ephemeral-resources virtual workspace (its kcp-side bootstrap runs through the gateway)."
else
  EPHEMERAL_IMAGE="ephemeral-vw:gateway-api-dev"
  log "Building the ephemeral-resources image from $EPHEMERAL_VW_DIR ($EPHEMERAL_IMAGE)..."
  docker build -t "$EPHEMERAL_IMAGE" -f - "$EPHEMERAL_VW_DIR" <<'EOF' >/dev/null
FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ephemeral-virtual-workspace ./cmd/ephemeral-virtual-workspace \
 && CGO_ENABLED=0 go build -o /out/endpointslice-controller ./cmd/endpointslice-controller \
 && CGO_ENABLED=0 go build -o /out/example-webhook ./examples/webhook
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/ /
USER 65532:65532
EOF
  kind load docker-image "$EPHEMERAL_IMAGE" --name "$CLUSTER_NAME" >/dev/null

  # ensure_workspace <parent path> <name> — create a workspace pinned to the
  # shard the virtual workspace is attached to, and wait for it to be Ready.
  #
  # The pinning is load-bearing, and both halves of it fail loudly if omitted.
  # Workspaces are scheduled round-robin across shards, while this virtual
  # workspace is attached to one (spec.target.rootShardRef):
  #
  #   * a provider workspace on another shard makes the consumer's shard resolve
  #     spec.resources[].storage.virtual.reference against its own CRD informer,
  #     which does not hold another shard's CRDs — the resource vanishes from
  #     discovery with "no matches for kind EphemeralResourceEndpointSlice";
  #   * a consumer workspace on another shard is invisible to this server's
  #     APIBinding informer, and a create is refused with "logical cluster ...
  #     has no APIBinding to APIExport ...".
  #
  # Hence everything lands on the root shard here. Lifting this is the
  # repository's known gap #2 ("not tested against a sharded kcp").
  ensure_workspace() {
    local parent="$1" name="$2"
    cat > "$WS_MANIFEST" <<WSEOF
apiVersion: tenancy.kcp.io/v1alpha1
kind: Workspace
metadata:
  name: $name
spec:
  location:
    selector:
      matchLabels:
        name: root
WSEOF
    retry 30 5 fpctl "$parent" apply -f "$WS_MANIFEST" >/dev/null
    retry 60 5 sh -c "[ \"\$(kubectl --kubeconfig '$KUBECONFIG_OUT' --server 'https://$BASE_DOMAIN:$PORT/clusters/$parent' get workspace $name -o jsonpath='{.status.phase}' 2>/dev/null)\" = Ready ]"
  }

  log "Bootstrapping the provider workspace root:providers:s3..."
  WS_MANIFEST="$(mktemp)"
  ensure_workspace root providers
  ensure_workspace root:providers s3

  # The provider-side objects, in the repository's documented order. The
  # identityHash filter tracks a kcp API change: ResourceSchemaStorageVirtual
  # lost the field, but the repository's example still carries it.
  ASSET_FILE="$(mktemp)"
  sed '/identityHash:/d' "$EPHEMERAL_VW_DIR/docs/example/00-endpointslice-crd.yaml" > "$ASSET_FILE"
  retry 30 5 fpctl root:providers:s3 apply -f "$ASSET_FILE" >/dev/null

  # The APIExport must not be created before the kind it references is served.
  # kcp's apiexport-reference controller resolves spec.resources[].storage.virtual.reference
  # through the RESTMapper; on a no-match it skips the reference silently and never
  # requeues (it watches APIExports and ClusterCachedResources, not CRDs). Applying
  # 00 and 02 in the same breath therefore leaves no ClusterCachedResource, the
  # EphemeralResourceEndpointSlice never reaches the cache server, and every shard
  # fails discovery for the virtual resource with "failed to retrieve virtual
  # workspace URL" — permanently, until something touches the APIExport again.
  endpointslice_crd_served() {
    fpctl root:providers:s3 api-resources --api-group=ephemeral.contrib.kcp.io 2>/dev/null \
      | grep -q '^ephemeralresourceendpointslices[[:space:]]'
  }
  log "Waiting for the EphemeralResourceEndpointSlice CRD to be served..."
  retry 60 2 endpointslice_crd_served >/dev/null

  for asset in 01-apiresourceschema 02-apiexport 03-endpointslice; do
    sed '/identityHash:/d' "$EPHEMERAL_VW_DIR/docs/example/$asset.yaml" > "$ASSET_FILE"
    retry 30 5 fpctl root:providers:s3 apply -f "$ASSET_FILE" >/dev/null
  done

  log "Issuing the example webhook's serving certificate..."
  retry 10 5 kubectl apply -f - <<EOF >/dev/null
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ephemeral-webhook-ca
spec:
  isCA: true
  commonName: ephemeral-webhook-ca
  secretName: ephemeral-webhook-ca
  issuerRef:
    group: cert-manager.io
    kind: Issuer
    name: selfsigned
---
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: ephemeral-webhook-ca
spec:
  ca:
    secretName: ephemeral-webhook-ca
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ephemeral-webhook
spec:
  dnsNames:
    - ephemeral-webhook.default.svc.cluster.local
    - ephemeral-webhook.default.svc
    - ephemeral-webhook
  secretName: ephemeral-webhook-tls
  issuerRef:
    group: cert-manager.io
    kind: Issuer
    name: ephemeral-webhook-ca
EOF

  log "Deploying the example webhook..."
  kubectl apply -f - <<EOF >/dev/null
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ephemeral-webhook
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ephemeral-webhook
  template:
    metadata:
      labels:
        app: ephemeral-webhook
    spec:
      containers:
        - name: webhook
          image: $EPHEMERAL_IMAGE
          command:
            - /example-webhook
          args:
            - --address=:18443
            - --tls-cert-file=/etc/webhook/tls/tls.crt
            - --tls-private-key-file=/etc/webhook/tls/tls.key
          volumeMounts:
            - name: tls
              mountPath: /etc/webhook/tls
              readOnly: true
      volumes:
        - name: tls
          secret:
            secretName: ephemeral-webhook-tls
---
apiVersion: v1
kind: Service
metadata:
  name: ephemeral-webhook
spec:
  selector:
    app: ephemeral-webhook
  ports:
    - port: 18443
      targetPort: 18443
EOF

  log "Deploying the ephemeral-resources virtual workspace and endpoint slice controller..."
  kubectl apply -f - <<EOF >/dev/null
# The registry of webhook-backed resources, read by the server at startup.
apiVersion: v1
kind: ConfigMap
metadata:
  name: ephemeral-vw-config
data:
  ephemeral-config.yaml: |
    apiVersion: ephemeral.contrib.kcp.io/v1alpha1
    kind: EphemeralWebhookConfiguration
    resources:
      - export:
          path: root:providers:s3
          name: s3.example.com
        group: s3.example.com
        resource: bucketinfos
        webhook:
          url: https://ephemeral-webhook.default.svc.cluster.local:18443/ephemeral/bucketinfos
          caBundleFile: /etc/ephemeral/webhook-ca/ca.crt
          timeoutSeconds: 10
---
# The identity the endpointslice-controller writes slice status with, already
# scoped to the provider workspace.
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: ephemeral-controller
spec:
  target:
    frontProxyRef:
      name: frontproxy
  targetWorkspace: root:providers:s3
  username: kcp-admin
  groups:
    - system:kcp:admin
  validity: 8766h
  secretRef:
    name: ephemeral-controller-kubeconfig
---
apiVersion: operator.kcp.io/v1alpha1
kind: VirtualWorkspace
metadata:
  name: ephemeral
spec:
  target:
    rootShardRef:
      name: root
  external:
    hostname: $BASE_DOMAIN
    port: $PORT
  image:
    repository: ${EPHEMERAL_IMAGE%:*}
    tag: "${EPHEMERAL_IMAGE##*:}"
  command:
    - /ephemeral-virtual-workspace
  replicas: 1
  extraArgs:
    - --ephemeral-config=/etc/ephemeral/config/ephemeral-config.yaml
    - --virtual-workspace-name=ephemeral-buckets
    # the webhook Service resolves to a cluster-internal address, which the
    # SSRF guard refuses by default
    - --allow-private-webhook-addresses
    # the APIExport may be owned by the other shard; only the cache server
    # sees every shard's objects. The operator mounts the kubeconfig, the flag
    # is ours to pass since this is not kcp's own virtual-workspace binary.
    - --cache-kubeconfig=/etc/cache-server/kubeconfig/kubeconfig
  extraVolumes:
    - name: ephemeral-config
      configMap:
        name: ephemeral-vw-config
    - name: webhook-ca
      secret:
        secretName: ephemeral-webhook-tls
        items:
          - key: ca.crt
            path: ca.crt
  extraVolumeMounts:
    - name: ephemeral-config
      mountPath: /etc/ephemeral/config
      readOnly: true
    - name: webhook-ca
      mountPath: /etc/ephemeral/webhook-ca
      readOnly: true
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
# Publishes this server's address into the EphemeralResourceEndpointSlice; a
# separate, provider-scoped process by design.
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ephemeral-endpointslice-controller
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ephemeral-endpointslice-controller
  template:
    metadata:
      labels:
        app: ephemeral-endpointslice-controller
    spec:
      hostAliases:
        - ip: $GATEWAY_IP
          hostnames:
            - $BASE_DOMAIN
      containers:
        - name: controller
          image: $EPHEMERAL_IMAGE
          command:
            - /endpointslice-controller
          args:
            - --kubeconfig=/etc/kcp/kubeconfig/kubeconfig
            - --external-url=https://$BASE_DOMAIN:$PORT
            - --virtual-workspace-name=ephemeral-buckets
            - --export=s3.example.com
          volumeMounts:
            - name: kubeconfig
              mountPath: /etc/kcp/kubeconfig
              readOnly: true
      volumes:
        - name: kubeconfig
          secret:
            secretName: ephemeral-controller-kubeconfig
EOF

  log "Waiting for the webhook, server and controller..."
  kubectl wait deployment/ephemeral-webhook --for=condition=Available --timeout=5m >/dev/null
  retry 60 10 kubectl get deployment ephemeral-virtual-workspace >/dev/null 2>&1
  kubectl wait deployment/ephemeral-virtual-workspace --for=condition=Available --timeout=10m >/dev/null
  kubectl wait deployment/ephemeral-endpointslice-controller --for=condition=Available --timeout=5m >/dev/null

  log "Waiting for the endpoint slice to carry this server's URL..."
  retry 60 10 sh -c "kubectl --kubeconfig '$KUBECONFIG_OUT' --server 'https://$BASE_DOMAIN:$PORT/clusters/root:providers:s3' get ephemeralresourceendpointslice s3.example.com -o jsonpath='{.status.endpoints[*].url}' 2>/dev/null | grep -q ephemeral-buckets"

  log "Binding the export in a consumer workspace..."
  ensure_workspace root consumer
  retry 30 5 fpctl root:consumer apply -f "$EPHEMERAL_VW_DIR/docs/example/04-apibinding.yaml" >/dev/null
  retry 60 5 sh -c "[ \"\$(kubectl --kubeconfig '$KUBECONFIG_OUT' --server 'https://$BASE_DOMAIN:$PORT/clusters/root:consumer' get apibinding s3.example.com -o jsonpath='{.status.phase}' 2>/dev/null)\" = Bound ]"

  log "Verifying a BucketInfo create is answered by the webhook (nothing is stored)..."
  retry 60 10 sh -c "kubectl --kubeconfig '$KUBECONFIG_OUT' --server 'https://$BASE_DOMAIN:$PORT/clusters/root:consumer' create -f '$EPHEMERAL_VW_DIR/docs/example/bucketinfo.yaml' -o yaml 2>/dev/null | grep -q 'kind: BucketInfo'"
  log "  BucketInfo answered by the webhook through /services/ephemeral-buckets."
fi

# --- MCP virtual workspace -----------------------------------------------
#
# Serves the Model Context Protocol at /services/mcp, scoped to what the calling
# user can see. See config/samples/virtual-workspaces/mcp.yaml.
#
# Deployed last because it depends on the access virtual workspace: every
# request fetches the caller's workspace list from /services/access, so that has
# to be serving first (it was verified above).
#
# The server holds no resource rights of its own. It impersonates the caller, so
# kcp authorizes each per-workspace request as the human who asked and the audit
# log records both identities. That is why the only grant below is `impersonate`.
log "Deploying the MCP virtual workspace (see config/samples/virtual-workspaces/mcp.yaml)..."

# The operator provisions the ClusterRoleBinding for the Kubeconfig's identity,
# but not the ClusterRole itself — that has to exist wherever the binding lands
# (root, for a front-proxy-targeted Kubeconfig with no targetWorkspace) and in
# every workspace whose resources the server should reach on a caller's behalf.
# Taken from deploy/kcp/rbac.yaml in the repository.
MCP_RBAC_MANIFEST="$(mktemp)"
cat > "$MCP_RBAC_MANIFEST" <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: mcp-virtual-workspace-impersonator
rules:
  - apiGroups: [""]
    resources: ["users", "groups", "serviceaccounts", "userextras"]
    verbs: ["impersonate"]
EOF

# root only: that is where a front-proxy-targeted Kubeconfig with no
# targetWorkspace puts its binding, and the impersonation this server does today
# — the SelfClusterAccessReview it sends to /services/access on the caller's
# behalf — is authorized there. Verified by removing it everywhere else and
# watching list_workspaces keep working.
#
# A tool that reads resources inside a consumer's workspace would need the role
# there too; none exists yet.
retry 30 5 kubectl --kubeconfig "$KUBECONFIG_OUT" \
  --server "https://$BASE_DOMAIN:$PORT/clusters/root" \
  apply -f "$MCP_RBAC_MANIFEST" >/dev/null

kubectl apply -f - <<EOF >/dev/null
# The identity the server connects to kcp with. Impersonation rights and nothing
# else; the ClusterRole above is what the operator's binding points at.
apiVersion: operator.kcp.io/v1alpha1
kind: Kubeconfig
metadata:
  name: mcp-vw-server
spec:
  target:
    frontProxyRef:
      name: frontproxy
  username: mcp-virtual-workspace
  validity: 8766h
  secretRef:
    name: mcp-vw-server-kubeconfig
  authorization:
    clusterRoleBindings:
      clusterRoles:
        - mcp-virtual-workspace-impersonator
---
apiVersion: operator.kcp.io/v1alpha1
kind: VirtualWorkspace
metadata:
  name: mcp
spec:
  target:
    rootShardRef:
      name: root
  external:
    hostname: $BASE_DOMAIN
    port: $PORT
  image:
    repository: ghcr.io/kcp-dev/contrib-mcp-virtual-workspace
    tag: latest
  # The entrypoint plus its subcommand. The operator's generated flags (TLS, the
  # requestheader CAs, --secure-port=6443, --kubeconfig) are all accepted, and
  # the requestheader configuration is what satisfies the server's "at least one
  # authentication method" startup check — no OIDC flags needed behind the
  # front-proxy.
  command:
    - /mcp-virtual-workspace
    - serve
  replicas: 1
  kubeconfigSecretRef:
    name: mcp-vw-server-kubeconfig
  extraArgs:
    - --access-url=https://$BASE_DOMAIN:$PORT/services/access
    # The front-proxy's serving certificate chains up to the root CA the
    # operator mounts into every virtual workspace pod.
    - --access-ca-file=/etc/kcp/tls/ca/root/tls.crt
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

log "Waiting for the MCP virtual workspace..."
retry 60 5 kubectl get deployment mcp-virtual-workspace >/dev/null 2>&1
kubectl wait deployment/mcp-virtual-workspace --for=condition=Available --timeout=10m >/dev/null
log "  mcp-virtual-workspace is ready."

if python3 -c "import socket; socket.gethostbyname('$BASE_DOMAIN')" >/dev/null 2>&1; then
  # An unauthenticated JSON-RPC initialize. 401/403 is the pass: it proves the
  # path routed to this server and its filter chain ran. A 404 would mean the
  # front-proxy mapping did not match and kcp answered instead — the one failure
  # this check exists to catch. Anything authenticated needs a client
  # certificate, which the e2e suite in the repository covers.
  log "Verifying /services/mcp routes and authenticates..."
  mcp_rejects_anonymous() {
    local code
    code="$(curl -sk -o /dev/null -w '%{http_code}' --max-time 10 \
      -X POST "https://$BASE_DOMAIN:$PORT/services/mcp" \
      -H 'Content-Type: application/json' \
      -H 'Accept: application/json, text/event-stream' \
      -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' 2>/dev/null)"
    [[ "$code" == "401" || "$code" == "403" ]]
  }
  retry 30 5 mcp_rejects_anonymous >/dev/null
  log "  /services/mcp is routed and rejects anonymous callers."
fi

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
