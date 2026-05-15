package telemetry_test

import (
	"context"
	"testing"
	"time"

	"github.com/BinHsu/aegis-greeter/internal/telemetry"
)

// shutdownTimeout is the per-test budget for provider Shutdown. The
// cross-repo contract says the app must not be sensitive to telemetry
// export failures; Shutdown blocks until the in-flight batch flushes
// or the context expires. With an unreachable endpoint the flush will
// never complete — the short budget bounds the test runtime, and the
// returned error is intentionally ignored (it is the expected outcome
// when there is no Alloy DaemonSet to talk to).
const shutdownTimeout = 200 * time.Millisecond

// TestInit_SamplerRatio_BVA exercises the sampler ratio boundary:
// negative (clamped to 0), zero, one, and above-one (clamped to 1).
// All four cases must succeed without panic.
func TestInit_SamplerRatio_BVA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		ratio float64
	}{
		{"negative (clamped to 0)", -0.5},
		{"zero", 0.0},
		{"one", 1.0},
		{"above one (clamped to 1)", 2.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := telemetry.Init(context.Background(), telemetry.Config{
				ServiceName:  "test",
				OTLPEndpoint: "",
				SamplerRatio: tc.ratio,
			})
			if err != nil {
				t.Fatalf("Init: %v", err)
			}
			if p == nil {
				t.Fatal("expected non-nil providers")
			}
			shutdownBounded(p)
		})
	}
}

// TestInit_OTLPEndpoint_BVA exercises the endpoint boundary: empty
// (no exporters wired), syntactically valid host:port (lazy-dialed by
// the gRPC exporter, so Init does not actually need the listener up),
// and an obviously unreachable address (same lazy-dial behavior).
// All three must succeed without panic — the request path must not
// be sensitive to a missing or unreachable Alloy DaemonSet.
//
// Shutdown errors are NOT asserted here. The cross-repo contract is
// explicit that the app must tolerate export failures; Shutdown will
// error when the endpoint is unreachable (it tried to flush and could
// not), and that is the correct degraded behavior, not a regression.
func TestInit_OTLPEndpoint_BVA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		endpoint string
	}{
		{"empty (no exporter wired)", ""},
		{"syntactically valid host:port", "127.0.0.1:4317"},
		{"unreachable address", "192.0.2.1:4317"}, // TEST-NET-1, RFC 5737 — guaranteed unroutable
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := telemetry.Init(context.Background(), telemetry.Config{
				ServiceName:  "test",
				OTLPEndpoint: tc.endpoint,
				SamplerRatio: 1.0,
			})
			if err != nil {
				t.Fatalf("Init endpoint=%q: %v", tc.endpoint, err)
			}
			if p == nil {
				t.Fatal("expected non-nil providers")
			}
			shutdownBounded(p)
		})
	}
}

// shutdownBounded calls Shutdown with a short deadline so test runs
// are not paced by export-flush timeouts. Error is intentionally
// ignored — see TestInit_OTLPEndpoint_BVA for the rationale.
func shutdownBounded(p *telemetry.Providers) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	_ = p.Shutdown(ctx)
}

// TestStartProfiler_Endpoint_BVA mirrors the OTLP endpoint BVA for
// Pyroscope: empty (no-op handle), valid, and unreachable. Stop must
// be safe to call in every case.
func TestStartProfiler_Endpoint_BVA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		endpoint string
	}{
		{"empty (no-op)", ""},
		{"syntactically valid", "http://127.0.0.1:4040"},
		{"unreachable", "http://192.0.2.1:4040"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, err := telemetry.StartProfiler(telemetry.ProfilerConfig{
				ApplicationName: "test",
				Endpoint:        tc.endpoint,
			})
			if err != nil {
				t.Fatalf("StartProfiler: %v", err)
			}
			if err := p.Stop(); err != nil {
				t.Errorf("Stop: %v", err)
			}
		})
	}
}
