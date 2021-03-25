# Build the manager binary
FROM quay.io/cybozu/golang:1.16-focal as builder

WORKDIR /workspace
COPY . .
RUN make build

FROM quay.io/cybozu/ubuntu:20.04

COPY --from=builder /workspace/bin/github-actions-controller /usr/local/bin

USER 10000:10000
ENTRYPOINT ["github-actions-controller"]
