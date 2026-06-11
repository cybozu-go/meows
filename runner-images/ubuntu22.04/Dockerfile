FROM ghcr.io/cybozu/golang:1.26.4.1_jammy@sha256:4273ab54d46bc2018b65785354589f6af6995c42b71c290b702035f2defa088d AS builder

WORKDIR /workspace
COPY . .
RUN make build

# Please update after confirming that placemat supports the relevant Ubuntu version.
FROM ghcr.io/cybozu/ubuntu:22.04.20260608@sha256:a796680ed45f1dd02abef0da037ad7a212a5781f300745ae6fffe4eeacff497e AS runner
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
