# AG-0010: Go layout — cmd/greeter + internal/

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

The greeter began as a single flat `greeter.go`. The observability work
(OpenTelemetry SDK, Pyroscope, the trace-aware logging handler, the
custom instruments) pushed the codebase toward roughly 600 lines across
several distinct concerns. A single file no longer reads well.

## Decision

Adopt a `cmd/` + `internal/` layout:

- `cmd/greeter/main.go` — process wiring only.
- `internal/telemetry` — OpenTelemetry providers, Pyroscope, the slog
  trace-context handler.
- `internal/handlers` — the greeter and the health probes.
- `internal/metrics` — the custom business instruments.

This was chosen over keeping a flat root with multiple files, and over
staying single-file.

## Consequences

- Each concern has a clear home; the upcoming-feature diffs stay
  scoped to one package.
- `internal/` enforces, at compile time, that these packages are not
  importable from outside — the codebase declares itself a service,
  not a library.
- Cost: one extra directory level for a project with a single binary.
  Accepted — the layout signals production intent to a reviewer and
  leaves room for a second command without restructuring.

## Alternatives considered

- **Single flat `greeter.go`** — where the project started. Fine at
  ~30 lines; illegible at ~600 across telemetry, handlers, and
  metrics concerns.
- **Flat root, multiple files in `package main`** — splits the
  concerns without the directory depth, but loses the `internal/`
  compile-time boundary and reads as a script rather than a service.
- **The full `golang-standards/project-layout`** (`pkg/`, `api/`,
  `configs/`, …) — ceremony far beyond a single-binary service.

## Out of scope / when to revisit

- A `pkg/` directory — only if code here becomes genuinely reusable by
  another module. `internal/` deliberately forbids that today; the day
  there is a real external consumer is the day to reconsider.
