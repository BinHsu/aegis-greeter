package telemetry

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Config is the input to Init. An empty OTLPEndpoint disables network
// export — the providers are still constructed so the rest of the app
// can rely on them, but no spans or metrics leave the process. A
// SamplerRatio outside [0.0, 1.0] is clamped silently without panic.
type Config struct {
	ServiceName  string
	OTLPEndpoint string
	SamplerRatio float64
}

// Providers groups the constructed TracerProvider and MeterProvider
// alongside a single Shutdown that flushes both.
type Providers struct {
	Tracer   trace.TracerProvider
	Meter    metric.MeterProvider
	shutdown []func(context.Context) error
}

// Shutdown flushes any pending batches in both providers. Errors are
// joined so a slow trace flush does not mask a metric flush failure.
func (p *Providers) Shutdown(ctx context.Context) error {
	var errs []error
	for _, fn := range p.shutdown {
		if err := fn(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Init builds the providers per cfg. When OTLPEndpoint is empty the
// providers are constructed without exporters — useful for local
// development, tests, and degraded environments where Alloy is
// unreachable but the app should still serve traffic.
func Init(ctx context.Context, cfg Config) (*Providers, error) {
	ratio := clamp01(cfg.SamplerRatio)

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes("",
			attribute.String("service.name", cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	p := &Providers{}

	tpOpts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	}
	mpOpts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}

	if cfg.OTLPEndpoint != "" {
		traceExp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		tpOpts = append(tpOpts, sdktrace.WithBatcher(traceExp))

		metricExp, err := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
		if err != nil {
			return nil, err
		}
		mpOpts = append(mpOpts, sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)))
	}

	tp := sdktrace.NewTracerProvider(tpOpts...)
	mp := sdkmetric.NewMeterProvider(mpOpts...)

	p.Tracer = tp
	p.Meter = mp
	p.shutdown = []func(context.Context) error{tp.Shutdown, mp.Shutdown}
	return p, nil
}

// clamp01 forces v into [0.0, 1.0]. Sampler ratios outside the range
// are programming errors but the right runtime answer is "do not
// crash, do not over-sample" rather than panic. The SDK itself clamps,
// but doing it explicitly here makes the BVA test cover this path.
func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
