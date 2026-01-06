# Go-Carbon

A lightweight, modular Solana blockchain indexing framework written in Go. Go-Carbon is a Go port of the [Carbon](https://github.com/sevenlabs-hq/carbon) framework, providing a flexible pipeline architecture for processing Solana blockchain data.

## ✨ Features

- **Modular Pipeline Architecture**: Flexible data processing with configurable datasources, processors, and pipes
- **Multiple Data Types**: Support for account updates, transactions, account deletions, and block details
- **Generic Processors**: Type-safe processors with Go generics
- **🔌 Plugin System**: Extensible decoder and event processor plugins
- **📝 Log Parser**: Extract and decode "Program data:" from transaction logs
- **🎯 Event Decoder**: Decode Anchor events with discriminators and Borsh serialization
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

- [Plugin Development Guide](docs/PLUGIN_DEVELOPMENT.md) - Create custom event decoders
- [Architecture](docs/architecture.md) - System architecture overview
- [Examples](examples/) - Complete working examples

## 🎯 Quick Start

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

## 📋 Examples

### Complete Examples

- [Basic Pipeline](examples/basic/) - Simple pipeline setup
- [Event Parser](examples/event-parser/) - Parse and decode events
- [Pipeline with Events](examples/pipeline-with-events/) - Full integration
- [Token Tracker](examples/token-tracker/) - Track token transfers
- [Alerts](examples/alerts/) - Alert system for specific events

### Running Examples

```bash
# Run event parser example
go run examples/event-parser/main.go

# Run pipeline with events
go run examples/pipeline-with-events/main.go
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

- **Total Code**: ~7,500+ lines of Go
- **Public Packages**: 3 (`log`, `decoder`, `plugin`)
- **Built-in Plugins**: 2 (SPL Token, Anchor)
- **Examples**: 5
- **Test Coverage**: Target >80% (work in progress)

## 🛣️ Roadmap

### ✅ Completed

- [x] Core pipeline architecture
- [x] Account, Transaction, Instruction processing
- [x] Filter system
- [x] Metrics collection
- [x] Instruction compilation implementation
- [x] Log parser framework
- [x] Event decoder system
- [x] Plugin architecture
- [x] SPL Token plugin
- [x] Anchor event plugin
- [x] Comprehensive examples
- [x] Plugin development documentation

### 🚧 In Progress

- [ ] Comprehensive test suite (>80% coverage)
- [ ] Yellowstone gRPC datasource
- [ ] Helius websocket datasource
- [ ] More protocol decoders (Metaplex, Serum, etc.)

### 📅 Planned

- [ ] Prometheus metrics backend
- [ ] WebSocket live updates
- [ ] GraphQL API
- [ ] Database integrations (PostgreSQL, MongoDB)
- [ ] CLI improvements
- [ ] Performance benchmarks

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
- [solana-go](https://github.com/gagliardetto/solana-go) - Go SDK for Solana
- [Anchor](https://www.anchor-lang.com/) - Solana development framework

## 📞 Support

- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)
- **Issues**: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions)

---

**Made with ❤️ for the Solana ecosystem**
