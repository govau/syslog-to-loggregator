package pulseemitter

import (
	"time"

	loggregator "code.cloudfoundry.org/go-loggregator"
)

// LogClient is the client used by PulseEmitter to emit metrics. This would
// usually be the go-loggregator v2 client.
type LoggClient interface {
	EmitCounter(name string, opts ...loggregator.EmitCounterOption)
	EmitGauge(opts ...loggregator.EmitGaugeOption)
}

type emitter interface {
	Emit(c LoggClient)
}

type PulseEmitterOption func(*PulseEmitter)

// WithPulseInterval is a PulseEmitterOption for setting the pulsing interval.
func WithPulseInterval(d time.Duration) PulseEmitterOption {
	return func(c *PulseEmitter) {
		c.pulseInterval = d
	}
}

// PulseEmitter will emit metrics on a given interval.
type PulseEmitter struct {
	loggClient LoggClient

	pulseInterval time.Duration
}

// New returns a PulseEmitter configured with the given LoggClient and
// PulseEmitterOptions. The default pulse interval is 60 seconds.
func New(c LoggClient, opts ...PulseEmitterOption) *PulseEmitter {
	pe := &PulseEmitter{
		pulseInterval: 60 * time.Second,
		loggClient:    c,
	}

	for _, opt := range opts {
		opt(pe)
	}

	return pe
}

// NewCounterMetric returns a CounterMetric that can be incremented. After
// calling NewCounterMetric the counter metric will begin to be emitted on the
// interval configured on the PulseEmitter. If the counter metrics value has
// not changed since last emitted a 0 value will be emitted. Every time the
// counter metric is emitted, its delta is reset to 0.
func (c *PulseEmitter) NewCounterMetric(name string, opts ...MetricOption) CounterMetric {
	m := NewCounterMetric(name, opts...)
	go c.pulse(m)

	return m
}

// NewGaugeMetric returns a GaugeMetric that has a value that can be set.
// After calling NewGaugeMetric the gauge metric will begin to be emitted on
// the interval configured on the PulseEmitter. When emitting the gauge
// metric, it will use the last value given when calling set on the gauge
// metric.
func (c *PulseEmitter) NewGaugeMetric(name, unit string, opts ...MetricOption) GaugeMetric {
	g := NewGaugeMetric(name, unit, opts...)
	go c.pulse(g)

	return g
}

func (c *PulseEmitter) pulse(e emitter) {
	for range time.Tick(c.pulseInterval) {
		e.Emit(c.loggClient)
	}
}
