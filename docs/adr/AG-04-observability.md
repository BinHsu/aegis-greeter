# AG-04: Observability

- **Status**: Accepted
- **Date**: 2026-05-16

## Context

The service emits all four observability signals — logs, metrics,
traces, profiles — to a Grafana Cloud stack (Loki, Mimir, Tempo,
Pyroscope) through a local Grafana Alloy collector. Two properties
shape every decision here: the instrumentation must be vendor-neutral
so the backend stays reversible, and it must be fail-soft so a missing
or unreachable telemetry endpoint never blocks or crashes the request
path.

## Decisions

### 1. Logging — log/slog, JSON, a flat field contract

Logging uses the standard library's `log/slog` — no third-party
logging dependency. Output is JSON with a flat key set, which the Loki
ingestion pipeline parses cheapest:

- `time`, `level`, `msg` — `slog` JSON defaults.
- `trace_id`, `span_id` — extracted from the request's OpenTelemetry
  span context; omitted entirely when there is no valid span, rather
  than emitted as all-zero values.
- `pod`, `node` — from the Downward API environment at startup.

The application emits no Kubernetes metadata beyond `pod` / `node`;
Alloy enriches the rest at ingestion. This field set is a contract with
the sibling repo's Alloy configuration — changing it is a cross-repo
coordination, not a local edit.

### 2. Metrics and traces — OpenTelemetry, one interface

Metrics and traces use the OpenTelemetry Go SDK over OTLP gRPC. The
`otelhttp` middleware wraps the greeter and auto-emits the standard
HTTP RED metrics plus one span per request; two custom business
instruments — `greeter_responses_total{personalized}` and
`greeter_build_info` — are hand-written. `prometheus/client_golang` is
deliberately **not** added: a second metrics stack causes cardinality
drift and dashboard confusion, so the OTel `MeterProvider` is the
single metrics interface.

### 3. Profiling — pyroscope-go

Continuous profiling uses `github.com/grafana/pyroscope-go` — CPU,
alloc-objects, and goroutine profiles.

### 4. Fail-soft everywhere

Every exporter degrades to a no-op when its endpoint is empty or
unreachable; the request path is never blocked by, and never crashes
on, a telemetry failure. This is covered by Boundary Value Analysis
tests over empty / valid / unreachable endpoints.

## Consequences

- Instrumentation is vendor-neutral: the backend can change without
  touching application code.
- Logs, traces, and metrics share a vocabulary — a log line links to
  its trace through `trace_id`.
- HTTP RED metrics come for free from the middleware; there are no
  hand-written request-count or latency instruments to maintain.
- Cost: the OpenTelemetry Go SDK surface is large and its
  initialization is verbose — this is isolated in `internal/telemetry`
  so it does not leak into the rest of the code.

## Alternatives considered

- **`uber-go/zap` / `rs/zerolog`** — faster loggers, but the speed
  matters only under logging volume this service does not have; adopt
  on a profile, not on reputation.
- **Nested JSON or `logfmt` log output** — Loki parses flat JSON
  cheapest; nesting and `logfmt` buy nothing the query side wants.
- **CloudWatch Logs Insights key names** — the original draft, before
  the observability backend was pinned to Grafana Cloud.
- **`prometheus/client_golang` alongside OTel** — a dual metrics
  stack; rejected for cardinality and dashboard reasons.
- **A vendor APM SDK (Datadog, AWS X-Ray)** — locks instrumentation to
  one backend and undercuts the vendor-neutrality this record exists to
  keep.

## Out of scope / when to revisit

- **Sampling strategy** — the tracer samples every request (ratio 1.0)
  at take-home scale; revisit under real traffic with a parent-based
  ratio plus an error-biased tail sampler.
- **A `schema_version` log field** — worth adding if the record grows
  past its current small fixed set; premature at seven keys.
- **A faster logger** — revisit the moment a CPU profile (Pyroscope is
  wired) shows logging on a hot path; `slog.Handler` is an interface,
  so the migration is mechanical.
