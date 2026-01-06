// Package main demonstrates the go-carbon plugin system with event parsing.
//
// This example shows how to:
// - Register custom decoder plugins
// - Parse "Program data:" from transaction logs
// - Decode Anchor events
// - Process events with custom handlers
// - Build a complete event indexing pipeline
package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/decoder/anchor"
	"github.com/lugondev/go-carbon/internal/decoder/spl_token"
	"github.com/lugondev/go-carbon/pkg/decoder"
	"github.com/lugondev/go-carbon/pkg/log"
	"github.com/lugondev/go-carbon/pkg/plugin"
)

// Example program ID (replace with your actual program)
var ExampleProgramID = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

// Example event: SwapExecuted
type SwapExecutedEvent struct {
	User      solana.PublicKey
	TokenIn   solana.PublicKey
	TokenOut  solana.PublicKey
	AmountIn  uint64
	AmountOut uint64
	Fee       uint64
	Timestamp int64
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting event parser example")

	// Step 1: Create plugin registry
	registry := plugin.NewRegistry()

	// Step 2: Register built-in plugins
	registerBuiltInPlugins(registry)

	// Step 3: Register custom plugins
	registerCustomPlugins(registry)

	// Step 4: Initialize all plugins
	ctx := context.Background()
	if err := registry.Initialize(ctx); err != nil {
		logger.Error("Failed to initialize plugins", "error", err)
		os.Exit(1)
	}

	// Step 5: Example transaction logs (simulate real data)
	exampleLogs := getExampleTransactionLogs()

	// Step 6: Parse logs
	logger.Info("Parsing transaction logs", "num_logs", len(exampleLogs))
	parseAndProcessLogs(ctx, registry, exampleLogs)

	// Step 7: Example with direct event decoding
	logger.Info("\n=== Direct Event Decoding Example ===")
	demonstrateDirectDecoding(registry)

	// Step 8: Shutdown
	if err := registry.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown plugins", "error", err)
	}

	logger.Info("Example completed successfully")
}

// registerBuiltInPlugins registers the built-in decoder plugins.
func registerBuiltInPlugins(registry *plugin.Registry) {
	// Register SPL Token plugin
	splTokenPlugin := spl_token.NewSPLTokenPlugin()
	if err := registry.Register(splTokenPlugin); err != nil {
		panic(fmt.Sprintf("failed to register SPL Token plugin: %v", err))
	}

	slog.Info("Registered SPL Token plugin")
}

// registerCustomPlugins registers custom decoder plugins for your program.
func registerCustomPlugins(registry *plugin.Registry) {
	// Create Anchor event decoders for your program
	decoders := createCustomDecoders()

	// Create Anchor event plugin
	anchorPlugin := anchor.NewAnchorEventPlugin(
		"my-dex",
		ExampleProgramID,
		decoders,
	)

	if err := registry.Register(anchorPlugin); err != nil {
		panic(fmt.Sprintf("failed to register anchor plugin: %v", err))
	}

	// Create and register event processor
	eventProcessor := createEventProcessor()
	if err := registry.Register(eventProcessor); err != nil {
		panic(fmt.Sprintf("failed to register event processor: %v", err))
	}

	slog.Info("Registered custom plugins", "program_id", ExampleProgramID.String())
}

// createCustomDecoders creates decoders for your custom Anchor events.
func createCustomDecoders() []decoder.Decoder {
	// Compute discriminator for SwapExecuted event
	// In Anchor: discriminator = sha256("event:SwapExecuted")[..8]
	swapDiscriminator := computeAnchorDiscriminator("SwapExecuted")

	swapDecoder := anchor.NewAnchorEventDecoder(
		"SwapExecuted",
		ExampleProgramID,
		swapDiscriminator,
		func(data []byte) (interface{}, error) {
			return decodeSwapExecutedEvent(data)
		},
	)

	// Add more decoders for other events
	return []decoder.Decoder{
		swapDecoder,
		// Add more event decoders here
	}
}

