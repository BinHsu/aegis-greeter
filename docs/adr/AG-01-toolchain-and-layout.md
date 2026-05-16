# AG-01: Toolchain and project layout

- **Status**: Accepted
- **Date**: 2026-05-16

## Context

A Go service should build the same way on every machine — a reviewer's
laptop, a teammate's, a CI runner — without depending on what is already
installed there and without leaving anything behind on it. Two facets
decide this: the toolchain (the compiler and the development tools) and
the layout of the code itself.

## Decisions

### 1. The toolchain is project-local

Every tool the project uses lives inside the project or in a
project-scoped cache; nothing installs into the host `~/go/bin`,
`/usr/local`, or shell profile.

- `go.mod` carries a `toolchain` directive. With `GOTOOLCHAIN=auto`
  (Go's default since 1.21) the pinned compiler is fetched on demand,
  so the host Go version is irrelevant.
- `tools.go`, behind a `//go:build tools` constraint, imports the dev
  tools (`golangci-lint`, `govulncheck`, `actionlint`) so their
  versions are tracked in `go.mod` / `go.sum` and hash-verified —
  without compiling them into the production binary.
- The `Makefile` exports `GOBIN=$PWD/bin`; `make dev-setup` installs
  those tools into `./bin/` (gitignored).

### 2. Pin the toolchain to the newest stable Go

The `toolchain` directive pins a specific release, and that release is
the newest stable Go — not an older "conservative" one. The same
"newest stable" preference governs the dev tools, the container base
image, and the GitHub Actions: default to current, deviate only for a
concrete blocking reason. A reviewer whose host Go is at or above the
pin needs no download; `GOTOOLCHAIN=auto` fetches only when the host is
behind.

### 3. cmd/ + internal/ layout

The binary lives at `cmd/greeter/main.go` (process wiring only); the
logic lives under `internal/`, split by concern — `internal/telemetry`,
`internal/handlers`, `internal/metrics`. The `internal/` boundary is
compiler-enforced: nothing here is importable from outside the module,
which is the codebase declaring itself a service, not a library.

## Consequences

- A reviewer runs `make dev-setup && make test` with no conflict
  against their existing toolchain, and `make image` needs no host Go
  at all.
- Tool and compiler versions are pinned and hash-verified — "passes on
  my machine" and "passes in CI" become the same statement.
- Each concern has one home, so a feature change stays a scoped diff.
- Cost: `./bin/` tools build per machine (the first `golangci-lint`
  build from source is slow, then cached); the pinned toolchain must
  be bumped as new Go releases land — a deliberate, reviewed one-liner,
  not drift.

## Alternatives considered

- **`nix flake` / `devbox` / `mise`** — solve host isolation more
  completely, but add a non-Go dependency and a second lockfile for a
  project whose only dev tools are Go binaries.
- **Commit the tool binaries into the repo** — zero reviewer setup,
  but bloats the repo, is not multi-arch, and the binaries go stale
  silently.
- **No `toolchain` directive — build with whatever host Go is present**
  — reproducibility evaporates; two Go versions can produce two
  behaviours.
- **Pin an older "safe" Go** — neither safe nor stable, just behind;
  and a current `golangci-lint` raises the floor anyway.
- **A single flat `greeter.go`, or a flat root of files in
  `package main`** — fine at thirty lines, illegible at six hundred,
  and the flat form loses the `internal/` compile-time boundary.
- **The full `golang-standards/project-layout`** (`pkg/`, `api/`, …) —
  ceremony far beyond a single-binary service.

## Out of scope / when to revisit

- A non-Go tool that is not `go install`-able — `hadolint` already
  runs via a pinned Docker image; if more such tools appear, reassess
  whether a general-purpose version manager earns its keep.
- A `pkg/` directory — only if code here becomes genuinely reusable by
  another module; `internal/` deliberately forbids that today.
- Automating the toolchain bump with a dependency bot — worth wiring
  if the pin starts lagging behind the newest stable.
