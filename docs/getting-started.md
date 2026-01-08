---
layout: default
title: Getting Started
nav_order: 2
description: "Quick start guide for Go-Carbon Solana indexing framework"
permalink: /getting-started
---

# Getting Started
{: .no_toc }

Get up and running with Go-Carbon in minutes
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

## Prerequisites

Before you begin, ensure you have:

- **Go 1.21 or higher** installed ([Download Go](https://go.dev/dl/))
- Basic understanding of Solana and blockchain concepts
- Familiarity with Go programming

Check your Go version:

```bash
go version
```

---

## Installation

### Option 1: Install CLI Tool (Recommended)

Install the `carbon` CLI tool globally:

```bash
go install github.com/lugondev/go-carbon/cmd/carbon@latest
```

Verify installation:

```bash
carbon --help
```

### Option 2: Clone Repository

Clone for development or to explore examples:

```bash
git clone https://github.com/lugondev/go-carbon.git
cd go-carbon
```

Build the CLI:

```bash
go build -o carbon ./cmd/carbon
```

### Option 3: As a Library

Add to your Go project:

```bash
go get github.com/lugondev/go-carbon
```

---

## Quick Start

### 1. Generate Code from Anchor IDL

If you have an Anchor program, generate Go code from its IDL:

```bash
carbon codegen --idl ./target/idl/my_program.json --output ./generated/myprogram
```

This creates a complete Go package with:
- Program metadata
- Type definitions
- Account decoders
- Event decoders
- Instruction builders

**Example output:**
```
✓ Generated program.go
✓ Generated types.go
✓ Generated accounts.go
✓ Generated events.go
✓ Generated instructions.go
```

[Learn more about code generation →](codegen)

### 2. Create Your First Pipeline

Create a new Go file `main.go`:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    
    "github.com/gagliardetto/solana-go"
    "github.com/lugondev/go-carbon/internal/account"
    "github.com/lugondev/go-carbon/internal/datasource"
    "github.com/lugondev/go-carbon/internal/datasource/rpc"
    "github.com/lugondev/go-carbon/internal/metrics"
    "github.com/lugondev/go-carbon/internal/pipeline"
    "github.com/lugondev/go-carbon/pkg/types"
)

type AccountData struct {
    Owner    types.Pubkey
    Lamports uint64
}

type AccountDecoder struct{}

func (d *AccountDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[AccountData] {
    if acc == nil {
        return nil
    }
    
    return &account.DecodedAccount[AccountData]{
        Lamports: acc.Lamports,
        Owner:    acc.Owner,
        Data: AccountData{
            Owner:    acc.Owner,
            Lamports: acc.Lamports,
        },
    }
}

type AccountProcessor struct {
    logger *slog.Logger
}

func (p *AccountProcessor) Process(
    ctx context.Context,
    input account.AccountProcessorInput[AccountData],
    m *metrics.Collection,
) error {
    p.logger.Info("Account updated",
        "pubkey", input.Metadata.Pubkey.String(),
        "lamports", input.DecodedAccount.Data.Lamports,
    )
    return nil
}

func main() {
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)
    
    rpcConfig := rpc.DefaultConfig("https://api.devnet.solana.com")
    
    accountsToMonitor := []solana.PublicKey{
        solana.MustPublicKeyFromBase58("11111111111111111111111111111111"),
    }
    
    rpcDatasource := rpc.NewAccountMonitorDatasource(rpcConfig, accountsToMonitor)
    rpcDatasource.WithLogger(logger)
    
    decoder := &AccountDecoder{}
    processor := &AccountProcessor{logger: logger}
    
    accountPipe := account.NewAccountPipe(decoder, processor)
    accountPipe.WithLogger(logger)
    
    metricsCollection := metrics.NewCollection(
        metrics.NewLogMetrics(logger),
    )
    
    p := pipeline.Builder().
        Datasource(datasource.NewNamedDatasourceID("rpc-devnet"), rpcDatasource).
        AccountPipe(accountPipe).
        Metrics(metricsCollection).
        WithGracefulShutdown().
        Logger(logger).
        Build()
    
    ctx := context.Background()
    if err := p.Run(ctx); err != nil {
        logger.Error("Pipeline error", "error", err)
        os.Exit(1)
    }
}
```

Run your pipeline:

```bash
go run main.go
```

---

## Understanding the Basics

### Pipeline Architecture

Go-Carbon uses a **pipeline architecture** where data flows through stages:

```
Datasource → Update Channel → Router → Pipes → Processors
```

**Key components:**

1. **Datasource**: Fetches data from blockchain (RPC, gRPC, WebSocket)
2. **Update Channel**: Buffers updates between datasource and router
3. **Router**: Routes updates to appropriate pipes based on type
4. **Pipes**: Process specific update types (accounts, transactions, instructions)
5. **Processors**: Your custom logic to handle decoded data

[Learn more about architecture →](architecture)

### Update Types

Go-Carbon supports four update types:

| Type | Description | Use Case |
|------|-------------|----------|
| **Account** | Account state changes | Track balances, token accounts |
| **Transaction** | Complete transaction data | Analyze transaction patterns |
| **Instruction** | Individual instructions | Decode program interactions |
| **Block** | Block/slot metadata | Track blockchain progress |

### Processing Flow

1. **Datasource** produces raw updates
2. **Decoder** converts binary data to structured types
3. **Processor** handles the decoded data (save to DB, send alerts, etc.)

```go
Raw Data → Decoder → Decoded Data → Processor → Your Logic
```

---

## Next Steps

### Explore Examples

Check out complete working examples:

```bash
cd examples/
```

- **[basic/](https://github.com/lugondev/go-carbon/tree/main/examples/basic)** - Simple pipeline setup
- **[event-parser/](https://github.com/lugondev/go-carbon/tree/main/examples/event-parser)** - Parse and decode events
- **[token-tracker/](https://github.com/lugondev/go-carbon/tree/main/examples/token-tracker)** - Track token movements
- **[alerts/](https://github.com/lugondev/go-carbon/tree/main/examples/alerts)** - DeFi alert system
- **[codegen-jennifer/](https://github.com/lugondev/go-carbon/tree/main/examples/codegen-jennifer)** - Code generation demo

Run any example:

```bash
cd examples/basic
go run main.go
```

### Read Core Documentation

- [Architecture Overview](architecture) - Understand system design
- [Code Generation Guide](codegen) - Generate Go code from IDL
- [Plugin Development](plugin-development) - Create custom plugins

### Build Your First Project

Common use cases:

**1. Token Tracker**
```bash
cd examples/token-tracker
cp config.yaml my-config.yaml
go run . -config my-config.yaml
```

**2. Event Indexer**
- Generate code from your program's IDL
- Create event processors
- Store events in database

**3. DeFi Monitor**
- Monitor specific programs (DEXs, lending protocols)
- Decode instructions and events
- Send alerts or webhooks

---

## Configuration

### RPC Datasource Configuration

```go
rpcConfig := rpc.DefaultConfig("https://api.mainnet-beta.solana.com")
rpcConfig.PollInterval = 5 * time.Second
rpcConfig.Timeout = 30 * time.Second
```

### Pipeline Configuration

```go
pipeline.Builder().
    Datasource(id, datasource).
    AccountPipe(accountPipe).
    Metrics(metrics).
    MetricsFlushInterval(10 * time.Second).
    ChannelBufferSize(1000).
    WithGracefulShutdown().
    Logger(logger).
    Build()
```

### Logging Configuration

```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

Log levels: `Debug`, `Info`, `Warn`, `Error`

---

## Troubleshooting

### Common Issues

**Issue: "connection refused"**
```
ERROR failed to connect to RPC endpoint
```
**Solution**: Check RPC endpoint URL and network connectivity

**Issue: "rate limit exceeded"**
```
WARN rate limit exceeded, retrying...
```
**Solution**: 
- Use a paid RPC provider (Helius, Quicknode, Alchemy)
- Increase `PollInterval` in RPC config
- Reduce number of monitored accounts

**Issue: "context canceled"**
```
Pipeline stopped: context canceled
```
**Solution**: This is normal for graceful shutdown (Ctrl+C)

### Getting Help

- **Documentation**: Browse this site
- **Examples**: Check `examples/` directory
- **Issues**: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions)

---

## What's Next?

Now that you have Go-Carbon running, explore advanced topics:

- [Architecture Deep Dive](architecture) - Understand internal design
- [Code Generation](codegen) - Generate type-safe code from IDL
- [Plugin Development](plugin-development) - Create custom decoders
- [Best Practices](#) - Production deployment tips

---

{: .note }
> Need help? Join our [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions) or [open an issue](https://github.com/lugondev/go-carbon/issues/new).

---

[Next: Architecture Overview →](architecture){: .btn .btn-purple }
