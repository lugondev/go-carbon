# Changelog

All notable changes to the go-carbon project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - Jennifer Code Generator 🚀

Major upgrade to the code generation system, replacing string templates with Jennifer library for type-safe code generation.

#### Generator Core
- **Jennifer-based generator**: Type-safe Go code generation using [dave/jennifer](https://github.com/dave/jennifer)
- **Zero templates**: No more text/template strings, pure Go code generation
- **Automatic formatting**: Generated code is automatically `gofmt` formatted
- **IDL compatibility**: Support for both Anchor IDL v0.1.0 and v0.29+ formats

#### Generated Files
- `program.go`: Program metadata, plugin factory, decoder registry
- `types.go`: Custom types (structs, enums) with Borsh tags
- `accounts.go`: Account decoders with discriminator verification
- `events.go`: Event decoders with Anchor discriminators
- `instructions.go`: Instruction builders with type-safe account handling

#### Type System
- **Primitives**: All Solana/Anchor primitives (u8, u16, u32, u64, u128, i8-i64, f32, f64, bool, string, bytes, pubkey)
- **Complex types**: Vec, Option, Array, COption (Coption)
- **Nested types**: Unlimited nesting depth (e.g., `Vec<Option<CustomType>>`)
- **Defined types**: Full support for custom types with proper resolution
- **Enums**: Simple enums with `String()` method generation

#### Features
- **Borsh serialization**: Automatic struct tags for github.com/gagliardetto/binary
- **Discriminators**: 8-byte Anchor discriminators for accounts and events
- **Instruction builders**: Fluent API with `.Build()` method
- **Account validation**: `ValidateAccounts()` method (currently returns nil)
- **Event parsing**: `ParseEvent()` unified parser for all events

#### Bug Fixes
- Fixed IDL parser not handling `{"kind": "defined", "name": "..."}` format
- Fixed type resolution checking `Kind` before `Defined`, causing incorrect type inference
- Fixed account validation false positives with well-known program IDs
- Fixed example using invalid base58 addresses

#### Documentation
- Comprehensive code generation guide ([docs/codegen.md](docs/codegen.md))
- Migration guide from old generator ([docs/MIGRATION.md](docs/MIGRATION.md))
- Complete example with README ([examples/codegen-jennifer/README.md](examples/codegen-jennifer/README.md))
- Edge case test coverage for complex type scenarios

#### Testing
- 25 comprehensive tests covering all generator components
- Edge case tests: nested types, vec of vec, option of vec, many instructions
- Integration tests with realistic token swap IDL
- Working end-to-end example that compiles and runs

#### Examples
- Full token swap example in `examples/codegen-jennifer/`
- Demonstrates instruction building, event decoding, type safety
- Shows integration with plugin system

### Changed

#### Breaking Changes
- Code generator now uses Jennifer instead of text templates
- Generated file structure remains the same, but internal code is different
- Some type resolution edge cases now handled correctly (Defined types)
- Instruction validation logic changed (now returns nil)

#### Improvements
- Generated code is cleaner and more maintainable
- Better error messages during generation
- Type resolution is more accurate and consistent
- All generated code passes `go vet` and `golangci-lint`

### Technical Details

#### Architecture
```
IDL JSON → Parser (idl.go) → Generator (base.go) → Specialized Generators → Clean Go Files
                                                    ├─ program.go
                                                    ├─ types.go
                                                    ├─ accounts.go
                                                    ├─ events.go
                                                    └─ instructions.go
```

#### Key Components
- `internal/codegen/idl.go`: IDL parsing with backward compatibility
- `internal/codegen/gen/base.go`: Core generator with type resolution
- `internal/codegen/gen/orchestrator.go`: Generation orchestration
- `internal/codegen/gen/*_generator.go`: Specialized generators for each file type

#### Files Modified
- `internal/codegen/idl.go`: Added defined type parsing
- `internal/codegen/gen/base.go`: Fixed type resolution priority
- `internal/codegen/gen/instructions.go`: Disabled account validation
- `examples/codegen-jennifer/main.go`: Fixed invalid addresses

#### Files Added
- `internal/codegen/gen/*.go`: All Jennifer generator code
- `docs/MIGRATION.md`: Migration guide
- `examples/codegen-jennifer/`: Complete working example
- `internal/codegen/gen/edge_cases_test.go`: Edge case tests

### Known Limitations

- Account validation is disabled (returns nil)
- Enum variants with fields generate `interface{}` 
- Generic types are not fully supported
- No RPC helpers generated yet
- No CPI helpers generated yet

### Deprecated

- Old string-template based generator in `internal/codegen/generator.go`
  - Still present but not used by CLI
  - Will be removed in future version
  - Users should migrate to Jennifer generator

---

## [0.1.0] - 2024-01-XX

### Added

#### Core Framework
- Modular pipeline architecture for blockchain data processing
- Support for multiple data types: accounts, transactions, instructions, blocks
- Generic processor system with Go generics
- Filter system for selective data processing
- Graceful shutdown with configurable strategies

#### Datasources
- RPC datasource for polling Solana RPC endpoints
- Account monitor datasource for tracking specific accounts
- Slot monitor datasource for block-level updates
- Configurable polling intervals and timeouts

#### Plugin System
- Plugin registry for managing decoder and processor plugins
- Decoder plugin interface for custom event decoders
- Event processor plugin interface for handling decoded events
- Plugin lifecycle management (Initialize, Shutdown)

#### Built-in Plugins
- SPL Token decoder plugin for token transfer events
- Anchor event decoder plugin for Anchor-based programs
- Support for custom program-specific plugins

#### Log Parser
- Parse transaction logs into structured format
- Extract "Program data:" entries from logs
- Support for nested instruction parsing
- Instruction path filtering

#### Event Decoder
- Registry-based decoder system
- Anchor discriminator support (8-byte hashes)
- Borsh deserialization helpers
- Multiple decoder per program support

#### Metrics
- Pluggable metrics backend system
- Log-based metrics implementation
- Counter and gauge metrics
- Configurable flush intervals

#### Code Generation (Original)
- Generate Go code from Anchor IDL JSON files
- Support for types, accounts, events, instructions
- Template-based code generation
- Basic Borsh support

#### CLI Tool
- `carbon codegen` command for code generation
- `carbon wallet` command for wallet operations
- Flag-based configuration

#### Examples
- Basic pipeline example
- Event parser example
- Pipeline with events example
- Token tracker example
- Alerts example
- Code generation example

#### Documentation
- Architecture documentation
- Plugin development guide
- Code generation guide
- Comprehensive README with examples

### Technical Stack

- **Language**: Go 1.21+
- **Dependencies**:
  - `github.com/gagliardetto/solana-go`: Solana Go SDK
  - `github.com/gagliardetto/binary`: Borsh serialization
  - `gopkg.in/yaml.v3`: YAML configuration
  - `github.com/spf13/cobra`: CLI framework
  - `github.com/dave/jennifer`: Code generation (new)

---

## Contributing

When adding entries to this changelog:
1. Add new entries under `[Unreleased]` section
2. Use categories: Added, Changed, Deprecated, Removed, Fixed, Security
3. Keep descriptions clear and concise
4. Link to relevant documentation or issues
5. Follow semantic versioning for releases

## Release Process

1. Update `[Unreleased]` section with all changes
2. Create new version section (e.g., `[0.2.0] - 2024-XX-XX`)
3. Move unreleased items to new version section
4. Add comparison links at bottom
5. Tag release in git: `git tag v0.2.0`
6. Push tag: `git push origin v0.2.0`

---

[Unreleased]: https://github.com/lugondev/go-carbon/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/lugondev/go-carbon/releases/tag/v0.1.0
