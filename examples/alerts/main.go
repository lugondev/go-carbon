// Package main demonstrates an alert system using the go-carbon framework.
//
// This example shows how to:
// - Monitor transactions for specific program interactions
// - Decode instruction data for different protocols
// - Send alerts via multiple channels (console, webhook, etc.)
// - Filter and process instruction-level events
//
// This is similar to the PumpFun alerts example from the Rust Carbon framework.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/datasource/rpc"
	"github.com/lugondev/go-carbon/internal/instruction"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/pipeline"
	"github.com/lugondev/go-carbon/internal/processor"
	"github.com/lugondev/go-carbon/pkg/types"
)

// RPC endpoint
const rpcEndpoint = "https://api.mainnet-beta.solana.com"

// Well-known program IDs
var (
	// PumpFun Program ID
	PumpFunProgramID = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")

	// Raydium AMM Program ID
	RaydiumAMMProgramID = solana.MustPublicKeyFromBase58("675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8")

	// Jupiter Aggregator V6
	JupiterV6ProgramID = solana.MustPublicKeyFromBase58("JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4")
)

// AlertType represents the type of alert.
type AlertType string

const (
	AlertTypeSwap        AlertType = "SWAP"
	AlertTypeLiquidity   AlertType = "LIQUIDITY"
	AlertTypeTokenLaunch AlertType = "TOKEN_LAUNCH"
	AlertTypeWhale       AlertType = "WHALE"
)

// Alert represents an alert to be sent.
type Alert struct {
	Type        AlertType         `json:"type"`
	Timestamp   time.Time         `json:"timestamp"`
	Signature   string            `json:"signature"`
	Slot        uint64            `json:"slot"`
	ProgramID   string            `json:"program_id"`
	Description string            `json:"description"`
	Data        map[string]string `json:"data,omitempty"`
}

// AlertSender defines the interface for sending alerts.
type AlertSender interface {
	Send(ctx context.Context, alert *Alert) error
}

// ConsoleAlertSender sends alerts to the console.
type ConsoleAlertSender struct {
	logger *slog.Logger
}

// NewConsoleAlertSender creates a new ConsoleAlertSender.
func NewConsoleAlertSender(logger *slog.Logger) *ConsoleAlertSender {
	return &ConsoleAlertSender{logger: logger}
}

// Send implements AlertSender.
func (s *ConsoleAlertSender) Send(ctx context.Context, alert *Alert) error {
	s.logger.Info("🚨 ALERT",
		"type", alert.Type,
		"program", alert.ProgramID,
		"signature", alert.Signature,
		"description", alert.Description,
		"data", alert.Data,
	)
	return nil
}

// WebhookAlertSender sends alerts to a webhook URL.
type WebhookAlertSender struct {
	webhookURL string
	client     *http.Client
	logger     *slog.Logger
}

