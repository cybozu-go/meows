FROM ghcr.io/cybozu/golang:1.26.3.1_noble@sha256:0da22bb6f9a876d774654892d411131272ae3dd14c530b6b4ad9598b0a74d1da AS builder

WORKDIR /workspace
COPY . .
RUN make build

FROM ghcr.io/cybozu/ubuntu:24.04.20260508@sha256:ab2735f6893fc167776587097de968c7d66c1cb052326d8cfeb3c4f8cd8bac00 AS controller
LABEL org.opencontainers.image.source="https://github.com/cybozu-go/meows"

COPY --from=builder /workspace/tmp/bin/controller /usr/local/bin
COPY --from=builder /workspace/tmp/bin/slack-agent /usr/local/bin
COPY --from=builder /workspace/tmp/bin/meows /usr/local/bin

USER 10000:10000
ENTRYPOINT ["controller"]
