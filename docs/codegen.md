## Supported & Unsupported Features

### ✅ Fully Supported

- **Primitive Types**: All Solana/Borsh primitives (u8-u128, i8-i128, f32, f64, bool, string, bytes, pubkey)
- **Custom Structs**: With nested fields and Borsh tags
- **Simple Enums**: Variants without fields, with String() methods
- **Arrays**: Fixed-size arrays `[T; N]`
- **Vectors**: Dynamic slices `vec<T>`
- **Options**: Pointer types `option<T>` → `*T`
- **Defined Types**: References to custom types
- **Accounts**: With discriminators and decoders
- **Instructions**: Type-safe builders with validation
- **Events**: With discriminators and parsers
- **Nested Types**: Unlimited nesting depth
- **Both IDL Formats**: v0.1.0 and v0.29+ automatically detected

### ⚠️ Partially Supported

**Enum Variants with Fields:**
```json
{
  "kind": "enum",
  "variants": [
    {"name": "success", "fields": [{"name": "value", "type": "u64"}]},
    {"name": "error", "fields": [{"name": "code", "type": "u32"}]}
  ]
}
```

**Current behavior:** Generated as `interface{}` type  
**Workaround:** Manual implementation or use simple enums

**Generic Types:**
```json
{"defined": "MyType", "generics": [{"kind": "type", "type": "u64"}]}
```

**Current behavior:** Generics are ignored, base type is used  
**Workaround:** Expand generics in IDL or handle manually

### ❌ Not Yet Supported

**COption Type:**
```json
{"coption": "pubkey"}
```

**Status:** Parser supports it, generates `*T`, but may need special handling  
**Workaround:** Use `option` instead

**Tuple Types:**
```json
{"tuple": ["u64", "pubkey"]}
```

**Status:** Parsed but generates anonymous struct  
**Workaround:** Use named struct types

**PDA Seeds:**
```json
{
  "pda": {
    "seeds": [
      {"kind": "const", "value": [112, 111, 111, 108]},
      {"kind": "account", "path": "authority"}
    ]
  }
}
```

**Status:** Not implemented  
**Workaround:** Implement PDA derivation manually

### Comparison: Old vs New Generator

| Feature | Old (Templates) | New (Jennifer) |
|---------|----------------|----------------|
| Type Safety | ❌ String templates | ✅ Type-checked code |
| Instruction Builders | ❌ Not generated | ✅ Fully functional |
| Account Validation | ❌ Manual | ✅ Automatic |
| Borsh Encoding | ⚠️ Partial | ✅ Full support |
| Error Messages | ❌ Template errors | ✅ Go compiler errors |
| IDE Support | ❌ No autocomplete | ✅ Full autocomplete |
| Standalone Code | ❌ Needs framework | ✅ No dependencies |
| Extensibility | ❌ Hard to modify | ✅ Easy to extend |
| Generated Code Quality | ⚠️ Verbose | ✅ Clean, idiomatic |
| Discriminators | ✅ Supported | ✅ Supported |
| Events | ✅ Decoders only | ✅ Full parsing |
| Both IDL Formats | ❌ v0.29+ only | ✅ v0.1.0 + v0.29+ |

## Known Limitations

### 1. Validation is Disabled

The current implementation has `ValidateAccounts()` as a no-op:

```go
func (ix *InitializePoolInstruction) ValidateAccounts() error {
    return nil
}
```

**Reason:** `solana.SystemProgramID.IsZero()` returns `true`, causing false positives

**Future:** Will add proper validation that handles well-known program IDs

### 2. No RPC Helpers

Generated code doesn't include RPC fetch helpers:

```go
// Not generated (yet):
func FetchSwapPool(ctx context.Context, client *rpc.Client, address solana.PublicKey) (*SwapPool, error)
```

**Workaround:** Use `client.GetAccountInfo()` + `DecodeSwapPool()`

### 3. No CPI Helpers

No Cross-Program Invocation helpers generated:

```go
// Not generated (yet):
func SwapCPI(ctx solana.Context, ...) error
```

**Workaround:** Build instructions manually and use `solana.Invoke()`

### 4. No Transaction Builders

No high-level transaction composition:

```go
// Not generated (yet):
func BuildSwapTransaction(params SwapParams) (*solana.Transaction, error)
```

**Workaround:** Use `solana.NewTransaction()` with generated instructions
## Troubleshooting

### Build Errors

**Problem:** `undefined: binary.MarshalBorsh`

```
./instructions.go:42:11: undefined: binary.MarshalBorsh
```

**Solution:** Run `go mod tidy` to fetch dependencies

```bash
go mod tidy
```

**Problem:** Import cycle

```
import cycle not allowed
```

**Solution:** Regenerate code with correct package name:

```bash
carbon codegen -i idl.json -o pkg/program -p program
```

### Type Errors

**Problem:** Cannot use enum constant as integer

**Solution:** Use the enum type, not raw integers:

```go
// Correct:
direction := tokenswap.SwapDirectionAToB
```

### Runtime Errors

**Problem:** `invalid discriminator for SwapPool account`

**Solution:** Verify account type and regenerate code:

```bash
anchor build
carbon codegen -i target/idl/program.json -o pkg/program
```

**Problem:** `failed to decode account: EOF`

**Solution:** Check account size matches struct, regenerate from latest IDL

### IDL Parsing Errors

**Problem:** `failed to parse IDL`

**Solution:** Validate IDL JSON:

```bash
cat target/idl/program.json | jq .
```

### Common Mistakes

**Mistake:** Editing generated files

**Solution:** Create separate file for custom code

**Mistake:** Wrong import path

**Solution:** Use full module path: `github.com/yourorg/project/pkg/program`

### Getting Help

1. Check Examples: `examples/codegen-jennifer/`
2. Read Tests: `internal/codegen/gen/*_test.go`
3. Open Issue: [GitHub Issues](https://github.com/lugondev/go-carbon/issues)

## See Also

- [Complete Example](../examples/codegen-jennifer/) - Working token swap example
- [Plugin Development](plugin-development.md) - Creating custom plugins
- [Architecture](architecture.md) - System architecture overview
- [Jennifer Library](https://github.com/dave/jennifer) - Code generation toolkit

