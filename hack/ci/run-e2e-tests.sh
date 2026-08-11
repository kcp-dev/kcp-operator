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

cd "$(dirname "$0")/../.."
source hack/lib.sh

# start docker so we can run kind
start_docker_daemon_ci

# For periodics especially it's important that we output what versions exactly
# we're testing against.
PRELOAD_IMAGE=

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
  fi

  echo "kcp image tag.......: $KCP_TAG"
fi

if [ -z "${PULL_BASE_REF:-}" ]; then
  echo "kcp-operator version: $(git rev-parse HEAD)"
fi

# build the image(s)
export IMAGE_TAG=local

echo "Building container images..."
ARCHITECTURES=arm64 DRY_RUN=yes ./hack/ci/build-image.sh

# create the local kind cluster(s)
export E2E_TOPOLOGY="${E2E_TOPOLOGY:-single}"

case "$E2E_TOPOLOGY" in
  single)
    # a single cluster runs both controller groups
    CONFIG_CLUSTER_NAME=e2e
    WORKLOAD_CLUSTER_NAME=""
    ;;
  config-workload)
    # a config cluster runs the config controller group, a workload cluster the workload group
    CONFIG_CLUSTER_NAME=e2e-config
    WORKLOAD_CLUSTER_NAME=e2e-workload
    ;;
  *)
    echo "ERROR: E2E_TOPOLOGY must be 'single' or 'config-workload', got '$E2E_TOPOLOGY'"
    exit 1
    ;;
esac

echo "Preloading the kindest/node image..."
docker load --input /kindest.tar

export KUBECONFIG=$(mktemp)
echo "Creating kind cluster $CONFIG_CLUSTER_NAME..."
create_kind_cluster "$CONFIG_CLUSTER_NAME" kindest/node:v1.32.2
chmod 600 "$KUBECONFIG"

if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
  export WORKLOAD_KUBECONFIG=$(mktemp)
  echo "Creating kind cluster $WORKLOAD_CLUSTER_NAME..."
  KUBECONFIG="$WORKLOAD_KUBECONFIG" create_kind_cluster "$WORKLOAD_CLUSTER_NAME" kindest/node:v1.32.2
  chmod 600 "$WORKLOAD_KUBECONFIG"
else
  export WORKLOAD_KUBECONFIG="$KUBECONFIG"
fi

# preload the custom kcp image, if requested
if [[ -n "$PRELOAD_IMAGE" ]]; then
  echo "Preloading kcp image $PRELOAD_IMAGE into kind clusters..."
  retry_linear 1 5 kind load docker-image "$PRELOAD_IMAGE" --name "$CONFIG_CLUSTER_NAME"
  if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
    retry_linear 1 5 kind load docker-image "$PRELOAD_IMAGE" --name "$WORKLOAD_CLUSTER_NAME"
  fi
fi

# apply kernel limits job on the workload cluster (where the kcp pods run)
# first and wait for completion
echo "Applying kernel limits job…"
KUBECTL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kubectl)"
KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" apply --filename hack/ci/kernel.yaml
KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" wait --for=condition=Complete job/kernel-limits --timeout=300s
echo "Kernel limits job completed."

# store logs as artifacts (optional; protokol may not be available)
if PROTOKOL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-protokol 2>/dev/null)"; then
  "$PROTOKOL" --output "$ARTIFACTS/logs" --namespace 'kcp-*' --namespace 'e2e-*' >/dev/null 2>&1 &
  if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
    KUBECONFIG="$WORKLOAD_KUBECONFIG" "$PROTOKOL" --output "$ARTIFACTS/logs-workload" --namespace 'kcp-*' --namespace 'e2e-*' >/dev/null 2>&1 &
  fi
else
  echo "WARNING: failed to install protokol, logs will not be collected as artifacts"
fi

# need Helm to setup etcd
HELM="$(UGET_PRINT_PATH=absolute make --no-print-directory install-helm)"

# load the operator image into the kind clusters
image="ghcr.io/kcp-dev/kcp-operator:$IMAGE_TAG"
archive=operator.tar

