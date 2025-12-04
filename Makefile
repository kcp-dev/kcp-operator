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

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

export CGO_ENABLED ?= 0
export GOFLAGS ?= -mod=readonly -trimpath
export GO111MODULE = on
GOBUILDFLAGS ?= -v
LDFLAGS += -extldflags '-static'
LDFLAGS_EXTRA ?= -w

ifdef DEBUG_BUILD
GOFLAGS = -mod=readonly
LDFLAGS_EXTRA =
GOTOOLFLAGS_EXTRA = -gcflags=all="-N -l"
endif

BUILD_DEST ?= _build
GOTOOLFLAGS ?= $(GOBUILDFLAGS) -ldflags '$(LDFLAGS) $(LDFLAGS_EXTRA)' $(GOTOOLFLAGS_EXTRA)

# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/kcp-dev/kcp-operator

GOIMPORTS_VERSION ?= c72f1dc2e3aacfa00aece3391d938c9bc734e791
GOLANGCI_LINT_VERSION ?= 2.1.6
HELM_VERSION ?= 3.18.6
KUBECTL_VERSION ?= v1.32.0
KUSTOMIZE_VERSION ?= v5.4.3
PROTOKOL_VERSION ?= 0.7.2

# codegen tooling
APPLYCONFIGURATION_GEN_VERSION ?= v0.32.0
CLIENT_GEN_VERSION ?= v0.32.0
CONTROLLER_GEN_VERSION ?= v0.16.1
KCP_CODEGEN_VERSION ?= v2.3.1
RECONCILER_GEN_VERSION ?= v0.5.0

export UGET_DIRECTORY ?= _tools
export UGET_CHECKSUMS ?= hack/tools.checksums
export UGET_VERSIONED_BINARIES = true

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: codegen
codegen: ## Generate manifest, code and the SDK.
	@hack/update-codegen.sh

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test-e2e  # Run the e2e tests against a kind k8s instance that is already spun up.
test-e2e:
	go test ./test/e2e/ -v

# Creates a kind cluster and runs the e2e tests in them. The kind cluster is destroyed after the tests.
.PHONY: test-e2e-with-kind  # Run the e2e tests against a temporary kind cluster.
test-e2e-with-kind:
	@hack/run-e2e-tests.sh

GOLANGCI_LINT = $(UGET_DIRECTORY)/golangci-lint-$(GOLANGCI_LINT_VERSION)

.PHONY: lint
lint: install-golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run --timeout 10m
	cd sdk && ../$(GOLANGCI_LINT) --config ../.golangci.yml run --timeout 10m

.PHONY: lint-fix
lint-fix: install-golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --timeout 10m --fix
	cd sdk && ../$(GOLANGCI_LINT) --config ../.golangci.yml run --timeout 10m --fix

.PHONY: modules
modules: ## Run go mod tidy to ensure modules are up to date.
	hack/update-go-modules.sh

GOIMPORTS = $(UGET_DIRECTORY)/goimports-$(GOIMPORTS_VERSION)

.PHONY: imports
imports: install-goimports ## Re-order Go import statements.
	$(GOIMPORTS) -m github.com/kcp-dev/kcp-operator

.PHONY: verify
verify: codegen fmt vet modules imports ## Run all codegen and formatting targets and check if files have changed.
	if ! git diff --quiet --exit-code ; then echo "ERROR: Found unexpected changes to git repository"; git diff; exit 1; fi

##@ Build

.PHONY: clean
clean: ## Remove all built binaries.
	rm -rf $(BUILD_DEST)

.PHONY: clean-tools
clean-tools: ## Remove all downloaded tools.
	rm -rf $(UGET_DIRECTORY)

.PHONY: build
build: ## Build manager binary.
	go build $(GOTOOLFLAGS) -o $(BUILD_DEST)/manager cmd/main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

KUSTOMIZE = $(abspath .)/$(UGET_DIRECTORY)/kustommize-$(KUSTOMIZE_VERSION)
KUBECTL = $(abspath .)/$(UGET_DIRECTORY)/kubectl-$(KUBECTL_VERSION)

.PHONY: build-installer
build-installer: manifests generate install-kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: kubectl install-kustomize install-kubectl ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: kubectl install-kustomize install-kubectl ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: kubectl install-kustomize install-kubectl ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kubectl install-kustomize install-kubectl ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

