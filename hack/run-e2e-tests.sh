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
export E2E_TOPOLOGY="${E2E_TOPOLOGY:-single}"

case "$E2E_TOPOLOGY" in
  single)
    # a single cluster runs both controller groups
    CONFIG_CLUSTER_NAME="$KIND_CLUSTER_NAME"
    WORKLOAD_CLUSTER_NAME=""
    ;;
  config-workload)
    # a config cluster runs the config controller group, a workload cluster the workload group
    CONFIG_CLUSTER_NAME="$KIND_CLUSTER_NAME-config"
    WORKLOAD_CLUSTER_NAME="$KIND_CLUSTER_NAME-workload"
    ;;
  *)
    echo "ERROR: E2E_TOPOLOGY must be 'single' or 'config-workload', got '$E2E_TOPOLOGY'"
    exit 1
    ;;
esac

DATA_DIR=".e2e-$KIND_CLUSTER_NAME-$E2E_TOPOLOGY"
OPERATOR_PID=0
PROTOKOL_PID=0
WORKLOAD_PROTOKOL_PID=0
SYNCER_PID=0
NO_TEARDOWN=${NO_TEARDOWN:-false}
USE_EXISTING_CLUSTER=${USE_EXISTING_CLUSTER:-false}

mkdir -p "$DATA_DIR"
rm -rf "$DATA_DIR/kind-logs" "$DATA_DIR/kind-logs-workload"
echo "Logs are stored in $DATA_DIR/."
DATA_DIR="$(realpath "$DATA_DIR")"

# create a local kind cluster or use existing one
if $USE_EXISTING_CLUSTER; then
  echo "Using existing clusters (USE_EXISTING_CLUSTER=true)..."
  if [[ -z "${KUBECONFIG:-}" ]]; then
    echo "ERROR: KUBECONFIG must be set when USE_EXISTING_CLUSTER=true"
    exit 1
  fi
  if [[ "$E2E_TOPOLOGY" == config-workload ]] && [[ -z "${WORKLOAD_KUBECONFIG:-}" ]]; then
    echo "ERROR: WORKLOAD_KUBECONFIG must be set when USE_EXISTING_CLUSTER=true and E2E_TOPOLOGY=config-workload"
    exit 1
  fi
  # Convert to absolute paths if relative
  export KUBECONFIG="$(realpath "$KUBECONFIG")"
  export WORKLOAD_KUBECONFIG="$(realpath "${WORKLOAD_KUBECONFIG:-$KUBECONFIG}")"
  echo "Using KUBECONFIG: $KUBECONFIG"
  echo "Using WORKLOAD_KUBECONFIG: $WORKLOAD_KUBECONFIG"
  CONFIG_CLUSTER_NAME="$KIND_CLUSTER_NAME"
  WORKLOAD_CLUSTER_NAME=""
else
  export KUBECONFIG="$DATA_DIR/kind-config.kubeconfig"
  echo "Creating kind cluster $CONFIG_CLUSTER_NAME (set \$USE_EXISTING_CLUSTER to true to use your own)"
  kind create cluster --name "$CONFIG_CLUSTER_NAME" --kubeconfig "$KUBECONFIG"
  chmod 600 "$KUBECONFIG"

  if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
    export WORKLOAD_KUBECONFIG="$DATA_DIR/kind-workload.kubeconfig"
    echo "Creating kind cluster $WORKLOAD_CLUSTER_NAME"
    kind create cluster --name "$WORKLOAD_CLUSTER_NAME" --kubeconfig "$WORKLOAD_KUBECONFIG"
    chmod 600 "$WORKLOAD_KUBECONFIG"
  else
    export WORKLOAD_KUBECONFIG="$KUBECONFIG"
  fi
fi

