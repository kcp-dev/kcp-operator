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

cd $(dirname $0)/..

BOILERPLATE_HEADER="$(realpath hack/boilerplate.go.txt)"

BASE=github.com/kcp-dev/kcp-operator
MODULE="$BASE/sdk"
SDK_DIR=sdk
SDK_PKG="$MODULE"
APIS_PKG="$MODULE/apis"

set -x

# generate reconciling helpers
_tools/reconciler-gen --config hack/reconciling.yaml > internal/reconciling/zz_generated_reconcile.go

# generate CRDs
go run sigs.k8s.io/controller-tools/cmd/controller-gen \
  rbac:roleName=manager-role crd webhook \
  paths="./..." \
  output:crd:artifacts:config=config/crd/bases

# generate SDK
rm -rf -- "$SDK_DIR/{applyconfiguration,clientset,informers,listers}"

go run k8s.io/code-generator/cmd/applyconfiguration-gen \
  --go-header-file "$BOILERPLATE_HEADER" \
  --output-dir $SDK_DIR/applyconfiguration \
  --output-pkg $SDK_PKG/applyconfiguration \
  $APIS_PKG/operator/v1alpha1

go run k8s.io/code-generator/cmd/client-gen \
  --input-base "" \
  --input $APIS_PKG/operator/v1alpha1 \
  --clientset-name versioned \
  --go-header-file "$BOILERPLATE_HEADER" \
  --output-dir $SDK_DIR/clientset \
  --output-pkg $SDK_PKG/clientset

go run github.com/kcp-dev/code-generator/v2 \
  "client:headerFile=$BOILERPLATE_HEADER,apiPackagePath=$APIS_PKG,outputPackagePath=$SDK_PKG,singleClusterClientPackagePath=$SDK_PKG/clientset/versioned,singleClusterApplyConfigurationsPackagePath=$SDK_PKG/applyconfiguration" \
  "informer:headerFile=$BOILERPLATE_HEADER,apiPackagePath=$APIS_PKG,outputPackagePath=$SDK_PKG,singleClusterClientPackagePath=$SDK_PKG/clientset/versioned" \
  "lister:headerFile=$BOILERPLATE_HEADER,apiPackagePath=$APIS_PKG" \
  "paths=./sdk/apis/..." \
  "output:dir=$SDK_DIR"

# Use openshift's import fixer because gimps fails to parse some of the files;
# its output is identical to how gimps would sort the imports, but it also fixes
# the misplaced go:build directives.
for submodule in "applyconfiguration" "clientset" "informers" "listers"; do
  _tools/openshift-goimports \
    --module "$MODULE/$submodule" \
    --path "$SDK_DIR/$submodule"
done
