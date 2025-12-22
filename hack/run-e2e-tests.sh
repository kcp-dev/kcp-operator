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

KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-e2e}"
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
  if [[ $OPERATOR_PID -gt 0 ]]; then
    echo "Stopping kcp-operator..."
    kill -TERM $OPERATOR_PID
    wait $OPERATOR_PID
  fi

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

KUBECTL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kubectl)"
HELM="$(UGET_PRINT_PATH=absolute make --no-print-directory install-helm)"
PROTOKOL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-protokol)"

# apply kernel limits job first and wait for completion
echo "Applying kernel limits job…"
"$KUBECTL" apply --filename hack/ci/kernel.yaml
"$KUBECTL" wait --for=condition=Complete job/kernel-limits --timeout=300s
echo "Kernel limits job completed."

# deploying operator CRDs
echo "Deploying operator CRDs..."
"$KUBECTL" apply --kustomize config/crd

# deploying cert-manager
echo "Deploying cert-manager..."

"$HELM" repo add jetstack https://charts.jetstack.io --force-update
"$HELM" repo update

"$HELM" upgrade \
  --install \
  --namespace cert-manager \
  --create-namespace \
  --version v1.18.2 \
  --set crds.enabled=true \
  --atomic \
  cert-manager jetstack/cert-manager

"$KUBECTL" apply --filename hack/ci/testdata/clusterissuer.yaml

# start the operator locally
echo "Starting kcp-operator..."
_build/manager \
  -kubeconfig "$KUBECONFIG" \
  -zap-log-level debug \
  -zap-encoder console \
  -zap-time-encoding iso8601 \
  -health-probe-bind-address="" \
  >"$DATA_DIR/kcp-operator.log" 2>&1 &
OPERATOR_PID=$!
echo "Running as process $OPERATOR_PID."

"$PROTOKOL" --namespace 'e2e-*' --output "$DATA_DIR/kind-logs" 2>/dev/null &
PROTOKOL_PID=$!

echo "Running e2e tests..."

export HELM_BINARY="$HELM"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-2}

(set -x; go test -tags e2e -parallel $E2E_PARALLELISM $TEST_ARGS "$WHAT")
