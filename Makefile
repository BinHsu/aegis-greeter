# Project-local Go toolchain: every tool lands in ./bin, never in ~/go/bin.
# Host Go version is irrelevant — go.mod toolchain directive pins the compiler.

GOBIN := $(CURDIR)/bin
PATH  := $(GOBIN):$(PATH)
export GOBIN PATH

.PHONY: dev-setup tidy vet test lint vuln secrets build image hadolint \
        actionlint precommit prepush hooks-install clean

# Container image identity. IMAGE_TAG defaults to the 7-char git short
# SHA so locally-built images are traceable to a commit. VERSION and
# COMMIT are baked into the binary via -ldflags so /greeter knows its
# own provenance — see internal/metrics greeter_build_info.
IMAGE_NAME := aegis-greeter
IMAGE_TAG  := $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse HEAD 2>/dev/null || echo unknown)

# Hadolint image pinned to multi-arch index digest fetched 2026-05-15.
# Refresh via: docker buildx imagetools inspect hadolint/hadolint:latest-alpine
HADOLINT_IMAGE := hadolint/hadolint:latest-alpine@sha256:7aba693c1442eb31c0b015c129697cb3b6cb7da589d85c7562f9deb435a6657c

dev-setup:
	go install -tags tools \
	  github.com/golangci/golangci-lint/cmd/golangci-lint \
	  golang.org/x/vuln/cmd/govulncheck \
	  github.com/rhysd/actionlint/cmd/actionlint \
	  github.com/zricethezav/gitleaks/v8

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

# secrets — scan the full git history for committed credentials.
secrets: dev-setup
	$(GOBIN)/gitleaks git --redact --no-banner .

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o ./bin/greeter ./cmd/greeter

image:
	docker buildx build \
	  --platform linux/amd64 \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --tag $(IMAGE_NAME):$(IMAGE_TAG) \
	  --tag $(IMAGE_NAME):latest \
	  --load \
	  .

hadolint:
	docker run --rm -i $(HADOLINT_IMAGE) < Dockerfile

actionlint: dev-setup
	$(GOBIN)/actionlint

# precommit — fast local gate (pre-commit hook target). gofmt + vet +
# build only, finishes in seconds.
precommit:
	@unformatted=$$(gofmt -l .); \
	  if [ -n "$$unformatted" ]; then \
	    echo "FAIL gofmt — run 'gofmt -w .' on:"; echo "$$unformatted"; \
	    exit 1; \
	  fi
	go vet ./...
	go build ./...

# prepush — comprehensive local gate (pre-push hook target). Reuses the
# same targets CI runs so "passes locally" predicts "passes CI".
prepush: precommit test lint vuln secrets actionlint hadolint
	@echo "prepush gate clean — safe to push"

# hooks-install — point git at the committed .githooks directory. Run
# once per clone. Idempotent.
hooks-install:
	git config core.hooksPath .githooks
	@echo "git hooks active: core.hooksPath -> .githooks"

clean:
	rm -rf ./bin