echo "Loading operator image into kind..."
buildah manifest push --all "$image" "oci-archive:$archive:$image"
kind load image-archive "$archive" --name "$CONFIG_CLUSTER_NAME"
if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
  kind load image-archive "$archive" --name "$WORKLOAD_CLUSTER_NAME"
fi

# The mock virtual workspace stands in for a server that is not kcp's own, so the e2e tests can
# cover VirtualWorkspace.spec.command, init containers and per-container kubeconfigs. It is built
# here rather than in hack/ci/build-image.sh so that it never enters the release pipeline, and it is
# single-architecture because it only ever runs on the local kind node.
export MOCK_VW_IMG="ghcr.io/kcp-dev/mock-virtualworkspace:$IMAGE_TAG"
mockArchive=mock-virtualworkspace.tar

echo "Building mock virtual workspace image $MOCK_VW_IMG..."
buildah build-using-dockerfile \
  --file hack/ci/testdata/mock-vw/Dockerfile \
  --tag "$MOCK_VW_IMG" \
  --format=docker \
  .

echo "Loading mock virtual workspace image into kind..."
buildah push "$MOCK_VW_IMG" "oci-archive:$mockArchive:$MOCK_VW_IMG"
kind load image-archive "$mockArchive" --name "$CONFIG_CLUSTER_NAME"
if [[ -n "$WORKLOAD_CLUSTER_NAME" ]]; then
  kind load image-archive "$mockArchive" --name "$WORKLOAD_CLUSTER_NAME"
fi

if [[ "$E2E_TOPOLOGY" == config-workload ]]; then
  # deploy the config cluster operator
  echo "Deploying config operator..."
  "$KUBECTL" kustomize hack/ci/testdata/config | "$KUBECTL" apply --server-side --filename -
  "$KUBECTL" --namespace kcp-operator-system wait deployment kcp-operator-controller-manager --for condition=Available
  "$KUBECTL" --namespace kcp-operator-system wait pod --all --for condition=Ready

  # the workload cluster only needs the deploy.operator.kcp.io CRDs
  echo "Deploying workload cluster operator..."
  KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" apply --server-side --kustomize config/crd/deploy
  "$KUBECTL" kustomize hack/ci/testdata/workload | KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" apply --server-side --filename -
  KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" --namespace kcp-operator-system wait deployment kcp-operator-controller-manager --for condition=Available
  KUBECONFIG="$WORKLOAD_KUBECONFIG" "$KUBECTL" --namespace kcp-operator-system wait pod --all --for condition=Ready

  # the syncer plays the role of an external GitOps tool, copying compiled
  # resources and secrets from the config to the workload cluster
  echo "Starting e2e syncer..."
  go build -o "$ARTIFACTS/e2e-syncer" ./test/syncer
  "$ARTIFACTS/e2e-syncer" --config-kubeconfig "$KUBECONFIG" --workload-kubeconfig "$WORKLOAD_KUBECONFIG" >"$ARTIFACTS/syncer.log" 2>&1 &
else
  echo "Deploying operator..."
  "$KUBECTL" kustomize hack/ci/testdata/single | "$KUBECTL" apply --server-side --filename -
  "$KUBECTL" --namespace kcp-operator-system wait deployment kcp-operator-controller-manager --for condition=Available
  "$KUBECTL" --namespace kcp-operator-system wait pod --all --for condition=Ready
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
  cert-manager jetstack/cert-manager

"$KUBECTL" apply --filename hack/ci/testdata/clusterissuer.yaml

echo "Running e2e tests..."

export HELM_BINARY="$HELM"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-1}

# Increase file descriptor limit for CI environments
ulimit -n 65536

# -parallel will only control how many tests run in parallel *within a single test package.*
# We however need to limit the overall amount of tests that can run at the same time, since
# the kind cluster does not have infinite capacity. The only way to tell Go to please not
# run all packages at the same time is setting GOMAXPROCS.
export GOMAXPROCS=$E2E_PARALLELISM

(set -x; go test -tags e2e -parallel $E2E_PARALLELISM $TEST_ARGS "$WHAT")

echo "Done. :-)"
