#!/usr/bin/env bash

# Copyright 2025 The KCP Authors.
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

export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-e2e}"
DATA_DIR=".e2e-$KIND_CLUSTER_NAME"
OPERATOR_PID=0
PROTOKOL_PID=0
NO_TEARDOWN=${NO_TEARDOWN:-false}

mkdir -p "$DATA_DIR"
rm -rf "$DATA_DIR/kind-logs"
echo "Logs are stored in $DATA_DIR/."
DATA_DIR="$(realpath "$DATA_DIR")"

# create a local kind cluster

export KUBECONFIG="$DATA_DIR/kind.kubeconfig"
echo "Creating kind cluster $KIND_CLUSTER_NAME..."
kind create cluster --name "$KIND_CLUSTER_NAME"
chmod 600 "$KUBECONFIG"

teardown_kind() {
  if [[ $PROTOKOL_PID -gt 0 ]]; then
    echo "Stopping protokol..."
    kill -TERM $PROTOKOL_PID
    # no wait because protokol ends quickly and wait would fail
  fi

  kind delete cluster --name "$KIND_CLUSTER_NAME"
  rm "$KUBECONFIG"
}

if ! $NO_TEARDOWN; then
  echo "Will tear down kind cluster once the script has finished."
  trap teardown_kind EXIT
fi

echo "Kubeconfig is in $KUBECONFIG."

# apply kernel limits job first and wait for completion
echo "Applying kernel limits job..."
kubectl apply --filename hack/ci/kernel.yaml
kubectl wait --for=condition=Complete job/kernel-limits --timeout=300s
echo "Kernel limits job completed."

# deploying operator CRDs
echo "Deploying operator CRDs..."
kubectl apply --kustomize config/crd

# deploying cert-manager
echo "Deploying cert-manager..."

_tools/helm repo add jetstack https://charts.jetstack.io --force-update
_tools/helm repo update

_tools/helm upgrade \
  --install \
  --namespace cert-manager \
  --create-namespace \
  --version v1.18.2 \
  --set crds.enabled=true \
  --atomic \
  cert-manager jetstack/cert-manager

kubectl apply --filename hack/ci/testdata/clusterissuer.yaml

# build operator image and deploy it into kind
echo "Building and deploying kcp-operator..."
export IMG="ghcr.io/kcp-dev/kcp-operator:e2e"
make --no-print-directory docker-build kind-load deploy

if command -v protokol &> /dev/null; then
  protokol --namespace 'e2e-*' --namespace kcp-operator-system --output "$DATA_DIR/kind-logs" 2>/dev/null &
  PROTOKOL_PID=$!
else
  echo "Install https://codeberg.org/xrstf/protokol to automatically"
  echo "collect logs from the kind cluster."
fi

echo "Running e2e tests..."

export HELM_BINARY="$(realpath _tools/helm)"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-2}

(set -x; go test -tags e2e -parallel $E2E_PARALLELISM $TEST_ARGS "$WHAT")
