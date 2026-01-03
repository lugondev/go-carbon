// Package metrics provides interfaces and implementations for collecting and managing
// performance metrics within the carbon pipeline.
//
// The Metrics interface defines methods for initializing, updating, flushing, and
// shutting down metrics. It supports various metric types including gauges, counters,
// and histograms for monitoring performance and operational health in real time.
package metrics

import (
	"context"
	"log/slog"
	"sync"
)

// Metrics defines the interface for collecting and managing pipeline metrics.
// Implementations can send metrics to various backends like Prometheus, DataDog, etc.
type Metrics interface {
	// Initialize prepares the metrics system for data collection.
	Initialize(ctx context.Context) error

	// Flush sends any buffered metrics data to ensure all metrics are reported.
	Flush(ctx context.Context) error

	// Shutdown gracefully shuts down the metrics system, performing cleanup.
	Shutdown(ctx context.Context) error

	// UpdateGauge sets a gauge metric to the specified value.
	// Gauges track values that can go up or down, like queue length.
	UpdateGauge(ctx context.Context, name string, value float64) error

	// IncrementCounter increments a counter metric by the specified value.
	// Counters track values that only increase, like total processed items.
	IncrementCounter(ctx context.Context, name string, value uint64) error

	// RecordHistogram records a value in a histogram metric.
	// Histograms track the distribution of values, like request latencies.
	RecordHistogram(ctx context.Context, name string, value float64) error
}

// Collection manages multiple Metrics implementations and delegates calls to all of them.
type Collection struct {
	metrics []Metrics
	mu      sync.RWMutex
}

// NewCollection creates a new Collection with the given metrics implementations.
func NewCollection(metrics ...Metrics) *Collection {
	return &Collection{
		metrics: metrics,
	}
}

// Add adds a new Metrics implementation to the collection.
func (c *Collection) Add(m Metrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, m)
}

// Initialize initializes all metrics in the collection.
func (c *Collection) Initialize(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, m := range c.metrics {
		if err := m.Initialize(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes all metrics in the collection.
func (c *Collection) Flush(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, m := range c.metrics {
		if err := m.Flush(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Shutdown shuts down all metrics in the collection.
func (c *Collection) Shutdown(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, m := range c.metrics {
		if err := m.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

// UpdateGauge updates a gauge metric across all implementations.
func (c *Collection) UpdateGauge(ctx context.Context, name string, value float64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, m := range c.metrics {
		if err := m.UpdateGauge(ctx, name, value); err != nil {
			return err
		}
	}
	return nil
}

// IncrementCounter increments a counter across all implementations.
func (c *Collection) IncrementCounter(ctx context.Context, name string, value uint64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, m := range c.metrics {
		if err := m.IncrementCounter(ctx, name, value); err != nil {
			return err
		}
	}
	return nil
}

// RecordHistogram records a histogram value across all implementations.
func (c *Collection) RecordHistogram(ctx context.Context, name string, value float64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, m := range c.metrics {
		if err := m.RecordHistogram(ctx, name, value); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the number of metrics implementations in the collection.
func (c *Collection) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.metrics)
}

// NoopMetrics is a Metrics implementation that does nothing.
// Useful for testing or when metrics are disabled.
type NoopMetrics struct{}

// NewNoopMetrics creates a new NoopMetrics.
func NewNoopMetrics() *NoopMetrics {
	return &NoopMetrics{}
}

func (n *NoopMetrics) Initialize(ctx context.Context) error                              { return nil }
func (n *NoopMetrics) Flush(ctx context.Context) error                                   { return nil }
func (n *NoopMetrics) Shutdown(ctx context.Context) error                                { return nil }
func (n *NoopMetrics) UpdateGauge(ctx context.Context, name string, value float64) error { return nil }
func (n *NoopMetrics) IncrementCounter(ctx context.Context, name string, value uint64) error {
	return nil
}
func (n *NoopMetrics) RecordHistogram(ctx context.Context, name string, value float64) error {
	return nil
}

// LogMetrics is a Metrics implementation that logs all metrics using slog.
type LogMetrics struct {
	logger   *slog.Logger
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]uint64
}

// NewLogMetrics creates a new LogMetrics with the given logger.
// If logger is nil, the default logger is used.
func NewLogMetrics(logger *slog.Logger) *LogMetrics {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogMetrics{
		logger:   logger,
		gauges:   make(map[string]float64),
		counters: make(map[string]uint64),
	}
}

// Initialize initializes the log metrics.
func (l *LogMetrics) Initialize(ctx context.Context) error {
	l.logger.Info("metrics initialized")
	return nil
}

// Flush logs all current metric values.
func (l *LogMetrics) Flush(ctx context.Context) error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	l.logger.Info("metrics flush",
		"gauges", l.gauges,
		"counters", l.counters,
	)
	return nil
}

// Shutdown shuts down the log metrics.
func (l *LogMetrics) Shutdown(ctx context.Context) error {
	l.logger.Info("metrics shutdown")
	return nil
}

// UpdateGauge logs the gauge update.
func (l *LogMetrics) UpdateGauge(ctx context.Context, name string, value float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.gauges[name] = value
	l.logger.Debug("gauge updated", "name", name, "value", value)
	return nil
}

// IncrementCounter logs the counter increment.
func (l *LogMetrics) IncrementCounter(ctx context.Context, name string, value uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.counters[name] += value
	l.logger.Debug("counter incremented", "name", name, "value", value, "total", l.counters[name])
	return nil
}

// RecordHistogram logs the histogram record.
func (l *LogMetrics) RecordHistogram(ctx context.Context, name string, value float64) error {
	l.logger.Debug("histogram recorded", "name", name, "value", value)
	return nil
}

// Metric names used by the pipeline.
const (
	MetricUpdatesReceived                = "updates_received"
	MetricUpdatesProcessed               = "updates_processed"
	MetricUpdatesSuccessful              = "updates_successful"
	MetricUpdatesFailed                  = "updates_failed"
	MetricUpdatesQueued                  = "updates_queued"
	MetricUpdatesProcessTimeNanoseconds  = "updates_process_time_nanoseconds"
	MetricUpdatesProcessTimeMilliseconds = "updates_process_time_milliseconds"
	MetricAccountUpdatesProcessed        = "account_updates_processed"
	MetricTransactionUpdatesProcessed    = "transaction_updates_processed"
	MetricAccountDeletionsProcessed      = "account_deletions_processed"
	MetricBlockDetailsProcessed          = "block_details_processed"
)
