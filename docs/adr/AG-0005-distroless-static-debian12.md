# AG-0005: Distroless static-debian12 runtime base

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

The runtime container needs a base image. `alpine` is small but ships
a shell, a package manager, and musl libc — surface that an attacker
can use and that a vulnerability scanner must track. The greeter binary
is built `CGO_ENABLED=0`, so it is fully static and needs no libc at
all.

## Decision

The runtime stage is `gcr.io/distroless/static-debian12`, pinned to a
multi-arch index digest. It contains the binary, CA certificates, and
`/etc/passwd` — no shell, no package manager, no libc. The container
runs as `USER nonroot:nonroot`.

## Consequences

- Minimal CVE surface; no shell means no shell-based exploitation or
  `kubectl exec` foothold.
- Final image is roughly 7 MB.
- No shell for in-container debugging. When debugging is genuinely
  needed, use the `:debug` distroless variant or a Kubernetes ephemeral
  container — the production image stays minimal.
