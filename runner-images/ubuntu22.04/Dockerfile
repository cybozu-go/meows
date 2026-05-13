FROM ghcr.io/cybozu/golang:1.26.3.1_jammy@sha256:5286ca13634ae4217fe033c1564c5c4751e7f8979b81f017306db7a6a2544791 AS builder

WORKDIR /workspace
COPY . .
RUN make build

# Please update after confirming that placemat supports the relevant Ubuntu version.
FROM ghcr.io/cybozu/ubuntu:22.04.20260508@sha256:bcab2b64f0bae142d2bb0d9bda83a90dd457aa40e2b487e02f083d0f2747e503 AS runner
LABEL org.opencontainers.image.source="https://github.com/cybozu-go/meows"

ARG RUNNER_VERSION
ARG RUNNER_SHA

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
  && echo "${RUNNER_SHA}  runner.tar.gz" | sha256sum -c - \
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
