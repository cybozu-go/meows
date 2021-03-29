CURL := curl -sSLf

CONTROLLER_RUNTIME_VERSION := $(shell awk '/sigs\.k8s\.io\/controller-runtime/ {print substr($$2, 2)}' go.mod)
CONTROLLER_GEN_VERSION := 0.4.1
KUSTOMIZE_VERSION := 3.8.7
K8S_VERSION := 1.19.7
KIND_VERSION := 0.10.0
CERT_MANAGER_VERSION := 1.2.0

PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
BIN_DIR := $(PROJECT_DIR)/bin
ENVTEST_ASSETS_DIR := $(PROJECT_DIR)/testbin
E2E_DIR := $(PROJECT_DIR)/e2e

CONTROLLER_GEN := $(BIN_DIR)/controller-gen
KUSTOMIZE := $(BIN_DIR)/kustomize
NILERR := $(BIN_DIR)/nilerr
STATICCHECK := $(BIN_DIR)/staticcheck
GINKGO := $(BIN_DIR)/ginkgo
KIND := $(BIN_DIR)/kind
KUBECTL := $(BIN_DIR)/kubectl
KUSTOMIZE := $(BIN_DIR)/kustomize
export KUBECTL

# Set the shell used to bash for better error handling.
SHELL = /bin/bash
.SHELLFLAGS = -e -o pipefail -c

# Image URL to use all building/pushing image targets
CONTROLLER_IMG ?= controller:latest
RUNNER_IMG ?= runner:latest
AGENT_IMG ?= slack-agent:latest

# kind envs
KIND_CLUSTER_NAME ?= e2e-actions
KIND_CONFIG := $(E2E_DIR)/kind.yaml

GITHUB_APP_ID ?=
GITHUB_APP_INSTALLATION_ID ?=
GITHUB_APP_PRIVATE_KEY_PATH ?=

SLACK_WEBHOOK_URL ?=
SLACK_APP_TOKEN ?=
SLACK_BOT_TOKEN ?=

CRD_OPTIONS ?=

.PHONY: all
all: help

##@ Basic
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Setup necessary tools.
	mkdir -p $(ENVTEST_ASSETS_DIR)
	$(CURL) -o $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v$(CONTROLLER_RUNTIME_VERSION)/hack/setup-envtest.sh
	mkdir -p $(BIN_DIR)
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION)
	GOBIN=$(BIN_DIR) go install honnef.co/go/tools/cmd/staticcheck@latest
	GOBIN=$(BIN_DIR) go install github.com/gostaticanalysis/nilerr/cmd/nilerr@latest
	GOBIN=$(BIN_DIR) go install github.com/onsi/ginkgo/ginkgo@latest
	$(CURL) https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BIN_DIR)
	$(CURL) -o $(BIN_DIR)/kind https://kind.sigs.k8s.io/dl/v$(KIND_VERSION)/kind-linux-amd64 && chmod a+x $(BIN_DIR)/kind
	$(CURL) -o $(BIN_DIR)/kubectl https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl && chmod a+x $(BIN_DIR)/kubectl

