# Go-Carbon Architecture

This document provides a detailed overview of the go-carbon framework architecture, its components, and how they interact.

## Overview

Go-Carbon is a modular, extensible framework for indexing and processing Solana blockchain data. It follows a pipeline architecture where data flows from datasources through processors, with support for filtering, metrics, and multiple output channels.

## Core Concepts

### 1. Pipeline

The Pipeline is the central orchestrator of the framework. It manages:
- Data sources that feed updates
- Processing pipes for different update types
- Metrics collection
- Lifecycle management (startup, shutdown)

```go
p := pipeline.Builder().
    Datasource(id, datasource).
    AccountPipe(accountPipe).
    InstructionPipe(instructionPipe).
    TransactionPipe(transactionPipe).
    Metrics(metricsCollection).
    Build()
```

### 2. Update Types

The framework supports four types of updates:

| Type | Description |
|------|-------------|
| `UpdateTypeAccount` | Account state changes |
| `UpdateTypeTransaction` | Complete transaction data |
| `UpdateTypeAccountDeletion` | Account deletion events |
| `UpdateTypeBlockDetails` | Block/slot metadata |

### 3. Data Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              Pipeline                                    │
│                                                                         │
│  ┌─────────────┐    ┌─────────────────┐    ┌──────────────────────────┐│
│  │  Datasource │───▶│  Update Channel │───▶│  Update Router           ││
│  │  (RPC/gRPC) │    │  (buffered)     │    │                          ││
│  └─────────────┘    └─────────────────┘    │  ┌─────────────────────┐ ││
│                                            │  │ Account Pipes       │ ││
│  ┌─────────────┐                          │  ├─────────────────────┤ ││
│  │  Datasource │───▶     ...     ───▶     │  │ Instruction Pipes   │ ││
│  │  (Helius)   │                          │  ├─────────────────────┤ ││
│  └─────────────┘                          │  │ Transaction Pipes   │ ││
│                                            │  ├─────────────────────┤ ││
│  ┌─────────────┐                          │  │ Block Details Pipes │ ││
│  │  Datasource │───▶     ...     ───▶     │  └─────────────────────┘ ││
│  │ (Yellowst.) │                          └──────────────────────────┘│
│  └─────────────┘                                       │               │
│                                                        ▼               │
│                           ┌────────────────────────────────────────┐   │
│                           │              Processors                │   │
│                           │  ┌────────┐ ┌────────┐ ┌────────┐     │   │
│                           │  │Decoder │ │Process │ │Output  │     │   │
│                           │  └────────┘ └────────┘ └────────┘     │   │
│                           └────────────────────────────────────────┘   │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                         Metrics Collection                       │   │
│  │   ┌──────────┐  ┌──────────┐  ┌──────────┐                      │   │
│  │   │  Logger  │  │Prometheus│  │  Custom  │                      │   │
│  │   └──────────┘  └──────────┘  └──────────┘                      │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

## Components

### Datasource

Datasources are responsible for fetching blockchain data and pushing it to the pipeline.

```go
type Datasource interface {
    // Consume starts consuming updates from the datasource
    Consume(
        ctx context.Context,
        id DatasourceID,
        updates chan<- UpdateWithSource,
        metrics *metrics.Collection,
    ) error
    
    // UpdateTypes returns the types of updates this datasource provides
    UpdateTypes() []UpdateType
}
```

**Built-in Datasources:**
- `AccountMonitorDatasource` - Polls accounts via RPC
- `TransactionFetcherDatasource` - Fetches specific transactions
- `SlotMonitorDatasource` - Monitors for new slots

**Creating a Custom Datasource:**

```go
type MyDatasource struct {
    // your fields
}

func (d *MyDatasource) Consume(
    ctx context.Context,
    id datasource.DatasourceID,
    updates chan<- datasource.UpdateWithSource,
    metrics *metrics.Collection,
) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Fetch data and send updates
            update := datasource.UpdateWithSource{
                DatasourceID: id,
                Update: datasource.NewAccountUpdate(&datasource.AccountUpdate{
                    // ...
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

### Decoder

Decoders transform raw blockchain data into structured types.

```go
type AccountDecoder[T any] interface {
    DecodeAccount(account *types.Account) *DecodedAccount[T]
}