// NewWebhookAlertSender creates a new WebhookAlertSender.
func NewWebhookAlertSender(webhookURL string, logger *slog.Logger) *WebhookAlertSender {
	return &WebhookAlertSender{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// Send implements AlertSender.
func (s *WebhookAlertSender) Send(ctx context.Context, alert *Alert) error {
	if s.webhookURL == "" {
		return nil
	}

	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	s.logger.Debug("Webhook sent successfully", "status", resp.StatusCode)
	return nil
}

// MultiAlertSender sends alerts to multiple senders.
type MultiAlertSender struct {
	senders []AlertSender
}

// NewMultiAlertSender creates a new MultiAlertSender.
func NewMultiAlertSender(senders ...AlertSender) *MultiAlertSender {
	return &MultiAlertSender{senders: senders}
}

// Send implements AlertSender.
func (s *MultiAlertSender) Send(ctx context.Context, alert *Alert) error {
	for _, sender := range s.senders {
		if err := sender.Send(ctx, alert); err != nil {
			// Log but don't fail on individual sender errors
			continue
		}
	}
	return nil
}

// InstructionEvent represents a decoded instruction event.
type InstructionEvent struct {
	ProgramID types.Pubkey
	Type      string
	Data      map[string]string
}

// ProgramInstructionDecoder decodes instructions for multiple programs.
type ProgramInstructionDecoder struct {
	targetPrograms map[string]bool
}

// NewProgramInstructionDecoder creates a new decoder for the specified programs.
func NewProgramInstructionDecoder(programs ...solana.PublicKey) *ProgramInstructionDecoder {
	targets := make(map[string]bool)
	for _, p := range programs {
		targets[p.String()] = true
	}
	return &ProgramInstructionDecoder{targetPrograms: targets}
}

// DecodeInstruction implements InstructionDecoder.
func (d *ProgramInstructionDecoder) DecodeInstruction(ix *types.Instruction) *instruction.DecodedInstruction[InstructionEvent] {
	if ix == nil {
		return nil
	}

	programStr := ix.ProgramID.String()
	if !d.targetPrograms[programStr] {
		return nil
	}

	// Basic decoding - in a real implementation, you would decode based on discriminator
	event := InstructionEvent{
		ProgramID: ix.ProgramID,
		Type:      "UNKNOWN",
		Data:      make(map[string]string),
	}

	// Try to identify instruction type based on discriminator (first 8 bytes typically)
	if len(ix.Data) >= 8 {
		discriminator := ix.Data[:8]
		event.Data["discriminator"] = fmt.Sprintf("%x", discriminator)

		// Example: Identify common instruction types
		// This would be expanded based on the specific program's IDL
		switch programStr {
		case PumpFunProgramID.String():
			event = decodePumpFunInstruction(ix, discriminator)
		case RaydiumAMMProgramID.String():
			event = decodeRaydiumInstruction(ix, discriminator)
		case JupiterV6ProgramID.String():
			event = decodeJupiterInstruction(ix, discriminator)
		}
	}

	event.Data["accounts_count"] = fmt.Sprintf("%d", len(ix.Accounts))
	event.Data["data_len"] = fmt.Sprintf("%d", len(ix.Data))

	return &instruction.DecodedInstruction[InstructionEvent]{
		ProgramID: ix.ProgramID,
		Data:      event,
		Accounts:  ix.Accounts,
	}
}

// decodePumpFunInstruction decodes PumpFun program instructions.
func decodePumpFunInstruction(ix *types.Instruction, discriminator []byte) InstructionEvent {
	event := InstructionEvent{
		ProgramID: ix.ProgramID,
		Type:      "PUMPFUN_UNKNOWN",
		Data:      make(map[string]string),
	}

	// Common PumpFun discriminators (example - actual values may differ)
	// These would come from the program's IDL
	switch {
	case isDiscriminator(discriminator, "buy"):
		event.Type = "PUMPFUN_BUY"
	case isDiscriminator(discriminator, "sell"):
		event.Type = "PUMPFUN_SELL"
	case isDiscriminator(discriminator, "create"):
		event.Type = "PUMPFUN_CREATE"
	}

	return event
}

// decodeRaydiumInstruction decodes Raydium AMM instructions.
func decodeRaydiumInstruction(ix *types.Instruction, discriminator []byte) InstructionEvent {
	event := InstructionEvent{
		ProgramID: ix.ProgramID,
		Type:      "RAYDIUM_UNKNOWN",
		Data:      make(map[string]string),
	}

	// Raydium uses single-byte instruction identifiers
	if len(ix.Data) >= 1 {
		switch ix.Data[0] {
		case 9:
			event.Type = "RAYDIUM_SWAP"
		case 10:
			event.Type = "RAYDIUM_SWAP_BASE_IN"
		case 11:
			event.Type = "RAYDIUM_SWAP_BASE_OUT"
		case 3:
			event.Type = "RAYDIUM_ADD_LIQUIDITY"
		case 4:
			event.Type = "RAYDIUM_REMOVE_LIQUIDITY"
		}
	}

	return event
}

// decodeJupiterInstruction decodes Jupiter aggregator instructions.
func decodeJupiterInstruction(ix *types.Instruction, discriminator []byte) InstructionEvent {
	event := InstructionEvent{
		ProgramID: ix.ProgramID,
		Type:      "JUPITER_UNKNOWN",
		Data:      make(map[string]string),
	}

	// Jupiter V6 uses Anchor-style discriminators
	switch {
	case isDiscriminator(discriminator, "route"):
		event.Type = "JUPITER_ROUTE"
	case isDiscriminator(discriminator, "sharedAccountsRoute"):
		event.Type = "JUPITER_SHARED_ACCOUNTS_ROUTE"
	case isDiscriminator(discriminator, "exactOutRoute"):
		event.Type = "JUPITER_EXACT_OUT_ROUTE"
	}

	return event
}

// isDiscriminator is a helper to check discriminators (simplified).
func isDiscriminator(data []byte, name string) bool {
	// In a real implementation, this would compute the Anchor discriminator
	// or compare against known values from the IDL
	_ = name
	return false // Placeholder
}

// AlertProcessor processes decoded instructions and generates alerts.
type AlertProcessor struct {
	alertSender AlertSender
	logger      *slog.Logger
}

// NewAlertProcessor creates a new AlertProcessor.
func NewAlertProcessor(alertSender AlertSender, logger *slog.Logger) *AlertProcessor {
	return &AlertProcessor{
		alertSender: alertSender,
		logger:      logger,
	}
}

// Process implements Processor.
func (p *AlertProcessor) Process(
	ctx context.Context,
	input instruction.InstructionProcessorInput[InstructionEvent],
	m *metrics.Collection,
) error {
	event := input.DecodedInstruction.Data

	// Determine alert type and generate alert
	var alertType AlertType
	var description string

	switch event.Type {
	case "PUMPFUN_BUY", "PUMPFUN_SELL":
		alertType = AlertTypeSwap
		description = fmt.Sprintf("PumpFun %s detected", event.Type)
	case "PUMPFUN_CREATE":
		alertType = AlertTypeTokenLaunch
		description = "New PumpFun token launched!"
	case "RAYDIUM_SWAP", "RAYDIUM_SWAP_BASE_IN", "RAYDIUM_SWAP_BASE_OUT":
		alertType = AlertTypeSwap
		description = fmt.Sprintf("Raydium %s detected", event.Type)
	case "RAYDIUM_ADD_LIQUIDITY", "RAYDIUM_REMOVE_LIQUIDITY":
		alertType = AlertTypeLiquidity
		description = fmt.Sprintf("Raydium liquidity event: %s", event.Type)
	case "JUPITER_ROUTE", "JUPITER_SHARED_ACCOUNTS_ROUTE", "JUPITER_EXACT_OUT_ROUTE":
		alertType = AlertTypeSwap
		description = fmt.Sprintf("Jupiter swap: %s", event.Type)
	default:
		// Log unknown events but don't alert
		p.logger.Debug("Unknown instruction type",
			"type", event.Type,
			"program", event.ProgramID.String(),
		)
		return nil
	}

	alert := &Alert{
		Type:        alertType,
		Timestamp:   time.Now(),
		Signature:   input.Metadata.TransactionMetadata.Signature.String(),
		Slot:        input.Metadata.TransactionMetadata.Slot,
		ProgramID:   event.ProgramID.String(),
		Description: description,
		Data:        event.Data,
	}

	// Send alert
	if err := p.alertSender.Send(ctx, alert); err != nil {
		p.logger.Error("Failed to send alert", "error", err)
	}

	// Record metrics
	_ = m.IncrementCounter(ctx, "alerts_sent", 1)
	_ = m.IncrementCounter(ctx, fmt.Sprintf("alerts_%s", alertType), 1)

	return nil
}

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting go-carbon alerts example")

	// Get webhook URL from environment (optional)
	webhookURL := os.Getenv("ALERT_WEBHOOK_URL")

	// Create alert senders
	consoleSender := NewConsoleAlertSender(logger)
	var alertSender AlertSender = consoleSender

	if webhookURL != "" {
		webhookSender := NewWebhookAlertSender(webhookURL, logger)
		alertSender = NewMultiAlertSender(consoleSender, webhookSender)
		logger.Info("Webhook alerts enabled", "url", webhookURL)
	}

	// Create RPC config - using mainnet for real data
	rpcConfig := rpc.DefaultConfig(rpcEndpoint)
	rpcConfig.PollInterval = 2 * time.Second

	// Create slot monitor to get new blocks
	slotMonitor := rpc.NewSlotMonitorDatasource(rpcConfig)
	slotMonitor.WithLogger(logger)

	// Create instruction decoder for target programs
	decoder := NewProgramInstructionDecoder(
		PumpFunProgramID,
		RaydiumAMMProgramID,
		JupiterV6ProgramID,
	)

	// Create alert processor
	alertProcessor := NewAlertProcessor(alertSender, logger)

	// Create instruction pipe
	instructionPipe := instruction.NewInstructionPipe(decoder, alertProcessor)
	instructionPipe.WithLogger(logger)

	// Create metrics collection
	metricsCollection := metrics.NewCollection(
		metrics.NewLogMetrics(logger),
	)

	// Build the pipeline
	p := pipeline.Builder().
		Datasource(datasource.NewNamedDatasourceID("rpc-mainnet"), slotMonitor).
		InstructionPipe(instructionPipe).
		Metrics(metricsCollection).
		MetricsFlushInterval(30 * time.Second).
		ChannelBufferSize(1000).
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

	logger.Info("Alert system started, monitoring for DeFi events...")

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

	logger.Info("Alert system stopped")
}

// ExampleAlertFiltering demonstrates filtering alerts by criteria.
func ExampleAlertFiltering() {
	logger := slog.Default()
	alertSender := NewConsoleAlertSender(logger)

	// Create a conditional processor that only alerts on specific conditions
	conditionalProcessor := processor.NewConditionalProcessor(
		processor.ProcessorFunc[instruction.InstructionProcessorInput[InstructionEvent]](
			func(ctx context.Context, input instruction.InstructionProcessorInput[InstructionEvent], m *metrics.Collection) error {
				event := input.DecodedInstruction.Data
				alert := &Alert{
					Type:        AlertTypeTokenLaunch,
					Timestamp:   time.Now(),
					ProgramID:   event.ProgramID.String(),
					Description: "Filtered alert",
				}
				return alertSender.Send(ctx, alert)
			},
		),
		func(input instruction.InstructionProcessorInput[InstructionEvent]) bool {
			// Only alert on token launches
			return input.DecodedInstruction.Data.Type == "PUMPFUN_CREATE"
		},
	)

	decoder := NewProgramInstructionDecoder(PumpFunProgramID)
	instructionPipe := instruction.NewInstructionPipe(decoder, conditionalProcessor)
	instructionPipe.WithLogger(logger)

	fmt.Println("Created filtered instruction pipe for token launches only")
}

// ExampleBatchAlerts demonstrates batching alerts for efficiency.
func ExampleBatchAlerts() {
	logger := slog.Default()
	alertSender := NewConsoleAlertSender(logger)

	// Batch processor that collects alerts
	batchProcessor := processor.NewBatchProcessor(
		processor.ProcessorFunc[[]instruction.InstructionProcessorInput[InstructionEvent]](
			func(ctx context.Context, inputs []instruction.InstructionProcessorInput[InstructionEvent], m *metrics.Collection) error {
				// Process batch of alerts
				for _, input := range inputs {
					event := input.DecodedInstruction.Data
					alert := &Alert{
						Type:        AlertTypeSwap,
						Timestamp:   time.Now(),
						ProgramID:   event.ProgramID.String(),
						Description: fmt.Sprintf("Batch alert: %s", event.Type),
					}
					if err := alertSender.Send(ctx, alert); err != nil {
						logger.Error("Failed to send batch alert", "error", err)
					}
				}
				return nil
			},
		),
		10, // Batch size of 10
	)

	_ = batchProcessor // Use in pipeline
	fmt.Println("Created batch alert processor")
}
