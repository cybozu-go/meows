FROM ghcr.io/cybozu/golang:1.25-jammy AS builder

WORKDIR /workspace
COPY . .
RUN make build

FROM ghcr.io/cybozu/ubuntu:22.04 AS controller
LABEL org.opencontainers.image.source="https://github.com/cybozu-go/meows"

COPY --from=builder /workspace/tmp/bin/controller /usr/local/bin
COPY --from=builder /workspace/tmp/bin/slack-agent /usr/local/bin
COPY --from=builder /workspace/tmp/bin/meows /usr/local/bin

USER 10000:10000
ENTRYPOINT ["controller"]
