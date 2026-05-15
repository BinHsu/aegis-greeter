# Project-local Go toolchain: every tool lands in ./bin, never in ~/go/bin.
# Host Go version is irrelevant — go.mod toolchain directive pins the compiler.

GOBIN := $(CURDIR)/bin
PATH  := $(GOBIN):$(PATH)
export GOBIN PATH

.PHONY: dev-setup vet test lint vuln build clean tidy

dev-setup:
	go install -tags tools \
	  github.com/golangci/golangci-lint/cmd/golangci-lint \
	  golang.org/x/vuln/cmd/govulncheck \
	  github.com/rhysd/actionlint/cmd/actionlint

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./... -race -count=1

lint: dev-setup
	$(GOBIN)/golangci-lint run ./...

vuln: dev-setup
	$(GOBIN)/govulncheck ./...

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o ./bin/greeter ./cmd/greeter

clean:
	rm -rf ./bin
