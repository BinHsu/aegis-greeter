package telemetry

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// TraceContextHandler wraps a slog.Handler to inject trace_id and
// span_id (extracted from the request context via the OTel API) along
// with static pod and node identifiers (set from Downward API env at
// startup). Records that have no valid span context omit the trace
// fields entirely rather than emitting all-zero IDs — keeps Loki log
// lines clean and prevents indexing a useless label.
type TraceContextHandler struct {
	inner     slog.Handler
	pod, node string
}

// NewTraceContextHandler wraps inner with the trace + pod + node
// injection. Pass empty strings for pod / node to omit them.
func NewTraceContextHandler(inner slog.Handler, pod, node string) *TraceContextHandler {
	return &TraceContextHandler{inner: inner, pod: pod, node: node}
}

// Enabled delegates to the wrapped handler.
func (h *TraceContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle adds pod, node, trace_id, and span_id attrs (when present
// and valid) and forwards to the inner handler.
func (h *TraceContextHandler) Handle(ctx context.Context, rec slog.Record) error {
	if h.pod != "" {
		rec.AddAttrs(slog.String("pod", h.pod))
	}
	if h.node != "" {
		rec.AddAttrs(slog.String("node", h.node))
	}
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		rec.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, rec)
}

// WithAttrs returns a new handler with the additional attrs applied
// to the inner handler; pod and node fields propagate.
func (h *TraceContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TraceContextHandler{
		inner: h.inner.WithAttrs(attrs),
		pod:   h.pod,
		node:  h.node,
	}
}

// WithGroup returns a new handler with a group applied to the inner
// handler; pod and node fields propagate.
func (h *TraceContextHandler) WithGroup(name string) slog.Handler {
	return &TraceContextHandler{
		inner: h.inner.WithGroup(name),
		pod:   h.pod,
		node:  h.node,
	}
}
