# OPENSHIFT_GIMPORTS_VER defines which version of openshift-goimports to use
# for checking import statements.
OPENSHIFT_GOIMPORTS_VER := c72f1dc2e3aacfa00aece3391d938c9bc734e791
RECONCILER_GEN_VER := v0.5.0
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.31.0
## Tool Versions
KUBECTL_VERSION ?= v1.32.0
KUSTOMIZE_VERSION ?= v5.4.3
CONTROLLER_TOOLS_VERSION ?= v0.16.1
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= 2.1.6
PROTOKOL_VERSION ?= 0.7.2

# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/kcp-dev/kcp-operator

TOOLS_DIR = $(shell pwd)/_tools

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

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
codegen: reconciler-gen openshift-goimports ## Generate manifest, code and the SDK.
	@hack/update-codegen.sh

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(TOOLS_DIR) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# Utilize Kind or modify the e2e tests to load the image locally, enabling compatibility with other vendors.
.PHONY: test-e2e  # Run the e2e tests against a Kind k8s instance that is spun up.
test-e2e:
	go test ./test/e2e/ -v -ginkgo.v

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter.
	$(GOLANGCI_LINT) run --timeout 10m

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes.
	$(GOLANGCI_LINT) run --timeout 10m --fix

.PHONY: modules
modules: ## Run go mod tidy to ensure modules are up to date.
	hack/update-go-modules.sh

.PHONY: imports
imports: openshift-goimports ## Re-order Go import statements.
	$(OPENSHIFT_GOIMPORTS) -m github.com/kcp-dev/kcp-operator

.PHONY: verify
verify: codegen fmt vet modules imports ## Run all codegen and formatting targets and check if files have changed.
	if ! git diff --quiet --exit-code ; then echo "ERROR: Found unexpected changes to git repository"; git diff; exit 1; fi

##@ Build

.PHONY: clean
clean: ## Remove all built binaries.
	rm -rf _build

.PHONY: build
build: ## Build manager binary.
	go build -o _build/manager cmd/main.go

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

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: kubectl kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: kubectl kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: kubectl kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kubectl kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Tool Binaries
KUBECTL ?= $(TOOLS_DIR)/kubectl
KUSTOMIZE ?= $(TOOLS_DIR)/kustomize
ENVTEST ?= $(TOOLS_DIR)/setup-envtest
GOLANGCI_LINT = $(TOOLS_DIR)/golangci-lint
PROTOKOL = $(TOOLS_DIR)/protokol
RECONCILER_GEN := $(TOOLS_DIR)/reconciler-gen
OPENSHIFT_GOIMPORTS := $(TOOLS_DIR)/openshift-goimports

.PHONY: kubectl
kubectl: $(KUBECTL) ## Download kubectl locally if necessary.

.PHONY: $(KUBECTL)
$(KUBECTL):
	@UNCOMPRESSED=true hack/download-tool.sh https://dl.k8s.io/$(KUBECTL_VERSION)/bin/$(shell go env GOOS)/$(shell go env GOARCH)/kubectl kubectl $(KUBECTL_VERSION) kubectl

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.

.PHONY: $(KUSTOMIZE)
$(KUSTOMIZE):
	@GO_MODULE=true hack/download-tool.sh sigs.k8s.io/kustomize/kustomize/v5 kustomize $(KUSTOMIZE_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.

.PHONY: $(ENVTEST)
$(ENVTEST):
	@GO_MODULE=true hack/download-tool.sh sigs.k8s.io/controller-runtime/tools/setup-envtest setup-envtest $(ENVTEST_VERSION)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.

.PHONY: $(GOLANGCI_LINT)
$(GOLANGCI_LINT):
	@hack/download-tool.sh https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-$(shell go env GOOS)-$(shell go env GOARCH).tar.gz golangci-lint $(GOLANGCI_LINT_VERSION)

.PHONY: protokol
protokol: $(PROTOKOL) ## Download protokol locally if necessary.

.PHONY: $(PROTOKOL)
$(PROTOKOL):
	@hack/download-tool.sh https://codeberg.org/xrstf/protokol/releases/download/v${PROTOKOL_VERSION}/protokol_${PROTOKOL_VERSION}_$(shell go env GOOS)_$(shell go env GOARCH).tar.gz protokol $(PROTOKOL_VERSION)

.PHONY: reconciler-gen
reconciler-gen: $(RECONCILER_GEN) ## Download reconciler-gen locally if necessary.

.PHONY: $(RECONCILER_GEN)
$(RECONCILER_GEN):
	@GO_MODULE=true hack/download-tool.sh k8c.io/reconciler/cmd/reconciler-gen reconciler-gen $(RECONCILER_GEN_VER)

.PHONY: openshift-goimports
openshift-goimports: $(OPENSHIFT_GOIMPORTS) ## Download openshift-goimports locally if necessary.

.PHONY: $(OPENSHIFT_GOIMPORTS)
$(OPENSHIFT_GOIMPORTS):
	@GO_MODULE=true hack/download-tool.sh github.com/openshift-eng/openshift-goimports openshift-goimports $(OPENSHIFT_GOIMPORTS_VER)
