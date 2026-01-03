// Package pipeline defines the Pipeline struct and related components for processing
// blockchain data updates.
//
// The Pipeline is central to the carbon-core framework, offering a flexible and
// extensible data processing architecture that supports various blockchain data types,
// including account updates, transaction details, and account deletions. The pipeline
// integrates multiple data sources and processing pipes to handle and transform incoming
// data, while recording performance metrics for monitoring and analysis.
//
// # Overview
//
// This package provides the Pipeline struct, which orchestrates data flow from multiple
// sources, processes it through designated pipes, and captures metrics at each stage.
// The pipeline is highly customizable and can be configured with various components
// to suit specific data handling requirements.
//
// # Key Components
//
//   - Datasources: Provide raw data updates, which may include account or transaction details.
//   - Account, Instruction, and Transaction Pipes: Modular units that decode and process
//     specific types of data.
//   - Metrics: Collects data on pipeline performance, such as processing times and error rates.
package pipeline

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/internal/datasource"
	cerrors "github.com/lugondev/go-carbon/internal/errors"
	"github.com/lugondev/go-carbon/internal/filter"
	"github.com/lugondev/go-carbon/internal/instruction"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/transaction"
	"github.com/lugondev/go-carbon/pkg/types"
)

// ShutdownStrategy defines the shutdown behavior for the pipeline.
type ShutdownStrategy int

const (
	// ShutdownStrategyProcessPending terminates the datasources and finishes
	// processing all pending updates. This is the default behavior.
	ShutdownStrategyProcessPending ShutdownStrategy = iota

	// ShutdownStrategyImmediate stops the entire pipeline immediately.
	ShutdownStrategyImmediate
)

// DefaultChannelBufferSize is the default size of the channel buffer for the pipeline.
const DefaultChannelBufferSize = 1000

// DefaultMetricsFlushInterval is the default interval for flushing metrics.
const DefaultMetricsFlushInterval = 5 * time.Second

// Pipeline represents the primary data processing pipeline in the carbon-core framework.
//
// The Pipeline struct is responsible for orchestrating the flow of data from various
// sources, processing it through multiple pipes (for accounts, transactions, instructions,
// and account deletions), and recording metrics at each stage.
type Pipeline struct {
	// Datasources are the data sources that provide updates to the pipeline.
	Datasources []DatasourceWithID

	// AccountPipes handle account updates.
	AccountPipes []account.AccountPipeRunner

	// AccountDeletionPipes handle account deletions.
	AccountDeletionPipes []AccountDeletionPipeRunner

	// BlockDetailsPipes handle block details updates.
	BlockDetailsPipes []BlockDetailsPipeRunner

	// InstructionPipes handle instructions within transactions.
	InstructionPipes []instruction.InstructionPipeRunner

	// TransactionPipes handle complete transaction payloads.
	TransactionPipes []transaction.TransactionPipeRunner

	// Metrics collects performance data.
	Metrics *metrics.Collection

	// MetricsFlushInterval defines how frequently metrics should be flushed.
	MetricsFlushInterval time.Duration

	// ShutdownStrategy determines how the pipeline behaves on shutdown.
	ShutdownStrategy ShutdownStrategy

	// ChannelBufferSize is the size of the channel buffer for updates.
	ChannelBufferSize int

	// Logger is used for logging.
	Logger *slog.Logger

	// cancelFunc is used to cancel the pipeline context.
	cancelFunc context.CancelFunc

	// mu protects pipeline state during modifications.
	mu sync.RWMutex
}

// DatasourceWithID pairs a datasource with its unique identifier.
type DatasourceWithID struct {
	ID         datasource.DatasourceID
	Datasource datasource.Datasource
}

// AccountDeletionPipeRunner is an interface for running account deletion pipes.
type AccountDeletionPipeRunner interface {
	RunAccountDeletion(
		ctx context.Context,
		deletion *datasource.AccountDeletion,
		metricsCollection *metrics.Collection,
	) error
	GetFilters() []filter.Filter
}

// BlockDetailsPipeRunner is an interface for running block details pipes.
type BlockDetailsPipeRunner interface {
	RunBlockDetails(
		ctx context.Context,
		details *datasource.BlockDetails,
		metricsCollection *metrics.Collection,
	) error
	GetFilters() []filter.Filter
}

