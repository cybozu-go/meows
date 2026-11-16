FROM ghcr.io/cybozu/golang:1.27.3.2_noble@sha256:d78937fddb26c4350824d4022f5c9dfe20bf762811ea31065f8c6d467c58bc30 AS builder

WORKDIR /workspace

ENV GOPROXY=https://golang.flatt.tech
ENV NETRC=/run/secrets/netrc

COPY go.mod go.sum ./
RUN --mount=type=secret,id=netrc,target=/run/secrets/netrc,required=false \
    go mod download

COPY . .
RUN GOPROXY=off make build

FROM ghcr.io/cybozu/ubuntu:24.04.20261113@sha256:74243e15b3f04c67d11f86a43092d1e54f73ee8a1049f3175ea8b87273abc723 AS controller
LABEL org.opencontainers.image.source="https://github.com/cybozu-go/meows"

COPY --from=builder /workspace/tmp/bin/controller /usr/local/bin
COPY --from=builder /workspace/tmp/bin/slack-agent /usr/local/bin
COPY --from=builder /workspace/tmp/bin/meows /usr/local/bin

USER 10000:10000
ENTRYPOINT ["controller"]
