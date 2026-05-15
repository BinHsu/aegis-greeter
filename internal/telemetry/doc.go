// Package telemetry initializes the OpenTelemetry TracerProvider and
// MeterProvider, the Grafana Pyroscope profiler client, and a
// log/slog handler that injects OTel trace context into every log
// record. All three components are independently fail-soft: an empty
// or unreachable endpoint degrades the corresponding subsystem to a
// no-op rather than crashing the request path.
//
// Wired by cmd/greeter/main.go per the cross-repo observability
// contract (aegis-stateless issue #1): metrics + traces to Grafana
// Cloud Mimir / Tempo via local Alloy DaemonSet on OTLP gRPC, logs to
// Loki via Alloy log-tailing, profiles to Pyroscope direct.
package telemetry
