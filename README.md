# Go-Carbon

A lightweight, modular Solana blockchain indexing framework written in Go. Go-Carbon is a Go port of the [Carbon](https://github.com/sevenlabs-hq/carbon) framework, providing a flexible pipeline architecture for processing Solana blockchain data.

## Features

- **Modular Pipeline Architecture**: Flexible data processing with configurable datasources, processors, and pipes
- **Multiple Data Types**: Support for account updates, transactions, account deletions, and block details
- **Generic Processors**: Type-safe processors with Go generics
- **Pluggable Metrics**: Support for multiple metrics backends (Prometheus, logging, etc.)
- **Graceful Shutdown**: Configurable shutdown strategies for clean termination
- **Filter System**: Powerful filtering for selective data processing

## Installation

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

## Quick Start

### Basic Pipeline Setup

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

### Creating a Custom Datasource

```go
package main

import (
    "context"

    "github.com/lugondev/go-carbon/internal/datasource"
    "github.com/lugondev/go-carbon/internal/metrics"
)

type MyDatasource struct {
    rpcURL string
}

func NewMyDatasource(rpcURL string) *MyDatasource {
    return &MyDatasource{rpcURL: rpcURL}
}

func (d *MyDatasource) Consume(
    ctx context.Context,
    id datasource.DatasourceID,
    updates chan<- datasource.UpdateWithSource,
    m *metrics.Collection,
) error {
    // Connect to your data source (RPC, WebSocket, gRPC, etc.)
    // Send updates to the channel
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Fetch and send updates
            update := datasource.UpdateWithSource{
                DatasourceID: id,
                Update: datasource.NewAccountUpdate(&datasource.AccountUpdate{
                    // ... account data
                }),
            }
            updates <- update
        }
    }
}

func (d *MyDatasource) UpdateTypes() []datasource.UpdateType {
    return []datasource.UpdateType{datasource.UpdateTypeAccount}
}
```

### Creating a Custom Processor

```go
package main

import (
    "context"
    "fmt"

    "github.com/lugondev/go-carbon/internal/metrics"
    "github.com/lugondev/go-carbon/internal/processor"
)

// TokenTransfer represents a decoded token transfer
type TokenTransfer struct {
    From   string
    To     string
    Amount uint64
    Mint   string
}

// TokenTransferProcessor processes token transfer events
type TokenTransferProcessor struct{}

func NewTokenTransferProcessor() *TokenTransferProcessor {
    return &TokenTransferProcessor{}
}

func (p *TokenTransferProcessor) Process(
    ctx context.Context,
    transfer TokenTransfer,
    m *metrics.Collection,
) error {
    fmt.Printf("Token Transfer: %s -> %s, Amount: %d, Mint: %s\n",
        transfer.From, transfer.To, transfer.Amount, transfer.Mint)
    return nil
}

// Using ProcessorFunc for simple cases
var simpleProcessor = processor.ProcessorFunc[TokenTransfer](
    func(ctx context.Context, data TokenTransfer, m *metrics.Collection) error {
        // Process the data
        return nil
    },
)
```

## Architecture

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

## Core Components

### Pipeline

The central orchestrator that manages data flow from datasources through processors.

```go
p := pipeline.Builder().
    Datasource(id, ds).
    AccountPipe(pipe).
    InstructionPipe(instrPipe).
    TransactionPipe(txPipe).
    Metrics(metricsCollection).
    ChannelBufferSize(5000).
    MetricsFlushInterval(10 * time.Second).
    WithGracefulShutdown().
    Build()
```

### Datasource

Interface for data providers that feed updates into the pipeline.

```go
type Datasource interface {
    Consume(
        ctx context.Context,
        id DatasourceID,
        updates chan<- UpdateWithSource,
        metrics *metrics.Collection,
    ) error
    
    UpdateTypes() []UpdateType
}
```

**Update Types:**
- `UpdateTypeAccount` - Account state changes
- `UpdateTypeTransaction` - Transaction data
- `UpdateTypeAccountDeletion` - Account deletion events
- `UpdateTypeBlockDetails` - Block metadata

### Processor

Generic interface for processing data with metrics support.

```go
type Processor[T any] interface {
    Process(ctx context.Context, data T, metrics *metrics.Collection) error
}
```

**Built-in Processor Types:**
- `ProcessorFunc[T]` - Function adapter
- `NoopProcessor[T]` - No-op for testing
- `ChainedProcessor[T]` - Sequential processing
- `ConditionalProcessor[T]` - Conditional execution
- `BatchProcessor[T]` - Batch processing

### Metrics

Interface for collecting pipeline performance metrics.

```go
type Metrics interface {
    Initialize(ctx context.Context) error
    Flush(ctx context.Context) error
    Shutdown(ctx context.Context) error
    UpdateGauge(ctx context.Context, name string, value float64) error
    IncrementCounter(ctx context.Context, name string, value uint64) error
    RecordHistogram(ctx context.Context, name string, value float64) error
}
```

**Built-in Implementations:**
- `NoopMetrics` - Disabled metrics
- `LogMetrics` - Logs metrics using slog

## CLI Usage

```bash
# Show help
carbon --help

# Show version
carbon version

# Wallet commands
carbon wallet generate              # Generate new wallet
carbon wallet balance <address>     # Check balance
carbon wallet airdrop <address>     # Request airdrop (devnet)

# With custom RPC
carbon --rpc https://api.mainnet-beta.solana.com wallet balance <address>

# With config file
carbon --config ~/.carbon.yaml wallet balance <address>
```

### Configuration

Create a config file at `~/.carbon.yaml`:

```yaml
rpc: https://api.mainnet-beta.solana.com
network: mainnet

# Pipeline settings
pipeline:
  channel_buffer_size: 1000
  metrics_flush_interval: 5s
  shutdown_strategy: graceful

# Metrics settings
metrics:
  enabled: true
  type: log  # log, prometheus, noop
```

## Examples

See the [examples](./examples) directory for complete examples:

- [Basic Pipeline](./examples/basic) - Simple pipeline setup
- [Token Tracker](./examples/token-tracker) - Track token transfers
- [Alerts](./examples/alerts) - Alert system for specific events

## Project Structure

```
go-carbon/
├── cmd/carbon/           # CLI application
│   ├── main.go
│   └── cmd/
│       ├── root.go       # Root command
│       ├── wallet.go     # Wallet commands
│       └── version.go    # Version command
├── internal/
│   ├── pipeline/         # Pipeline implementation
│   │   ├── pipeline.go   # Main Pipeline struct
│   │   └── builder.go    # PipelineBuilder
│   ├── datasource/       # Datasource interface
│   ├── processor/        # Processor interface
│   ├── metrics/          # Metrics implementations
│   ├── account/          # Account processing
│   ├── instruction/      # Instruction processing
│   ├── transaction/      # Transaction processing
│   ├── filter/           # Filter system
│   ├── errors/           # Error handling
│   ├── config/           # Configuration
│   └── solana/           # Solana client utilities
├── pkg/types/            # Solana types
├── configs/              # Config files
├── examples/             # Example implementations
└── docs/                 # Documentation
```

## Comparison with Rust Carbon

| Feature | Rust Carbon | Go-Carbon |
|---------|-------------|-----------|
| Pipeline Architecture | ✅ | ✅ |
| Account Processing | ✅ | ✅ |
| Transaction Processing | ✅ | ✅ |
| Instruction Processing | ✅ | ✅ |
| Metrics System | ✅ | ✅ |
| Filter System | ✅ | ✅ |
| Yellowstone gRPC | ✅ | 🚧 Planned |
| Helius Datasource | ✅ | 🚧 Planned |
| 60+ Protocol Decoders | ✅ | 🚧 Planned |
| CLI Tools | ✅ | ✅ |

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Carbon](https://github.com/sevenlabs-hq/carbon) - The original Rust implementation by SevenLabs
- [solana-go](https://github.com/gagliardetto/solana-go) - Go SDK for Solana
