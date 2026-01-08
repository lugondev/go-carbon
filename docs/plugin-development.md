---
layout: default
title: Plugin Development
nav_order: 6
description: "Create custom event decoders and processors for go-carbon"
permalink: /plugin-development
---

# Plugin Development Guide

This guide explains how to create custom plugins for go-carbon to decode and process blockchain events.

## Table of Contents

1. [Plugin System Overview](#plugin-system-overview)
2. [Creating a Decoder Plugin](#creating-a-decoder-plugin)
3. [Creating an Event Processor Plugin](#creating-an-event-processor-plugin)
4. [Working with Anchor Events](#working-with-anchor-events)
5. [Log Parsing](#log-parsing)
6. [Complete Example](#complete-example)
7. [Best Practices](#best-practices)

## Plugin System Overview

The go-carbon plugin system is modular and consists of three main components:

- **Log Parser** (`pkg/log`): Extracts "Program data:" from transaction logs
- **Decoder** (`pkg/decoder`): Decodes binary event data into structured types
- **Plugin** (`pkg/plugin`): Registers and manages decoders and event processors

### Architecture

```
Transaction Logs
       ↓
   Log Parser → Extract "Program data:"
       ↓
   Decoder → Decode binary data
       ↓
   Event Processor → Handle decoded events
```

## Creating a Decoder Plugin

### Step 1: Define Your Event Structure

```go
package myprogram

import "github.com/gagliardetto/solana-go"

// SwapEvent represents a swap event from your program
type SwapEvent struct {
    User      solana.PublicKey
    TokenIn   solana.PublicKey
    TokenOut  solana.PublicKey
    AmountIn  uint64
    AmountOut uint64
}
```

### Step 2: Implement the Decoder

For Anchor programs, use `AnchorEventDecoder`:

```go
import (
    "crypto/sha256"
    "fmt"
    
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/internal/decoder/anchor"
)

// computeDiscriminator computes Anchor event discriminator
func computeDiscriminator(eventName string) decoder.AnchorDiscriminator {
    data := []byte(fmt.Sprintf("event:%s", eventName))
    hash := sha256.Sum256(data)
    return decoder.NewAnchorDiscriminator(hash[:8])
}

// decodeSwapEvent decodes SwapEvent from Borsh-serialized data
func decodeSwapEvent(data []byte) (*SwapEvent, error) {
    if len(data) < 104 { // 32 + 32 + 32 + 8 + 8
        return nil, fmt.Errorf("insufficient data")
    }
    
    event := &SwapEvent{}
    offset := 0
    
    // Decode each field (little-endian)
    copy(event.User[:], data[offset:offset+32])
    offset += 32
    
    copy(event.TokenIn[:], data[offset:offset+32])
    offset += 32
    
    copy(event.TokenOut[:], data[offset:offset+32])
    offset += 32
    
    event.AmountIn, _ = decoder.DecodeU64LE(data[offset:offset+8])
    offset += 8
    
    event.AmountOut, _ = decoder.DecodeU64LE(data[offset:offset+8])
    
    return event, nil
}

// NewSwapEventDecoder creates a decoder for SwapEvent
func NewSwapEventDecoder(programID solana.PublicKey) decoder.Decoder {
    disc := computeDiscriminator("SwapExecuted")
    
    return anchor.NewAnchorEventDecoder(
        "SwapExecuted",
        programID,
        disc,
        func(data []byte) (interface{}, error) {
            return decodeSwapEvent(data)
        },
    )
}
```

### Step 3: Create the Plugin

```go
import (
    "github.com/lugondev/go-carbon/pkg/plugin"
    "github.com/lugondev/go-carbon/internal/decoder/anchor"
)

func NewMyProgramPlugin(programID solana.PublicKey) plugin.Plugin {
    // Create all your event decoders
    decoders := []decoder.Decoder{
        NewSwapEventDecoder(programID),
        // Add more decoders...
    }
    
    // Create Anchor plugin
    return anchor.NewAnchorEventPlugin(
        "my-program",
        programID,
        decoders,
    )
}
```

### Step 4: Register the Plugin

```go
func main() {
    registry := plugin.NewRegistry()
    
    programID := solana.MustPublicKeyFromBase58("YourProgramID...")
    myPlugin := NewMyProgramPlugin(programID)
    
    registry.MustRegister(myPlugin)
    
    ctx := context.Background()
    registry.Initialize(ctx)
}
```

## Creating an Event Processor Plugin

Event processors handle decoded events with custom business logic.

```go
import (
    "context"
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/internal/decoder/anchor"
)

func NewEventProcessor(programID solana.PublicKey) plugin.Plugin {
    return anchor.NewEventProcessorPlugin(
        "my-event-processor",
        programID,
        []string{"SwapExecuted"}, // Event types to handle
        func(ctx context.Context, event *decoder.Event) error {
            // Type assert to your event type
            swapEvent, ok := event.Data.(*SwapEvent)
            if !ok {
                return nil
            }
            
            // Your custom logic
            fmt.Printf("Swap: %d → %d\n", swapEvent.AmountIn, swapEvent.AmountOut)
            
            // Save to database
            // Send webhook
            // Update cache
            // etc.
            
            return nil
        },
    )
}
```

## Working with Anchor Events

Anchor programs emit events with an 8-byte discriminator:

```
[8 bytes discriminator][N bytes Borsh-serialized event data]
```

### Computing Discriminators

```go
import "crypto/sha256"

func computeDiscriminator(eventName string) [8]byte {
    // Anchor: sha256("event:{EventName}")[..8]
    data := []byte(fmt.Sprintf("event:%s", eventName))
    hash := sha256.Sum256(data)
    
    var disc [8]byte
    copy(disc[:], hash[:8])
    return disc
}
```

### Decoding Borsh Data

```go
// For simple types, decode manually:
func decodeSimpleEvent(data []byte) (*MyEvent, error) {
    event := &MyEvent{}
    offset := 0
    
    // u64 field (8 bytes, little-endian)
    event.Amount, _ = decoder.DecodeU64LE(data[offset:offset+8])
    offset += 8
    
    // Pubkey field (32 bytes)
    copy(event.User[:], data[offset:offset+32])
    offset += 32
    
    // String (4 bytes length + data)
    strLen := binary.LittleEndian.Uint32(data[offset:offset+4])
    offset += 4
    event.Message = string(data[offset:offset+int(strLen)])
    
    return event, nil
}

// For complex types, use a Borsh library
// https://github.com/gagliardetto/solana-go
```

## Log Parsing

### Extracting Program Data

```go
import "github.com/lugondev/go-carbon/pkg/log"

parser := log.NewParser()

// Transaction logs
logs := []string{
    "Program XXX invoke [1]",
    "Program data: AQAAAAAAAAA...", // Base64 encoded
    "Program XXX success",
}

// Extract all "Program data:" entries
programData := parser.ExtractProgramData(logs)

for _, data := range programData {
    // data is now []byte (decoded from base64)
    event, _ := myDecoder.Decode(data)
    // Process event...
}
```

### Filtering by Instruction Path

For nested instructions:

```go
// Path [0] = first top-level instruction
// Path [0, 1] = second inner instruction of first top-level
instructionPath := log.InstructionPath{0, 1}

filteredLogs := parser.FilterByInstructionPath(logs, instructionPath)
programData := parser.ExtractProgramData(filteredLogs)
```

## Complete Example

See `examples/event-parser/main.go` for a complete working example.

### Quick Start

```go
package main

import (
    "context"
    "github.com/lugondev/go-carbon/pkg/plugin"
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/pkg/log"
)

func main() {
    // 1. Create registry
    registry := plugin.NewRegistry()
    
    // 2. Register plugins
    registry.MustRegister(NewMyPlugin())
    
    // 3. Initialize
    ctx := context.Background()
    registry.Initialize(ctx)
    
    // 4. Parse logs
    parser := log.NewParser()
    programData := parser.ExtractProgramData(transactionLogs)
    
    // 5. Decode events
    decoderRegistry := registry.GetDecoderRegistry()
    events, _ := decoderRegistry.DecodeAll(programData, nil)
    
    // 6. Process events
    for _, event := range events {
        registry.ProcessEvent(ctx, event)
    }
}
```

## Best Practices

### 1. Validate Data Length

Always check data length before decoding:

```go
if len(data) < expectedSize {
    return nil, fmt.Errorf("insufficient data: need %d, got %d", expectedSize, len(data))
}
```

### 2. Handle Unknown Events Gracefully

```go
func (d *MyDecoder) Decode(data []byte) (*decoder.Event, error) {
    if !d.CanDecode(data) {
        return nil, nil // Not an error, just can't decode
    }
    // ... decode
}
```

### 3. Use Type-Safe Decoding

```go
// Good: Type-safe
event, ok := decodedEvent.Data.(*SwapEvent)
if !ok {
    return fmt.Errorf("unexpected event type")
}

// Bad: Type assertion without check
event := decodedEvent.Data.(*SwapEvent) // May panic!
```

### 4. Log Decoding Errors at Debug Level

```go
if err := decoder.Decode(data); err != nil {
    logger.Debug("Failed to decode", "error", err) // Not Error!
    // Many decoders may fail - this is normal
}
```

### 5. Test with Real Data

```bash
# Get transaction logs from Solana
solana confirm -v <signature>

# Copy "Log Messages" to your test
```

### 6. Document Your Event Schema

```go
// SwapExecuted event (120 bytes total)
// 
// Layout:
//   [0..8]    discriminator (0x1234567890abcdef)
//   [8..40]   user: Pubkey (32 bytes)
//   [40..72]  token_in: Pubkey (32 bytes)
//   [72..104] token_out: Pubkey (32 bytes)
//   [104..112] amount_in: u64 (8 bytes, LE)
//   [112..120] amount_out: u64 (8 bytes, LE)
type SwapExecuted struct { /* ... */ }
```

## Testing

Create unit tests for your decoders:

```go
func TestSwapEventDecoder(t *testing.T) {
    decoder := NewSwapEventDecoder(programID)
    
    // Create test data
    data := make([]byte, 8+120) // discriminator + event
    copy(data[0:8], computeDiscriminator("SwapExecuted")[:])
    // ... fill test data
    
    // Decode
    event, err := decoder.Decode(data)
    assert.NoError(t, err)
    assert.NotNil(t, event)
    
    // Verify
    swapEvent := event.Data.(*SwapEvent)
    assert.Equal(t, expectedUser, swapEvent.User)
}
```

## Troubleshooting

### Event Not Decoded

1. Check discriminator matches
2. Verify data length is sufficient
3. Ensure correct byte order (little-endian)
4. Check if decoder is registered

### Wrong Data Decoded

1. Verify struct field order matches Anchor/Borsh layout
2. Check for padding bytes
3. Validate string/vec length prefixes
4. Test with known good transaction

## Resources

- [Anchor Events Documentation](https://www.anchor-lang.com/docs/events)
- [Borsh Specification](https://borsh.io/)
- [Solana Transaction Structure](https://docs.solana.com/developing/programming-model/transactions)
- [Example Plugins](../../examples/)

## Support

For questions or issues:
1. Check existing examples in `examples/`
2. Review built-in plugins in `internal/decoder/`
3. Open an issue on GitHub
