# Code Generation Example

This example demonstrates the **go-carbon Jennifer-based code generator** that generates type-safe Go code from Anchor IDL files.

## Overview

The code generator produces standalone, compilable Go packages from Anchor program IDLs. The generated code includes:

- **Program metadata** - Program ID, name, version
- **Custom types** - Structs and enums with String() methods
- **Account types** - With discriminators and decoders
- **Instructions** - Type-safe builders with account validation
- **Events** - Decoders for program events

## Project Structure

```
examples/codegen-jennifer/
├── README.md              # This file
├── token_swap.idl.json    # Example Anchor IDL
├── main.go                # Usage demonstration
├── go.mod                 # Module definition
└── generated/             # Generated Go code
    ├── program.go         # Program metadata
    ├── types.go           # Custom types (SwapState, SwapDirection)
    ├── accounts.go        # Account types (SwapPool)
    ├── instructions.go    # Instruction builders
    └── events.go          # Event decoders
```

## The Example Program

The `token_swap.idl.json` defines a simple token swap program with:

**Types:**
- `SwapState` - Struct containing swap pool configuration
- `SwapDirection` - Enum for swap direction (A→B or B→A)

**Accounts:**
- `SwapPool` - On-chain state containing reserves and configuration

**Instructions:**
- `InitializePool` - Creates a new swap pool
- `Swap` - Executes a token swap

**Events:**
- `PoolInitialized` - Emitted when pool is created
- `SwapExecuted` - Emitted when swap occurs

## Running the Example

### 1. Generate Code from IDL

```bash
# From project root
go run cmd/carbon/main.go codegen \
  -i examples/codegen-jennifer/token_swap.idl.json \
  -o examples/codegen-jennifer/generated \
  -p tokenswap
```

### 2. Build and Run

```bash
cd examples/codegen-jennifer
go mod tidy
go build -o example main.go
./example
```

### Expected Output

```
=== Token Swap Example ===

Program ID: SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8
Program Name: token_swap
Program Version: 1.0.0

✓ Initialize Pool Instruction created:
  Program ID: SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8
  Accounts: 5
  Data size: 176 bytes

✓ Swap Instruction created:
  Program ID: SwaPpA9LAaLfeLi3a68M4DjnLqgtticKg6CnyNwgAC8
  Accounts: 7
  Data size: 249 bytes
  Amount In: 1000000
  Min Amount Out: 950000
  Direction: a_to_b

=== Summary ===
✓ Generated code compiles successfully
✓ Instruction builders work correctly
✓ Type-safe instruction creation
✓ Account validation included
✓ Event decoders available
```

## Code Walkthrough

### 1. Import Generated Package

```go
import (
    "github.com/gagliardetto/solana-go"
    "github.com/lugondev/go-carbon/examples/codegen-jennifer/generated"
)
```

### 2. Access Program Metadata

```go
fmt.Printf("Program ID: %s\n", tokenswap.ProgramID)
fmt.Printf("Program Name: %s\n", tokenswap.ProgramName)
fmt.Printf("Program Version: %s\n", tokenswap.ProgramVersion)
```

### 3. Create Instructions

```go
// Initialize pool instruction
initIx := tokenswap.NewInitializePoolInstruction(
    swapPool,      // Pool account
    authority,     // Authority
    tokenAMint,    // Token A mint
    tokenBMint,    // Token B mint
    systemProgram, // System program
    feeRate,       // Fee rate parameter
)

// Build Solana instruction
instruction, err := initIx.Build()
if err != nil {
    log.Fatalf("Failed to build: %v", err)
}
```

### 4. Work with Custom Types

```go
// Use generated enum
direction := tokenswap.SwapDirectionAToB

// Enum has String() method
fmt.Printf("Direction: %s\n", direction.String()) // Output: "a_to_b"

// Create swap instruction with typed parameters
swapIx := tokenswap.NewSwapInstruction(
    swapPool,
    user,
    userSource,
    userDest,
    poolSource,
    poolDest,
    tokenProgram,
    amountIn,      // uint64
    minAmountOut,  // uint64
    direction,     // SwapDirection enum
)
```

### 5. Decode Events (from transaction logs)

```go
// Get event data from transaction logs
eventData := parseEventFromLogs(txLogs)

// Decode specific event
event, err := tokenswap.DecodeSwapExecutedEvent(eventData)
if err != nil {
    log.Printf("Failed to decode: %v", err)
}

fmt.Printf("User: %s\n", event.User)
fmt.Printf("Amount In: %d\n", event.AmountIn)
fmt.Printf("Amount Out: %d\n", event.AmountOut)
```

### 6. Decode Accounts (from RPC)

