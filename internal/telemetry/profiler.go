package telemetry

import (
	"github.com/grafana/pyroscope-go"
)

// ProfilerConfig configures the Pyroscope client. An empty Endpoint
// disables profiling — the returned Profiler is a no-op handle whose
// Stop is safe to call.
type ProfilerConfig struct {
	ApplicationName string
	Endpoint        string
}

// Profiler wraps the upstream pyroscope.Profiler so a nil-or-empty
// configuration produces a handle that satisfies the same interface.
type Profiler struct {
	inner *pyroscope.Profiler
}

// StartProfiler begins continuous CPU, alloc-objects, and goroutine
// profiling against the configured Pyroscope endpoint. When Endpoint
// is empty, no upstream client is started and Stop is a no-op.
func StartProfiler(cfg ProfilerConfig) (*Profiler, error) {
	if cfg.Endpoint == "" {
		return &Profiler{}, nil
	}
	p, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: cfg.ApplicationName,
		ServerAddress:   cfg.Endpoint,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileGoroutines,
		},
	})
	if err != nil {
		return nil, err
	}
	return &Profiler{inner: p}, nil
}

// Stop flushes any pending profile batches and shuts down the client.
// Safe to call on a nil receiver or on a no-op Profiler.
func (p *Profiler) Stop() error {
	if p == nil || p.inner == nil {
		return nil
	}
	return p.inner.Stop()
}
