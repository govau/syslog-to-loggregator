package pulseemitter_test

import (
	"sync"

	loggregator "code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/pulseemitter"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CounterMetric", func() {
	Context("Emit", func() {
		It("prepares an envelope for delivery", func() {
			metric := pulseemitter.NewCounterMetric("name", pulseemitter.WithVersion(1, 2))

			metric.Increment(10)

			spy := newSpyLoggClient()
			metric.Emit(spy)
			Expect(spy.CounterName()).To(Equal("name"))

			e := &loggregator_v2.Envelope{
				Message: &loggregator_v2.Envelope_Counter{
					Counter: &loggregator_v2.Counter{},
				},
				Tags: make(map[string]string),
			}
			for _, o := range spy.CounterOpts() {
				o(e)
			}

			Expect(e.GetCounter().GetDelta()).To(Equal(uint64(10)))
			Expect(e.Tags["metric_version"]).To(Equal("1.2"))
		})

		It("decrements its value on success", func() {
			metric := pulseemitter.NewCounterMetric("name")
			spy := newSpyLoggClient()

			metric.Increment(10)
			metric.Emit(spy)

			metric.Emit(spy)
			e := &loggregator_v2.Envelope{
				Message: &loggregator_v2.Envelope_Counter{
					&loggregator_v2.Counter{},
				},
			}

			for _, o := range spy.counterOpts {
				o(e)
			}

			Expect(e.GetCounter().GetDelta()).To(Equal(uint64(0)))
		})
	})
})

type spyLoggClient struct {
	mu             sync.Mutex
	name           string
	counterOpts    []loggregator.EmitCounterOption
	gaugeOpts      []loggregator.EmitGaugeOption
	gaugeCallCount int
}

func newSpyLoggClient() *spyLoggClient {
	return &spyLoggClient{}
}

func (s *spyLoggClient) EmitCounter(name string, opts ...loggregator.EmitCounterOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.name = name
	s.counterOpts = opts
}

func (s *spyLoggClient) EmitGauge(opts ...loggregator.EmitGaugeOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gaugeCallCount++
	s.gaugeOpts = opts
}

func (s *spyLoggClient) CounterName() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.name
}

func (s *spyLoggClient) CounterOpts() []loggregator.EmitCounterOption {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.counterOpts
}

func (s *spyLoggClient) GaugeOpts() []loggregator.EmitGaugeOption {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.gaugeOpts
}

func (s *spyLoggClient) GaugeCallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.gaugeCallCount
}
