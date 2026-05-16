# AG-0008: Grafana-stack structured log format contract

- **Status**: Accepted
- **Date**: 2026-05-15

## Context

Logs flow to Grafana Cloud Loki via a Grafana Alloy DaemonSet that
tails container stdout. Loki and the trace backend (Tempo) are only
useful together if the log records carry a stable, agreed set of
fields. An earlier draft of this contract targeted AWS CloudWatch Logs
Insights; the observability stack was then pinned to Grafana Cloud
(cross-repo issue #1), and this record was rewritten to match.

## Decision

Logs are `log/slog` JSON written to stdout, with **flat** keys — no
nested objects, so Loki label extraction stays simple:

- `time`, `level`, `msg` — `slog` JSON defaults.
- `trace_id`, `span_id` — extracted from the request's OpenTelemetry
  span context. Omitted entirely when there is no valid span, rather
  than emitted as all-zero values.
- `pod`, `node` — taken from the Downward API environment at startup.

The application does not emit Kubernetes metadata beyond `pod` / `node`;
Alloy enriches the rest at ingestion.

## Consequences

- Log lines correlate to traces through `trace_id`.
- Flat JSON keeps Loki parsing and labelling cheap.
- The field set is a contract with the sibling repo's Alloy
  configuration — changing it is a cross-repo coordination, not a local
  edit.

## Alternatives considered

- **Nested JSON** (`{"trace": {"id": ...}}`) — structurally tidy, but
  Loki label and field extraction is cheapest against flat keys; the
  nesting buys nothing the query side wants.
- **`logfmt`** — lighter than JSON, but `slog`'s first-class handler
  is JSON and Loki parses JSON natively; `logfmt` would be a custom
  handler for no gain.
- **CloudWatch Logs Insights key names** — the original draft, retired
  when the observability backend was pinned to Grafana Cloud
  (cross-repo issue #1).
- **Emitting K8s metadata** (`namespace`, `labels`) from the app —
  rejected; Alloy's `kubernetes` enrichment adds these at ingestion,
  and duplicating them in the app is redundant and a drift risk.

## Out of scope / when to revisit

- An explicit `schema_version` field — worth adding if the record
  grows past the current small fixed set and consumers must handle
  more than one shape. At seven keys, it is premature.
