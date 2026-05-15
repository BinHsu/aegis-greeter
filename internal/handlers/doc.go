// Package handlers implements the HTTP request handlers for the
// aegis-greeter service: the greeter at /, the liveness probe at
// /healthz, and the drain-aware readiness probe at /readyz.
//
// All handlers in this package are stdlib-only; the OTel instrumentation
// is applied via otelhttp.NewHandler in cmd/greeter when the server is
// wired (see Step A3b).
package handlers
