# AG-0009: OpenTelemetry SDK + pyroscope-go adoption

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

Cross-repo issue #1 pinned the observability stack to Grafana Cloud
(Mimir / Loki / Tempo / Pyroscope) with a Grafana Alloy in-cluster
collector. The application is the missing piece — without SDK-level
instrumentation the collector has no application traces, metrics, or
profiles to forward.

## Decision

Adopt the **OpenTelemetry Go SDK** for traces and metrics over OTLP
gRPC, plus **`github.com/grafana/pyroscope-go`** for continuous
profiling:

- `otelhttp.NewHandler` wraps the greeter and auto-emits the standard
  HTTP RED metrics and one span per request.
- Two custom business instruments are hand-written:
  `greeter_responses_total{personalized}` and `greeter_build_info`.
- Pyroscope captures CPU, alloc-objects, and goroutine profiles.

`prometheus/client_golang` is deliberately **not** added alongside
OTel — a dual metrics stack causes cardinality and dashboard
confusion. The OTel `MeterProvider` is the single metrics interface.

Every exporter is fail-soft: an empty or unreachable endpoint degrades
to a no-op and never blocks or crashes the request path.

## Consequences

- Vendor-neutral instrumentation; the backend can change without
  touching application code.
- HTTP RED metrics come for free from the middleware — no hand-written
  request-count or latency instruments.
- The OpenTelemetry Go SDK surface is large and its initialization is
  verbose; this is isolated in `internal/telemetry`.
- Fail-soft handling is required at every exporter boundary and is
  covered by Boundary Value Analysis tests (empty / valid / unreachable
  endpoints).

## Alternatives considered

- **`prometheus/client_golang`** — the established Go metrics library.
  Rejected as a *second* metrics stack alongside OTel: dual pipelines
  cause cardinality drift and dashboard confusion. OTel's
  `MeterProvider` is the single interface, by decision.
- **A vendor APM SDK** (Datadog, AWS X-Ray SDK) — ergonomic, but locks
  instrumentation to one backend and undercuts the vendor-neutrality
  this ADR exists to keep.
- **DIY instrumentation** (hand-rolled Prometheus counters + manual
  trace headers) — re-implements a large fraction of OTel, worse.

## Out of scope / when to revisit

- **Sampling strategy** — the tracer samples at ratio 1.0 (every
  request) for take-home scale. Revisit under real traffic: a
  `parentbased(traceidratio)` with an error-biased tail sampler is the
  next step once trace volume has a cost.
- **Span-to-metrics, browser RUM, profiling tuning** — deferred; the
  current SDK wiring is the foundation they would build on, not a
  rework.
