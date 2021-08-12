CURL := curl -sSLf

KUSTOMIZE_VERSION := 4.1.3
K8S_VERSION := 1.19.7
KIND_VERSION := 0.11.1
CERT_MANAGER_VERSION := 1.3.1

PROJECT_DIR := $(CURDIR)/../
BIN_DIR := $(PROJECT_DIR)/tmp/bin
LOGS_DIR := $(PROJECT_DIR)/tmp/logs
REPO_DIR := $(PROJECT_DIR)/tmp/repo

KIND := $(BIN_DIR)/kind
KUBECTL := $(BIN_DIR)/kubectl
KUSTOMIZE := $(BIN_DIR)/kustomize

# Set the shell used to bash for better error handling.
SHELL = /bin/bash
.SHELLFLAGS = -e -o pipefail -c

# kind envs
KIND_CLUSTER_NAME ?= kindtest-meows

.PHONY: all
all: help

##@ Basic
.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: setup
setup: ## Setup necessary tools.
	mkdir -p $(BIN_DIR)
	$(CURL) https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv$(KUSTOMIZE_VERSION)/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BIN_DIR)
	$(CURL) -o $(BIN_DIR)/kind https://kind.sigs.k8s.io/dl/v$(KIND_VERSION)/kind-linux-amd64 && chmod a+x $(BIN_DIR)/kind
	$(CURL) -o $(BIN_DIR)/kubectl https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl && chmod a+x $(BIN_DIR)/kubectl

.PHONY: clean
clean: ## Clean files
	@echo Nothing. Run \"make clean\" in the parent directory.

##@ Test

.PHONY: start
start: ## Start kind cluster.
	$(KIND) create cluster --image kindest/node:v$(K8S_VERSION) --name $(KIND_CLUSTER_NAME) --config kind.yaml --wait 1m
	$(MAKE) cert-manager
	$(MAKE) load

.PHONY: test
test: ## Run all test on kind.
	mkdir -p $(REPO_DIR)
	rm -rf $(REPO_DIR)/.git $(REPO_DIR)/.github
	. ${PROJECT_DIR}/.secret.env.sh; \
		KINDTEST=1 BIN_DIR=$(BIN_DIR) \
		TEST_REPO_WORK_DIR=$(REPO_DIR) \
		GITHUB_APP_PRIVATE_KEY_PATH=$(PROJECT_DIR)/.secret.private-key.pem \
		go test . -v -timeout=15m -ginkgo.v -ginkgo.failFast $(KINDTEST_OPT)

.PHONY: bootstrap
bootstrap: ## Run bootstrap test on kind.
	$(MAKE) test KINDTEST_OPT="-ginkgo.focus 'bootstrap'"

.PHONY: logs
logs:
	$(KIND) export logs --name=$(KIND_CLUSTER_NAME) $(LOGS_DIR)

.PHONY: stop
stop: ## Stop kind cluster
	$(KIND) delete cluster --name $(KIND_CLUSTER_NAME)
	-docker image rm meows-controller:kindtest
	-docker image rm meows-runner:kindtest

.PHONY: load
load: ## Load docker images onto kind cluster.
	$(MAKE) -C $(PROJECT_DIR) image tag IMAGE_TAG=kindtest
	$(KIND) load docker-image meows-controller:kindtest --name $(KIND_CLUSTER_NAME)
	$(KIND) load docker-image meows-runner:kindtest --name $(KIND_CLUSTER_NAME)

.PHONY: cert-manager
cert-manager:
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
