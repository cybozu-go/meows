CONTROLLER_GEN_VERSION := 0.18.0
ENVTEST_K8S_VERSION := 1.33.0

PROJECT_DIR := $(CURDIR)
TMP_DIR := $(PROJECT_DIR)/tmp
BIN_DIR := $(TMP_DIR)/bin
KINDTEST_DIR := $(PROJECT_DIR)/kindtest

CONTROLLER_GEN := $(BIN_DIR)/controller-gen
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
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION)
	GOBIN=$(BIN_DIR) go install honnef.co/go/tools/cmd/staticcheck@latest
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: clean
clean: ## Clean files
	rm -rf $(TMP_DIR)/*

##@ Build

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) crd paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) webhook paths="./..." output:stdout > config/controller/webhook.yaml
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./..." output:rbac:artifacts:config=config/controller

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: build
build: generate ## Build all binaries.
	go build -o $(BIN_DIR)/ -trimpath ./cmd/controller
	go build -o $(BIN_DIR)/ -trimpath ./cmd/entrypoint
	go build -o $(BIN_DIR)/ -trimpath ./cmd/job-started
	go build -o $(BIN_DIR)/ -trimpath ./cmd/slack-agent
	go build -o $(BIN_DIR)/ -trimpath ./cmd/meows

.PHONY: image
image: ## Build container images.
	docker build --target controller -t meows-controller:devel .
	docker build --target runner -t meows-runner:devel .

.PHONY: tag
tag: ## Tag container images.
	docker tag meows-controller:devel $(IMAGE_PREFIX)meows-controller:$(IMAGE_TAG)
	docker tag meows-runner:devel $(IMAGE_PREFIX)meows-runner:$(IMAGE_TAG)

.PHONY: push
push: ## Push container images.
	docker push $(IMAGE_PREFIX)meows-controller:$(IMAGE_TAG)
	docker push $(IMAGE_PREFIX)meows-runner:$(IMAGE_TAG)

##@ Test

.PHONY: lint
lint: ## Run gofmt, staticcheck and vet.
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	$(STATICCHECK) ./...
	go vet ./...

.PHONY: check-generate
check-generate: ## Generate manifests and code, and check if diff exists.
	$(MAKE) manifests
	$(MAKE) generate
	git diff --exit-code --name-only

.PHONY: test
test: ## Run unit tests.
	source <($(BIN_DIR)/setup-envtest use -p env $(ENVTEST_K8S_VERSION)) \
		&& go test -v -count=1 ./... -coverprofile $(TMP_DIR)/cover.out
