

# For Go
GO111MODULE = on
export GO111MODULE

.PHONY: all
all: test

.PHONY: test
test: test-tools
	test -z "$$(gofmt -s -l . | tee /dev/stderr)"
	staticcheck ./...
	test -z "$$(nilerr ./... 2>&1 | tee /dev/stderr)"
	go install ./...
	go test -race -v ./...
	go vet ./...

.PHONY: test-tools
test-tools: staticcheck nilerr

.PHONY: staticcheck
staticcheck:
	if ! which staticcheck >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck; \
	fi

.PHONY: nilerr
nilerr:
	if ! which nilerr >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get github.com/gostaticanalysis/nilerr/cmd/nilerr; \
	fi
