KINDTEST_IMAGE_REF := "v1.35.0@sha256:452d707d4862f52530247495d180205e029056831160e22870e37e3f6c1ac31f"
CERT_MANAGER_VERSION := 1.20.2
CERT_MANAGER_GPG_KEY_URL := https://cert-manager.io/public-keys/cert-manager-keyring-2021-09-20-1020CF3C033D4F35BAE1C19E1226061C665DF13E.gpg
KUBECTL_EVICT_COMMIT_SHA := 46aa0fb4296b93a92cda27f405f84677976b897e

PROJECT_DIR := $(CURDIR)/../
BIN_DIR := $(PROJECT_DIR)/tmp/bin
LOGS_DIR := $(PROJECT_DIR)/tmp/logs
REPO_DIR := $(PROJECT_DIR)/tmp/repo
CHART_DIR := $(PROJECT_DIR)/tmp/charts
# Default RUNNER for kindtest
RUNNER_IMAGE_DIR := $(PROJECT_DIR)/runner-images/ubuntu22.04
RUNNER_VERSION := $(shell cat $(PROJECT_DIR)/runner-images/RUNNER_VERSION)

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
	GOBIN=$(BIN_DIR) go install github.com/ueokande/kubectl-evict@$(KUBECTL_EVICT_COMMIT_SHA)

.PHONY: clean
clean: ## Clean files
	@echo Nothing. Run \"make clean\" in the parent directory.

##@ Test

$(CHART_DIR):
	mkdir -p $(CHART_DIR)

$(CHART_DIR)/cert-manager-keyring.gpg: | $(CHART_DIR)
	curl -sSLf -o $@ $(CERT_MANAGER_GPG_KEY_URL)

$(CHART_DIR)/cert-manager/Chart.yaml: $(CHART_DIR)/cert-manager-keyring.gpg
	helm pull oci://quay.io/jetstack/charts/cert-manager --version v$(CERT_MANAGER_VERSION) --verify --keyring $(CHART_DIR)/cert-manager-keyring.gpg --untar --untardir $(CHART_DIR)

.PHONY: start
start: ## Start kind cluster.
	kind create cluster --image kindest/node:$(KINDTEST_IMAGE_REF) --name $(KIND_CLUSTER_NAME) --config kind.yaml --wait 1m
	$(MAKE) cert-manager
	$(MAKE) load

.PHONY: test
test: ## Run all test on kind.
	mkdir -p $(REPO_DIR)
	rm -rf $(REPO_DIR)/.git $(REPO_DIR)/.github
	. ${PROJECT_DIR}/.secret.env.sh; \
		PATH=$(BIN_DIR):${PATH} \
		KINDTEST=1 \
		TEST_REPO_WORK_DIR=$(REPO_DIR) \
		GITHUB_APP_PRIVATE_KEY_PATH=$(PROJECT_DIR)/.secret.private-key.pem \
		go test . -v -timeout=15m -ginkgo.v --ginkgo.fail-fast $(KINDTEST_OPT)

.PHONY: bootstrap
bootstrap: ## Run bootstrap test on kind.
	$(MAKE) test KINDTEST_OPT="-ginkgo.focus 'bootstrap'"

.PHONY: logs
logs:
	kind export logs --name=$(KIND_CLUSTER_NAME) $(LOGS_DIR)

.PHONY: stop
stop: ## Stop kind cluster
	kind delete cluster --name $(KIND_CLUSTER_NAME)
	-docker image rm meows-controller:kindtest
	-docker image rm meows-runner:kindtest

.PHONY: load
load: ## Load docker images onto kind cluster.
	$(MAKE) -C $(PROJECT_DIR) image tag IMAGE_TAG=kindtest
	docker build --build-arg RUNNER_VERSION=$(RUNNER_VERSION) -f $(RUNNER_IMAGE_DIR)/Dockerfile -t meows-runner:kindtest $(PROJECT_DIR)
	kind load docker-image meows-controller:kindtest --name $(KIND_CLUSTER_NAME)
	kind load docker-image meows-runner:kindtest --name $(KIND_CLUSTER_NAME)

.PHONY: cert-manager
cert-manager: $(CHART_DIR)/cert-manager/Chart.yaml
	helm install cert-manager $(CHART_DIR)/cert-manager/ -n cert-manager --create-namespace --set crds.enabled=true
	kubectl -n cert-manager wait --for=condition=available --timeout=180s --all deployments
