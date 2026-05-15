package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Instruments holds the custom business instruments for aegis-greeter.
// HTTP-level RED metrics (request duration, in-flight, body size) are
// emitted by otelhttp middleware in the calling code — those are NOT
// re-implemented here. Anything in this struct is business-domain.
type Instruments struct {
	responses metric.Int64Counter
}

// New constructs the custom instruments against the supplied Meter.
// The build_info Gauge is observable: it reports 1 at every collection
// interval with version and commit as labels — the Prometheus "info"
// metric convention. version and commit are baked into the binary at
// build time via -ldflags and do not change for the process lifetime.
func New(meter metric.Meter, version, commit string) (*Instruments, error) {
	responses, err := meter.Int64Counter(
		"greeter_responses_total",
		metric.WithDescription("Total greetings sent, labeled by personalization state."),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.Int64ObservableGauge(
		"greeter_build_info",
		metric.WithDescription("Build identity gauge — always 1, labels carry version/commit."),
		metric.WithInt64Callback(func(_ context.Context, o metric.Int64Observer) error {
			o.Observe(1, metric.WithAttributes(
				attribute.String("version", version),
				attribute.String("commit", commit),
			))
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Instruments{responses: responses}, nil
}

// RecordResponse increments greeter_responses_total with the
// personalized={true,false} label. Pass true only when the request
// carried a non-empty ?name= query parameter.
func (i *Instruments) RecordResponse(ctx context.Context, personalized bool) {
	label := "false"
	if personalized {
		label = "true"
	}
	i.responses.Add(ctx, 1, metric.WithAttributes(attribute.String("personalized", label)))
}
