// Package metrics defines the custom OpenTelemetry business
// instruments for the greeter service:
//
//   - greeter_responses_total{personalized="true|false"} Counter,
//     incremented per request from the handler. "personalized" is true
//     only when the request carried a non-empty ?name= URL parameter.
//   - greeter_build_info{version,commit,image_sha} ObservableGauge set
//     to 1 at every collection interval — Prometheus "info" metric
//     convention. Labels are captured at startup from build-time
//     ldflags + the IMAGE_SHA env var.
//
// HTTP-level RED metrics (request duration, in-flight, body size) are
// emitted by the otelhttp middleware in cmd/greeter/main.go — they
// are NOT re-implemented in this package, per the cross-repo
// observability contract that forbids duplicate instruments.
package metrics