// createEventProcessor creates a custom event processor.
func createEventProcessor() plugin.Plugin {
	return anchor.NewEventProcessorPlugin(
		"my-dex-processor",
		ExampleProgramID,
		[]string{"SwapExecuted"}, // Event types to handle
		func(ctx context.Context, event *decoder.Event) error {
			// Process the decoded event
			slog.Info("Processing event",
				"name", event.Name,
				"program_id", event.ProgramID.String(),
			)

			// Type assert and handle specific event
			if swapEvent, ok := event.Data.(*SwapExecutedEvent); ok {
				handleSwapEvent(ctx, swapEvent)
			}

			return nil
		},
	)
}

// parseAndProcessLogs demonstrates parsing logs and extracting events.
func parseAndProcessLogs(ctx context.Context, registry *plugin.Registry, logs []string) {
	// Create log parser
	parser := log.NewParser()

	// Parse all logs
	parsedLogs := parser.ParseAll(logs)

	slog.Info("Parsed logs", "total", len(parsedLogs))

	// Extract "Program data:" entries
	programData := parser.ExtractProgramData(logs)
	slog.Info("Extracted program data", "count", len(programData))

	// Get decoder registry from plugins
	decoderRegistry := registry.GetDecoderRegistry()

	// Decode each program data entry
	for i, data := range programData {
		event, err := decoderRegistry.Decode(data, &ExampleProgramID)
		if err != nil {
			slog.Debug("Failed to decode event", "index", i, "error", err)
			continue
		}

		if event != nil {
			slog.Info("Decoded event",
				"name", event.Name,
				"program_id", event.ProgramID.String(),
			)

			// Process through event processors
			if err := registry.ProcessEvent(ctx, event); err != nil {
				slog.Error("Failed to process event", "error", err)
			}
		}
	}
}

// demonstrateDirectDecoding shows direct event decoding without logs.
func demonstrateDirectDecoding(registry *plugin.Registry) {
	// Create example event data (simulate Anchor event with discriminator)
	eventData := createMockSwapEventData()

	decoderRegistry := registry.GetDecoderRegistry()

	// Decode the event
	event, err := decoderRegistry.Decode(eventData, &ExampleProgramID)
	if err != nil {
		slog.Error("Failed to decode event", "error", err)
		return
	}

	if event != nil {
		slog.Info("Successfully decoded event",
			"name", event.Name,
			"data", event.Data,
		)
	}
}

// computeAnchorDiscriminator computes the 8-byte Anchor event discriminator.
func computeAnchorDiscriminator(eventName string) decoder.AnchorDiscriminator {
	// Anchor uses: sha256(format!("event:{}", event_name))[..8]
	data := []byte(fmt.Sprintf("event:%s", eventName))
	hash := sha256.Sum256(data)
	return decoder.NewAnchorDiscriminator(hash[:8])
}

// decodeSwapExecutedEvent decodes the SwapExecuted event from Borsh data.
func decodeSwapExecutedEvent(data []byte) (*SwapExecutedEvent, error) {
	// Expected structure (all little-endian):
	// - user: 32 bytes (Pubkey)
	// - token_in: 32 bytes (Pubkey)
	// - token_out: 32 bytes (Pubkey)
	// - amount_in: 8 bytes (u64)
	// - amount_out: 8 bytes (u64)
	// - fee: 8 bytes (u64)
	// - timestamp: 8 bytes (i64)
	// Total: 120 bytes

	if len(data) < 120 {
		return nil, fmt.Errorf("insufficient data for SwapExecuted event: need 120 bytes, got %d", len(data))
	}

	event := &SwapExecutedEvent{}
	offset := 0

	// Decode user pubkey (32 bytes)
	copy(event.User[:], data[offset:offset+32])
	offset += 32

	// Decode token_in pubkey (32 bytes)
	copy(event.TokenIn[:], data[offset:offset+32])
	offset += 32

	// Decode token_out pubkey (32 bytes)
	copy(event.TokenOut[:], data[offset:offset+32])
	offset += 32

	// Decode amount_in (8 bytes)
	if amount, err := decoder.DecodeU64LE(data[offset : offset+8]); err == nil {
		event.AmountIn = amount
	} else {
		return nil, err
	}
	offset += 8

	// Decode amount_out (8 bytes)
	if amount, err := decoder.DecodeU64LE(data[offset : offset+8]); err == nil {
		event.AmountOut = amount
	} else {
		return nil, err
	}
	offset += 8

	// Decode fee (8 bytes)
	if fee, err := decoder.DecodeU64LE(data[offset : offset+8]); err == nil {
		event.Fee = fee
	} else {
		return nil, err
	}
	offset += 8

	// Decode timestamp (8 bytes, signed)
	if ts, err := decoder.DecodeU64LE(data[offset : offset+8]); err == nil {
		event.Timestamp = int64(ts)
	} else {
		return nil, err
	}

	return event, nil
}

