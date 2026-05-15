# AG-0011: Go toolchain pin policy — newest stable

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

`go.mod` carries a `toolchain` directive that pins the Go compiler. The
initial plan pinned an older release for conservatism. During toolchain
setup, `go mod tidy` raised the `go` directive floor because a current
`golangci-lint` requires a recent Go in its own `go.mod`, and that
constraint propagates transitively to any consumer.

## Decision

Pin the toolchain to the **newest stable Go release**, not an older
"safe" version. The pin landed at `go1.26.3` — the version on the
development host — after the transitive bump and a deliberate choice to
prefer current tooling.

The same "newest stable" preference applies to dev tools, container
base images, and GitHub Actions: default to the latest stable, deviate
only for a concrete blocking reason.

## Consequences

- Modern Go and modern tooling, with the security and performance work
  that comes with them.
- A reviewer whose host Go is at or above the pin needs no toolchain
  download — `GOTOOLCHAIN=auto` only fetches when the host is behind.
- The pin must be bumped as new Go releases land. This is a deliberate,
  low-cost maintenance task, not drift — staying current is the goal.
