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

## Alternatives considered

- **`nix flake` / `devbox` / `mise`** — solve host isolation more
  completely (they pin non-Go tools too), but add a non-Go dependency
  and a second lockfile for a project whose only dev tools are Go
  binaries.
- **Commit the tool binaries into the repo** — zero setup for a
  reviewer, but bloats the repo, is not multi-arch, and the binaries
  go stale silently.
- **Document the versions and let each developer install them** — no
  enforcement; versions drift between machines, which is the exact
  problem this ADR exists to prevent.

## Out of scope / when to revisit

- Prebuilt-binary or devcontainer distribution of the dev tools —
  revisit if `make dev-setup` build-from-source time becomes real
  friction (today it is a one-time cost the Go build cache absorbs).
- Pinning a non-Go tool — `hadolint` already runs via a pinned Docker
  image rather than a host install; if more such tools appear,
  reassess whether a general-purpose version manager earns its keep.
