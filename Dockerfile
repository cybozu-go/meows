FROM ghcr.io/cybozu/golang:1.27.1.1_noble@sha256:e38fe3b72f61d034394ee2c2592d41fa753226718bbc111bb6bb9a21601f0859 AS builder

WORKDIR /workspace
COPY . .
RUN make build

FROM ghcr.io/cybozu/ubuntu:24.04.20260902@sha256:182a16198fafecbce2f813a07de269d28efb885aa4843e611ce44c25ce46cc2b AS controller
LABEL org.opencontainers.image.source="https://github.com/cybozu-go/meows"

COPY --from=builder /workspace/tmp/bin/controller /usr/local/bin
COPY --from=builder /workspace/tmp/bin/slack-agent /usr/local/bin
COPY --from=builder /workspace/tmp/bin/meows /usr/local/bin

USER 10000:10000
ENTRYPOINT ["controller"]
