FROM ghcr.io/cybozu/golang:1.27.1.1_jammy@sha256:cf6a363d2a421e18a4e0038b69f0b44061903a98ab4c58ea46a68c271e81b327 AS builder

WORKDIR /workspace
COPY . .
RUN make build

FROM ghcr.io/cybozu/ubuntu:24.04.20260902@sha256:182a16198fafecbce2f813a07de269d28efb885aa4843e611ce44c25ce46cc2b AS runner
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
