# Go-Carbon

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Code Generation](https://img.shields.io/badge/Codegen-Jennifer-blue?style=flat)](docs/codegen.md)
[![Performance](https://img.shields.io/badge/Performance-11x_faster-brightgreen?style=flat)](docs/performance.md)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

A lightweight, modular, **high-performance** Solana blockchain indexing framework written in Go. Go-Carbon is a Go port of the [Carbon](https://github.com/sevenlabs-hq/carbon) framework, providing a flexible pipeline architecture for processing Solana blockchain data.

> 🚀 **NEW**: Jennifer-based code generator for type-safe Go code from Anchor IDL!  
> ⚡ **NEW**: Performance optimizations - **11x faster** event parsing with **zero allocations**!

## ✨ Features

### Core Features
- **🔥 Type-Safe Code Generation**: Generate production-ready Go code from Anchor IDL with Jennifer
- **⚡ High Performance**: 11x faster data processing with Pinocchio-inspired optimizations
  - Zero-copy views for account/event parsing
  - Buffer pooling with 98% less memory allocation
  - Batch decoding with discriminator matching
- **💾 Database Storage**: Persistent storage with MongoDB and PostgreSQL support
  - Batch operations with optimized helpers
  - Connection pooling and transaction support
  - Schema migrations for PostgreSQL
- **Modular Pipeline Architecture**: Flexible data processing with configurable datasources, processors, and pipes
- **Multiple Data Types**: Support for account updates, transactions, account deletions, and block details
- **Generic Processors**: Type-safe processors with Go generics

### Event Processing
- **🔌 Plugin System**: Extensible decoder and event processor plugins
- **📝 Log Parser**: Extract and decode "Program data:" from transaction logs
- **🎯 Event Decoder**: Decode Anchor events with discriminators and Borsh serialization
- **🚀 Batch Decoding**: Optimized batch processing for high-throughput scenarios

### Performance Optimizations
- **Zero-Copy Views**: Direct memory access without allocations (11x faster)
- **Buffer Pooling**: Reusable buffers reduce GC pressure (57% faster, 98% less memory)
- **Fast Discriminator Matching**: O(1) event routing (5x faster)
- **Batch Processing**: Optimized decoding for large event sets (5-8% faster)

See [Performance Guide](docs/performance.md) for detailed benchmarks.

### Developer Experience
- **Reusable Utilities**: Centralized helpers for common operations
  - Batch operation helpers (PostgreSQL & MongoDB)
  - Filter checking utilities
  - String case conversion utilities
- **Pluggable Metrics**: Support for multiple metrics backends (Prometheus, logging, etc.)
- **Graceful Shutdown**: Configurable shutdown strategies for clean termination
- **Filter System**: Powerful filtering for selective data processing

## 🚀 New: Event Parsing & Plugin System

Go-Carbon now includes a powerful plugin system for parsing and decoding Solana program events:

### Event Parsing Flow

```
Transaction Logs → Log Parser → Event Decoder → Event Processor
```

### Built-in Plugins

- **SPL Token Decoder**: Decode SPL Token program events
- **Anchor Decoder**: Decode Anchor framework events with discriminators
- **Custom Plugins**: Easy-to-create custom decoders for any program

### Quick Event Parsing Example

```go
import (
    "github.com/lugondev/go-carbon/pkg/log"
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

// 1. Create plugin registry
registry := plugin.NewRegistry()

// 2. Register plugins
registry.MustRegister(NewMyProgramPlugin())
registry.Initialize(ctx)

// 3. Parse transaction logs
parser := log.NewParser()
programData := parser.ExtractProgramData(transactionLogs)

// 4. Decode events
decoderRegistry := registry.GetDecoderRegistry()
events, _ := decoderRegistry.DecodeAll(programData, nil)

// 5. Process events
for _, event := range events {
    registry.ProcessEvent(ctx, event)
}
```

## 📦 Installation

### From Source

```bash
git clone https://github.com/lugondev/go-carbon.git
cd go-carbon
go build -o carbon ./cmd/carbon
```

### Using Go Install

```bash
go install github.com/lugondev/go-carbon/cmd/carbon@latest
```

### As a Library

```bash
go get github.com/lugondev/go-carbon
```

## 📚 Documentation

- **[Performance Guide](docs/performance.md)** - ⚡ **NEW** High-performance optimization patterns (11x faster!)
- [Code Generation](docs/codegen.md) - Generate Go code from Anchor IDL
- [Database Storage](docs/database.md) - MongoDB and PostgreSQL integration
- [Plugin Development](docs/plugin-development.md) - Create custom event decoders
- [Architecture](docs/architecture.md) - System architecture overview
- [Examples](examples/) - Complete working examples

## 🎯 Quick Start

### 0. High-Performance Event Decoding (NEW!)

```go
import (
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/pkg/view"
)

// Traditional approach
events, _ := registry.DecodeAll(dataList, &programID)

// Optimized: 5-8% faster with zero-copy views
events, _ := registry.DecodeAllFast(dataList, &programID)

// For large batches (1000+ events)
events, _ := registry.DecodeAllParallel(dataList, &programID, 4)

// Zero-copy account parsing (11x faster!)
accountView := view.NewAccountView(rawData)
lamports := accountView.Lamports()  // No allocations!

// Fast discriminator check (54% faster)
eventView, _ := view.NewEventView(data)
if decoder.FastCanDecodeWithView(eventView) {
    event, _ := decoder.DecodeFromView(eventView)
}
```

**Performance**: 11x faster parsing, 98% less memory, zero allocations. See [Performance Guide](docs/performance.md).

### 1. Basic Pipeline

```go
package main

import (
    "context"
    "log"
    "log/slog"

    "github.com/lugondev/go-carbon/internal/datasource"
    "github.com/lugondev/go-carbon/internal/metrics"
    "github.com/lugondev/go-carbon/internal/pipeline"
)

func main() {
    // Create a pipeline with the builder pattern
    p := pipeline.Builder().
        Datasource(
            datasource.NewNamedDatasourceID("my-datasource"),
            NewMyDatasource(), // Your custom datasource
        ).
        AccountPipe(NewMyAccountPipe()). // Your account processor
        Metrics(metrics.NewCollection(metrics.NewLogMetrics(slog.Default()))).
        WithGracefulShutdown().
        Build()

    // Run the pipeline
    ctx := context.Background()
    if err := p.Run(ctx); err != nil {
        log.Fatalf("Pipeline error: %v", err)
    }
}
```

### 2. Event Parsing

```go
package main

import (
    "context"
    "crypto/sha256"
    "fmt"
    
    "github.com/gagliardetto/solana-go"
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/pkg/log"
    "github.com/lugondev/go-carbon/internal/decoder/anchor"
)

// Define your event struct
type SwapEvent struct {
    User      solana.PublicKey
    TokenIn   solana.PublicKey
    TokenOut  solana.PublicKey
    AmountIn  uint64
    AmountOut uint64
}

func main() {
    programID := solana.MustPublicKeyFromBase58("YourProgramID...")
    
    // Create decoder for your Anchor event
    discriminator := computeDiscriminator("SwapExecuted")
    swapDecoder := anchor.NewAnchorEventDecoder(
        "SwapExecuted",
        programID,
        discriminator,
        decodeSwapEvent,
    )
    
    // Register decoder
    registry := decoder.NewRegistry()
    registry.Register("swap", swapDecoder)
    
    // Parse transaction logs
    parser := log.NewParser()
    programData := parser.ExtractProgramData(transactionLogs)
    
    // Decode events
    for _, data := range programData {
        event, _ := registry.Decode(data, &programID)
        if event != nil {
            fmt.Printf("Event: %s\n", event.Name)
            // Process event...
        }
    }
}

func computeDiscriminator(eventName string) decoder.AnchorDiscriminator {
    data := []byte(fmt.Sprintf("event:%s", eventName))
    hash := sha256.Sum256(data)
    return decoder.NewAnchorDiscriminator(hash[:8])
}

func decodeSwapEvent(data []byte) (interface{}, error) {
    // Decode Borsh-serialized data
    event := &SwapEvent{}
    // ... decode fields
    return event, nil
}
```

### 3. Database Storage (Optional)

Store blockchain data to MongoDB or PostgreSQL:

```go
import (
    "github.com/lugondev/go-carbon/internal/storage"
    "github.com/lugondev/go-carbon/internal/processor/database"
    _ "github.com/lugondev/go-carbon/internal/storage/mongo"
    _ "github.com/lugondev/go-carbon/internal/storage/postgres"
)

func main() {
    cfg := &config.Config{
        Database: config.DatabaseConfig{
            Enabled: true,
            Type:    "postgres",
            Postgres: config.PostgresConfig{
                Host:     "localhost",
                Port:     5432,
                User:     "carbon",
                Password: "carbon123",
                Database: "carbon_db",
            },
        },
    }

    connMgr, _ := storage.NewConnectionManager(&cfg.Database)
    repo, _ := connMgr.Connect(ctx)
    defer connMgr.Close()

    dbProcessor := database.NewDatasourceProcessor(repo, logger)
    dbProcessor.ProcessAccountUpdate(ctx, accountUpdate)
}
```

See [Database Documentation](docs/database.md) for details.

## 🔌 Creating a Custom Plugin

### Step 1: Define Your Event

```go
type MyEvent struct {
    User   solana.PublicKey
    Amount uint64
}
```

### Step 2: Create Decoder

```go
func NewMyEventDecoder(programID solana.PublicKey) decoder.Decoder {
    disc := computeDiscriminator("MyEvent")
    
    return anchor.NewAnchorEventDecoder(
        "MyEvent",
        programID,
        disc,
        func(data []byte) (interface{}, error) {
            return decodeMyEvent(data)
        },
    )
}
```

### Step 3: Create Plugin

```go
func NewMyPlugin(programID solana.PublicKey) plugin.Plugin {
    decoders := []decoder.Decoder{
        NewMyEventDecoder(programID),
    }
    
    return anchor.NewAnchorEventPlugin(
        "my-plugin",
        programID,
        decoders,
    )
}
```

### Step 4: Use Plugin

```go
registry := plugin.NewRegistry()
registry.MustRegister(NewMyPlugin(programID))
registry.Initialize(ctx)

// Now all "MyEvent" events will be automatically decoded!
```

See [Plugin Development Guide](docs/PLUGIN_DEVELOPMENT.md) for complete documentation.

## 🛠️ Code Generation from IDL

**NEW**: Generate type-safe Go code from Anchor IDL JSON files using our Jennifer-based generator!

### Quick Start

```bash
# Generate from IDL file
carbon codegen --idl ./target/idl/my_program.json --output ./pkg/myprogram

# With custom package name
carbon codegen -i idl.json -o ./generated/swap -p tokenswap
```

### What's Generated?

| File | Description | Features |
|------|-------------|----------|
| `program.go` | Program ID, plugin factory, decoder registry | ✅ Ready-to-use plugin |
| `types.go` | Custom types (structs, enums) | ✅ Borsh serialization |
| `accounts.go` | Account structs with discriminators | ✅ Type-safe decoding |
| `events.go` | Event structs, decoders, factory | ✅ Anchor discriminators |
| `instructions.go` | Instruction builders with accounts | ✅ Instruction builders |

### Generator Highlights

✨ **Jennifer-Powered**: Type-safe code generation, no templates  
🎯 **Zero Config**: Works with any Anchor IDL (v0.1.0 - v0.29+)  
🔒 **Type Safety**: Full Go type checking at compile time  
📦 **Borsh Support**: Built-in Borsh serialization tags  
🔑 **Discriminators**: Automatic Anchor discriminator handling  
🏗️ **Clean Code**: Production-ready, `gofmt` formatted  

### Example: From IDL to Working Code

**Input**: `token_swap.idl.json`
```json
{
  "name": "token_swap",
  "instructions": [
    {
      "name": "swap",
      "accounts": [...],
      "args": [...]
    }
  ],
  "events": [...]
}
```

**Output**: Ready-to-use Go package
```go
// Use generated code immediately
registry := plugin.NewRegistry()
registry.MustRegister(tokenswap.NewTokenSwapPlugin(programID))

// Decode events from logs
events, _ := tokenswap.GetDecoderRegistry(programID).DecodeAll(logs, &programID)

// Build instructions with type safety
swapIx := tokenswap.NewSwapInstructionBuilder().
    SetAmountIn(1000000).
    SetMinAmountOut(950000).
    Build()
```

📖 **Full Documentation**: [Code Generation Guide](docs/codegen.md) | [Migration Guide](docs/MIGRATION.md)

### Using Generated Code

```go
package main

import (
    "context"
    
    "github.com/gagliardetto/solana-go"
    "github.com/yourorg/myprogram/generated/tokenswap"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

func main() {
    // Create plugin registry
    registry := plugin.NewRegistry()
    
    // Register the generated plugin
    programID := tokenswap.ProgramID
    registry.MustRegister(tokenswap.NewTokenSwapPlugin(programID))
    
    // Initialize
    ctx := context.Background()
    registry.Initialize(ctx)
    
    // Or use decoder registry directly
    decoderRegistry := tokenswap.GetDecoderRegistry(programID)
    
    // Decode events from transaction logs
    events, _ := decoderRegistry.DecodeAll(programDataList, &programID)
    for _, event := range events {
        switch e := event.Data.(type) {
        case *tokenswap.SwapExecutedEvent:
            fmt.Printf("Swap: %d -> %d\n", e.AmountIn, e.AmountOut)
        }
    }
}
```

## 📋 Examples

### Complete Examples

- [Basic Pipeline](examples/basic/) - Simple pipeline setup
- [Event Parser](examples/event-parser/) - Parse and decode events
- [Pipeline with Events](examples/pipeline-with-events/) - Full integration
- [Token Tracker](examples/token-tracker/) - Track token transfers
- [Database Storage](examples/database-storage/) - Store data in MongoDB/PostgreSQL
- [Alerts](examples/alerts/) - Alert system for specific events
- [Code Generation](examples/codegen/) - Generate code from IDL

### Running Examples

```bash
# Run event parser example
go run examples/event-parser/main.go

# Run pipeline with events
go run examples/pipeline-with-events/main.go

# Run codegen example
go run examples/codegen/main.go
```

## 🏗️ Architecture

### Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Pipeline                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐     ┌─────────────────────────────────┐   │
│  │ Datasource  │────▶│        Update Channel           │   │
│  │ (RPC/gRPC)  │     └─────────────┬───────────────────┘   │
│  └─────────────┘                   │                       │
│                                    ▼                       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Router                            │   │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐ │   │
│  │  │ Account │  │  Tx     │  │ Instr   │  │ Block   │ │   │
│  │  │ Pipes   │  │ Pipes   │  │ Pipes   │  │ Pipes   │ │   │
│  │  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘ │   │
│  └───────┼────────────┼────────────┼────────────┼──────┘   │
│          │            │            │            │          │
│          ▼            ▼            ▼            ▼          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Filters & Processors                    │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Metrics                           │   │
│  │     (Prometheus / Logging / Custom)                  │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Event Parsing Architecture

```
Transaction
     │
     ├── Logs[]
     │    │
     │    ▼
     │  Log Parser
     │    │
     │    ├─▶ "Program data: BASE64"
     │    │        │
     │    │        ▼
     │    │   Decoder Registry
     │    │        │
     │    │        ├─▶ Anchor Decoder
     │    │        ├─▶ SPL Token Decoder
     │    │        └─▶ Custom Decoder
     │    │             │
     │    │             ▼
     │    │        Decoded Event
     │    │             │
     │    │             ▼
     │    │     Event Processor
     │    │             │
     │    │             ▼
     │    │        Your Logic
     │    │        (Save DB, Send Webhook, etc.)
     │    │
     │    └─▶ "Program log: MESSAGE"
     │
     └── Instructions[]
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/decoder/...
go test ./pkg/log/...
go test ./pkg/plugin/...
```

## 📊 Project Statistics

- **Total Code**: ~10,300+ lines of Go (+2,300 from optimizations)
- **Public Packages**: 6 (`log`, `decoder`, `plugin`, `utils`, `buffer`, `view`)
- **Built-in Plugins**: 2 (SPL Token, Anchor)
- **CLI Tools**: codegen, wallet
- **Examples**: 7
- **Performance**: 11x faster parsing, 98% less memory allocation
- **Benchmarks**: Complete suite with before/after comparisons
- **Code Quality**: Refactored with DRY principles (-212 lines duplication)
- **Test Coverage**: Target >80% (work in progress)

## 🏗️ Code Architecture

### Performance-Optimized Components

Go-Carbon includes high-performance utilities inspired by Pinocchio:

#### Zero-Copy Views
```go
import "github.com/lugondev/go-carbon/pkg/view"

// 11x faster account parsing
accountView := view.NewAccountView(rawData)
lamports := accountView.Lamports()  // Direct memory access, 0 allocations

// 11x faster event parsing
eventView, _ := view.NewEventView(data)
disc := eventView.Discriminator()   // Zero-copy discriminator
```

#### Buffer Pooling
```go
import "github.com/lugondev/go-carbon/pkg/buffer"

// 57% faster, 98% less memory
buf := buffer.GetBuffer(1024)
defer buffer.PutBuffer(buf)
// Use buffer...
```

#### Batch Decoding
```go
import "github.com/lugondev/go-carbon/pkg/decoder"

// 5-8% faster batch decoding
events, _ := registry.DecodeAllFast(dataList, &programID)

// Parallel for large batches (1000+)
events, _ := registry.DecodeAllParallel(dataList, &programID, 4)
```

### Reusable Components

Go-Carbon provides several reusable utilities to simplify development:

#### Batch Operations
```go
import "github.com/lugondev/go-carbon/internal/storage"

// PostgreSQL batch helper
helper := storage.NewPostgresBatchHelper(pool)
helper.BatchInsert(ctx, query, itemCount, func(batch *pgx.Batch, i int) {
    batch.Queue(query, items[i].Field1, items[i].Field2)
})

// MongoDB batch helper
helper := storage.NewMongoBatchHelper[MyModel](collection)
helper.BatchInsert(ctx, items)
```

#### Filter Utilities
```go
import "github.com/lugondev/go-carbon/internal/filter"

// Centralized filter checking
if filter.CheckAccountFilters(dsID, filters, metadata, account) {
    // Process account
}
```

#### String Utilities
```go
import "github.com/lugondev/go-carbon/pkg/utils"

utils.ToPascalCase("my_field_name")  // MyFieldName
utils.ToSnakeCase("MyFieldName")     // my_field_name
```

## 🛣️ Roadmap

### ✅ Completed

- [x] Core pipeline architecture
- [x] Account, Transaction, Instruction processing
- [x] Filter system with helper utilities
- [x] Metrics collection
- [x] Instruction compilation implementation
- [x] Log parser framework
- [x] Event decoder system
- [x] Plugin architecture
- [x] SPL Token plugin
- [x] Anchor event plugin
- [x] Comprehensive examples
- [x] Plugin development documentation
- [x] **Code generation from Anchor IDL**
- [x] **Database storage layer (MongoDB & PostgreSQL)**
- [x] **Schema migrations for PostgreSQL**
- [x] **Batch operations for high throughput**
- [x] **Code refactoring: DRY principles applied**
- [x] **Reusable utility packages**
- [x] **⚡ Performance optimizations (Pinocchio-inspired)**
  - [x] Zero-copy views (11x faster parsing)
  - [x] Buffer pooling (57% faster, 98% less memory)
  - [x] Fast discriminator matching (5x faster routing)
  - [x] Batch decoding optimization (5-8% faster)
  - [x] Complete performance documentation

### 🚧 In Progress

- [ ] Comprehensive test suite (>80% coverage)
- [ ] Yellowstone gRPC datasource
- [ ] Helius websocket datasource
- [ ] More protocol decoders (Metaplex, Serum, etc.)

### 📅 Planned

- [ ] Prometheus metrics backend
- [ ] WebSocket live updates
- [ ] GraphQL API
- [ ] SIMD-based discriminator matching (future optimization)
- [ ] Arena allocator for event batches (future optimization)

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write tests for new features
- Update documentation
- Use meaningful commit messages
- Keep PRs focused on a single feature/fix

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Carbon](https://github.com/sevenlabs-hq/carbon) - The original Rust implementation by SevenLabs
- [Pinocchio](https://github.com/anza-xyz/pinocchio) - Inspiration for zero-copy optimizations
- [solana-go](https://github.com/gagliardetto/solana-go) - Go SDK for Solana
- [Anchor](https://www.anchor-lang.com/) - Solana development framework

## 📞 Support

- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)
- **Issues**: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions)

---

**Made with ❤️ for the Solana ecosystem**
