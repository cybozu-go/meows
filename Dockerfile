# Build the manager binary
FROM quay.io/cybozu/golang:1.15-focal as builder

WORKDIR /workspace
COPY . .
RUN make build

FROM quay.io/cybozu/ubuntu:20.04
WORKDIR /
COPY --from=builder /workspace/bin/github-actions-controller .

USER 10000:10000
ENTRYPOINT ["/github-actions-controller"]
