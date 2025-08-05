# Maintenance

## How to update supported Kubernetes

Meows supports the three latest Kubernetes versions.
If a new Kubernetes version is released, please update the followings:

### 1. Update supported kubernetes and dependencies versions

- Kubernetes versions: You can check the versions at <https://hub.docker.com/r/kindest/node/tags>.
  - `k8s-version` in [.github/workflows/main.yaml](/.github/workflows/main.yaml)
  - "Supported software" in [README.md](/README.md)
- Tools versions:
  - Update `CONTROLLER_GEN_VERSION` in [Makefile](/Makefile) to the latest version from <https://github.com/kubernetes-sigs/controller-tools/releases>.
  - Update `RUNNER_VERSION` in [runner-images/RUNNER_VERSION](/runner-images/RUNNER_VERSION) to the latest version from <https://github.com/actions/runner/releases>.
  - In [kindtest/Makefile](/kindtest/Makefile):
    - Update `KINDTEST_IMAGE_REF` to the latest supported version of [kindest/node](https://hub.docker.com/r/kindest/node/tags) tag and digest.
    - Update `KUSTOMIZE_VERSION` to the latest version from <https://github.com/kubernetes-sigs/kustomize/releases>.
    - Update `KIND_VERSION` to the latest version from <https://github.com/kubernetes-sigs/kind/releases>.
    - Update `CERT_MANAGER_VERSION` to the latest version from <https://github.com/cert-manager/cert-manager/releases>.
- After saving the changes above, update `ENVTEST_K8S_VERSION` in [Makefile](/Makefile) to the latest patch version among the latest supported kubernetes minor versions listed by running `make setup && tmp/bin/setup-envtest list` at the root of this repository. If the latest minor supported version is `1.30.Z`, find `1.30.Z+` from the output but not `1.31.Z`.
- Other dependencies versions:
  - Update `ghcr.io/cybozu/golang` image in [Dockerfile](/Dockerfile) to the latest version from <https://github.com/cybozu/neco-containers/pkgs/container/golang>.
- `go.mod` and `go.sum`:
  - Run [update-gomod](https://github.com/masa213f/tools/tree/main/cmd/update-gomod).

If Kubernetes or controller-runtime API has changed, please update the relevant source code accordingly.

### 2. Update meows by running `make`

You can update meows by running the following `make` commands:

```sh
make setup
make manifests
make build
```

### 3. Fix test code if tests fail

After pushing the change, if the CI fails, fix the tests and push the changes again.

_e.g._, <https://github.com/cybozu-go/meows/pull/185>

### 4. Release the new version

After merging the changes above, follow the procedures written in [Release.md](/RELEASE.md) and release the new version.

_e.g._, <https://github.com/cybozu-go/meows/pull/186>
