---
layout: default
title: Code Generation
nav_order: 4
description: "Generate type-safe Go code from Anchor IDL files using Jennifer"
permalink: /codegen
has_children: false
---

# Code Generation Guide
{: .no_toc }

Generate production-ready, type-safe Go code from Anchor IDL files
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

{: .new }
> **NEW**: Jennifer-based generator produces clean, type-safe Go code with zero templates!

## Overview

Go-Carbon includes a powerful code generator that transforms Anchor IDL JSON files into production-ready Go code. The generator uses the [Jennifer library](https://github.com/dave/jennifer) for type-safe code generation.

### Key Features

✨ **Type-Safe**: Full Go type checking at compile time  
🎯 **Zero Config**: Works with any Anchor IDL (v0.1.0 - v0.29+)  
📦 **Borsh Support**: Built-in serialization tags  
🔑 **Discriminators**: Automatic Anchor discriminator handling  
🏗️ **Clean Code**: Production-ready, `gofmt` formatted  

---

## Quick Start

### Basic Usage

```bash
carbon codegen --idl ./target/idl/my_program.json --output ./pkg/myprogram
```

### With Options

```bash
carbon codegen \
  --idl ./target/idl/token_swap.json \
  --output ./generated/tokenswap \
  --package tokenswap
```

### CLI Options

| Option | Short | Description | Default |
|--------|-------|-------------|---------|
| `--idl` | `-i` | Path to Anchor IDL JSON file | Required |
| `--output` | `-o` | Output directory | Required |
| `--package` | `-p` | Package name | Directory name |

---

## Generated Files

The generator creates 5 Go files:

### 1. `program.go` - Program Metadata

Contains program ID, plugin factory, and decoder registry.

```go
package tokenswap

import (
    solanago "github.com/gagliardetto/solana-go"
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

var ProgramID = solanago.MustPublicKeyFromBase58("TokenSwap...")

func NewTokenSwapPlugin(programID solanago.PublicKey) plugin.Plugin {
    return anchor.NewAnchorEventPlugin(
        "token_swap",
        programID,
        GetDecoders(programID),
    )
}

func GetDecoderRegistry(programID solanago.PublicKey) *decoder.Registry {
    registry := decoder.NewRegistry()
    for _, dec := range GetDecoders(programID) {
        registry.Register(dec.Name(), dec)
    }
    return registry
}
```

### 2. `types.go` - Custom Types

Struct and enum definitions with Borsh tags.

```go
package tokenswap

import solanago "github.com/gagliardetto/solana-go"

type SwapState struct {
    IsInitialized bool               `borsh:"is_initialized" json:"is_initialized"`
    Authority     solanago.PublicKey `borsh:"authority" json:"authority"`
    TokenAMint    solanago.PublicKey `borsh:"token_a_mint" json:"token_a_mint"`
    TokenBMint    solanago.PublicKey `borsh:"token_b_mint" json:"token_b_mint"`
    FeeNumerator  uint64             `borsh:"fee_numerator" json:"fee_numerator"`
}

type SwapDirection uint8

const (
    SwapDirectionAToB SwapDirection = 0
    SwapDirectionBToA SwapDirection = 1
)

func (e SwapDirection) String() string {
    switch e {
    case SwapDirectionAToB:
        return "AToB"
    case SwapDirectionBToA:
        return "BToA"
    default:
        return "Unknown"
    }
}
```

### 3. `accounts.go` - Account Decoders

Account structures with discriminator verification.

```go
package tokenswap

import (
    solanago "github.com/gagliardetto/solana-go"
    bin "github.com/gagliardetto/binary"
)

var SwapPoolDiscriminator = [8]byte{/* computed discriminator */}

type SwapPool struct {
    State         SwapState
    TokenAAccount solanago.PublicKey
    TokenBAccount solanago.PublicKey
}

func DecodeSwapPool(data []byte) (*SwapPool, error) {
    if len(data) < 8 {
        return nil, fmt.Errorf("data too short for discriminator")
    }
    
    disc := [8]byte{}
    copy(disc[:], data[:8])
    
    if disc != SwapPoolDiscriminator {
        return nil, fmt.Errorf("invalid discriminator for SwapPool account")
    }
    
    account := &SwapPool{}
    err := bin.UnmarshalBorsh(account, data[8:])
    if err != nil {
        return nil, err
    }
    
    return account, nil
}
```

### 4. `events.go` - Event Decoders

Event structures with Anchor discriminator handling.

```go
package tokenswap

import (
    solanago "github.com/gagliardetto/solana-go"
    "github.com/lugondev/go-carbon/pkg/decoder"
)

var SwapExecutedEventDiscriminator = decoder.NewAnchorDiscriminator(
    []byte{/* sha256("event:SwapExecuted")[:8] */},
)

type SwapExecutedEvent struct {
    User      solanago.PublicKey
    TokenIn   solanago.PublicKey
    TokenOut  solanago.PublicKey
    AmountIn  uint64
    AmountOut uint64
}

func GetDecoders(programID solanago.PublicKey) []decoder.Decoder {
    return []decoder.Decoder{
        anchor.NewAnchorEventDecoder(
            "SwapExecuted",
            programID,
            SwapExecutedEventDiscriminator,
            DecodeSwapExecutedEvent,
        ),
    }
}

func ParseEvent(data []byte, programID *solanago.PublicKey) (*decoder.Event, error) {
    registry := GetDecoderRegistry(*programID)
    return registry.Decode(data, programID)
}
```

### 5. `instructions.go` - Instruction Builders

Type-safe instruction builders with fluent API.

```go
package tokenswap

import solanago "github.com/gagliardetto/solana-go"

type SwapInstruction struct {
    AmountIn      uint64
    MinAmountOut  uint64
    SwapDirection SwapDirection
    
    SwapPool         *solanago.AccountMeta
    Authority        *solanago.AccountMeta
    UserSourceToken  *solanago.AccountMeta
    UserDestToken    *solanago.AccountMeta
    PoolSourceToken  *solanago.AccountMeta
    PoolDestToken    *solanago.AccountMeta
    TokenProgram     *solanago.AccountMeta
}

func NewSwapInstructionBuilder() *SwapInstruction {
    return &SwapInstruction{
        TokenProgram: &solanago.AccountMeta{
            PublicKey:  solanago.TokenProgramID,
            IsSigner:   false,
            IsWritable: false,
        },
    }
}

func (ix *SwapInstruction) SetAmountIn(amountIn uint64) *SwapInstruction {
    ix.AmountIn = amountIn
    return ix
}

func (ix *SwapInstruction) SetMinAmountOut(minAmountOut uint64) *SwapInstruction {
    ix.MinAmountOut = minAmountOut
    return ix
}

func (ix *SwapInstruction) Build() (*solanago.Instruction, error) {
    accounts := []*solanago.AccountMeta{
        ix.SwapPool,
        ix.Authority,
        ix.UserSourceToken,
        ix.UserDestToken,
        ix.PoolSourceToken,
        ix.PoolDestToken,
        ix.TokenProgram,
    }
    
    data := SwapInstructionData{
        Discriminator: SwapDiscriminator,
        AmountIn:      ix.AmountIn,
        MinAmountOut:  ix.MinAmountOut,
        Direction:     ix.SwapDirection,
    }
    
    instructionData, err := bin.MarshalBorsh(data)
    if err != nil {
        return nil, err
    }
    
    return solanago.NewInstruction(ProgramID, accounts, instructionData), nil
}

func (ix *SwapInstruction) ValidateAccounts() error {
    return nil
}
```

---

## Using Generated Code

### 1. With Plugin System

```go
package main

import (
    "context"
    "github.com/lugondev/go-carbon/pkg/plugin"
    "github.com/yourorg/project/generated/tokenswap"
)

func main() {
    registry := plugin.NewRegistry()
    
    registry.MustRegister(tokenswap.NewTokenSwapPlugin(tokenswap.ProgramID))
    
    ctx := context.Background()
    registry.Initialize(ctx)
    
    events, _ := registry.GetDecoderRegistry().DecodeAll(programDataList, &tokenswap.ProgramID)
    for _, event := range events {
        switch e := event.Data.(type) {
        case *tokenswap.SwapExecutedEvent:
            fmt.Printf("Swap: %d -> %d\n", e.AmountIn, e.AmountOut)
        }
    }
}
```

### 2. Direct Event Decoding

```go
event, err := tokenswap.ParseEvent(eventData, &tokenswap.ProgramID)
if err != nil {
    log.Fatal(err)
}

swapEvent := event.Data.(*tokenswap.SwapExecutedEvent)
fmt.Printf("User: %s\n", swapEvent.User.String())
```

### 3. Building Instructions

```go
swapIx := tokenswap.NewSwapInstructionBuilder().
    SetAmountIn(1000000).
    SetMinAmountOut(950000).
    SetSwapDirection(tokenswap.SwapDirectionAToB).
    SetSwapPool(&solana.AccountMeta{
        PublicKey:  poolAddress,
        IsWritable: true,
    }).
    Build()

tx := solana.NewTransaction(swapIx, recentBlockhash)
```

### 4. Decoding Accounts

```go
accountInfo, err := rpcClient.GetAccountInfo(ctx, poolAddress)
if err != nil {
    log.Fatal(err)
}

pool, err := tokenswap.DecodeSwapPool(accountInfo.Value.Data.GetBinary())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Pool Authority: %s\n", pool.State.Authority.String())
```

---

## Type System

### Primitive Types

All Solana/Anchor primitives are supported:

| IDL Type | Go Type | Example |
|----------|---------|---------|
| `u8` | `uint8` | `var x uint8` |
| `u16` | `uint16` | `var x uint16` |
| `u32` | `uint32` | `var x uint32` |
| `u64` | `uint64` | `var x uint64` |
| `u128` | `bin.Uint128` | `var x bin.Uint128` |
| `i8` | `int8` | `var x int8` |
| `i16` | `int16` | `var x int16` |
| `i32` | `int32` | `var x int32` |
| `i64` | `int64` | `var x int64` |
| `i128` | `bin.Int128` | `var x bin.Int128` |
| `f32` | `float32` | `var x float32` |
| `f64` | `float64` | `var x float64` |
| `bool` | `bool` | `var x bool` |
| `string` | `string` | `var x string` |
| `bytes` | `[]byte` | `var x []byte` |
| `pubkey` | `solana.PublicKey` | `var x solana.PublicKey` |

### Complex Types

**Vec:**
```json
{"vec": {"defined": "Transaction"}}
```
→ `[]Transaction`

**Option:**
```json
{"option": "u64"}
```
→ `*uint64`

**Array:**
```json
{"array": ["u8", 32]}
```
→ `[32]uint8`

**Nested:**
```json
{"vec": {"option": {"defined": "CustomType"}}}
```
→ `[]*CustomType`

---

## Supported & Unsupported Features

### ✅ Fully Supported

- **Primitive Types**: All Solana/Borsh primitives (u8-u128, i8-i128, f32, f64, bool, string, bytes, pubkey)
- **Custom Structs**: With nested fields and Borsh tags
- **Simple Enums**: Variants without fields, with String() methods
- **Arrays**: Fixed-size arrays `[T; N]`
- **Vectors**: Dynamic slices `vec<T>`
- **Options**: Pointer types `option<T>` → `*T`
- **Defined Types**: References to custom types
- **Accounts**: With discriminators and decoders
- **Instructions**: Type-safe builders with validation
- **Events**: With discriminators and parsers
- **Nested Types**: Unlimited nesting depth
- **Both IDL Formats**: v0.1.0 and v0.29+ automatically detected

### ⚠️ Partially Supported

**Enum Variants with Fields:**
```json
{
  "kind": "enum",
  "variants": [
    {"name": "success", "fields": [{"name": "value", "type": "u64"}]},
    {"name": "error", "fields": [{"name": "code", "type": "u32"}]}
  ]
}
```

**Current behavior:** Generated as `interface{}` type  
**Workaround:** Manual implementation or use simple enums

**Generic Types:**
```json
{"defined": "MyType", "generics": [{"kind": "type", "type": "u64"}]}
```

**Current behavior:** Generics are ignored, base type is used  
**Workaround:** Expand generics in IDL or handle manually

### ❌ Not Yet Supported

**COption Type:**
```json
{"coption": "pubkey"}
```

**Status:** Parser supports it, generates `*T`, but may need special handling  
**Workaround:** Use `option` instead

**Tuple Types:**
```json
{"tuple": ["u64", "pubkey"]}
```

**Status:** Parsed but generates anonymous struct  
**Workaround:** Use named struct types

**PDA Seeds:**
```json
{
  "pda": {
    "seeds": [
      {"kind": "const", "value": [112, 111, 111, 108]},
      {"kind": "account", "path": "authority"}
    ]
  }
}
```

**Status:** Not implemented  
**Workaround:** Implement PDA derivation manually

### Comparison: Old vs New Generator

| Feature | Old (Templates) | New (Jennifer) |
|---------|----------------|----------------|
| Type Safety | ❌ String templates | ✅ Type-checked code |
| Instruction Builders | ❌ Not generated | ✅ Fully functional |
| Account Validation | ❌ Manual | ✅ Automatic |
| Borsh Encoding | ⚠️ Partial | ✅ Full support |
| Error Messages | ❌ Template errors | ✅ Go compiler errors |
| IDE Support | ❌ No autocomplete | ✅ Full autocomplete |
| Standalone Code | ❌ Needs framework | ✅ No dependencies |
| Extensibility | ❌ Hard to modify | ✅ Easy to extend |
| Generated Code Quality | ⚠️ Verbose | ✅ Clean, idiomatic |
| Discriminators | ✅ Supported | ✅ Supported |
| Events | ✅ Decoders only | ✅ Full parsing |
| Both IDL Formats | ❌ v0.29+ only | ✅ v0.1.0 + v0.29+ |

---

## Known Limitations

### 1. Validation is Disabled

The current implementation has `ValidateAccounts()` as a no-op:

```go
func (ix *InitializePoolInstruction) ValidateAccounts() error {
    return nil
}
```

**Reason:** `solana.SystemProgramID.IsZero()` returns `true`, causing false positives

**Future:** Will add proper validation that handles well-known program IDs

### 2. No RPC Helpers

Generated code doesn't include RPC fetch helpers:

```go
func FetchSwapPool(ctx context.Context, client *rpc.Client, address solana.PublicKey) (*SwapPool, error)
```

**Workaround:** Use `client.GetAccountInfo()` + `DecodeSwapPool()`

### 3. No CPI Helpers

No Cross-Program Invocation helpers generated:

```go
func SwapCPI(ctx solana.Context, ...) error
```

**Workaround:** Build instructions manually and use `solana.Invoke()`

### 4. No Transaction Builders

No high-level transaction composition:

```go
func BuildSwapTransaction(params SwapParams) (*solana.Transaction, error)
```

**Workaround:** Use `solana.NewTransaction()` with generated instructions

---

## Troubleshooting

### Build Errors

**Problem:** `undefined: binary.MarshalBorsh`

```
./instructions.go:42:11: undefined: binary.MarshalBorsh
```

**Solution:** Run `go mod tidy` to fetch dependencies

```bash
go mod tidy
```

**Problem:** Import cycle

```
import cycle not allowed
```

**Solution:** Regenerate code with correct package name:

```bash
carbon codegen -i idl.json -o pkg/program -p program
```

### Type Errors

**Problem:** Cannot use enum constant as integer

**Solution:** Use the enum type, not raw integers:

```go
direction := tokenswap.SwapDirectionAToB
```

### Runtime Errors

**Problem:** `invalid discriminator for SwapPool account`

**Solution:** Verify account type and regenerate code:

```bash
anchor build
carbon codegen -i target/idl/program.json -o pkg/program
```

**Problem:** `failed to decode account: EOF`

**Solution:** Check account size matches struct, regenerate from latest IDL

### IDL Parsing Errors

**Problem:** `failed to parse IDL`

**Solution:** Validate IDL JSON:

```bash
cat target/idl/program.json | jq .
```

### Common Mistakes

**Mistake:** Editing generated files

**Solution:** Create separate file for custom code

**Mistake:** Wrong import path

**Solution:** Use full module path: `github.com/yourorg/project/pkg/program`

### Getting Help

1. Check Examples: `examples/codegen-jennifer/`
2. Read Tests: `internal/codegen/gen/*_test.go`
3. Open Issue: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)

---

## See Also

- [Migration Guide](migration) - Migrate from old generator
- [Getting Started](getting-started) - Quick start tutorial
- [Plugin Development](plugin-development) - Creating custom plugins
- [Architecture](architecture) - System architecture overview
- [Jennifer Library](https://github.com/dave/jennifer) - Code generation toolkit

---

[View Complete Example →](https://github.com/lugondev/go-carbon/tree/main/examples/codegen-jennifer){: .btn .btn-primary }
