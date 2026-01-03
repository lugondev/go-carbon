// Package pipeline provides the PipelineBuilder for constructing Pipeline instances.
package pipeline

import (
	"log/slog"
	"time"

	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/instruction"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/transaction"
)

// PipelineBuilder provides a fluent API for constructing a Pipeline.
type PipelineBuilder struct {
	pipeline *Pipeline
}

// NewPipelineBuilder creates a new PipelineBuilder with default settings.
func NewPipelineBuilder() *PipelineBuilder {
	return &PipelineBuilder{
		pipeline: NewPipeline(),
	}
}

// Datasource adds a data source to the pipeline.
func (b *PipelineBuilder) Datasource(id datasource.DatasourceID, ds datasource.Datasource) *PipelineBuilder {
	b.pipeline.Datasources = append(b.pipeline.Datasources, DatasourceWithID{
		ID:         id,
		Datasource: ds,
	})
	return b
}

// AccountPipe adds an account pipe to the pipeline.
// The pipe must implement AccountPipeRunner interface.
func (b *PipelineBuilder) AccountPipe(pipe account.AccountPipeRunner) *PipelineBuilder {
	b.pipeline.AccountPipes = append(b.pipeline.AccountPipes, pipe)
	return b
}

// AccountDeletionPipe adds an account deletion pipe to the pipeline.
func (b *PipelineBuilder) AccountDeletionPipe(pipe AccountDeletionPipeRunner) *PipelineBuilder {
	b.pipeline.AccountDeletionPipes = append(b.pipeline.AccountDeletionPipes, pipe)
	return b
}

// BlockDetailsPipe adds a block details pipe to the pipeline.
func (b *PipelineBuilder) BlockDetailsPipe(pipe BlockDetailsPipeRunner) *PipelineBuilder {
	b.pipeline.BlockDetailsPipes = append(b.pipeline.BlockDetailsPipes, pipe)
	return b
}

// InstructionPipe adds an instruction pipe to the pipeline.
// The pipe must implement InstructionPipeRunner interface.
func (b *PipelineBuilder) InstructionPipe(pipe instruction.InstructionPipeRunner) *PipelineBuilder {
	b.pipeline.InstructionPipes = append(b.pipeline.InstructionPipes, pipe)
	return b
}

// TransactionPipe adds a transaction pipe to the pipeline.
// The pipe must implement TransactionPipeRunner interface.
func (b *PipelineBuilder) TransactionPipe(pipe transaction.TransactionPipeRunner) *PipelineBuilder {
	b.pipeline.TransactionPipes = append(b.pipeline.TransactionPipes, pipe)
	return b
}

// Metrics sets a custom metrics collection for the pipeline.
func (b *PipelineBuilder) Metrics(mc *metrics.Collection) *PipelineBuilder {
	b.pipeline.Metrics = mc
	return b
}

// MetricsFlushInterval sets the interval for flushing metrics.
func (b *PipelineBuilder) MetricsFlushInterval(interval time.Duration) *PipelineBuilder {
	b.pipeline.MetricsFlushInterval = interval
	return b
}

// ShutdownStrategy sets the shutdown strategy for the pipeline.
func (b *PipelineBuilder) ShutdownStrategy(strategy ShutdownStrategy) *PipelineBuilder {
	b.pipeline.ShutdownStrategy = strategy
	return b
}

// ChannelBufferSize sets the buffer size for the update channel.
func (b *PipelineBuilder) ChannelBufferSize(size int) *PipelineBuilder {
	b.pipeline.ChannelBufferSize = size
	return b
}

// Logger sets a custom logger for the pipeline.
func (b *PipelineBuilder) Logger(logger *slog.Logger) *PipelineBuilder {
	b.pipeline.Logger = logger
	return b
}

// Build returns the constructed Pipeline.
func (b *PipelineBuilder) Build() *Pipeline {
	return b.pipeline
}

// WithDefaultMetrics adds default metrics to the pipeline.
func (b *PipelineBuilder) WithDefaultMetrics() *PipelineBuilder {
	b.pipeline.Metrics = metrics.NewCollection()
	return b
}

// WithGracefulShutdown configures the pipeline for graceful shutdown.
func (b *PipelineBuilder) WithGracefulShutdown() *PipelineBuilder {
	b.pipeline.ShutdownStrategy = ShutdownStrategyProcessPending
	return b
}

// WithImmediateShutdown configures the pipeline for immediate shutdown.
func (b *PipelineBuilder) WithImmediateShutdown() *PipelineBuilder {
	b.pipeline.ShutdownStrategy = ShutdownStrategyImmediate
	return b
}

// AddAccountPipeTyped is a generic helper that creates and adds an AccountPipe.
// This is useful when you want to add a typed account pipe directly.
func AddAccountPipeTyped[T any](
	b *PipelineBuilder,
	pipe *account.AccountPipe[T],
) *PipelineBuilder {
	b.pipeline.AccountPipes = append(b.pipeline.AccountPipes, pipe)
	return b
}

// AddInstructionPipeTyped is a generic helper that creates and adds an InstructionPipe.
// This is useful when you want to add a typed instruction pipe directly.
func AddInstructionPipeTyped[T any](
	b *PipelineBuilder,
	pipe *instruction.InstructionPipe[T],
) *PipelineBuilder {
	b.pipeline.InstructionPipes = append(b.pipeline.InstructionPipes, pipe)
	return b
}

// AddTransactionPipeTyped is a generic helper that creates and adds a TransactionPipe.
// This is useful when you want to add a typed transaction pipe directly.
func AddTransactionPipeTyped[T any, U any](
	b *PipelineBuilder,
	pipe *transaction.TransactionPipe[T, U],
) *PipelineBuilder {
	b.pipeline.TransactionPipes = append(b.pipeline.TransactionPipes, pipe)
	return b
}
