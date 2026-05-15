package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/BinHsu/aegis-greeter/internal/telemetry"
)

// TestTraceContextHandler_BVA exercises the span-context boundary in
// the slog handler: no span in ctx → trace fields omitted; valid span
// → trace_id + span_id populated. The "zero TraceID" boundary
// collapses to "no span" semantically (both have invalid SpanContext),
// so it is the same code path; covered by the no-span case.
func TestTraceContextHandler_BVA(t *testing.T) {
	t.Parallel()

	t.Run("no span in ctx: trace fields omitted", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		handler := telemetry.NewTraceContextHandler(
			slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
			"test-pod", "test-node",
		)
		logger := slog.New(handler)

		logger.InfoContext(context.Background(), "no span")

		fields := decode(t, buf.Bytes())
		if _, ok := fields["trace_id"]; ok {
			t.Errorf("trace_id should be absent: %v", fields)
		}
		if _, ok := fields["span_id"]; ok {
			t.Errorf("span_id should be absent: %v", fields)
		}
		if got := fields["pod"]; got != "test-pod" {
			t.Errorf("pod: got %v, want test-pod", got)
		}
		if got := fields["node"]; got != "test-node" {
			t.Errorf("node: got %v, want test-node", got)
		}
	})

	t.Run("active span: trace_id + span_id populated", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		handler := telemetry.NewTraceContextHandler(
			slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
			"", "",
		)
		logger := slog.New(handler)

		tp := sdktrace.NewTracerProvider()
		t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

		ctx, span := tp.Tracer("test").Start(context.Background(), "op")
		defer span.End()

		logger.InfoContext(ctx, "with span")

		fields := decode(t, buf.Bytes())
		traceID, ok := fields["trace_id"].(string)
		if !ok || traceID == "" || isAllZero(traceID) {
			t.Errorf("trace_id should be a non-zero hex: got %v", fields["trace_id"])
		}
		spanID, ok := fields["span_id"].(string)
		if !ok || spanID == "" || isAllZero(spanID) {
			t.Errorf("span_id should be a non-zero hex: got %v", fields["span_id"])
		}
		if _, ok := fields["pod"]; ok {
			t.Errorf("pod should be absent when empty: %v", fields)
		}
	})
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode %q: %v", raw, err)
	}
	return m
}

func isAllZero(hex string) bool {
	for _, c := range hex {
		if c != '0' {
			return false
		}
	}
	return true
}
