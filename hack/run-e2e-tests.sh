#!/usr/bin/env bash

# Copyright 2025 The kcp Authors.
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

cd $(dirname $0)/..
source hack/lib.sh

export KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME:-e2e}"
DATA_DIR=".e2e-$KIND_CLUSTER_NAME"
OPERATOR_PID=0
PROTOKOL_PID=0
NO_TEARDOWN=${NO_TEARDOWN:-false}
USE_EXISTING_CLUSTER=${USE_EXISTING_CLUSTER:-false}

mkdir -p "$DATA_DIR"
rm -rf "$DATA_DIR/kind-logs"
echo "Logs are stored in $DATA_DIR/."
DATA_DIR="$(realpath "$DATA_DIR")"

# create a local kind cluster or use existing one
if $USE_EXISTING_CLUSTER; then
  echo "Using existing cluster (USE_EXISTING_CLUSTER=true)..."
  if [[ -z "${KUBECONFIG:-}" ]]; then
    echo "ERROR: KUBECONFIG must be set when USE_EXISTING_CLUSTER=true"
    exit 1
  fi
  # Convert to absolute path if relative
  export KUBECONFIG="$(realpath "$KUBECONFIG")"
  echo "Using KUBECONFIG: $KUBECONFIG"
else
  export KUBECONFIG="$DATA_DIR/kind.kubeconfig"
  echo "Creating kind cluster $KIND_CLUSTER_NAME (set \$USE_EXISTING_CLUSTER to true to use your own)"
  kind create cluster --name "$KIND_CLUSTER_NAME"
  chmod 600 "$KUBECONFIG"
fi

teardown_kind() {
  if [[ $PROTOKOL_PID -gt 0 ]]; then
    echo "Stopping protokol..."
    kill -TERM $PROTOKOL_PID
    # no wait because protokol ends quickly and wait would fail
  fi

  if ! $USE_EXISTING_CLUSTER; then
    kind delete cluster --name "$KIND_CLUSTER_NAME"
    rm "$KUBECONFIG"
  fi
}

if ! $NO_TEARDOWN; then
  if $USE_EXISTING_CLUSTER; then
    echo "Will stop operator and protokol once the script has finished (keeping existing cluster)."
  else
    echo "Will tear down kind cluster once the script has finished."
  fi
  trap teardown_kind EXIT
fi

echo "Kubeconfig is in $KUBECONFIG."

if [ -n "${KCP_TAG:-}" ]; then
  # resolve what looks like branch names
  if [[ "$KCP_TAG" == main ]] || [[ "$KCP_TAG" =~ ^release- ]]; then
    if [[ "$KCP_TAG" == main ]]; then
      # rely on a KCP_RELEASE env in the Prowjob
      KCP_TAG_ALIAS="v$KCP_RELEASE.999"
    else
      KCP_TAG_ALIAS="v$(echo "$KCP_TAG" | sed -E 's/release-//g').999"
    fi

    echo "Resolving kcp $KCP_TAG (as $KCP_TAG_ALIAS)..."

    tmpdir="$(mktemp -d)"
    here="$(pwd)"

    cd "$tmpdir"
    git clone --quiet --depth 1 --branch "$KCP_TAG" --single-branch https://github.com/kcp-dev/kcp .
    gitHead="$(git rev-parse HEAD)"
    cd "$here"
    rm -rf "$tmpdir"

    # kcp's containers are tagged with the first 9 characters of the Git hash
    ORIGINAL_TAG="${gitHead:0:9}"

    echo "Going to use kcp image $ORIGINAL_TAG as $KCP_TAG_ALIAS."

    # Due to the process above, we might now run the tests against "kcp:d6ab2dc"
    # or whatever random hash might be the most recent build. This interferes with
    # the operator's version detection. To work around this, we pull the image first,
    # retag it with a dummy version, load it into kind and then use that image.
    KCP_TAG="$KCP_TAG_ALIAS"
    ORIGINAL_IMAGE="ghcr.io/kcp-dev/kcp:$ORIGINAL_TAG"
    PRELOAD_IMAGE="ghcr.io/kcp-dev/kcp:$KCP_TAG"
    docker pull "$ORIGINAL_IMAGE"
    docker tag "$ORIGINAL_IMAGE" "$PRELOAD_IMAGE"

    retry_linear 1 5 kind load docker-image "$PRELOAD_IMAGE" --name "$KIND_CLUSTER_NAME"
  fi

  echo "kcp image tag: $KCP_TAG"
fi

KUBECTL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kubectl)"
KUSTOMIZE="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kustomize)"
HELM="$(UGET_PRINT_PATH=absolute make --no-print-directory install-helm)"
PROTOKOL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-protokol)"

# apply kernel limits job first and wait for completion
echo "Applying kernel limits job..."
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
  --version v1.20.2 \
  --set crds.enabled=true \
  --atomic \
  cert-manager jetstack/cert-manager

"$KUBECTL" apply --filename hack/ci/testdata/clusterissuer.yaml

# build operator image it into kind
echo "Building and loading kcp-operator..."
export IMG="ghcr.io/kcp-dev/kcp-operator:local"
make --no-print-directory docker-build kind-load

echo "Deploying kcp-operator..."
"$KUSTOMIZE" build hack/ci/testdata | "$KUBECTL" apply --filename -

"$PROTOKOL" --namespace 'e2e-*' --namespace kcp-operator-system --output "$DATA_DIR/kind-logs" 2>/dev/null &
PROTOKOL_PID=$!

echo "Running e2e tests..."

export HELM_BINARY="$HELM"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"
export KCP_RELEASE="${KCP_TAG:-}"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-1}

# -parallel will only control how many tests run in parallel *within a single test package.*
# We however need to limit the overall amount of tests that can run at the same time, since
# the kind cluster does not have infinite capacity. The only way to tell Go to please not
# run all packages at the same time is setting GOMAXPROCS.
export GOMAXPROCS=$E2E_PARALLELISM

(set -x; go test -tags e2e -parallel $E2E_PARALLELISM $TEST_ARGS "$WHAT")
