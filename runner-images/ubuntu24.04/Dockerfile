FROM ghcr.io/cybozu/golang:1.27.3.2_noble@sha256:d78937fddb26c4350824d4022f5c9dfe20bf762811ea31065f8c6d467c58bc30 AS builder

WORKDIR /workspace

ENV GOPROXY=https://golang.flatt.tech
ENV NETRC=/run/secrets/netrc

COPY go.mod go.sum ./
RUN --mount=type=secret,id=netrc,target=/run/secrets/netrc,required=false \
    go mod download

COPY . .
RUN GOPROXY=off make build

FROM ghcr.io/cybozu/ubuntu:24.04.20261113@sha256:74243e15b3f04c67d11f86a43092d1e54f73ee8a1049f3175ea8b87273abc723 AS runner
LABEL org.opencontainers.image.source="https://github.com/cybozu-go/meows"

ARG RUNNER_VERSION
ARG RUNNER_SHA256

ENV DEBIAN_FRONTEND=noninteractive
# hadolint ignore=DL3015
RUN apt-get update -y \
  && apt-get install -y software-properties-common \
  && add-apt-repository -y ppa:git-core/ppa \
  && apt-get update -y \
  && apt-get install -y --no-install-recommends libyaml-dev \
  && rm -rf /var/lib/apt/lists/*

ARG RUNNER_ASSETS_DIR=/runner
RUN mkdir -p ${RUNNER_ASSETS_DIR} \
  && cd ${RUNNER_ASSETS_DIR} \
  && curl -L -O https://raw.githubusercontent.com/actions/runner/${RUNNER_VERSION}/LICENSE \
  && curl -L -o runner.tar.gz https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_VERSION}.tar.gz \
  && echo "${RUNNER_SHA256}  runner.tar.gz" | sha256sum -c --strict - \
  && tar xzf ./runner.tar.gz \
  && rm runner.tar.gz \
  && ./bin/installdependencies.sh \
  && chown -R 10000 ${RUNNER_ASSETS_DIR}

ENV AGENT_TOOLSDIRECTORY=/opt/hostedtoolcache
RUN mkdir -p ${AGENT_TOOLSDIRECTORY} \
  && chmod g+rwx ${AGENT_TOOLSDIRECTORY}

USER 10000
COPY scripts/job-cancelled /usr/local/bin
COPY scripts/job-failure   /usr/local/bin
COPY scripts/job-success   /usr/local/bin

COPY --from=builder /workspace/tmp/bin/meows /usr/local/bin
COPY --from=builder /workspace/tmp/bin/job-started /usr/local/bin
COPY --from=builder /workspace/tmp/bin/entrypoint /usr/local/bin

CMD ["/usr/local/bin/entrypoint"]
