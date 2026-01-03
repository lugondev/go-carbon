// Package processor defines the Processor interface for processing data within
// the carbon pipeline.
//
// The Processor interface provides a standardized way to handle various types of
// data. It includes support for metric tracking, enabling real-time insights into
// processing performance.
package processor

import (
	"context"

	"github.com/lugondev/go-carbon/internal/metrics"
)

// Processor defines the interface for processing data within the pipeline.
//
// Implementations of this interface handle specific types of data and can record
// metrics during processing. The type parameter T specifies the input data type.
type Processor[T any] interface {
	// Process handles the given data.
	// The context is used for cancellation and timeouts.
	// The metrics collection is used for recording performance metrics.
	Process(ctx context.Context, data T, metrics *metrics.Collection) error
}

// ProcessorFunc is a function type that implements the Processor interface.
// It allows using functions as processors without creating a new type.
type ProcessorFunc[T any] func(ctx context.Context, data T, metrics *metrics.Collection) error

// Process implements the Processor interface.
func (f ProcessorFunc[T]) Process(ctx context.Context, data T, metrics *metrics.Collection) error {
	return f(ctx, data, metrics)
}

// NoopProcessor is a processor that does nothing.
// Useful for testing or as a placeholder.
type NoopProcessor[T any] struct{}

// NewNoopProcessor creates a new NoopProcessor.
func NewNoopProcessor[T any]() *NoopProcessor[T] {
	return &NoopProcessor[T]{}
}

// Process does nothing and returns nil.
func (p *NoopProcessor[T]) Process(ctx context.Context, data T, metrics *metrics.Collection) error {
	return nil
}

// ChainedProcessor chains multiple processors together.
// Each processor is called in sequence with the same input.
type ChainedProcessor[T any] struct {
	processors []Processor[T]
}

// NewChainedProcessor creates a new ChainedProcessor with the given processors.
func NewChainedProcessor[T any](processors ...Processor[T]) *ChainedProcessor[T] {
	return &ChainedProcessor[T]{processors: processors}
}

// Add adds a processor to the chain.
func (c *ChainedProcessor[T]) Add(p Processor[T]) {
	c.processors = append(c.processors, p)
}

// Process calls each processor in sequence.
func (c *ChainedProcessor[T]) Process(ctx context.Context, data T, metrics *metrics.Collection) error {
	for _, p := range c.processors {
		if err := p.Process(ctx, data, metrics); err != nil {
			return err
		}
	}
	return nil
}

// ConditionalProcessor wraps a processor with a condition function.
// The processor is only called if the condition returns true.
type ConditionalProcessor[T any] struct {
	processor Processor[T]
	condition func(T) bool
}

// NewConditionalProcessor creates a new ConditionalProcessor.
func NewConditionalProcessor[T any](processor Processor[T], condition func(T) bool) *ConditionalProcessor[T] {
	return &ConditionalProcessor[T]{
		processor: processor,
		condition: condition,
	}
}

// Process calls the wrapped processor only if the condition returns true.
func (c *ConditionalProcessor[T]) Process(ctx context.Context, data T, metrics *metrics.Collection) error {
	if c.condition(data) {
		return c.processor.Process(ctx, data, metrics)
	}
	return nil
}

// ErrorHandlingProcessor wraps a processor with error handling.
type ErrorHandlingProcessor[T any] struct {
	processor    Processor[T]
	errorHandler func(error) error
}

// NewErrorHandlingProcessor creates a new ErrorHandlingProcessor.
func NewErrorHandlingProcessor[T any](processor Processor[T], errorHandler func(error) error) *ErrorHandlingProcessor[T] {
	return &ErrorHandlingProcessor[T]{
		processor:    processor,
		errorHandler: errorHandler,
	}
}

// Process calls the wrapped processor and handles any errors.
func (e *ErrorHandlingProcessor[T]) Process(ctx context.Context, data T, metrics *metrics.Collection) error {
	err := e.processor.Process(ctx, data, metrics)
	if err != nil && e.errorHandler != nil {
		return e.errorHandler(err)
	}
	return err
}

// BatchProcessor collects items and processes them in batches.
type BatchProcessor[T any] struct {
	processor Processor[[]T]
	batchSize int
	buffer    []T
}

// NewBatchProcessor creates a new BatchProcessor.
func NewBatchProcessor[T any](processor Processor[[]T], batchSize int) *BatchProcessor[T] {
	return &BatchProcessor[T]{
		processor: processor,
		batchSize: batchSize,
		buffer:    make([]T, 0, batchSize),
	}
}

// Process adds an item to the buffer and processes the batch when full.
func (b *BatchProcessor[T]) Process(ctx context.Context, data T, metrics *metrics.Collection) error {
	b.buffer = append(b.buffer, data)
	if len(b.buffer) >= b.batchSize {
		return b.FlushBatch(ctx, metrics)
	}
	return nil
}

// FlushBatch processes any remaining items in the buffer.
func (b *BatchProcessor[T]) FlushBatch(ctx context.Context, metrics *metrics.Collection) error {
	if len(b.buffer) == 0 {
		return nil
	}
	err := b.processor.Process(ctx, b.buffer, metrics)
	b.buffer = b.buffer[:0]
	return err
}

// BufferSize returns the current number of items in the buffer.
func (b *BatchProcessor[T]) BufferSize() int {
	return len(b.buffer)
}