.PHONY: clean
clean: ## Clean files>
	rm -f bin/*
	rm -rf testbin/*

##@ Build

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: build
build: generate ## Build manager binary.
	go build -o bin/github-actions-controller ./cmd/controller

.PHONY: build-agent
build-agent:
	go build -o bin/slack-agent ./cmd/slack-agent

.PHONY: build-annotator
build-annotator:
	go build -o bin/deltime-annotate ./cmd/deltime-annotate

.PHONY: run
run: manifests generate ## Run a controller from your host.
	go run ./cmd/controller

.PHONY: images
images: controller-image runner-image agent-image ## Build both container and runner docker images.

.PHONY: controller-image
controller-image:
	docker build -t ${CONTROLLER_IMG} .

.PHONY: runner-image
runner-image:
	docker build -f Dockerfile.runner -t ${RUNNER_IMG} .

.PHONY: agent-image
agent-image:
	docker build -f Dockerfile.agent -t ${AGENT_IMG} .

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
	{ \
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh && \
	fetch_envtest_tools $(notdir $(ENVTEST_ASSETS_DIR)) && \
	setup_envtest_env $(ENVTEST_ASSETS_DIR) && \
	go test -v -count=1 ./... -coverprofile cover.out ; \
	}

.PHONY: prepare
prepare: ## Prepare for e2e test.
	if [ -z "$${GITHUB_APP_ID}" ]; then \
	  echo "GITHUB_APP_ID must be set" 1>&2; \
	  exit 1; \
	fi
	if [ -z "$${GITHUB_APP_INSTALLATION_ID}" ]; then \
	  echo "GITHUB_APP_INSTALLATION_ID must be set" 1>&2; \
	  exit 1; \
	fi
	if [ -z "$${GITHUB_APP_PRIVATE_KEY_PATH}" ]; then \
	  echo "GITHUB_APP_PRIVATE_KEY_PATH must be set" 1>&2; \
	  exit 1; \
	fi
	if [ -z "$${SLACK_APP_TOKEN}" ]; then \
	  echo "SLACK_APP_TOKEN must be set" 1>&2; \
	  exit 1; \
	fi
	if [ -z "$${SLACK_BOT_TOKEN}" ]; then \
	  echo "SLACK_BOT_TOKEN must be set" 1>&2; \
	  exit 1; \
	fi
	if [ -z "$${SLACK_WEBHOOK_URL}" ]; then \
	  echo "SLACK_WEBHOOK_URL must be set" 1>&2; \
	  exit 1; \
	fi
	$(MAKE) start-kind
	$(MAKE) load
	$(KUBECTL) create ns actions-system
	$(KUBECTL) label ns default actions.cybozu.com/pod-mutate=true
	$(KUBECTL) create secret generic github-app-secret \
		-n actions-system \
		--from-literal=app-id=$(GITHUB_APP_ID) \
		--from-literal=app-installation-id=$(GITHUB_APP_INSTALLATION_ID) \
		--from-file=app-private-key=$(GITHUB_APP_PRIVATE_KEY_PATH)
	$(KUBECTL) create secret generic slack-app-secret \
		--from-literal=SLACK_WEBHOOK_URL=$(SLACK_WEBHOOK_URL) \
		--from-literal=SLACK_APP_TOKEN=$(SLACK_APP_TOKEN) \
		--from-literal=SLACK_BOT_TOKEN=$(SLACK_BOT_TOKEN)
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) wait pods -n cert-manager -l app=cert-manager --for=condition=Ready --timeout=1m
	$(KUBECTL) wait pods -n cert-manager -l app=cainjector --for=condition=Ready --timeout=1m
	$(KUBECTL) wait pods -n cert-manager -l app=webhook --for=condition=Ready --timeout=1m

.PHONY: e2e
e2e: ## Run e2e test.
	$(MAKE) install
	$(KUSTOMIZE) build --load_restrictor='none' $(E2E_DIR)/manifests | $(KUBECTL) apply -f -
	env E2ETEST=1 BIN_DIR=$(BIN_DIR) $(GINKGO) --failFast -v $(E2E_DIR)

##@ Deployment

.PHONY: start-kind
start-kind: ## Start kind cluster.
	$(KIND) create cluster --image kindest/node:v$(K8S_VERSION) --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG)

.PHONY: stop-kind
stop-kind: ## Stop kind cluster
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: load
load: load-controller-image load-runner-image load-agent-image ## Load docker images onto k8s cluster.

.PHONY: load-controller-image
load-controller-image:
	$(KIND) load docker-image $(CONTROLLER_IMG) --name $(KIND_CLUSTER_NAME)

.PHONY: load-runner-image
load-runner-image:
	$(KIND) load docker-image $(RUNNER_IMG) --name $(KIND_CLUSTER_NAME)

.PHONY: load-agent-image
load-agent-image:
	$(KIND) load docker-image $(AGENT_IMG) --name $(KIND_CLUSTER_NAME)

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${CONTROLLER_IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete -f -

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
