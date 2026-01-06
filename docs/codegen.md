# Code Generation from Anchor IDL

Generate Go code from Anchor IDL JSON files to automatically create type-safe structs, decoders, and plugins.

## Quick Start

```bash
# Generate from IDL file
carbon codegen --idl ./target/idl/my_program.json --output ./pkg/myprogram

# With custom package name
carbon codegen -i idl.json -o ./generated/swap -p tokenswap
```

## Command Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--idl` | `-i` | Path to Anchor IDL JSON file | (required) |
| `--output` | `-o` | Output directory | `./generated` |
| `--package` | `-p` | Go package name | Program name from IDL |

## Generated Files

| File | Description |
|------|-------------|
| `program.go` | Program ID, plugin factory, decoder registry |
| `types.go` | Custom types (structs, enums) |
| `accounts.go` | Account structs with discriminators |
| `events.go` | Event structs, decoders, decoder factory |
| `instructions.go` | Instruction structs with accounts |

## IDL Structure

The generator supports Anchor IDL v0.1.0+ format:

```json
{
  "address": "YourProgramId...",
  "metadata": {
    "name": "my_program",
    "version": "0.1.0",
    "spec": "0.1.0"
  },
  "instructions": [...],
  "accounts": [...],
  "events": [...],
  "types": [...],
  "errors": [...]
}
```

## Type Mappings

| IDL Type | Go Type |
|----------|---------|
| `u8` | `uint8` |
| `u16` | `uint16` |
| `u32` | `uint32` |
| `u64` | `uint64` |
| `u128` | `[16]byte` |
| `i8` | `int8` |
| `i16` | `int16` |
| `i32` | `int32` |
| `i64` | `int64` |
| `i128` | `[16]byte` |
| `f32` | `float32` |
| `f64` | `float64` |
| `bool` | `bool` |
| `string` | `string` |
| `bytes` | `[]byte` |
| `pubkey` | `solana.PublicKey` |
| `option<T>` | `*T` |
| `vec<T>` | `[]T` |
| `[T; N]` | `[N]T` |
| defined type | `PascalCaseTypeName` |

## Usage Examples

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/project/generated/myprogram"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

func main() {
    // Use the generated ProgramID
    programID := myprogram.ProgramID
    
    // Create plugin registry
    registry := plugin.NewRegistry()
    
    // Register the generated plugin
    registry.MustRegister(myprogram.NewMyProgramPlugin(programID))
    
    // Initialize
    ctx := context.Background()
    registry.Initialize(ctx)
    
    fmt.Printf("Registered plugin for program: %s\n", programID)
}
```

### Decoding Events

```go
package main

import (
    "fmt"
    
    "github.com/yourorg/project/generated/myprogram"
    "github.com/lugondev/go-carbon/pkg/log"
)

func main() {
    programID := myprogram.ProgramID
    
    // Get decoder registry
    decoderRegistry := myprogram.GetDecoderRegistry(programID)
    
    // Parse transaction logs
    parser := log.NewParser()
    transactionLogs := []string{
        "Program XXX invoke [1]",
        "Program data: AQAAAAAAAAA...", // Base64 encoded event
        "Program XXX success",
    }
    programData := parser.ExtractProgramData(transactionLogs)
    
    // Decode all events
    events, err := decoderRegistry.DecodeAll(programData, &programID)
    if err != nil {
        fmt.Printf("Error decoding: %v\n", err)
        return
    }
    
    // Process decoded events
    for _, event := range events {
        fmt.Printf("Event: %s\n", event.Name)
        
        // Type switch for specific handling
        switch e := event.Data.(type) {
        case *myprogram.SwapExecutedEvent:
            fmt.Printf("  Swap: %d -> %d\n", e.AmountIn, e.AmountOut)
        case *myprogram.PoolInitializedEvent:
            fmt.Printf("  Pool: %s\n", e.Pool)
        }
    }
}
```

### With Pipeline Integration

```go
package main

