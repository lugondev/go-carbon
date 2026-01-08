---
layout: default
title: Migration Guide
nav_order: 5
description: "Migrate from old generator to Jennifer-based code generator"
permalink: /migration
---

# Migration Guide: Old Generator → Jennifer Generator

This guide helps you migrate from the old string-template based code generator to the new Jennifer-based generator.

## TL;DR

**The new generator is a drop-in replacement** with better type safety and cleaner code. Most users can simply regenerate and fix minor API changes.

```bash
# Old way
carbon codegen -i idl.json -o pkg/program

# New way (same command!)
carbon codegen -i idl.json -o pkg/program
```

## What Changed

### Breaking Changes

#### 1. Generated File Structure

**Old Generator:**
```
pkg/program/
├── program.go    # Contains plugin, decoder registry
├── types.go
├── accounts.go
├── events.go
└── instructions.go
```

**New Generator:**
```
pkg/program/
├── program.go    # Only program metadata (ID, name, version)
├── types.go      # Custom types with String() methods
├── accounts.go   # Account decoders
├── events.go     # Event decoders + ParseEvent()
└── instructions.go  # Instruction BUILDERS (new!)
```

**Impact:** Low - same files, different contents

**Action:** Regenerate code, no changes needed in most cases

#### 2. Plugin System Removed from Generated Code

**Old Generator:**
```go
// Generated code included plugin factory
func NewMyProgramPlugin(programID solana.PublicKey) plugin.Plugin {
    // ... plugin implementation
}

func GetDecoderRegistry(programID solana.PublicKey) *decoder.Registry {
    // ... decoder registry
}
```

**New Generator:**
```go
// Only program metadata
const ProgramName = "my_program"
const ProgramVersion = "1.0.0"
var ProgramID = solana.MustPublicKeyFromBase58("...")
```

**Impact:** High - if you used plugin system

**Action Required:**

**Before (with plugin):**
```go
registry := plugin.NewRegistry()
registry.MustRegister(myprogram.NewMyProgramPlugin(myprogram.ProgramID))
registry.Initialize(ctx)
```

**After (direct usage):**
```go
// Just use the generated types directly
ix := myprogram.NewSwapInstruction(...)
instruction, _ := ix.Build()

event, _ := myprogram.DecodeSwapExecutedEvent(data)
```

**If you need plugin system**, wrap generated code:

```go
// custom_plugin.go
package myprogram

import (
    "github.com/lugondev/go-carbon/pkg/plugin"
    "github.com/lugondev/go-carbon/pkg/decoder"
)

func NewPlugin() plugin.Plugin {
    return &myProgramPlugin{
        programID: ProgramID,
    }
}

type myProgramPlugin struct {
    programID solana.PublicKey
}

func (p *myProgramPlugin) Name() string {
    return ProgramName
}

func (p *myProgramPlugin) ProgramID() solana.PublicKey {
    return p.programID
}

// Implement other plugin.Plugin methods...
```

#### 3. Instruction Building API

**Old Generator:**
```go
// Only instruction structs generated
type SwapInstruction struct {
    AmountIn     uint64
    MinAmountOut uint64
}

type SwapAccounts struct {
    Pool solana.PublicKey
    User solana.PublicKey
}

// You had to build manually
accounts := []*solana.AccountMeta{...}
data := encodeInstruction(...)
ix := solana.NewInstruction(programID, accounts, data)
```

**New Generator:**
```go
// Constructor + Build() method generated
func NewSwapInstruction(
    pool solana.PublicKey,
    user solana.PublicKey,
    userSource solana.PublicKey,
    userDest solana.PublicKey,
    amountIn uint64,
    minAmountOut uint64,
) *SwapInstruction

func (ix *SwapInstruction) Build() (*solana.GenericInstruction, error)
func (ix *SwapInstruction) ValidateAccounts() error

// Usage
ix := myprogram.NewSwapInstruction(pool, user, ..., amountIn, minAmountOut)
instruction, err := ix.Build()
```

**Impact:** High - API completely different

**Action Required:**

**Before:**
```go
// Old manual approach
ix := myprogram.SwapInstruction{
    AmountIn:     1000000,
    MinAmountOut: 950000,
}

accounts := &myprogram.SwapAccounts{
    Pool: poolPubkey,
    User: userPubkey,
}

data, _ := borsh.Serialize(ix)
fullData := append(myprogram.SwapDiscriminator[:], data...)

accountMetas := []*solana.AccountMeta{
    {PublicKey: accounts.Pool, IsWritable: true, IsSigner: false},
    {PublicKey: accounts.User, IsWritable: false, IsSigner: true},
}

instruction := solana.NewInstruction(myprogram.ProgramID, accountMetas, fullData)
```

