FROM ghcr.io/cybozu/golang:1.26.4.1_noble@sha256:add9d704d4b75df2c51328615be89b61a3e71e4833321aa02c3f325a30d3eb8f AS builder

WORKDIR /workspace
COPY . .
RUN make build

# Please update after confirming that placemat supports the relevant Ubuntu version.
FROM ghcr.io/cybozu/ubuntu:24.04.20260608@sha256:2137d223a483a2870dae87054a21314a69c6d8b9583a9a4ab25ea6e87b178b4a AS runner
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
