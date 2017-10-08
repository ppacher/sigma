package metrics

import (
	"fmt"
	"sync"

	"github.com/homebot/sigma/node"
)

// Metric is some metric collected for functions
type Metric interface {
	// Update re-calculates the metrics value and is called on every
	// control loop iteration
	Update(map[string]node.Controller) float64

	// String returns a string representation of the metric
	String() string

	// IsAbs returns true if the metric value is absolute, false if it's
	// relative (precentage)
	IsAbs() bool
}

// MetricFactory creates a new metrics map for the given controller
type MetricFactory func() Metric

// Metrics keeps track of various matrics for a function running on sigma
type Metrics struct {
	rw      sync.RWMutex
	metrics map[string]Metric

	lastResult map[string]float64
}

// Update recalculates all metric values and returns the result
func (m *Metrics) Update(states map[string]node.Controller) map[string]float64 {
	m.rw.Lock()
	defer m.rw.Unlock()

	res := make(map[string]float64)

	for key, metric := range m.metrics {
		res[key] = metric.Update(states)
	}

	m.lastResult = res

	return res
}

// Last returns the last result computed
func (m *Metrics) Last() map[string]float64 {
	m.rw.RLock()
	defer m.rw.RUnlock()

	return m.lastResult
}

type metricTypes struct {
	rw sync.RWMutex

	factories map[string]MetricFactory
}

func (t *metricTypes) getMetrics() *Metrics {
	t.rw.RLock()
	defer t.rw.RUnlock()

	m := &Metrics{
		metrics: make(map[string]Metric),
	}

	for key, factory := range t.factories {
		m.metrics[key] = factory()
	}

	return m
}

var factories *metricTypes

// GetMetrics returns a new Metrics object for the given function
// controller
func GetMetrics() *Metrics {
	return factories.getMetrics()
}

// Register registers a new metric factory
func Register(name string, factory MetricFactory) {
	factories.rw.Lock()
	defer factories.rw.Unlock()

	if _, ok := factories.factories[name]; ok {
		panic(fmt.Sprintf("metric with name %q already registered", name))
	}

	factories.factories[name] = factory
}

func init() {
	factories = &metricTypes{
		factories: make(map[string]MetricFactory),
	}
}
