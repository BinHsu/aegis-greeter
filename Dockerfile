# syntax=docker/dockerfile:1.7

# --- Build stage ---
# Pinned to the multi-arch index digest fetched 2026-05-15. To refresh:
#   docker buildx imagetools inspect golang:1.26-alpine
FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

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
