// Package main demonstrates a basic pipeline setup for the go-carbon framework.
//
// This example shows how to:
// - Create a pipeline using the builder pattern
// - Set up a simple account monitor datasource
// - Process account updates with a custom processor
// - Configure metrics and logging
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/datasource/rpc"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/pipeline"
	"github.com/lugondev/go-carbon/internal/processor"
	"github.com/lugondev/go-carbon/pkg/types"
)

// RPC endpoint - use devnet for testing
const rpcEndpoint = "https://api.devnet.solana.com"

// AccountData represents decoded account data.
// In a real application, this would match your specific account structure.
type AccountData struct {
	Owner    types.Pubkey
	Lamports uint64
	DataLen  int
}

// AccountDataDecoder decodes account data into AccountData.
type AccountDataDecoder struct{}

// DecodeAccount implements the AccountDecoder interface.
func (d *AccountDataDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[AccountData] {
	if acc == nil {
		return nil
	}

	return &account.DecodedAccount[AccountData]{
		Lamports:   acc.Lamports,
		Owner:      acc.Owner,
		Executable: acc.Executable,
		RentEpoch:  acc.RentEpoch,
		Data: AccountData{
			Owner:    acc.Owner,
			Lamports: acc.Lamports,
			DataLen:  len(acc.Data),
		},
	}
}

// AccountProcessor processes decoded account data.
type AccountProcessor struct {
	logger *slog.Logger
}

// NewAccountProcessor creates a new AccountProcessor.
func NewAccountProcessor(logger *slog.Logger) *AccountProcessor {
	return &AccountProcessor{logger: logger}
}

// Process implements the Processor interface.
func (p *AccountProcessor) Process(
	ctx context.Context,
	input account.AccountProcessorInput[AccountData],
	m *metrics.Collection,
) error {
	data := input.DecodedAccount.Data

	p.logger.Info("Account updated",
		"pubkey", input.Metadata.Pubkey.String(),
		"slot", input.Metadata.Slot,
		"owner", data.Owner.String(),
		"lamports", data.Lamports,
		"data_len", data.DataLen,
		"sol_balance", types.LamportsToSOL(data.Lamports),
	)

	// Record custom metrics
	_ = m.IncrementCounter(ctx, "accounts_processed", 1)
	_ = m.UpdateGauge(ctx, "last_account_balance", float64(data.Lamports))

	return nil
}

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting go-carbon basic example")

	// Create RPC config
	rpcConfig := rpc.DefaultConfig(rpcEndpoint)
	rpcConfig.PollInterval = 5 * time.Second // Poll every 5 seconds

	// Accounts to monitor (you can add your own accounts here)
	accountsToMonitor := []solana.PublicKey{
		// System Program
		solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
		// Token Program
		solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"),
	}

	// Create the RPC datasource
	rpcDatasource := rpc.NewAccountMonitorDatasource(rpcConfig, accountsToMonitor)
	rpcDatasource.WithLogger(logger)

	// Create account decoder and processor
	decoder := &AccountDataDecoder{}
	proc := NewAccountProcessor(logger)

	// Create account pipe
	accountPipe := account.NewAccountPipe(decoder, proc)
	accountPipe.WithLogger(logger)

	// Create metrics collection with log metrics
	metricsCollection := metrics.NewCollection(
		metrics.NewLogMetrics(logger),
	)

	// Build the pipeline
	p := pipeline.Builder().
		Datasource(datasource.NewNamedDatasourceID("rpc-devnet"), rpcDatasource).
		AccountPipe(accountPipe).
		Metrics(metricsCollection).
		MetricsFlushInterval(10 * time.Second).
		ChannelBufferSize(100).
		WithGracefulShutdown().
		Logger(logger).
		Build()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run pipeline in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- p.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			logger.Error("Pipeline error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Pipeline stopped")
}

// ExampleWithProcessorFunc demonstrates using a ProcessorFunc for simple cases.
func ExampleWithProcessorFunc() {
	logger := slog.Default()

	// Create a simple processor using ProcessorFunc
	simpleProcessor := processor.ProcessorFunc[account.AccountProcessorInput[AccountData]](
		func(ctx context.Context, input account.AccountProcessorInput[AccountData], m *metrics.Collection) error {
			fmt.Printf("Account %s has %d lamports\n",
				input.Metadata.Pubkey.String(),
				input.DecodedAccount.Lamports,
			)
			return nil
		},
	)

	// Create account pipe with the simple processor
	decoder := &AccountDataDecoder{}
	accountPipe := account.NewAccountPipe(decoder, simpleProcessor)
	accountPipe.WithLogger(logger)

	fmt.Println("Created account pipe with ProcessorFunc")
}

// ExampleWithChainedProcessor demonstrates using chained processors.
func ExampleWithChainedProcessor() {
	logger := slog.Default()

	// Create multiple processors
	logProcessor := processor.ProcessorFunc[account.AccountProcessorInput[AccountData]](
		func(ctx context.Context, input account.AccountProcessorInput[AccountData], m *metrics.Collection) error {
			logger.Info("Log processor", "pubkey", input.Metadata.Pubkey.String())
			return nil
		},
	)

	metricsProcessor := processor.ProcessorFunc[account.AccountProcessorInput[AccountData]](
		func(ctx context.Context, input account.AccountProcessorInput[AccountData], m *metrics.Collection) error {
			_ = m.IncrementCounter(ctx, "processed", 1)
			return nil
		},
	)

	// Chain them together
	chainedProc := processor.NewChainedProcessor(logProcessor, metricsProcessor)

	// Create account pipe with chained processor
	decoder := &AccountDataDecoder{}
	accountPipe := account.NewAccountPipe(decoder, chainedProc)
	accountPipe.WithLogger(logger)

	fmt.Println("Created account pipe with ChainedProcessor")
}

// ExampleWithConditionalProcessor demonstrates using conditional processing.
func ExampleWithConditionalProcessor() {
	logger := slog.Default()

	// Create a processor that only logs
	baseProcessor := processor.ProcessorFunc[account.AccountProcessorInput[AccountData]](
		func(ctx context.Context, input account.AccountProcessorInput[AccountData], m *metrics.Collection) error {
			logger.Info("High-value account detected!",
				"pubkey", input.Metadata.Pubkey.String(),
				"lamports", input.DecodedAccount.Lamports,
			)
			return nil
		},
	)

	// Only process accounts with more than 1 SOL
	conditionalProc := processor.NewConditionalProcessor(
		baseProcessor,
		func(input account.AccountProcessorInput[AccountData]) bool {
			return input.DecodedAccount.Lamports > types.LamportsPerSOL
		},
	)

	// Create account pipe with conditional processor
	decoder := &AccountDataDecoder{}
	accountPipe := account.NewAccountPipe(decoder, conditionalProc)
	accountPipe.WithLogger(logger)

	fmt.Println("Created account pipe with ConditionalProcessor")
}
