// Package telemetry initializes the OpenTelemetry TracerProvider and
// MeterProvider, the Grafana Pyroscope profiler client, and a log/slog
// JSON handler that injects trace context. Wiring is added in Step A3
// (cross-repo observability contract, sibling issue #1).
package telemetry
