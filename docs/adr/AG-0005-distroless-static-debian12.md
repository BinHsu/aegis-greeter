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

## Alternatives considered

- **`alpine`** — small, but ships a shell, `apk`, and musl libc:
  exploitation surface and scanner noise the static binary does not
  need.
- **`scratch`** — even smaller than distroless static, but carries no
  CA certificates and no `/etc/passwd`. The greeter makes outbound TLS
  calls (OTLP) and runs as a named non-root user, so it needs both —
  distroless static provides exactly those and nothing more.
- **`debian-slim` / `ubuntu`** — a full userland for a static binary
  that uses none of it. Rejected on CVE surface.
- **`distroless/static-debian12:nonroot`** — the same image with the
  `nonroot` user preset. Functionally equivalent; `USER nonroot` is
  set explicitly in the Dockerfile instead, so the choice is visible
  in the build rather than inherited from a tag.

## Out of scope / when to revisit

- If the build ever needs `CGO_ENABLED=1` (a C dependency), the static
  base no longer suffices — `gcr.io/distroless/base` (with glibc)
  becomes the floor. No such dependency is on the horizon.
