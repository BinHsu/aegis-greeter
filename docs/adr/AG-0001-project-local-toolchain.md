# AG-0001: Project-local toolchain via tools.go + GOBIN + Makefile

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

The project depends on developer tools — `golangci-lint`, `govulncheck`,
`actionlint` — that are not part of the production binary. Installing
them globally (`brew install`, `go install ...@latest` into `~/go/bin`)
pollutes the host, drifts between machines, and conflicts with whatever
versions a reviewer already has. A reviewer should be able to clone the
repo and run it without it touching their existing toolchain.

## Decision

Every tool, the compiler included, lives inside the project:

- `go.mod` declares a `toolchain` directive. With `GOTOOLCHAIN=auto`
  (Go's default since 1.21) the pinned compiler is fetched on demand;
  the host Go version is irrelevant.
- `tools.go`, behind a `//go:build tools` constraint, imports the dev
  tools so their versions are tracked in `go.mod` / `go.sum` and
  hash-verified — without compiling them into the production binary.
- The `Makefile` exports `GOBIN=$PWD/bin`, so `make dev-setup` installs
  tools into `./bin/` (gitignored), never `~/go/bin`.

This was chosen over `nix flake` / `devbox` / `mise`: those solve the
same problem but add a non-Go dependency for a project whose isolation
needs are fully met by Go-native primitives.

## Consequences

- A reviewer runs `make dev-setup && make test` with no host conflict.
- Tool versions are pinned and hash-verified in `go.sum`.
- Container builds need no host Go at all (`make image`).
- Cost: `./bin/` tools are rebuilt per machine, and the first
  `golangci-lint` build from source is slow (mitigated by the Go build
  cache on repeat runs).
