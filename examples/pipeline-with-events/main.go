// Package main demonstrates a simplified event indexer using the plugin system.
//
// This example shows:
// - How to register and use decoder plugins
// - How to parse transaction logs
// - How to extract and decode events
// - How to process events with custom handlers
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/decoder/anchor"
	"github.com/lugondev/go-carbon/internal/decoder/spl_token"
	"github.com/lugondev/go-carbon/pkg/decoder"
	"github.com/lugondev/go-carbon/pkg/log"
	"github.com/lugondev/go-carbon/pkg/plugin"
)

// EventIndexer indexes and processes blockchain events.
type EventIndexer struct {
	logger          *slog.Logger
	pluginRegistry  *plugin.Registry
	decoderRegistry *decoder.Registry
	logParser       *log.LogParser
	eventsProcessed uint64
}

// NewEventIndexer creates a new event indexer.
func NewEventIndexer(logger *slog.Logger) *EventIndexer {
	return &EventIndexer{
		logger:          logger,
		pluginRegistry:  plugin.NewRegistry(),
		decoderRegistry: decoder.NewRegistry(),
		logParser:       log.NewParser(),
		eventsProcessed: 0,
	}
}

// Initialize initializes the indexer and registers plugins.
func (idx *EventIndexer) Initialize(ctx context.Context) error {
	idx.logger.Info("Initializing event indexer")

	// Register built-in plugins
	if err := idx.registerBuiltInPlugins(); err != nil {
		return fmt.Errorf("failed to register built-in plugins: %w", err)
	}

	// Register custom plugins for your programs
	if err := idx.registerCustomPlugins(); err != nil {
		return fmt.Errorf("failed to register custom plugins: %w", err)
	}

	// Initialize all plugins
	if err := idx.pluginRegistry.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}

	// Get decoder registry from plugins
	idx.decoderRegistry = idx.pluginRegistry.GetDecoderRegistry()

	idx.logger.Info("Event indexer initialized",
		"plugins", len(idx.pluginRegistry.ListPlugins()),
		"decoders", len(idx.decoderRegistry.ListDecoders()),
	)

	return nil
}

// registerBuiltInPlugins registers built-in decoder plugins.
func (idx *EventIndexer) registerBuiltInPlugins() error {
	// Register SPL Token plugin
	splPlugin := spl_token.NewSPLTokenPlugin()
	if err := idx.pluginRegistry.Register(splPlugin); err != nil {
		return err
	}

	idx.logger.Info("Registered built-in plugins",
		"plugins", []string{"spl-token"},
	)

	return nil
}

// registerCustomPlugins registers custom plugins for your specific programs.
func (idx *EventIndexer) registerCustomPlugins() error {
	// Example: Register Anchor event plugin for your DEX program
	// Replace with your actual program ID
	programID := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	// Create your custom decoders
	customDecoders := []decoder.Decoder{
		// Add your event decoders here
	}

	anchorPlugin := anchor.NewAnchorEventPlugin(
		"my-custom-program",
		programID,
		customDecoders,
	)

	if err := idx.pluginRegistry.Register(anchorPlugin); err != nil {
		return err
	}

	// Register event processor
	eventProcessor := idx.createEventProcessor(programID)
	if err := idx.pluginRegistry.Register(eventProcessor); err != nil {
		return err
	}

	return nil
}

// createEventProcessor creates an event processor for handling decoded events.
func (idx *EventIndexer) createEventProcessor(programID solana.PublicKey) plugin.Plugin {
	return anchor.NewEventProcessorPlugin(
		"event-handler",
		programID,
		[]string{}, // Handle all event types
		func(ctx context.Context, event *decoder.Event) error {
			return idx.handleEvent(ctx, event)
		},
	)
}

// handleEvent handles a decoded event.
func (idx *EventIndexer) handleEvent(ctx context.Context, event *decoder.Event) error {
	idx.eventsProcessed++

	idx.logger.Info("Event received",
		"name", event.Name,
		"program", event.ProgramID.String(),
		"total_processed", idx.eventsProcessed,
	)

	// Your custom event handling logic:
	// - Save to database
	// - Send to message queue
	// - Update cache
	// - Trigger webhooks
	// - Update analytics

	return nil
}

// ProcessTransactionLogs processes transaction logs and extracts events.
func (idx *EventIndexer) ProcessTransactionLogs(ctx context.Context, logs []string, signature string) error {
	idx.logger.Debug("Processing transaction logs",
		"signature", signature,
		"num_logs", len(logs),
	)

	// Extract "Program data:" from logs
	programData := idx.logParser.ExtractProgramData(logs)
	if len(programData) == 0 {
		return nil // No events to process
	}

	idx.logger.Debug("Extracted program data",
		"count", len(programData),
		"signature", signature,
	)

	// Decode each event
	for i, data := range programData {
		event, err := idx.decoderRegistry.Decode(data, nil)
		if err != nil {
			idx.logger.Debug("Could not decode event",
				"index", i,
				"data_length", len(data),
				"error", err,
			)
			continue
		}

		if event != nil {
			// Process event through plugin registry
			if err := idx.pluginRegistry.ProcessEvent(ctx, event); err != nil {
				idx.logger.Error("Failed to process event",
					"event_name", event.Name,
					"error", err,
				)
			}
		}
	}

	return nil
}

