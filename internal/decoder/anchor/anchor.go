// Package anchor provides a generic Anchor event decoder plugin.
//
// Anchor programs emit events with an 8-byte discriminator followed by the event data.
// This plugin provides utilities to decode Anchor events from "Program data:" logs.
//
// Example usage:
//
//	// Create decoder for a specific event
//	discriminator := decoder.NewAnchorDiscriminator([]byte{...})
//	eventDecoder := anchor.NewAnchorEventDecoder(
//	    "my-event",
//	    programID,
//	    discriminator,
//	    func(data []byte) (interface{}, error) {
//	        // Decode Borsh data
//	        var event MyEvent
//	        if err := borsh.Deserialize(&event, data); err != nil {
//	            return nil, err
//	        }
//	        return event, nil
//	    },
//	)
package anchor

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/decoder"
	"github.com/lugondev/go-carbon/pkg/log"
	"github.com/lugondev/go-carbon/pkg/plugin"
)

// AnchorEventPlugin is a plugin for decoding Anchor program events.
type AnchorEventPlugin struct {
	*plugin.BasePlugin
	decoders      []decoder.Decoder
	logProcessors []log.LogProcessor
	programID     solana.PublicKey
}

// NewAnchorEventPlugin creates a new Anchor event plugin for a specific program.
func NewAnchorEventPlugin(
	name string,
	programID solana.PublicKey,
	eventDecoders []decoder.Decoder,
) *AnchorEventPlugin {
	base := plugin.NewBasePlugin(
		name,
		"1.0.0",
		fmt.Sprintf("Anchor event decoder for %s", programID.String()),
	)

	return &AnchorEventPlugin{
		BasePlugin:    base,
		decoders:      eventDecoders,
		logProcessors: []log.LogProcessor{NewAnchorLogProcessor()},
		programID:     programID,
	}
}

// GetDecoders implements DecoderPlugin interface.
func (p *AnchorEventPlugin) GetDecoders() []decoder.Decoder {
	return p.decoders
}

// GetLogProcessors implements DecoderPlugin interface.
func (p *AnchorEventPlugin) GetLogProcessors() []log.LogProcessor {
	return p.logProcessors
}

// AnchorLogProcessor processes Anchor event logs.
type AnchorLogProcessor struct{}

// NewAnchorLogProcessor creates a new AnchorLogProcessor.
func NewAnchorLogProcessor() *AnchorLogProcessor {
	return &AnchorLogProcessor{}
}

// ProcessLog implements LogProcessor interface.
func (p *AnchorLogProcessor) ProcessLog(logEntry *log.ParsedLog) bool {
	if logEntry.Type != log.LogTypeData {
		return false
	}

	// Check if this is an Anchor event (has 8-byte discriminator)
	if len(logEntry.Data) < 8 {
		return false
	}

	// Anchor events have discriminator as first 8 bytes
	// We just mark it as processed, actual decoding happens in decoders
	return true
}

// GetName implements LogProcessor interface.
func (p *AnchorLogProcessor) GetName() string {
	return "anchor:event-log"
}

// AnchorEventDecoder decodes a specific Anchor event type.
type AnchorEventDecoder struct {
	name          string
	programID     solana.PublicKey
	discriminator decoder.AnchorDiscriminator
	decodeFunc    func([]byte) (interface{}, error)
}

// NewAnchorEventDecoder creates a new Anchor event decoder.
func NewAnchorEventDecoder(
	name string,
	programID solana.PublicKey,
	discriminator decoder.AnchorDiscriminator,
	decodeFunc func([]byte) (interface{}, error),
) *AnchorEventDecoder {
	return &AnchorEventDecoder{
		name:          name,
		programID:     programID,
		discriminator: discriminator,
		decodeFunc:    decodeFunc,
	}
}

// Decode implements Decoder interface.
func (d *AnchorEventDecoder) Decode(data []byte) (*decoder.Event, error) {
	if !d.CanDecode(data) {
		return nil, fmt.Errorf("discriminator mismatch for event %s", d.name)
	}

	// Skip discriminator (first 8 bytes)
	eventData := data[8:]

	// Decode using custom function
	decoded, err := d.decodeFunc(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode event %s: %w", d.name, err)
	}

	return &decoder.Event{
		Name:          d.name,
		Data:          decoded,
		RawData:       data,
		ProgramID:     d.programID,
		Discriminator: d.discriminator[:],
	}, nil
}

// CanDecode implements Decoder interface.
func (d *AnchorEventDecoder) CanDecode(data []byte) bool {
	if len(data) < 8 {
		return false
	}

	dataDisc := decoder.NewAnchorDiscriminator(data)
	return dataDisc.Equals(d.discriminator)
}

// GetName implements Decoder interface.
func (d *AnchorEventDecoder) GetName() string {
	return d.name
}

// GetProgramID implements Decoder interface.
func (d *AnchorEventDecoder) GetProgramID() solana.PublicKey {
	return d.programID
}