type InstructionDecoder[T any] interface {
    DecodeInstruction(instruction *types.Instruction) *DecodedInstruction[T]
}
```

**Example Decoder:**

```go
type TokenAccountDecoder struct{}

func (d *TokenAccountDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[TokenAccount] {
    // Check program ID
    if acc.Owner != TokenProgramID {
        return nil
    }
    
    // Decode data
    tokenData := decodeTokenAccount(acc.Data)
    
    return &account.DecodedAccount[TokenAccount]{
        Lamports: acc.Lamports,
        Owner:    acc.Owner,
        Data:     tokenData,
    }
}
```

### Processor

Processors handle the business logic for decoded data.

```go
type Processor[T any] interface {
    Process(ctx context.Context, data T, metrics *metrics.Collection) error
}
```

**Built-in Processor Types:**

| Type | Description |
|------|-------------|
| `ProcessorFunc[T]` | Function adapter for simple cases |
| `NoopProcessor[T]` | No-op processor for testing |
| `ChainedProcessor[T]` | Chains multiple processors |
| `ConditionalProcessor[T]` | Conditionally executes processor |
| `BatchProcessor[T]` | Batches items for bulk processing |
| `ErrorHandlingProcessor[T]` | Wraps processor with error handling |

**Example Processor:**

```go
type TokenAlertProcessor struct {
    alerter Alerter
}

func (p *TokenAlertProcessor) Process(
    ctx context.Context,
    input account.AccountProcessorInput[TokenAccount],
    metrics *metrics.Collection,
) error {
    if input.DecodedAccount.Data.Amount > threshold {
        p.alerter.Send(ctx, "Large balance detected!")
    }
    return nil
}
```

### Pipe

Pipes combine decoders and processors into processing units.

```go
// AccountPipe for account updates
accountPipe := account.NewAccountPipe(decoder, processor)

// InstructionPipe for instruction updates
instructionPipe := instruction.NewInstructionPipe(decoder, processor)

// TransactionPipe for transaction updates
transactionPipe := transaction.NewTransactionPipe(schema, decoder, processor)
```

### Filter

Filters control which updates are processed by each pipe.

```go
type Filter interface {
    FilterAccount(datasourceID, metadata, account) bool
    FilterInstruction(datasourceID, nestedInstruction) bool
    FilterTransaction(datasourceID, metadata, instructions) bool
    FilterAccountDeletion(datasourceID, deletion) bool
    FilterBlockDetails(datasourceID, details) bool
}
```

**Built-in Filters:**
- `BaseFilter` - Allows all (embed for partial implementation)
- `DatasourceFilter` - Filters by datasource ID
- `AllowAllFilter` - Allows everything
- `FilterChain` - Chains multiple filters (AND logic)

### Metrics

Metrics track pipeline performance and custom application metrics.

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

**Built-in Metrics:**
- `NoopMetrics` - Disabled metrics
- `LogMetrics` - Logs metrics via slog
- `Collection` - Aggregates multiple metrics implementations

**Built-in Metric Names:**

| Metric | Type | Description |
|--------|------|-------------|
| `updates_received` | Counter | Total updates received |
| `updates_processed` | Counter | Total updates processed |
| `updates_successful` | Counter | Successfully processed updates |
| `updates_failed` | Counter | Failed updates |
| `updates_queued` | Gauge | Current queue size |
| `updates_process_time_ms` | Histogram | Processing time in milliseconds |

## Advanced Topics

### Nested Instructions

The framework supports nested instruction processing, maintaining parent-child relationships:

```go
type NestedInstruction struct {
    Metadata          *InstructionMetadata
    Instruction       *types.Instruction
    InnerInstructions *NestedInstructions
}
```

### Transaction Schema Matching

Define schemas to match specific transaction patterns:

```go
schema := transaction.NewTransactionSchema[MyInstructionType](
    &transaction.InstructionSchemaNode[MyInstructionType]{
        Name: "swap",
        Matcher: func(ix *instruction.DecodedInstruction[MyInstructionType]) bool {
            return ix.Data.Type == "swap"
        },
    },
    &transaction.AnySchemaNode[MyInstructionType]{}, // Match any
    &transaction.InstructionSchemaNode[MyInstructionType]{
        Name: "transfer",
        // ...
    },
)
```

### Graceful Shutdown

The pipeline supports two shutdown strategies:

1. **ProcessPending** (default) - Processes all pending updates before shutdown
2. **Immediate** - Stops immediately

```go
p := pipeline.Builder().
    WithGracefulShutdown().  // ProcessPending
    // or
    WithImmediateShutdown(). // Immediate
    Build()
```

### Concurrent Processing

Datasources run concurrently, each in its own goroutine. Updates are processed sequentially from a buffered channel:

```go
p := pipeline.Builder().
    ChannelBufferSize(5000). // Configure buffer size
    Build()
```

## Best Practices

### 1. Use Type-Safe Decoders

Always return `nil` from decoders when data doesn't match expected format:

```go
func (d *MyDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[MyData] {
    if acc.Owner != expectedProgram {
        return nil // Skip, don't error
    }
    // ...
}
```

### 2. Handle Context Cancellation

Always check context in long-running operations:

```go
func (p *MyProcessor) Process(ctx context.Context, data MyData, m *metrics.Collection) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Process data
    }
}
```

### 3. Use Metrics Liberally

Track custom metrics for observability:

```go
func (p *MyProcessor) Process(ctx context.Context, data MyData, m *metrics.Collection) error {
    start := time.Now()
    defer func() {
        _ = m.RecordHistogram(ctx, "my_process_time_ms", float64(time.Since(start).Milliseconds()))
    }()
    
    _ = m.IncrementCounter(ctx, "my_items_processed", 1)
    // ...
}
```

### 4. Compose Processors

Use processor composition for reusable logic:

```go
// Chain: Log -> Validate -> Process -> Store
chainedProcessor := processor.NewChainedProcessor(
    logProcessor,
    validateProcessor,
    processProcessor,
    storeProcessor,
)

