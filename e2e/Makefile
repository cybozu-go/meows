K8S_VERSION := 1.19.7
KIND_VERSION := 0.10.0
KUSTOMIZE_VERSION := 3.7.0

KIND_NODE_IMAGE := kindest/node

BINDIR := $(abspath $(PWD)/bin)
KIND := $(BINDIR)/kind
KUBECTL := $(BINDIR)/kubectl
KUSTOMIZE := $(BINDIR)/kustomize
export KUBECTL

KIND_CONFIG := ./config/kind.yaml
CLUSTER_NAME := e2e-actions

.PHONY: start
start:
	$(KIND) create cluster --image $(KIND_NODE_IMAGE):v$(K8S_VERSION) --name $(CLUSTER_NAME) --config $(KIND_CONFIG)

.PHONY: stop
stop:
	$(KIND) delete cluster --name $(CLUSTER_NAME)

.PHONY: load-images
load-images:
	echo "not implemented"

.PHONY: test
test:
	go test -count 1 -v . -args -ginkgo.progress -ginkgo.v

.PHONY: setup
setup:
	mkdir -p $(BINDIR)
	curl -o $(BINDIR)/kind -sfL https://kind.sigs.k8s.io/dl/v$(KIND_VERSION)/kind-linux-amd64
	curl -o $(BINDIR)/kubectl -sfL https://storage.googleapis.com/kubernetes-release/release/v$(K8S_VERSION)/bin/linux/amd64/kubectl
	curl -sfL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.7.0/kustomize_v$(KUSTOMIZE_VERSION)_linux_amd64.tar.gz | tar -xz -C $(BINDIR)
	chmod a+x $(BINDIR)/kubectl $(BINDIR)/kind
