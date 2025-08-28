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

mkdir -p "$DATA_DIR"
echo "Logs are stored in $DATA_DIR/."
DATA_DIR="$(realpath "$DATA_DIR")"

# create a local kind cluster

export KUBECONFIG="$DATA_DIR/kind.kubeconfig"
echo "Creating kind cluster $KIND_CLUSTER_NAME…"
kind create cluster --name "$KIND_CLUSTER_NAME"
chmod 600 "$KUBECONFIG"

teardown_kind() {
  echo "Stopping kcp-operator…"
  kill -TERM $OPERATOR_PID
  wait $OPERATOR_PID

  kind delete cluster --name "$KIND_CLUSTER_NAME"
  rm "$KUBECONFIG"
}
trap teardown_kind EXIT

echo "Kubeconfig is in $KUBECONFIG."

# deploying operator CRDs
echo "Deploying operator CRDs…"
kubectl apply --kustomize config/crd

# deploying cert-manager
echo "Deploying cert-manager…"

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

# start the operator locally
echo "Starting kcp-operator…"
_build/manager \
  -kubeconfig "$KUBECONFIG" \
  -zap-log-level debug \
  -zap-encoder console \
  -zap-time-encoding iso8601 \
  >"$DATA_DIR/kcp-operator.log" 2>&1 &
OPERATOR_PID=$!
echo "Running as process $OPERATOR_PID."

echo "Running e2e tests…"

export HELM_BINARY="$(realpath _tools/helm)"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-2}

(set -x; go test -tags e2e -p $E2E_PARALLELISM $TEST_ARGS "$WHAT")