**After:**
```go
// New builder approach
ix := myprogram.NewSwapInstruction(
    poolPubkey,
    userPubkey,
    userSourcePubkey,
    userDestPubkey,
    1000000,  // amountIn
    950000,   // minAmountOut
)

instruction, err := ix.Build()
if err != nil {
    return err
}
```

**Benefits:**
- ✅ Type-safe parameters
- ✅ Compile-time checking
- ✅ Automatic validation
- ✅ Less boilerplate

#### 4. Enum String() Methods

**Old Generator:**
```go
type SwapDirection uint8

const (
    SwapDirectionAToB SwapDirection = 0
    SwapDirectionBToA SwapDirection = 1
)

// No String() method
```

**New Generator:**
```go
type SwapDirection uint8

const (
    SwapDirectionAToB SwapDirection = 0
    SwapDirectionBToA SwapDirection = 1
)

func (e SwapDirection) String() string {
    switch e {
    case SwapDirectionAToB:
        return "a_to_b"
    case SwapDirectionBToA:
        return "b_to_a"
    default:
        return fmt.Sprintf("SwapDirection(%d)", e)
    }
}
```

**Impact:** Low - additional feature

**Action:** None required, but you can now use:

```go
direction := myprogram.SwapDirectionAToB
fmt.Printf("Direction: %s", direction) // "Direction: a_to_b"
```

#### 5. Event Parsing

**Old Generator:**
```go
// Only decode functions
func DecodeSwapExecutedEvent(data []byte) (*SwapExecutedEvent, error)

// You had to check discriminators manually
```

**New Generator:**
```go
// Decode functions + unified parser
func DecodeSwapExecutedEvent(data []byte) (*SwapExecutedEvent, error)

func ParseEvent(data []byte) (interface{}, string, error) {
    // Automatically detects event type by discriminator
    // Returns: (event, eventName, error)
}
```

**Impact:** Low - additional feature

**Action:** Optionally use `ParseEvent()` for auto-detection:

**Before:**
```go
// Manual type checking
if bytes.HasPrefix(data, SwapExecutedEventDiscriminator[:]) {
    event, _ := DecodeSwapExecutedEvent(data)
    // ...
}
```

**After:**
```go
// Automatic detection
event, eventName, err := myprogram.ParseEvent(data)
switch e := event.(type) {
case *myprogram.SwapExecutedEvent:
    fmt.Printf("Swap: %d -> %d\n", e.AmountIn, e.AmountOut)
case *myprogram.PoolInitializedEvent:
    fmt.Printf("Pool: %s\n", e.Pool)
}
```

### Non-Breaking Changes

#### 1. Defined Type Resolution

**Old Generator:** Sometimes failed to resolve `{"defined": "TypeName"}`

**New Generator:** Always resolves correctly

**Impact:** Fixes bugs where types appeared as `interface{}`

**Action:** Regenerate - types will now be correct

#### 2. IDL Format Support

**Old Generator:** Only Anchor v0.29+ format

**New Generator:** Both v0.1.0 and v0.29+ formats

**Impact:** Works with older IDL files

**Action:** None

#### 3. Code Quality

**Old Generator:**
- String concatenation
- Verbose output
- Hard to read

**New Generator:**
- Jennifer-based
- Clean, idiomatic Go
- Easy to read and maintain

**Action:** Enjoy better generated code!

## Step-by-Step Migration

### Step 1: Backup Current Generated Code

```bash
cp -r pkg/myprogram pkg/myprogram.old
```

### Step 2: Update go-carbon

```bash
go get -u github.com/lugondev/go-carbon@latest
go install github.com/lugondev/go-carbon/cmd/carbon@latest
```

### Step 3: Regenerate Code

```bash
carbon codegen \
    -i ../anchor-project/target/idl/my_program.json \
    -o pkg/myprogram \
    -p myprogram
```

### Step 4: Update Imports (if needed)

No changes needed - package name stays the same

### Step 5: Update Instruction Building Code

Find all instruction creation code and replace with new builder API.

**Find:**
```bash
grep -r "SwapInstruction{" ./
grep -r "solana.NewInstruction" ./
```

**Replace with:**
```go
ix := myprogram.NewSwapInstruction(...)
instruction, err := ix.Build()
```

### Step 6: Remove Plugin Code (if not needed)

If you only need generated types and don't use the plugin system:

**Remove:**
```go
registry := plugin.NewRegistry()
registry.MustRegister(myprogram.NewMyProgramPlugin(...))
```

**Use directly:**
```go
ix := myprogram.NewSwapInstruction(...)
instruction, _ := ix.Build()
```

### Step 7: Test

```bash
go test ./...
go build ./...
```

### Step 8: Clean Up

```bash
rm -rf pkg/myprogram.old
```

## Example Migration

### Before: Using Old Generator