.PHONY: install-kubectl
install-kubectl: ## Download kubectl locally if necessary.
	@UNCOMPRESSED=true hack/uget.sh https://dl.k8s.io/release/{VERSION}/bin/{GOOS}/{GOARCH}/kubectl kubectl $(KUBECTL_VERSION) kubectl

.PHONY: install-kustomize
install-kustomize: ## Download kustomize locally if necessary.
	@GO_MODULE=true hack/uget.sh sigs.k8s.io/kustomize/kustomize/v5 kustomize $(KUSTOMIZE_VERSION)

.PHONY: install-golangci-lint
install-golangci-lint: ## Download golangci-lint locally if necessary.
	@hack/uget.sh https://github.com/golangci/golangci-lint/releases/download/v{VERSION}/golangci-lint-{VERSION}-{GOOS}-{GOARCH}.tar.gz golangci-lint $(GOLANGCI_LINT_VERSION)

.PHONY: install-protokol
install-protokol: ## Download protokol locally if necessary.
	@hack/uget.sh https://codeberg.org/xrstf/protokol/releases/download/v{VERSION}/protokol_{VERSION}_{GOOS}_{GOARCH}.tar.gz protokol $(PROTOKOL_VERSION)

.PHONY: install-reconciler-gen
install-reconciler-gen: ## Download reconciler-gen locally if necessary.
	@GO_MODULE=true hack/uget.sh k8c.io/reconciler/cmd/reconciler-gen reconciler-gen $(RECONCILER_GEN_VERSION)

.PHONY: install-applyconfiguration-gen
install-applyconfiguration-gen: ## Download applyconfiguration-gen locally if necessary.
	@GO_MODULE=true hack/uget.sh k8s.io/code-generator/cmd/applyconfiguration-gen applyconfiguration-gen $(APPLYCONFIGURATION_GEN_VERSION)

.PHONY: install-client-gen
install-client-gen: ## Download client-gen locally if necessary.
	@GO_MODULE=true hack/uget.sh k8s.io/code-generator/cmd/client-gen client-gen $(CLIENT_GEN_VERSION)

.PHONY: install-controller-gen
install-controller-gen: ## Download controller-gen locally if necessary.
	@GO_MODULE=true hack/uget.sh sigs.k8s.io/controller-tools/cmd/controller-gen controller-gen $(CONTROLLER_GEN_VERSION)

.PHONY: install-kcp-codegen
install-kcp-codegen: ## Download kcp code-generator locally if necessary.
	@GO_MODULE=true hack/uget.sh github.com/kcp-dev/code-generator/v2 kcp-code-generator $(KCP_CODEGEN_VERSION) code-generator

.PHONY: install-goimports
install-goimports: ## Download openshift goimports locally if necessary.
	@GO_MODULE=true hack/uget.sh github.com/openshift-eng/openshift-goimports goimports $(GOIMPORTS_VERSION)

.PHONY: install-helm
install-helm: ## Download Helm locally if necessary.
	@hack/uget.sh https://get.helm.sh/helm-v{VERSION}-{GOOS}-{GOARCH}.tar.gz helm $(HELM_VERSION)

# This target can be used to conveniently update the checksums for all checksummed tools.
# Combine with GOARCH to update for other archs, like "GOARCH=arm64 make update-tools".

.PHONY: update-tools
update-tools: UGET_UPDATE=true
update-tools: clean-tools install-kubectl install-golangci-lint install-protokol install-helm

##@ Documentation

VENVDIR=$(abspath docs/venv)
REQUIREMENTS_TXT=docs/requirements.txt

.PHONY: generate-api-docs
generate-api-docs: ## Generate api docs from CRDs.
	git clean -fdX docs/content/reference
	docs/generators/crd-ref/run-crd-ref-gen.sh

.PHONY: local-docs
local-docs: venv ## Serve documentation locally.
	. $(VENV)/activate; \
	VENV=$(VENV) cd docs && mkdocs serve

.PHONY: deploy-docs
deploy-docs: venv ## Deploy documentation (CI make target).
	. $(VENV)/activate; \
	REMOTE=$(REMOTE) BRANCH=$(BRANCH) docs/scripts/deploy-docs.sh

include Makefile.venv