// ProcessInstructionLogs processes logs for a specific instruction.
func (idx *EventIndexer) ProcessInstructionLogs(
	ctx context.Context,
	allLogs []string,
	instructionPath log.InstructionPath,
	programID solana.PublicKey,
) error {
	// Filter logs for this specific instruction
	filteredLogs := idx.logParser.FilterByInstructionPath(allLogs, instructionPath)
	if len(filteredLogs) == 0 {
		return nil
	}

	// Extract program data from filtered logs
	programData := idx.logParser.ExtractProgramData(filteredLogs)

	// Decode and process events
	for _, data := range programData {
		event, err := idx.decoderRegistry.Decode(data, &programID)
		if err != nil {
			continue
		}

		if event != nil {
			_ = idx.pluginRegistry.ProcessEvent(ctx, event)
		}
	}

	return nil
}

func main() {
	// Setup logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting event indexer demo")

	// Create event indexer
	indexer := NewEventIndexer(logger)

	// Initialize indexer and plugins
	ctx := context.Background()
	if err := indexer.Initialize(ctx); err != nil {
		logger.Error("Failed to initialize indexer", "error", err)
		os.Exit(1)
	}

	// Example 1: Process transaction logs
	logger.Info("\n=== Example 1: Processing Transaction Logs ===")
	exampleTransactionLogs := getExampleTransactionLogs()
	if err := indexer.ProcessTransactionLogs(ctx, exampleTransactionLogs, "example-signature-1"); err != nil {
		logger.Error("Failed to process transaction logs", "error", err)
	}

	// Example 2: Process instruction-specific logs
	logger.Info("\n=== Example 2: Processing Instruction-Specific Logs ===")
	instructionPath := log.InstructionPath{0, 0} // First outer, first inner
	programID := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	if err := indexer.ProcessInstructionLogs(ctx, exampleTransactionLogs, instructionPath, programID); err != nil {
		logger.Error("Failed to process instruction logs", "error", err)
	}

	// Example 3: Direct event decoding
	logger.Info("\n=== Example 3: Direct Event Decoding ===")
	demonstrateDirectDecoding(indexer)

	// Shutdown plugins
	if err := indexer.pluginRegistry.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown plugins", "error", err)
	}

	logger.Info("Demo completed",
		"events_processed", indexer.eventsProcessed,
	)
}

// getExampleTransactionLogs returns example transaction logs.
func getExampleTransactionLogs() []string {
	return []string{
		"Program 11111111111111111111111111111111 invoke [1]",
		"Program log: Instruction: Transfer",
		"Program data: AQAAAAAAAADoBBEAAAAAAA==", // Mock event data
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]",
		"Program log: Instruction: Transfer",
		"Program data: AgAAAAAAAADoAwAAAAAAAA==", // Another event
		"Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success",
		"Program 11111111111111111111111111111111 success",
		"Program log: Compute units consumed: 15000",
	}
}

// demonstrateDirectDecoding shows direct event decoding.
func demonstrateDirectDecoding(indexer *EventIndexer) {
	// Create example event data (8-byte discriminator + payload)
	eventData := make([]byte, 16)
	// In real scenario, this would be actual event data from logs

	// Try to decode
	event, err := indexer.decoderRegistry.Decode(eventData, nil)
	if err != nil {
		slog.Debug("Could not decode example event", "error", err)
		return
	}

	if event != nil {
		slog.Info("Successfully decoded event",
			"name", event.Name,
		)
	}
}

// ExampleWithCustomProcessor demonstrates creating a custom processor.
func ExampleWithCustomProcessor() {
	logger := slog.Default()

	// Create custom processor
	customProcessor := func(ctx context.Context, event *decoder.Event) error {
		logger.Info("Custom processor",
			"event", event.Name,
			"program", event.ProgramID.String(),
		)
		return nil
	}

	// Create event processor plugin
	programID := solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	processor := anchor.NewEventProcessorPlugin(
		"my-processor",
		programID,
		[]string{"MyEvent"},
		customProcessor,
	)

	// Register with registry
	registry := plugin.NewRegistry()
	registry.MustRegister(processor)

	logger.Info("Custom processor registered")
}

// ExampleLogFiltering demonstrates log filtering by instruction path.
func ExampleLogFiltering() {
	parser := log.NewParser()

	logs := []string{
		"Program AAA invoke [1]",
		"Program log: Outer",
		"Program data: outer_data",
		"Program BBB invoke [2]",
		"Program log: Inner",
		"Program data: inner_data",
		"Program BBB success",
		"Program AAA success",
	}

	// Extract all program data
	allData := parser.ExtractProgramData(logs)
	fmt.Printf("All data: %d items\n", len(allData))

	// Filter for inner instruction [0, 0]
	filteredLogs := parser.FilterByInstructionPath(logs, log.InstructionPath{0, 0})
	filteredData := parser.ExtractProgramData(filteredLogs)
	fmt.Printf("Filtered data: %d items\n", len(filteredData))
}