// Conditional: Only process large amounts
conditionalProcessor := processor.NewConditionalProcessor(
    expensiveProcessor,
    func(input MyInput) bool { return input.Amount > threshold },
)
```

### 5. Filter Early

Apply filters at the pipe level to skip unnecessary decoding:

```go
accountPipe := account.NewAccountPipeWithFilters(
    decoder,
    processor,
    []filter.Filter{
        filter.NewDatasourceFilter(myDatasourceID),
        myCustomFilter,
    },
)
```

## Example: Complete Pipeline

```go
func main() {
    logger := slog.Default()
    
    // Datasources
    rpcDS := rpc.NewAccountMonitorDatasource(config, accounts)
    
    // Decoders
    tokenDecoder := &TokenAccountDecoder{}
    
    // Processors
    alertProcessor := NewAlertProcessor(alerter)
    storeProcessor := NewStoreProcessor(db)
    chainedProcessor := processor.NewChainedProcessor(alertProcessor, storeProcessor)
    
    // Pipes
    tokenPipe := account.NewAccountPipe(tokenDecoder, chainedProcessor)
    
    // Metrics
    metricsCollection := metrics.NewCollection(
        metrics.NewLogMetrics(logger),
        prometheus.NewPrometheusMetrics(),
    )
    
    // Pipeline
    p := pipeline.Builder().
        Datasource(datasource.NewNamedDatasourceID("rpc"), rpcDS).
        AccountPipe(tokenPipe).
        Metrics(metricsCollection).
        MetricsFlushInterval(10 * time.Second).
        ChannelBufferSize(1000).
        WithGracefulShutdown().
        Logger(logger).
        Build()
    
    // Run
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    if err := p.Run(ctx); err != nil && err != context.Canceled {
        log.Fatal(err)
    }
}
```