// NewPipeline creates a new Pipeline with default settings.
func NewPipeline() *Pipeline {
	return &Pipeline{
		Datasources:          make([]DatasourceWithID, 0),
		AccountPipes:         make([]account.AccountPipeRunner, 0),
		AccountDeletionPipes: make([]AccountDeletionPipeRunner, 0),
		BlockDetailsPipes:    make([]BlockDetailsPipeRunner, 0),
		InstructionPipes:     make([]instruction.InstructionPipeRunner, 0),
		TransactionPipes:     make([]transaction.TransactionPipeRunner, 0),
		Metrics:              metrics.NewCollection(),
		MetricsFlushInterval: DefaultMetricsFlushInterval,
		ShutdownStrategy:     ShutdownStrategyProcessPending,
		ChannelBufferSize:    DefaultChannelBufferSize,
		Logger:               slog.Default(),
	}
}

// Builder returns a new PipelineBuilder for constructing a Pipeline.
func Builder() *PipelineBuilder {
	return NewPipelineBuilder()
}

// Run starts the Pipeline and processes updates from data sources.
//
// The Run method initializes the pipeline's metrics system and starts listening
// for updates from the configured data sources. It processes each update received
// from the data sources, logging and updating metrics based on the success or
// failure of each operation.
func (p *Pipeline) Run(ctx context.Context) error {
	p.Logger.Info("starting pipeline",
		"num_datasources", len(p.Datasources),
		"num_metrics", p.Metrics.Len(),
		"num_account_pipes", len(p.AccountPipes),
		"num_account_deletion_pipes", len(p.AccountDeletionPipes),
		"num_instruction_pipes", len(p.InstructionPipes),
		"num_transaction_pipes", len(p.TransactionPipes),
	)

	// Initialize metrics
	if err := p.Metrics.Initialize(ctx); err != nil {
		return cerrors.Wrap(err, "failed to initialize metrics")
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(ctx)
	p.cancelFunc = cancel
	defer cancel()

	// Create the update channel
	updateChan := make(chan datasource.UpdateWithSource, p.ChannelBufferSize)

	// Start datasources
	var wg sync.WaitGroup
	for _, ds := range p.Datasources {
		wg.Add(1)
		go func(dsWithID DatasourceWithID) {
			defer wg.Done()
			if err := dsWithID.Datasource.Consume(ctx, dsWithID.ID, updateChan, p.Metrics); err != nil {
				p.Logger.Error("error consuming datasource",
					"datasource_id", dsWithID.ID.String(),
					"error", err,
				)
			}
		}(ds)
	}

	// Close the update channel when all datasources are done
	go func() {
		wg.Wait()
		close(updateChan)
	}()

	// Set up metrics flush ticker
	flushTicker := time.NewTicker(p.MetricsFlushInterval)
	defer flushTicker.Stop()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main processing loop
	for {
		select {
		case <-ctx.Done():
			p.Logger.Info("context cancelled, shutting down")
			return p.shutdown(ctx)

		case sig := <-sigChan:
			p.Logger.Info("received signal, shutting down", "signal", sig)
			cancel() // Cancel the context to stop datasources

			if p.ShutdownStrategy == ShutdownStrategyImmediate {
				p.Logger.Info("shutting down immediately")
				return p.shutdown(ctx)
			}

			p.Logger.Info("shutting down after processing pending updates")
			// Continue processing until channel is closed

		case <-flushTicker.C:
			if err := p.Metrics.Flush(ctx); err != nil {
				p.Logger.Error("failed to flush metrics", "error", err)
			}

		case update, ok := <-updateChan:
			if !ok {
				// Channel closed, all datasources finished
				p.Logger.Info("update channel closed, shutting down")
				return p.shutdown(ctx)
			}

			// Record metrics
			if err := p.Metrics.IncrementCounter(ctx, metrics.MetricUpdatesReceived, 1); err != nil {
				p.Logger.Error("failed to increment counter", "error", err)
			}

			// Process the update
			start := time.Now()
			err := p.process(ctx, update)
			elapsed := time.Since(start)

			// Record processing time
			_ = p.Metrics.RecordHistogram(ctx, metrics.MetricUpdatesProcessTimeNanoseconds, float64(elapsed.Nanoseconds()))
			_ = p.Metrics.RecordHistogram(ctx, metrics.MetricUpdatesProcessTimeMilliseconds, float64(elapsed.Milliseconds()))

			if err != nil {
				p.Logger.Error("error processing update",
					"type", update.Update.Type.String(),
					"error", err,
				)
				_ = p.Metrics.IncrementCounter(ctx, metrics.MetricUpdatesFailed, 1)
			} else {
				_ = p.Metrics.IncrementCounter(ctx, metrics.MetricUpdatesSuccessful, 1)
			}

			_ = p.Metrics.IncrementCounter(ctx, metrics.MetricUpdatesProcessed, 1)
			_ = p.Metrics.UpdateGauge(ctx, metrics.MetricUpdatesQueued, float64(len(updateChan)))
		}
	}
}

// Stop gracefully stops the pipeline.
func (p *Pipeline) Stop() {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
}

// shutdown performs cleanup operations before the pipeline exits.
func (p *Pipeline) shutdown(ctx context.Context) error {
	p.Logger.Info("pipeline shutdown starting")

	// Flush final metrics
	if err := p.Metrics.Flush(ctx); err != nil {
		p.Logger.Error("failed to flush metrics during shutdown", "error", err)
	}

	// Shutdown metrics
	if err := p.Metrics.Shutdown(ctx); err != nil {
		p.Logger.Error("failed to shutdown metrics", "error", err)
	}

	p.Logger.Info("pipeline shutdown complete")
	return nil
}

// process handles a single update, routing it to the appropriate pipes.
func (p *Pipeline) process(ctx context.Context, update datasource.UpdateWithSource) error {
	p.Logger.Debug("processing update",
		"type", update.Update.Type.String(),
		"datasource_id", update.DatasourceID.String(),
	)

	switch update.Update.Type {
	case datasource.UpdateTypeAccount:
		return p.processAccountUpdate(ctx, update.DatasourceID, update.Update.Account)

	case datasource.UpdateTypeTransaction:
		return p.processTransactionUpdate(ctx, update.DatasourceID, update.Update.Transaction)

	case datasource.UpdateTypeAccountDeletion:
		return p.processAccountDeletion(ctx, update.DatasourceID, update.Update.AccountDeletion)

	case datasource.UpdateTypeBlockDetails:
		return p.processBlockDetails(ctx, update.DatasourceID, update.Update.BlockDetails)

	default:
		p.Logger.Warn("unknown update type", "type", update.Update.Type)
		return nil
	}
}

// processAccountUpdate processes an account update through all account pipes.
func (p *Pipeline) processAccountUpdate(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	update *datasource.AccountUpdate,
) error {
	if update == nil {
		return nil
	}

	metadata := account.NewAccountMetadata(update)

	for _, pipe := range p.AccountPipes {
		// Check filters
		filters := pipe.GetFilters()
		passesFilters := true
		for _, f := range filters {
			if !f.FilterAccount(datasourceID, &filter.AccountMetadata{
				Slot:                 metadata.Slot,
				Pubkey:               metadata.Pubkey,
				TransactionSignature: metadata.TransactionSignature,
			}, &update.Account) {
				passesFilters = false
				break
			}
		}

		if !passesFilters {
			continue
		}

		if err := pipe.RunAccount(ctx, metadata, &update.Account, p.Metrics); err != nil {
			return err
		}
	}

	_ = p.Metrics.IncrementCounter(ctx, metrics.MetricAccountUpdatesProcessed, 1)
	return nil
}

// processTransactionUpdate processes a transaction update through instruction and transaction pipes.
func (p *Pipeline) processTransactionUpdate(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	update *datasource.TransactionUpdate,
) error {
	if update == nil {
		return nil
	}

	// Create transaction metadata
	txMetadata, err := transaction.NewTransactionMetadataFromUpdate(update)
	if err != nil {
		return cerrors.Wrap(err, "failed to create transaction metadata")
	}

	// Extract and nest instructions
	instructionsWithMetadata := p.extractInstructionsWithMetadata(txMetadata, update)
	nestedInstructions := instruction.NestInstructions(instructionsWithMetadata)

	// Process through instruction pipes
	for _, pipe := range p.InstructionPipes {
		for _, nestedIx := range nestedInstructions.Instructions {
			// Check filters
			filters := pipe.GetFilters()
			passesFilters := true
			for _, f := range filters {
				if !f.FilterInstruction(datasourceID, nestedIx) {
					passesFilters = false
					break
				}
			}

			if !passesFilters {
				continue
			}

			if err := pipe.RunInstruction(ctx, nestedIx, p.Metrics); err != nil {
				return err
			}
		}
	}

	// Process through transaction pipes
	for _, pipe := range p.TransactionPipes {
		// Check filters
		filters := pipe.GetFilters()
		passesFilters := true
		for _, f := range filters {
			if !f.FilterTransaction(datasourceID, txMetadata, nestedInstructions) {
				passesFilters = false
				break
			}
		}

		if !passesFilters {
			continue
		}

		if err := pipe.RunTransaction(ctx, txMetadata, nestedInstructions, p.Metrics); err != nil {
			return err
		}
	}

	_ = p.Metrics.IncrementCounter(ctx, metrics.MetricTransactionUpdatesProcessed, 1)
	return nil
}

// extractInstructionsWithMetadata extracts instructions from a transaction update with metadata.
func (p *Pipeline) extractInstructionsWithMetadata(
	txMetadata *transaction.TransactionMetadata,
	update *datasource.TransactionUpdate,
) instruction.InstructionsWithMetadata {
	var result instruction.InstructionsWithMetadata

	if update.Transaction == nil || update.Transaction.Message.Instructions == nil {
		return result
	}

	metadataRef := txMetadata.ToInstructionMetadataRef()

	// Process outer instructions
	for i, compiledIx := range update.Transaction.Message.Instructions {
		// Convert compiled instruction to full instruction
		ix := p.compiledToInstruction(compiledIx, txMetadata.AccountKeys)
		if ix == nil {
			continue
		}

		metadata := &instruction.InstructionMetadata{
			TransactionMetadata: metadataRef,
			StackHeight:         1,
			Index:               uint32(i + 1),
			AbsolutePath:        []uint8{uint8(i)},
		}

		result = append(result, instruction.InstructionWithMetadata{
			Metadata:    metadata,
			Instruction: ix,
		})
	}

	// Process inner instructions
	if update.Meta.InnerInstructions != nil {
		for _, innerGroup := range update.Meta.InnerInstructions {
			for j, innerIx := range innerGroup.Instructions {
				ix := p.compiledInnerToInstruction(innerIx, txMetadata.AccountKeys)
				if ix == nil {
					continue
				}

				stackHeight := uint32(2)
				if innerIx.StackHeight != nil {
					stackHeight = *innerIx.StackHeight
				}

				metadata := &instruction.InstructionMetadata{
					TransactionMetadata: metadataRef,
					StackHeight:         stackHeight,
					Index:               uint32(j + 1),
					AbsolutePath:        []uint8{innerGroup.Index, uint8(j)},
				}

				result = append(result, instruction.InstructionWithMetadata{
					Metadata:    metadata,
					Instruction: ix,
				})
			}
		}
	}

	return result
}

// compiledToInstruction converts a compiled instruction to a full instruction.
func (p *Pipeline) compiledToInstruction(
	compiled interface{},
	accountKeys []types.Pubkey,
) *types.Instruction {
	// This is a simplified implementation - in practice you'd need to handle
	// the specific transaction format from solana-go
	return nil
}

// compiledInnerToInstruction converts a compiled inner instruction to a full instruction.
func (p *Pipeline) compiledInnerToInstruction(
	inner types.InnerInstruction,
	accountKeys []types.Pubkey,
) *types.Instruction {
	// This is a simplified implementation - in practice you'd need to handle
	// the specific transaction format from solana-go
	return nil
}

// processAccountDeletion processes an account deletion through all deletion pipes.
func (p *Pipeline) processAccountDeletion(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	deletion *datasource.AccountDeletion,
) error {
	if deletion == nil {
		return nil
	}

	for _, pipe := range p.AccountDeletionPipes {
		// Check filters
		filters := pipe.GetFilters()
		passesFilters := true
		for _, f := range filters {
			if !f.FilterAccountDeletion(datasourceID, deletion) {
				passesFilters = false
				break
			}
		}

		if !passesFilters {
			continue
		}

		if err := pipe.RunAccountDeletion(ctx, deletion, p.Metrics); err != nil {
			return err
		}
	}

	_ = p.Metrics.IncrementCounter(ctx, metrics.MetricAccountDeletionsProcessed, 1)
	return nil
}

// processBlockDetails processes block details through all block details pipes.
func (p *Pipeline) processBlockDetails(
	ctx context.Context,
	datasourceID datasource.DatasourceID,
	details *datasource.BlockDetails,
) error {
	if details == nil {
		return nil
	}

	for _, pipe := range p.BlockDetailsPipes {
		// Check filters
		filters := pipe.GetFilters()
		passesFilters := true
		for _, f := range filters {
			if !f.FilterBlockDetails(datasourceID, details) {
				passesFilters = false
				break
			}
		}

		if !passesFilters {
			continue
		}

		if err := pipe.RunBlockDetails(ctx, details, p.Metrics); err != nil {
			return err
		}
	}

	_ = p.Metrics.IncrementCounter(ctx, metrics.MetricBlockDetailsProcessed, 1)
	return nil
}
