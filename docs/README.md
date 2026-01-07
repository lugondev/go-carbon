# Go-Carbon Documentation

Comprehensive documentation for the go-carbon Solana indexing framework.

## 📚 Documentation Index

### Getting Started
- **[README.md](../README.md)** - Main project overview, installation, quick start

### Code Generation
- **[codegen.md](./codegen.md)** - Current code generation guide from Anchor IDL
- **[codegen-upgrade-summary.md](./codegen-upgrade-summary.md)** - ⭐ **NEW** Summary of codegen improvements study
- **[codegen-improvements.md](./codegen-improvements.md)** - ⭐ **NEW** Detailed improvements learned from anchor-go
- **[implementation-plan-jennifer.md](./implementation-plan-jennifer.md)** - ⭐ **NEW** 10-day implementation plan
- **[jennifer-quick-reference.md](./jennifer-quick-reference.md)** - ⭐ **NEW** Jennifer code generation patterns (40+ examples)

### Plugin Development
- **[plugin-development.md](./plugin-development.md)** - Guide to creating custom event decoders

### Architecture
- **[architecture.md](./architecture.md)** - System architecture and design decisions

## 🆕 Latest Updates (2026-01-07)

### Codegen Upgrade Documentation

Chúng tôi đã nghiên cứu chi tiết [anchor-go](https://github.com/gagliardetto/anchor-go) và tạo một bộ tài liệu hoàn chỉnh về việc nâng cấp codegen:

#### 1. Codegen Upgrade Summary
- **File**: `codegen-upgrade-summary.md`
- **Content**: Tổng quan nhanh về findings, recommendations, next steps
- **For**: Decision makers, tech leads
- **Read time**: 5 phút

#### 2. Codegen Improvements Deep Dive
- **File**: `codegen-improvements.md`
- **Content**: 
  - 7 điểm mạnh chính của anchor-go
  - Ví dụ code chi tiết
  - 6 phases cải tiến
  - Timeline 12-17 ngày
  - Success metrics
- **For**: Developers, architects
- **Read time**: 20-30 phút

#### 3. Implementation Plan
- **File**: `implementation-plan-jennifer.md`
- **Content**:
  - Setup & dependencies
  - Phase-by-phase code examples
  - Testing strategy
  - Day-by-day checklist
- **For**: Implementing developers
- **Read time**: 30-40 phút

#### 4. Jennifer Quick Reference
- **File**: `jennifer-quick-reference.md`
- **Content**:
  - 40+ code generation patterns
  - Basic to advanced examples
  - Anchor/Solana specific patterns
  - Best practices
- **For**: Developers writing generators
- **Read time**: Reference material

## 🎯 Quick Navigation

### I want to...

**Generate code from IDL**
→ [codegen.md](./codegen.md)

**Understand the upgrade plan**
→ [codegen-upgrade-summary.md](./codegen-upgrade-summary.md)

**Learn Jennifer patterns**
→ [jennifer-quick-reference.md](./jennifer-quick-reference.md)

**Implement the upgrade**
→ [implementation-plan-jennifer.md](./implementation-plan-jennifer.md)

**Create a custom plugin**
→ [plugin-development.md](./plugin-development.md)

**Understand system design**
→ [architecture.md](./architecture.md)

## 📖 Reading Order

### For New Contributors

1. Start: `../README.md` - Project overview
2. Then: `architecture.md` - Understand the system
3. Next: `codegen.md` - Current state
4. Finally: `plugin-development.md` - Create extensions

### For Codegen Upgrade

1. Start: `codegen-upgrade-summary.md` - Quick overview (5 min)
2. Deep dive: `codegen-improvements.md` - What to improve (20 min)
3. Plan: `implementation-plan-jennifer.md` - How to do it (30 min)
4. Reference: `jennifer-quick-reference.md` - Code patterns (ongoing)

### For Plugin Developers

1. `../README.md` - Setup project
2. `architecture.md` - Understand pipeline
3. `plugin-development.md` - Create plugin
4. `../examples/` - Study examples

## 🔧 Documentation by Feature

### Pipeline & Processing
- Architecture overview: `architecture.md`
- Pipeline patterns: See `../examples/basic/`
- Metrics & monitoring: `architecture.md` (Metrics section)

### Event Decoding
- Decoder system: `plugin-development.md`
- Log parsing: See `../pkg/log/`
- Anchor events: See `../internal/decoder/anchor/`

### Code Generation
- **Current**: `codegen.md`
- **Future (Jennifer)**: `codegen-improvements.md`, `implementation-plan-jennifer.md`
- **Patterns**: `jennifer-quick-reference.md`

### Plugin System
- Guide: `plugin-development.md`
- Examples: `../examples/event-parser/`, `../examples/pipeline-with-events/`

## 🛠️ Development Guides

### Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/codegen/...
```

### Building
```bash
# Build CLI
go build -o carbon ./cmd/carbon

# Build for production
go build -ldflags="-s -w" -o carbon ./cmd/carbon
```

### Code Generation
```bash
# Generate from IDL
./carbon codegen --idl path/to/idl.json --output ./generated

# With package name
./carbon codegen -i idl.json -o ./gen -p myprogram
```

## 📝 Contributing to Docs

### Adding New Documentation

1. Create markdown file in `docs/`
2. Add entry to this README
3. Link from relevant documents
4. Update table of contents

### Documentation Standards

- Use clear, concise language
- Include code examples
- Add diagrams where helpful
- Link to related documents
- Keep updated with code changes

### Markdown Style

- Use `#` for titles (not underlines)
- Use `##` for sections
- Use code blocks with language tags
- Use bullet points for lists
- Use tables for comparisons

## 🔗 External Resources

### Solana Development
- [Solana Docs](https://docs.solana.com/)
- [Anchor Framework](https://www.anchor-lang.com/)
- [Solana Cookbook](https://solanacookbook.com/)

### Go Libraries
- [solana-go](https://github.com/gagliardetto/solana-go)
- [jennifer](https://github.com/dave/jennifer) - Code generation
- [borsh-go](https://github.com/gagliardetto/binary) - Borsh serialization

### Reference Implementations
- [anchor-go](https://github.com/gagliardetto/anchor-go) - Anchor Go client generator
- [carbon](https://github.com/sevenlabs-hq/carbon) - Original Rust implementation

## 📊 Documentation Stats

- **Total docs**: 8 files
- **Total size**: ~50KB
- **Code examples**: 100+
- **Diagrams**: 3
- **Last update**: 2026-01-07

## 🤝 Getting Help

- **Issues**: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)
- **Discussions**: [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions)
- **Examples**: See `../examples/` directory

## 📅 Maintenance

This documentation is maintained alongside the codebase. When making code changes:

1. Update relevant docs
2. Check cross-references
3. Update examples if needed
4. Verify code snippets compile

---

**Last Updated**: 2026-01-07  
**Maintained by**: go-carbon contributors  
**License**: MIT
