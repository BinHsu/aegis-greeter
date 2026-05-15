package metrics_test

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/BinHsu/aegis-greeter/internal/metrics"
)

// TestRecordResponse_PersonalizedLabel_BVA exercises the personalized
// label boundary: name absent, name present-but-empty, name present
// with a non-empty value. The first two collapse to personalized=false
// (the handler treats both as "fall back to IP"); only the third
// emits personalized=true.
func TestRecordResponse_PersonalizedLabel_BVA(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		personalized bool
		wantLabel    string
	}{
		{"absent: personalized=false", false, "false"},
		{"empty value: personalized=false", false, "false"},
		{"non-empty value: personalized=true", true, "true"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

			ins, err := metrics.New(mp.Meter("test"), "v", "c", "s")
			if err != nil {
				t.Fatalf("metrics.New: %v", err)
			}

			ins.RecordResponse(context.Background(), tc.personalized)

			var rm metricdata.ResourceMetrics
			if err := reader.Collect(context.Background(), &rm); err != nil {
				t.Fatalf("Collect: %v", err)
			}

			label := personalizedLabel(t, rm)
			if label != tc.wantLabel {
				t.Errorf("personalized label: got %q, want %q", label, tc.wantLabel)
			}
		})
	}
}

// TestBuildInfo_AlwaysOne asserts the greeter_build_info gauge emits
// 1 with the configured labels — the Prometheus "info" convention.
func TestBuildInfo_AlwaysOne(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	if _, err := metrics.New(mp.Meter("test"), "v1.2.3", "abcdef0", "sha256:cafe"); err != nil {
		t.Fatalf("metrics.New: %v", err)
	}

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Collect: %v", err)
	}

	value, attrs := buildInfoSample(t, rm)
	if value != 1 {
		t.Errorf("build_info value: got %d, want 1", value)
	}
	for _, want := range []struct{ key, val string }{
		{"version", "v1.2.3"},
		{"commit", "abcdef0"},
		{"image_sha", "sha256:cafe"},
	} {
		got, ok := attrs[want.key]
		if !ok {
			t.Errorf("build_info label %q missing", want.key)
			continue
		}
		if got != want.val {
			t.Errorf("build_info label %q: got %q, want %q", want.key, got, want.val)
		}
	}
}

func personalizedLabel(t *testing.T, rm metricdata.ResourceMetrics) string {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "greeter_responses_total" {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("greeter_responses_total wrong data type: %T", m.Data)
			}
			if len(sum.DataPoints) != 1 {
				t.Fatalf("expected exactly one data point, got %d", len(sum.DataPoints))
			}
			val, ok := sum.DataPoints[0].Attributes.Value("personalized")
			if !ok {
				t.Fatal("personalized attribute missing")
			}
			return val.AsString()
		}
	}
	t.Fatal("greeter_responses_total not found")
	return ""
}

func buildInfoSample(t *testing.T, rm metricdata.ResourceMetrics) (int64, map[string]string) {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "greeter_build_info" {
				continue
			}
			g, ok := m.Data.(metricdata.Gauge[int64])
			if !ok {
				t.Fatalf("greeter_build_info wrong data type: %T", m.Data)
			}
			if len(g.DataPoints) != 1 {
				t.Fatalf("expected exactly one data point, got %d", len(g.DataPoints))
			}
			dp := g.DataPoints[0]
			attrs := make(map[string]string)
			for _, kv := range dp.Attributes.ToSlice() {
				attrs[string(kv.Key)] = kv.Value.AsString()
			}
			return dp.Value, attrs
		}
	}
	t.Fatal("greeter_build_info not found")
	return 0, nil
}