import (
    "context"
    "log/slog"
    
    "github.com/yourorg/project/generated/myprogram"
    "github.com/lugondev/go-carbon/internal/datasource"
    "github.com/lugondev/go-carbon/internal/metrics"
    "github.com/lugondev/go-carbon/internal/pipeline"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

func main() {
    // Setup plugin registry
    registry := plugin.NewRegistry()
    registry.MustRegister(myprogram.NewMyProgramPlugin(myprogram.ProgramID))
    registry.Initialize(context.Background())
    
    // Create pipeline with event processing
    p := pipeline.Builder().
        Datasource(
            datasource.NewNamedDatasourceID("my-source"),
            NewMyDatasource(),
        ).
        TransactionPipe(NewEventProcessingPipe(registry)).
        Metrics(metrics.NewCollection(metrics.NewLogMetrics(slog.Default()))).
        WithGracefulShutdown().
        Build()
    
    // Run
    if err := p.Run(context.Background()); err != nil {
        panic(err)
    }
}
```

### Manual Event Decoding

```go
package main

import (
    "encoding/base64"
    "fmt"
    
    "github.com/yourorg/project/generated/myprogram"
)

func main() {
    // Raw event data (after base64 decoding, skip 8-byte discriminator)
    rawData := []byte{...}
    
    // Decode specific event type directly
    event, err := myprogram.DecodeSwapExecutedEvent(rawData)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("User: %s\n", event.User)
    fmt.Printf("Amount In: %d\n", event.AmountIn)
    fmt.Printf("Amount Out: %d\n", event.AmountOut)
}
```

## Generated Code Structure

### program.go

```go
package myprogram

const ProgramName = "my_program"
const ProgramVersion = "0.1.0"
var ProgramID = solana.MustPublicKeyFromBase58("...")

func NewMyProgramPlugin(programID solana.PublicKey) plugin.Plugin
func GetDecoderRegistry(programID solana.PublicKey) *decoder.Registry
```

### events.go

```go
package myprogram

// Event discriminator
var SwapExecutedEventDiscriminator = [8]byte{0x40, 0xc6, ...}

// Event struct
type SwapExecutedEvent struct {
    User      solana.PublicKey `json:"user" borsh:"user"`
    AmountIn  uint64           `json:"amount_in" borsh:"amount_in"`
    AmountOut uint64           `json:"amount_out" borsh:"amount_out"`
}

// Discriminator getter
func (e *SwapExecutedEvent) Discriminator() [8]byte

// Decoder function
func DecodeSwapExecutedEvent(data []byte) (*SwapExecutedEvent, error)

// Decoder factory
func NewEventDecoders(programID solana.PublicKey) []decoder.Decoder
func NewSwapExecutedDecoder(programID solana.PublicKey) decoder.Decoder
```

### accounts.go

```go
package myprogram

var PoolDiscriminator = [8]byte{0xf1, 0x9a, ...}

type Pool struct {
    Authority  solana.PublicKey `json:"authority" borsh:"authority"`
    Fee        uint64           `json:"fee" borsh:"fee"`
    TotalSwaps uint64           `json:"total_swaps" borsh:"total_swaps"`
}

func (a *Pool) Discriminator() [8]byte
func DecodePool(data []byte) (*Pool, error)
```

### instructions.go

```go
package myprogram

var SwapDiscriminator = [8]byte{0xf8, 0xc6, ...}

type SwapInstruction struct {
    AmountIn     uint64 `json:"amount_in" borsh:"amount_in"`
    MinAmountOut uint64 `json:"min_amount_out" borsh:"min_amount_out"`
}

type SwapAccounts struct {
    Pool         solana.PublicKey
    User         solana.PublicKey
    TokenProgram solana.PublicKey
}
```

### types.go

```go
package myprogram

// Enum type
type SwapDirection uint8

const (
    SwapDirectionAtoB SwapDirection = iota
    SwapDirectionBtoA
)

// Struct type
type PoolConfig struct {
    Fee         uint64 `json:"fee" borsh:"fee"`
    MaxSlippage uint16 `json:"max_slippage" borsh:"max_slippage"`
    Paused      bool   `json:"paused" borsh:"paused"`
}
```

## Workflow

### 1. Get IDL from Anchor Project

```bash
# Build Anchor program
anchor build

# IDL is generated at
ls target/idl/my_program.json
```

### 2. Generate Go Code

```bash
# From project root
carbon codegen \
    --idl ./target/idl/my_program.json \
    --output ./pkg/myprogram \
    --package myprogram
```

### 3. Use Generated Code

```go
import "github.com/yourorg/project/pkg/myprogram"

// Ready to use!
registry.MustRegister(myprogram.NewMyProgramPlugin(myprogram.ProgramID))
```

## Best Practices

### 1. Regenerate on IDL Changes

When your Anchor program changes, regenerate the Go code:

```bash
anchor build && carbon codegen -i target/idl/my_program.json -o pkg/myprogram
```

### 2. Don't Edit Generated Files

Generated files will be overwritten. For custom logic, create separate files:

```
pkg/myprogram/
  accounts.go      # Generated
  events.go        # Generated
  program.go       # Generated
  types.go         # Generated
  instructions.go  # Generated
  custom.go        # Your custom code
  handlers.go      # Your event handlers
```

### 3. Version Control Generated Code

Commit generated code to version control for reproducible builds:

```bash
git add pkg/myprogram/
git commit -m "Regenerate code from IDL v0.2.0"
```

### 4. Add to Build Pipeline

```makefile
.PHONY: codegen
codegen:
	anchor build
	carbon codegen -i target/idl/my_program.json -o pkg/myprogram
```

## Limitations

1. **Complex Borsh Types**: Some complex Borsh types (nested options, custom serialization) may need manual decoder implementation.

2. **Account Deserialization**: Account `Decode*` functions have TODO placeholders for full Borsh deserialization. For production use, implement complete deserialization or use a Borsh library.

3. **Instruction Building**: Generated code focuses on decoding. For building instructions, use [solana-go](https://github.com/gagliardetto/solana-go) directly.

## Troubleshooting

### "IDL file not found"

Ensure the IDL path is correct and the file exists:

```bash
ls -la ./target/idl/my_program.json
```

### "Failed to parse IDL JSON"

Validate your IDL JSON format:

```bash
cat target/idl/my_program.json | jq .
```

### Build Errors in Generated Code

1. Run `go mod tidy` to fetch dependencies
2. Check import paths match your module
3. Ensure go-carbon is properly imported

```bash
go mod tidy
go build ./pkg/myprogram/...
```

## See Also

- [Plugin Development Guide](plugin-development.md) - Creating custom plugins
- [Architecture](architecture.md) - System architecture overview
- [Examples](../examples/codegen/) - Complete codegen example
