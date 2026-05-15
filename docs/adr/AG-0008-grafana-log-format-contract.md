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