teardown_kind() {
  if [[ $PROTOKOL_PID -gt 0 ]]; then
    echo "Stopping protokol..."
    kill -TERM $PROTOKOL_PID
    # no wait because protokol ends quickly and wait would fail
  fi

  if [[ $WORKLOAD_PROTOKOL_PID -gt 0 ]]; then
    kill -TERM $WORKLOAD_PROTOKOL_PID
  fi

  if [[ $SYNCER_PID -gt 0 ]]; then
    echo "Stopping e2e syncer..."
    kill -TERM $SYNCER_PID
  fi

  if ! $USE_EXISTING_CLUSTER; then
    kind delete cluster --name "$CONFIG_CLUSTER_NAME"
    rm "$KUBECONFIG"
    if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
      kind delete cluster --name "$WORKLOAD_CLUSTER_NAME"
      rm "$WORKLOAD_KUBECONFIG"
    fi
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

    retry_linear 1 5 kind load docker-image "$PRELOAD_IMAGE" --name "$CONFIG_CLUSTER_NAME"
    if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
      retry_linear 1 5 kind load docker-image "$PRELOAD_IMAGE" --name "$WORKLOAD_CLUSTER_NAME"
    fi
  fi

  echo "kcp image tag: $KCP_TAG"
fi

KUBECTL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kubectl)"
KUSTOMIZE="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kustomize)"
HELM="$(UGET_PRINT_PATH=absolute make --no-print-directory install-helm)"
PROTOKOL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-protokol)"

# apply kernel limits job first and wait for completion (kcp pods run on the workload cluster)
echo "Applying kernel limits job..."
KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" apply --filename hack/ci/kernel.yaml
KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" wait --for=condition=Complete job/kernel-limits --timeout=300s
echo "Kernel limits job completed."

# deploying operator CRDs
echo "Deploying operator CRDs..."
"$KUBECTL" apply --server-side --kustomize config/crd

if [[ "$E2E_TOPOLOGY" == config-workload ]]; then
  echo "Deploying compiled resource CRDs on the workload cluster..."
  KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" apply --server-side --kustomize config/crd/deploy
else
  echo "Deploying compiled resource CRDs..."
  "$KUBECTL" apply --server-side --kustomize config/crd/deploy
fi

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
make --no-print-directory docker-build
KIND_CLUSTER_NAME="$CONFIG_CLUSTER_NAME" make --no-print-directory kind-load
if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
  KIND_CLUSTER_NAME="$WORKLOAD_CLUSTER_NAME" make --no-print-directory kind-load
fi

if [[ "$E2E_TOPOLOGY" == config-workload ]]; then
  echo "Deploying config kcp-operator..."
  "$KUSTOMIZE" build hack/ci/testdata/config | "$KUBECTL" apply --server-side --filename -

  echo "Deploying workload-only kcp-operator..."
  "$KUSTOMIZE" build hack/ci/testdata/workload | KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" apply --server-side --filename -

  echo "Building and starting e2e syncer..."
  go build -o "$DATA_DIR/e2e-syncer" ./test/syncer
  "$DATA_DIR/e2e-syncer" --config-kubeconfig "$KUBECONFIG" --workload-kubeconfig "$WORKLOAD_KUBECONFIG" >"$DATA_DIR/syncer.log" 2>&1 &
  SYNCER_PID=$!
else
  echo "Deploying kcp-operator..."
  "$KUSTOMIZE" build hack/ci/testdata/single | "$KUBECTL" apply --server-side --filename -
fi

"$PROTOKOL" --namespace 'e2e-*' --namespace kcp-operator-system --output "$DATA_DIR/kind-logs" 2>/dev/null &
PROTOKOL_PID=$!

if [[ "$E2E_TOPOLOGY" == config-workload ]]; then
  KUBECONFIG="$WORKLOAD_KUBECONFIG" "$PROTOKOL" --namespace 'e2e-*' --namespace kcp-operator-system --output "$DATA_DIR/kind-logs-workload" 2>/dev/null &
  WORKLOAD_PROTOKOL_PID=$!
fi

echo "Running e2e tests..."

export HELM_BINARY="$HELM"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-1}

# -parallel will only control how many tests run in parallel *within a single test package.*
# We however need to limit the overall amount of tests that can run at the same time, since
# the kind cluster does not have infinite capacity. The only way to tell Go to please not
# run all packages at the same time is setting GOMAXPROCS.
export GOMAXPROCS=$E2E_PARALLELISM

(set -x; go test -tags e2e -parallel $E2E_PARALLELISM $TEST_ARGS "$WHAT")
