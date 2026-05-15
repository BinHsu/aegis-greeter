# AG-0002: log/slog over zap / zerolog

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

The service needs structured logging. `zap` and `zerolog` are the
common high-performance choices; `log/slog` has been in the standard
library since Go 1.21. A greeter at this scale has no measured logging
hot path.

## Decision

Use `log/slog` with its JSON handler. The application wraps that
handler (see AG-0008) to inject OpenTelemetry trace context. No
third-party logging dependency is added.

## Consequences

- Zero dependency surface for logging; stability tracks the standard
  library.
- The JSON handler emits flat records that the Grafana Loki pipeline
  ingests cleanly.
- `slog` is measurably slower than `zap` in microbenchmarks. There is
  no evidence this matters for this service. The decision is
  pain-driven: adopt a faster logger only when a profile shows logging
  on a hot path. Until then, the dependency is not worth its cost.
