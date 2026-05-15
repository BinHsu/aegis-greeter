// Package metrics defines the custom OpenTelemetry business
// instruments for the greeter service:
//
//   - greeter_responses_total{personalized="true|false"} Counter,
//     incremented per request from the handler. "personalized" is true
//     only when the request carried a non-empty ?name= URL parameter.
//   - greeter_build_info{version,commit} ObservableGauge set to 1 at
//     every collection interval — the Prometheus "info" metric
//     convention. Labels are baked into the binary at build time via
//     -ldflags.
//
// HTTP-level RED metrics (request duration, in-flight, body size) are
// emitted by the otelhttp middleware in cmd/greeter/main.go — they
// are NOT re-implemented in this package, per the cross-repo
// observability contract that forbids duplicate instruments.
package metrics