// ComputeDiscriminator computes the Anchor event discriminator from event name.
// Anchor uses: sha256("event:{EventName}")[..8]
// This is a simplified version - in production, use proper sha256 hashing.
func ComputeDiscriminator(eventName string) decoder.AnchorDiscriminator {
	// This is a placeholder - actual implementation should use:
	// hash := sha256.Sum256([]byte(fmt.Sprintf("event:%s", eventName)))
	// return decoder.NewAnchorDiscriminator(hash[:8])

	// For now, return a zero discriminator
	return decoder.AnchorDiscriminator{}
}

// Generic event structures for common patterns

// TransferEvent is a common event structure for token/asset transfers.
type TransferEvent struct {
	From   solana.PublicKey `json:"from"`
	To     solana.PublicKey `json:"to"`
	Amount uint64           `json:"amount"`
}

// DecodeTransferEvent decodes a transfer event from Borsh data.
func DecodeTransferEvent(data []byte) (*TransferEvent, error) {
	if len(data) < 72 { // 32 + 32 + 8 bytes
		return nil, fmt.Errorf("insufficient data for transfer event")
	}

	event := &TransferEvent{}

	// Decode From (32 bytes)
	copy(event.From[:], data[0:32])

	// Decode To (32 bytes)
	copy(event.To[:], data[32:64])

	// Decode Amount (8 bytes, little-endian)
	event.Amount = binary.LittleEndian.Uint64(data[64:72])

	return event, nil
}

// SwapEvent is a common event structure for DEX swaps.
type SwapEvent struct {
	User         solana.PublicKey `json:"user"`
	TokenIn      solana.PublicKey `json:"token_in"`
	TokenOut     solana.PublicKey `json:"token_out"`
	AmountIn     uint64           `json:"amount_in"`
	AmountOut    uint64           `json:"amount_out"`
	MinAmountOut uint64           `json:"min_amount_out"`
}

// DecodeSwapEvent decodes a swap event from Borsh data.
func DecodeSwapEvent(data []byte) (*SwapEvent, error) {
	if len(data) < 120 { // 32 + 32 + 32 + 8 + 8 + 8 bytes
		return nil, fmt.Errorf("insufficient data for swap event")
	}

	event := &SwapEvent{}
	offset := 0

	// User (32 bytes)
	copy(event.User[:], data[offset:offset+32])
	offset += 32

	// TokenIn (32 bytes)
	copy(event.TokenIn[:], data[offset:offset+32])
	offset += 32

	// TokenOut (32 bytes)
	copy(event.TokenOut[:], data[offset:offset+32])
	offset += 32

	// AmountIn (8 bytes)
	event.AmountIn = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// AmountOut (8 bytes)
	event.AmountOut = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// MinAmountOut (8 bytes)
	event.MinAmountOut = binary.LittleEndian.Uint64(data[offset : offset+8])

	return event, nil
}

// CreateAccountEvent is a common event for account creation.
type CreateAccountEvent struct {
	Account   solana.PublicKey `json:"account"`
	Owner     solana.PublicKey `json:"owner"`
	Timestamp int64            `json:"timestamp"`
}

// DecodeCreateAccountEvent decodes an account creation event.
func DecodeCreateAccountEvent(data []byte) (*CreateAccountEvent, error) {
	if len(data) < 72 { // 32 + 32 + 8 bytes
		return nil, fmt.Errorf("insufficient data for create account event")
	}

	event := &CreateAccountEvent{}
	offset := 0

	// Account (32 bytes)
	copy(event.Account[:], data[offset:offset+32])
	offset += 32

	// Owner (32 bytes)
	copy(event.Owner[:], data[offset:offset+32])
	offset += 32

	// Timestamp (8 bytes, signed)
	event.Timestamp = int64(binary.LittleEndian.Uint64(data[offset : offset+8]))

	return event, nil
}

// EventProcessorPlugin processes decoded Anchor events.
type EventProcessorPlugin struct {
	*plugin.BasePlugin
	programID   solana.PublicKey
	eventTypes  []string
	processFunc func(context.Context, *decoder.Event) error
}

// NewEventProcessorPlugin creates a new event processor plugin.
func NewEventProcessorPlugin(
	name string,
	programID solana.PublicKey,
	eventTypes []string,
	processFunc func(context.Context, *decoder.Event) error,
) *EventProcessorPlugin {
	base := plugin.NewBasePlugin(
		name,
		"1.0.0",
		fmt.Sprintf("Event processor for %s", programID.String()),
	)

	return &EventProcessorPlugin{
		BasePlugin:  base,
		programID:   programID,
		eventTypes:  eventTypes,
		processFunc: processFunc,
	}
}

// ProcessEvent implements EventProcessorPlugin interface.
func (p *EventProcessorPlugin) ProcessEvent(ctx context.Context, event *decoder.Event) (bool, error) {
	// Check if event is from our program
	if event.ProgramID != p.programID {
		return false, nil
	}

	// Check if we handle this event type
	if len(p.eventTypes) > 0 {
		handled := false
		for _, eventType := range p.eventTypes {
			if eventType == event.Name {
				handled = true
				break
			}
		}
		if !handled {
			return false, nil
		}
	}

	// Process the event
	if err := p.processFunc(ctx, event); err != nil {
		return false, err
	}

	return true, nil
}

// GetEventTypes implements EventProcessorPlugin interface.
func (p *EventProcessorPlugin) GetEventTypes() []string {
	return p.eventTypes
}
