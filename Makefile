CURL := curl -sSLf

CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' go.mod)
CONTROLLER_GEN_VERSION := 0.4.1
K8S_VERSION := 1.19.7

PROJECT_DIR := $(CURDIR)
TMP_DIR := $(PROJECT_DIR)/tmp
BIN_DIR := $(TMP_DIR)/bin
ENVTEST_ASSETS_DIR := $(TMP_DIR)/envtest
KINDTEST_DIR := $(PROJECT_DIR)/kindtest

CONTROLLER_GEN := $(BIN_DIR)/controller-gen
NILERR := $(BIN_DIR)/nilerr
STATICCHECK := $(BIN_DIR)/staticcheck

CRD_OPTIONS ?=

IMAGE_PREFIX :=
IMAGE_TAG := latest

# Set the shell used to bash for better error handling.
SHELL = /bin/bash
.SHELLFLAGS = -e -o pipefail -c

.PHONY: all
all: help

##@ Basic
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Setup necessary tools.
	$(MAKE) -C kindtest setup
	mkdir -p $(BIN_DIR) $(ENVTEST_ASSETS_DIR)
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION)
	GOBIN=$(BIN_DIR) go install honnef.co/go/tools/cmd/staticcheck@latest
	GOBIN=$(BIN_DIR) go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest
	$(CURL) -o $(BIN_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v$(CONTROLLER_RUNTIME_VERSION)/hack/setup-envtest.sh
	source $(BIN_DIR)/setup-envtest.sh && fetch_envtest_tools $(ENVTEST_ASSETS_DIR)

.PHONY: clean
clean: ## Clean files
	rm -rf $(TMP_DIR)/*

##@ Build

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: build
build: generate ## Build all binaries.
	go build -o $(BIN_DIR)/github-actions-controller ./cmd/controller
	go build -o $(BIN_DIR)/ ./cmd/slack-agent
	go build -o $(BIN_DIR)/ ./cmd/slack-agent-client
	go build -o $(BIN_DIR)/ ./cmd/deltime-annotate
	go build -o $(BIN_DIR)/ ./cmd/job-started
	go build -o $(BIN_DIR)/ ./cmd/entrypoint

.PHONY: image
image: ## Build container images.
	docker build -t actions-controller:devel -f Dockerfile.controller  .
	docker build -t actions-runner:devel -f Dockerfile.runner .

.PHONY: tag
tag: ## Tag container images.
	docker tag actions-controller:devel $(IMAGE_PREFIX)actions-controller:$(IMAGE_TAG)
	docker tag actions-runner:devel $(IMAGE_PREFIX)actions-runner:$(IMAGE_TAG)

.PHONY: push
push: ## Push container images.
	docker push $(IMAGE_PREFIX)actions-controller:$(IMAGE_TAG)
	docker push $(IMAGE_PREFIX)actions-runner:$(IMAGE_TAG)

##@ Test

.PHONY: lint
lint: ## Run gofmt, staticcheck, nilerr and vet.
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	$(STATICCHECK) ./...
	test -z "$$($(NILERR) ./... 2>&1 | tee /dev/stderr)"
	go vet ./...

.PHONY: check-generate
check-generate: ## Generate manifests and code, and check if diff exists.
	$(MAKE) manifests
	$(MAKE) generate
	git diff --exit-code --name-only

.PHONY: test
test: ## Run unit tests.
	source $(BIN_DIR)/setup-envtest.sh \
		&& setup_envtest_env $(ENVTEST_ASSETS_DIR) \
		&& go test -v -count=1 ./... -coverprofile $(TMP_DIR)/cover.out
