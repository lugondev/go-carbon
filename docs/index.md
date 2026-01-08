---
layout: default
title: Home
nav_order: 1
description: "Go-Carbon is a lightweight, modular Solana blockchain indexing framework written in Go"
permalink: /
---

# Go-Carbon Documentation
{: .fs-9 }

A lightweight, modular Solana blockchain indexing framework written in Go
{: .fs-6 .fw-300 }

[Get Started](/go-carbon/getting-started){: .btn .btn-primary .fs-5 .mb-4 .mb-md-0 .mr-2 }
[View on GitHub](https://github.com/lugondev/go-carbon){: .btn .fs-5 .mb-4 .mb-md-0 }

---

{: .new }
> **NEW in 2026**: Jennifer-based code generator for type-safe Go code from Anchor IDL!

## ✨ Features

- **🔥 Type-Safe Code Generation**: Generate production-ready Go code from Anchor IDL
- **Modular Pipeline Architecture**: Flexible data processing with configurable components
- **🔌 Plugin System**: Extensible decoder and event processor plugins
- **📝 Log Parser**: Extract and decode events from transaction logs
- **🎯 Event Decoder**: Decode Anchor events with discriminators
- **Generic Processors**: Type-safe processors with Go generics
- **Pluggable Metrics**: Support for multiple metrics backends
- **Graceful Shutdown**: Clean termination with configurable strategies

---

## Quick Start

### Installation

```bash
go install github.com/lugondev/go-carbon/cmd/carbon@latest
```

### Generate Code from IDL

```bash
carbon codegen --idl ./target/idl/my_program.json --output ./pkg/myprogram
```

### Basic Pipeline

```go
package main

import (
    "context"
    "github.com/lugondev/go-carbon/internal/pipeline"
    "github.com/lugondev/go-carbon/internal/datasource"
)

func main() {
    p := pipeline.Builder().
        Datasource(datasource.NewNamedDatasourceID("my-source"), myDatasource).
        AccountPipe(myAccountPipe).
        WithGracefulShutdown().
        Build()
    
    ctx := context.Background()
    if err := p.Run(ctx); err != nil {
        panic(err)
    }
}
```

---

## Documentation Structure

<div class="code-example" markdown="1">

### 🚀 Getting Started
Learn the basics and set up your first pipeline
- [Getting Started Guide](getting-started)
- [Installation](getting-started#installation)
- [Quick Start Tutorial](getting-started#quick-start)

### 📚 Core Concepts
Understand the framework architecture
- [Architecture Overview](architecture)
- [Pipeline System](architecture#pipeline)
- [Data Flow](architecture#data-flow)

### 🛠️ Code Generation
Generate type-safe Go code from Anchor IDL
- [Code Generation Guide](codegen)
- [Migration Guide](migration)
- [Generated Code Usage](codegen#using-generated-code)

### 🔌 Plugin Development
Create custom event decoders and processors
- [Plugin Development Guide](plugin-development)
- [Decoder Plugins](plugin-development#creating-a-decoder-plugin)
- [Event Processors](plugin-development#creating-an-event-processor-plugin)

### 📖 API Reference
Detailed API documentation
- Types & Interfaces
- Processor API
- Decoder API
- Plugin API

</div>

---

## Code Generation Highlights

The new Jennifer-based code generator produces production-ready Go code:

✨ **Type-Safe**: Full Go type checking at compile time  
🎯 **Zero Config**: Works with any Anchor IDL (v0.1.0 - v0.29+)  
📦 **Borsh Support**: Built-in serialization tags  
🔑 **Discriminators**: Automatic Anchor discriminator handling  
🏗️ **Clean Code**: `gofmt` formatted, ready to use  

### What's Generated?

| File | Description |
|------|-------------|
| `program.go` | Program ID, plugin factory, decoder registry |
| `types.go` | Custom types with Borsh serialization |
| `accounts.go` | Account decoders with discriminators |
| `events.go` | Event decoders with Anchor discriminators |
| `instructions.go` | Instruction builders with type safety |

[Learn more about code generation →](codegen){: .btn .btn-outline }

---

## Community & Support

<div class="code-example" markdown="1">

### 💬 Get Help

- **Documentation**: Browse this documentation site
- **GitHub Issues**: [Report bugs or request features](https://github.com/lugondev/go-carbon/issues)
- **GitHub Discussions**: [Ask questions and share ideas](https://github.com/lugondev/go-carbon/discussions)
- **Examples**: Check the [examples directory](https://github.com/lugondev/go-carbon/tree/main/examples)

### 🤝 Contributing

We welcome contributions! See our [Contributing Guide](https://github.com/lugondev/go-carbon/blob/main/CONTRIBUTING.md) to get started.

### 📄 License

Go-Carbon is open source software licensed under the [MIT License](https://github.com/lugondev/go-carbon/blob/main/LICENSE).

</div>

---

## Project Statistics

<div class="code-example" markdown="1">

- **Language**: Go 1.21+
- **Total Code**: ~8,000+ lines
- **Public Packages**: `log`, `decoder`, `plugin`
- **Built-in Plugins**: SPL Token, Anchor Events
- **CLI Tools**: `codegen`, `wallet`
- **Examples**: 6 complete working examples
- **Test Coverage**: 25+ tests for code generator

</div>

---

<div class="text-center">

**Made with ❤️ for the Solana ecosystem**

[Get Started](/go-carbon/getting-started){: .btn .btn-primary }
[View on GitHub](https://github.com/lugondev/go-carbon){: .btn .btn-outline }

</div>
