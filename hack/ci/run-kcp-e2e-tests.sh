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

cd "$(dirname "$0")/../.."
source hack/lib.sh

# build the image(s)
export IMAGE_TAG=local

export CI_ARCH="$(go env GOARCH)"

echo "Building container images…"
ARCHITECTURES=$CI_ARCH DRY_RUN=yes ./hack/ci/build-image.sh

export KCP_E2E_TEST_IMAGE="ghcr.io/kcp-dev/kcp:e2e"
buildah build-using-dockerfile \
  --file test/kcp/Dockerfile \
  --tag "$KCP_E2E_TEST_IMAGE-$CI_ARCH" \
  --arch "$CI_ARCH" \
  --override-arch "$CI_ARCH" \
  --squash \
  --build-arg "TARGETOS=linux" \
  --build-arg "TARGETARCH=$CI_ARCH" \
  --format=docker \
  .

echo "Creating manifest $KCP_E2E_TEST_IMAGE..."
buildah manifest create "$KCP_E2E_TEST_IMAGE"
buildah manifest add "$KCP_E2E_TEST_IMAGE" "$KCP_E2E_TEST_IMAGE-$CI_ARCH"

# start docker so we can run kind
start_docker_daemon_ci

# create a local kind cluster
KIND_CLUSTER_NAME=e2e

echo "Preloading the kindest/node image…"
docker load --input /kindest.tar

export KUBECONFIG=$(mktemp)
echo "Creating kind cluster $KIND_CLUSTER_NAME…"
create_kind_cluster "$KIND_CLUSTER_NAME" kindest/node:v1.32.2
chmod 600 "$KUBECONFIG"

# apply kernel limits job first and wait for completion
echo "Applying kernel limits job…"
KUBECTL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-kubectl)"
"$KUBECTL" apply --filename hack/ci/kernel.yaml
"$KUBECTL" wait --for=condition=Complete job/kernel-limits --timeout=300s
echo "Kernel limits job completed."

# store logs as artifacts
PROTOKOL="$(UGET_PRINT_PATH=absolute make --no-print-directory install-protokol)"
"$PROTOKOL" --output "$ARTIFACTS/logs" --namespace 'kcp-*' --namespace 'e2e-*' >/dev/null 2>&1 &

# load the operator image into the kind cluster
image="ghcr.io/kcp-dev/kcp-operator:$IMAGE_TAG"
archive=operator.tar

echo "Loading operator image into kind…"
buildah manifest push "$image" "oci-archive:$archive:$image"
retry_linear 1 5 kind load image-archive "$archive" --name "$KIND_CLUSTER_NAME"

# load the tester image
echo "Loading tester image into kind…"
archive=tester.tar
buildah manifest push "$KCP_E2E_TEST_IMAGE" "oci-archive:$archive:$KCP_E2E_TEST_IMAGE"
retry_linear 1 5 kind load image-archive "$archive" --name "$KIND_CLUSTER_NAME"

# deploy the operator

echo "Deploying operator…"
"$KUBECTL" kustomize hack/ci/testdata | "$KUBECTL" apply --filename -
"$KUBECTL" --namespace kcp-operator-system wait deployment kcp-operator-controller-manager --for condition=Available
"$KUBECTL" --namespace kcp-operator-system wait pod --all --for condition=Ready

# deploying cert-manager
echo "Deploying cert-manager…"

HELM="$(UGET_PRINT_PATH=absolute make --no-print-directory install-helm)"

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

# Increase file descriptor limit for CI environments
ulimit -n 65536

echo "Running kcp e2e tests…"

export KUBECTL_BINARY="$KUBECTL"
export HELM_BINARY="$HELM"
export ETCD_HELM_CHART="$(realpath hack/ci/testdata/etcd)"

(set -x; go test -tags kcpe2e -timeout 2h -v ./test/kcp/...)

echo "Done. :-)"
