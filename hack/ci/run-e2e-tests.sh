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

# For periodics especially it's important that we output what versions exactly
# we're testing against.
if [ -n "${KCP_TAG:-}" ]; then
  # resolve what looks like branch names
  if [[ "$KCP_TAG" == main ]] || [[ "$KCP_TAG" =~ ^release- ]]; then
    echo "Resolving kcp $KCP_TAG ..."

    tmpdir="$(mktemp -d)"
    here="$(pwd)"

    cd "$tmpdir"
    git clone --quiet --depth 1 --branch "$KCP_TAG" --single-branch https://github.com/kcp-dev/kcp .
    KCP_TAG="$(git rev-parse HEAD)"
    cd "$here"
    rm -rf "$tmpdir"

    # kcp's containers are tagged with the first 9 characters of the Git hash
    KCP_TAG="${KCP_TAG:0:9}"
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

# start docker so we can run kind
start-docker.sh

# create a local kind cluster
KIND_CLUSTER_NAME=e2e

echo "Preloading the kindest/node image..."
docker load --input /kindest.tar

export KUBECONFIG=$(mktemp)
echo "Creating kind cluster $KIND_CLUSTER_NAME..."
kind create cluster --name "$KIND_CLUSTER_NAME" --image kindest/node:v1.32.2
chmod 600 "$KUBECONFIG"

# store logs as artifacts
make protokol
_tools/protokol --output "$ARTIFACTS/logs" --namespace 'kcp-*' --namespace 'e2e-*' >/dev/null 2>&1 &

# need Helm to setup etcd
make helm

# load the operator image into the kind cluster
image="ghcr.io/kcp-dev/kcp-operator:$IMAGE_TAG"
archive=operator.tar

echo "Loading operator image into kind..."
buildah manifest push --all "$image" "oci-archive:$archive:$image"
kind load image-archive "$archive" --name "$KIND_CLUSTER_NAME"

# deploy the operator
echo "Deploying operator..."
kubectl kustomize hack/ci/testdata | kubectl apply --filename -
kubectl --namespace kcp-operator-system wait deployment kcp-operator-controller-manager --for condition=Available
kubectl --namespace kcp-operator-system wait pod --all --for condition=Ready

# deploying cert-manager
echo "Deploying cert-manager..."

_tools/helm repo add jetstack https://charts.jetstack.io --force-update
_tools/helm repo update

_tools/helm upgrade \
  --install \
  --namespace cert-manager \
  --create-namespace \
  --version v1.16.2 \
  --set crds.enabled=true \
  cert-manager jetstack/cert-manager

kubectl apply --filename hack/ci/testdata/clusterissuer.yaml

echo "Running e2e tests..."

export HELM_BINARY="$(realpath _tools/helm)"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

WHAT="${WHAT:-./test/e2e/...}"
TEST_ARGS="${TEST_ARGS:--timeout 2h -v}"
E2E_PARALLELISM=${E2E_PARALLELISM:-2}

(set -x; go test -tags e2e -parallel $E2E_PARALLELISM $TEST_ARGS "$WHAT")

echo "Done. :-)"