// handleSwapEvent processes a swap event (your custom business logic).
func handleSwapEvent(ctx context.Context, event *SwapExecutedEvent) {
	slog.Info("Swap executed",
		"user", event.User.String(),
		"token_in", event.TokenIn.String(),
		"token_out", event.TokenOut.String(),
		"amount_in", event.AmountIn,
		"amount_out", event.AmountOut,
		"fee", event.Fee,
		"timestamp", time.Unix(event.Timestamp, 0).Format(time.RFC3339),
	)

	// Your custom logic here:
	// - Save to database
	// - Send notifications
	// - Update analytics
	// - Trigger webhooks
	// etc.
}

// getExampleTransactionLogs returns example transaction logs.
func getExampleTransactionLogs() []string {
	return []string{
		"Program 11111111111111111111111111111111 invoke [1]",
		"Program log: Instruction: Swap",
		"Program data: AwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==", // Mock event data
		"Program 11111111111111111111111111111111 success",
		"Program log: Compute units consumed: 15000",
	}
}

// createMockSwapEventData creates mock Anchor event data with discriminator.
func createMockSwapEventData() []byte {
	// Compute discriminator
	disc := computeAnchorDiscriminator("SwapExecuted")

	// Create mock event data (discriminator + event fields)
	data := make([]byte, 8+120) // 8 bytes discriminator + 120 bytes event data

	// Copy discriminator
	copy(data[0:8], disc[:])

	// Fill with example data (in real scenario, this comes from transaction logs)
	// For demo purposes, we'll use zero values
	// In production, this would be actual event data from Solana

	return data
}

// ExampleWithCustomFilter demonstrates filtering events by instruction path.
func ExampleWithCustomFilter() {
	parser := log.NewParser()

	// Example logs with nested instructions
	logs := []string{
		"Program AAA invoke [1]",
		"Program log: Outer instruction",
		"Program BBB invoke [2]",
		"Program data: inner_event_data",
		"Program BBB success",
		"Program AAA success",
	}

	// Filter logs from specific instruction path
	// Path [0, 0] means: first outer instruction, first inner instruction
	targetPath := log.InstructionPath{0, 0}
	filteredLogs := parser.FilterByInstructionPath(logs, targetPath)

	slog.Info("Filtered logs", "count", len(filteredLogs))
}

// ExampleWithLogProcessor demonstrates custom log processing.
func ExampleWithLogProcessor() {
	// Create custom log processor
	customProcessor := log.ProcessorFunc(func(logEntry *log.ParsedLog) bool {
		if logEntry.Type == log.LogTypeLog && logEntry.Message == "Instruction: Swap" {
			slog.Info("Detected swap instruction log")
			return true
		}
		return false
	})

	// Create parser with processor
	parser := log.NewParserWithProcessors(customProcessor)

	logs := []string{
		"Program log: Instruction: Swap",
		"Program log: Other message",
	}

	parser.ParseWithProcessors(logs)
}