```go
package main

import (
    "context"
    
    "github.com/gagliardetto/solana-go"
    "github.com/yourorg/project/pkg/tokenswap"
    "github.com/lugondev/go-carbon/pkg/plugin"
)

func main() {
    // Plugin-based approach
    registry := plugin.NewRegistry()
    registry.MustRegister(tokenswap.NewTokenSwapPlugin(tokenswap.ProgramID))
    registry.Initialize(context.Background())
    
    // Manual instruction building
    ix := tokenswap.SwapInstruction{
        AmountIn:     1000000,
        MinAmountOut: 950000,
    }
    
    accounts := &tokenswap.SwapAccounts{
        Pool: poolKey,
        User: userKey,
    }
    
    data, _ := borsh.Serialize(ix)
    fullData := append(tokenswap.SwapDiscriminator[:], data...)
    
    accountMetas := []*solana.AccountMeta{
        {PublicKey: accounts.Pool, IsWritable: true, IsSigner: false},
        {PublicKey: accounts.User, IsWritable: false, IsSigner: true},
        // ... more accounts
    }
    
    instruction := solana.NewInstruction(
        tokenswap.ProgramID,
        accountMetas,
        fullData,
    )
    
    // Send transaction...
}
```

### After: Using New Generator

```go
package main

import (
    "github.com/gagliardetto/solana-go"
    "github.com/yourorg/project/pkg/tokenswap"
)

func main() {
    // Direct type-safe instruction building
    ix := tokenswap.NewSwapInstruction(
        poolKey,        // pool
        userKey,        // user
        userSourceKey,  // userSource
        userDestKey,    // userDestination
        poolSourceKey,  // poolSource
        poolDestKey,    // poolDestination
        tokenProgramID, // tokenProgram
        1000000,        // amountIn
        950000,         // minAmountOut
        tokenswap.SwapDirectionAToB, // direction
    )
    
    instruction, err := ix.Build()
    if err != nil {
        panic(err)
    }
    
    // Send transaction...
}
```

**Lines of code:** 40 → 15 (62% reduction)  
**Type safety:** Manual → Compile-time checked  
**Boilerplate:** High → Minimal

## Timeline & Compatibility

### Recommended Timeline

- **Week 1:** Update go-carbon, regenerate code
- **Week 2:** Update instruction building code
- **Week 3:** Remove plugin code (if not needed)
- **Week 4:** Test and deploy

### Backward Compatibility

**The CLI command is backward compatible:**

```bash
# Works with both old and new generator
carbon codegen -i idl.json -o pkg/program
```

**Generated code is NOT backward compatible:**

You must update your application code when regenerating.

### Can I Use Both?

**No.** The new generator completely replaces the old one.

**Option 1:** Migrate fully (recommended)

**Option 2:** Stay on old version until ready:

```bash
# Pin to old version
go get github.com/lugondev/go-carbon@v0.x.x
```

## FAQ

### Q: Do I have to migrate?

**A:** Not immediately, but recommended for:
- Better type safety
- Cleaner code
- Future features
- Bug fixes

### Q: How long does migration take?

**A:** 
- Small projects: 1-2 hours
- Medium projects: 1 day
- Large projects: 2-3 days

### Q: Will my transactions still work?

**A:** Yes! On-chain format is unchanged. Only Go API changed.

### Q: Can I migrate incrementally?

**A:** No. Generate all or none. But you can migrate one program at a time if you have multiple.

### Q: What if I find bugs?

**A:** Report issues: https://github.com/lugondev/go-carbon/issues

### Q: Can I customize generated code?

**A:** Don't edit generated files. Create separate files for custom logic:

```
pkg/myprogram/
├── accounts.go      # Generated - don't edit
├── events.go        # Generated - don't edit
├── instructions.go  # Generated - don't edit
├── custom.go        # Your code - edit freely
└── helpers.go       # Your code - edit freely
```

## Getting Help

**Documentation:**
- [Code Generation Guide](codegen.md)
- [Example Project](../examples/codegen-jennifer/)
- [Jennifer Library Docs](https://github.com/dave/jennifer)

**Support:**
- [GitHub Issues](https://github.com/lugondev/go-carbon/issues)
- [GitHub Discussions](https://github.com/lugondev/go-carbon/discussions)

## Summary

| Aspect | Old Generator | New Generator |
|--------|--------------|---------------|
| **Type Safety** | Manual, error-prone | Compile-time checked |
| **API** | Low-level, verbose | High-level builders |
| **Code Quality** | Template artifacts | Clean, idiomatic Go |
| **IDL Support** | v0.29+ only | v0.1.0 + v0.29+ |
| **Instruction Building** | Manual | Automatic |
| **Validation** | Manual | Built-in |
| **Dependencies** | go-carbon framework | Standalone |
| **Migration Effort** | N/A | Low-Medium |

**Bottom line:** The new generator produces better code with better APIs. Migration is straightforward for most projects.