```go
// Get account data from RPC
accountInfo, err := rpcClient.GetAccountInfo(ctx, poolAddress)
if err != nil {
    log.Fatal(err)
}

// Decode account data
pool, err := tokenswap.DecodeSwapPool(accountInfo.Value.Data.GetBinary())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Token A Reserve: %d\n", pool.TokenAReserve)
fmt.Printf("Token B Reserve: %d\n", pool.TokenBReserve)
```

## Generated Code Features

### Type Safety

All generated code is fully type-safe:

```go
// Compile-time type checking
var direction SwapDirection = SwapDirectionAToB

// Parameters are strongly typed
func NewSwapInstruction(
    swapPool solanago.PublicKey,
    user solanago.PublicKey,
    // ... more accounts
    amountIn uint64,              // Not interface{}
    minimumAmountOut uint64,      // Not interface{}
    direction SwapDirection,      // Custom enum type
) *SwapInstruction
```

### Borsh Serialization

All types implement Borsh encoding/decoding automatically:

```go
type SwapState struct {
    IsInitialized bool               `borsh:"is_initialized" json:"is_initialized"`
    Authority     solanago.PublicKey `borsh:"authority" json:"authority"`
    TokenAMint    solanago.PublicKey `borsh:"token_a_mint" json:"token_a_mint"`
    TokenBMint    solanago.PublicKey `borsh:"token_b_mint" json:"token_b_mint"`
    FeeRate       uint64             `borsh:"fee_rate" json:"fee_rate"`
}
```

### Instruction Building

Instructions are built in a type-safe, validated way:

```go
ix := tokenswap.NewInitializePoolInstruction(...)

// Build() validates and serializes
instruction, err := ix.Build()

// ValidateAccounts() checks required accounts
if err := ix.ValidateAccounts(); err != nil {
    log.Fatal(err)
}
```

### Discriminators

All accounts and events use discriminators for type identification:

```go
var SwapPoolDiscriminator = []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8}
var SwapExecutedEventDiscriminator = []byte{0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f}
```

## Using with Your Own IDL

### 1. Export IDL from Anchor Project

```bash
# In your Anchor project
anchor build
anchor idl parse
```

This creates `target/idl/your_program.json`

### 2. Generate Go Code

```bash
carbon codegen \
  -i ./target/idl/your_program.json \
  -o ./pkg/yourprogram \
  -p yourprogram
```

### 3. Use in Your Go Application

```go
package main

import "yourproject/pkg/yourprogram"

func main() {
    // Use generated code
    ix := yourprogram.NewYourInstruction(...)
    instruction, _ := ix.Build()
    
    // Send transaction
    // ...
}
```

## Advantages Over String Templates

The Jennifer-based generator provides:

1. **Type Safety** - All code is type-safe Go, checked at compile time
2. **No Runtime Dependencies** - Generated code is standalone
3. **Better IDE Support** - Full autocomplete and type hints
4. **Easier Debugging** - Clear error messages, no template errors
5. **Maintainable** - Generator code is readable Go, not string templates
6. **Extensible** - Easy to add new features using Jennifer API

## Supported IDL Features

### Fully Supported

- ✅ Primitive types (u8, u16, u32, u64, u128, i8, i16, i32, i64, i128, bool, string, pubkey)
- ✅ Custom struct types
- ✅ Enum types (simple variants)
- ✅ Arrays (fixed size)
- ✅ Vectors (dynamic arrays)
- ✅ Option types
- ✅ Nested types
- ✅ Account types with discriminators
- ✅ Instructions with accounts
- ✅ Events with discriminators
- ✅ Both Anchor v0.1.0 and v0.29+ IDL formats

### Partially Supported

- ⚠️ Enums with fields (treated as interface{})
- ⚠️ Generic types (generics ignored, use base type)

### Not Yet Supported

- ❌ PDA seed generation
- ❌ CPI instruction helpers
- ❌ Complex nested generics

## Troubleshooting

### Import Errors

If you see import errors:

```bash
go mod tidy
```

### Generated Code Won't Compile

Regenerate with latest version:

```bash
# From project root
go run cmd/carbon/main.go codegen -i your.idl.json -o ./generated -p pkgname
```

### Type Mismatch Errors

Check that your IDL matches the expected Anchor format. The generator supports both old (v0.1.0) and new (v0.29+) formats.

## Next Steps

1. Try generating code from your own Anchor program's IDL
2. Integrate generated code into your Go application
3. See [docs/codegen.md](../../docs/codegen.md) for detailed documentation
4. Check out [internal/codegen/gen/](../../internal/codegen/gen/) to understand the generator

## Related Documentation

- [Code Generation Guide](../../docs/codegen.md)
- [Generator Architecture](../../docs/architecture.md)
- [Plugin Development](../../docs/plugin-development.md)

## License

MIT
