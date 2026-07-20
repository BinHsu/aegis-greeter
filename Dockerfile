# syntax=docker/dockerfile:1.7

# --- Build stage ---
# Pinned to the multi-arch index digest fetched 2026-07-21 (golang:1.26.5-alpine3.24)
# — bumped from 1.26.4 to clear GO-2026-5856 (crypto/tls stdlib CVE).
# To refresh:
#   docker buildx imagetools inspect golang:1.26-alpine
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

WORKDIR /src

# Module metadata first so the dependency layer is cached independently
# of the source. BuildKit cache mounts persist the module + build cache
# across builds outside the image layers — much faster iteration with
# no impact on the final image size.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY cmd ./cmd
COPY internal ./internal

# Build-time identity. Populated by CI from git state; falls back to
# defaults for hand-run docker build.
ARG VERSION=dev
ARG COMMIT=unknown

# CGO off → fully static binary that runs on a distroless image with
# no libc. -ldflags strips debug info + embeds VERSION/COMMIT. -trimpath
# removes local paths from the binary for reproducibility.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
      -trimpath \
      -o /out/greeter \
      ./cmd/greeter

# --- Runtime stage ---
# Distroless static — no shell, no package manager, no libc. Multi-arch
# index pinned to the digest fetched 2026-05-15. Same refresh command
# pattern as the builder stage above.
FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838

COPY --from=builder /out/greeter /greeter

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/greeter"]
