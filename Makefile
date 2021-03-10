CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' go.mod)
CONTROLLER_GEN_VERSION := 0.4.1

ENVTEST_ASSETS_DIR := testbin
ENVTEST_SCRIPT_URL := https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v$(CONTROLLER_RUNTIME_VERSION)/hack/setup-envtest.sh

# Set the shell used to bash for better error handling.
SHELL = /bin/bash
.SHELLFLAGS = -e -o pipefail -c

# Image URL to use all building/pushing image targets
IMG ?= controller:latest

CRD_OPTIONS ?=

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GO111MODULE = on
export GO111MODULE

.PHONY: all
all: help

##@ Basic
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Setup necessary tools
	if ! which staticcheck >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck; \
	fi
	if ! which nilerr >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gostaticanalysis/nilerr/cmd/nilerr; \
	fi

.PHONY: clean
clean: ## clean files
	rm -f bin/*
	rm -f testbin/*

##@ Test

.PHONY: lint
lint: ## Run gofmt, staticcheck, nilerr and vet
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	go vet ./...

.PHONY: check-generate
check-generate: ## Generate manifests and code, and check if diff exists
	$(MAKE) manifests
	$(MAKE) generate
	git diff --exit-code --name-only

.PHONY: test
test: ## Set up envtest if not done already, and run tests.
ifeq (,$(wildcard $(ENVTEST_ASSETS_DIR)/setup-envtest.sh))
	mkdir -p $(ENVTEST_ASSETS_DIR)
	curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh $(ENVTEST_SCRIPT_URL)
endif
	{ \
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh && \
	fetch_envtest_tools $(ENVTEST_ASSETS_DIR) && \
	setup_envtest_env $(PWD)/$(ENVTEST_ASSETS_DIR) && \
	go test ./... -coverprofile cover.out ; \
	}
	{ \
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh && \
	fetch_envtest_tools $(ENVTEST_ASSETS_DIR) && \
	setup_envtest_env $(PWD)/$(ENVTEST_ASSETS_DIR) && \
	TEST_PERMISSIVE=true go test -v -count 1 ./... ; \
	}

##@ Build

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: build
build: generate ## Build manager binary.
	go build -o bin/github-actions-controller main.go

.PHONY: run
run: manifests generate ## Run a controller from your host.
	go run ./main.go

.PHONY: build-controller-image
build-cotroller-image: test ## Build docker image with the controller.
	docker build -t ${IMG} .

.PHONY: build-runner-image
build-runner-image: test ## Build docker image with the runner.
	docker build -t ${IMG} .

##@ Deployment

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


.PHONY: controller-gen
CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION))

.PHONY: kustomize
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
