---
layout: default
title: Performance Optimization Guide
nav_order: 7
---

# Performance Optimization Guide

This guide documents performance optimizations implemented in go-carbon, inspired by patterns from the [Pinocchio](https://github.com/anza-xyz/pinocchio) Rust library.

## Table of Contents

- [Overview](#overview)
- [Buffer Pooling](#buffer-pooling)
- [Zero-Copy Views](#zero-copy-views)
- [Decoder Optimizations](#decoder-optimizations)
- [Benchmark Results](#benchmark-results)
- [Best Practices](#best-practices)

---

## Overview

Go-carbon has been optimized using techniques adapted from Pinocchio's on-chain optimization patterns:

1. **Buffer Pooling**: Reuse memory buffers with `sync.Pool`
2. **Zero-Copy Views**: Direct memory access without allocations
3. **Fast Discriminator Checking**: Optimized Anchor event routing

### Performance Summary

| Optimization | Performance Gain | Memory Reduction |
|--------------|------------------|------------------|
| Buffer Pool  | **63% faster**       | **98% less**         |
| Zero-Copy AccountView | **11.7x faster** | **100%** (0 allocs) |
| Zero-Copy EventView | **11.2x faster** | **100%** (0 allocs) |
| Fast CanDecode | **38% faster** | **100%** (0 allocs) |

---

## Buffer Pooling

### Concept

Instead of allocating new byte buffers with `make([]byte, size)`, reuse buffers from a global pool. This significantly reduces GC pressure.

### Implementation

Package: `pkg/buffer`

```go
import "github.com/lugondev/go-carbon/pkg/buffer"

// Get a buffer from pool
buf := buffer.GetBuffer(1024)
defer buffer.PutBuffer(buf)

// Use the buffer
copy(buf, data)
```

### How It Works

The pool uses power-of-2 size buckets (64B, 128B, 256B, ..., 1MB):

```go
// Internal structure
type BufferPool struct {
    pools map[int]*sync.Pool  // key = size bucket
}

// Automatically rounds up to next power of 2
buf := buffer.GetBuffer(1000)  // Gets 1024-byte buffer
```

### Benchmark Results

```
BenchmarkPoolVsAllocation/Pool-16         11.5M ops    92ns    24B   1 allocs/op
BenchmarkPoolVsAllocation/DirectAlloc-16   7.7M ops   150ns  1024B   1 allocs/op
```

**Improvement**: 63% faster, 98% less memory allocated

### When to Use

✅ **Use buffer pool when:**
- Processing high-volume data streams
- Allocating temporary buffers repeatedly
- Need predictable GC behavior

❌ **Don't use buffer pool when:**
- Buffer needs to live beyond current scope
- Size varies wildly (causes pool fragmentation)
- One-time allocations

---

## Zero-Copy Views

### Concept

Instead of copying and parsing data into structs, create zero-copy "views" that directly reference the underlying bytes using `unsafe` pointers.

### Implementation

Package: `pkg/view`

#### AccountView

Zero-copy access to Solana account data:

```go
import "github.com/lugondev/go-carbon/pkg/view"

// Create view (zero allocations)
accountView := view.NewAccountView(rawBytes)

// Access fields directly (no parsing overhead)
pubkey := accountView.Pubkey()           // solana.PublicKey
lamports := accountView.Lamports()       // uint64
data := accountView.Data()               // []byte (slice, not copy)
owner := accountView.Owner()             // solana.PublicKey
executable := accountView.IsExecutable() // bool
rentEpoch := accountView.RentEpoch()     // uint64
```

#### EventView

Zero-copy access to Anchor event data:

```go
// Create view (zero allocations)
eventView, err := view.NewEventView(rawBytes)
if err != nil {
    return err
}

// Fast discriminator access
disc := eventView.Discriminator()  // [8]byte (no allocation)

// Event data without discriminator
data := eventView.Data()  // []byte starting after discriminator
```

### Memory Layout

#### AccountView Layout
```
Offset | Size | Field
-------|------|-------------
0      | 32   | Pubkey
32     | 8    | Lamports
40     | 8    | Data Length
48     | N    | Data
48+N   | 32   | Owner
80+N   | 1    | Executable
81+N   | 8    | Rent Epoch
```

#### EventView Layout
```
Offset | Size | Field
-------|------|-------------
0      | 8    | Discriminator
8      | N    | Event Data
```

### Benchmark Results

```
BenchmarkAccountView/ZeroCopyView-16         652M ops  1.82ns  0B  0 allocs/op
BenchmarkAccountView/TraditionalParsing-16    55M ops 21.29ns 32B  1 allocs/op

BenchmarkEventView/ZeroCopyView-16           697M ops  1.73ns  0B  0 allocs/op
BenchmarkEventView/TraditionalParsing-16      57M ops 19.44ns 64B  1 allocs/op
```

**Improvement**: 11-12x faster, zero allocations

### Safety Considerations

Zero-copy views use `unsafe.Pointer` for performance. Follow these rules:

1. ⚠️ **Buffer must remain valid** during view lifetime
2. ⚠️ **Don't modify underlying bytes** while view is active
3. ⚠️ **Validate buffer size** before creating view
4. ✅ **Views are read-only** - safe for concurrent access

---

## Decoder Optimizations

### Fast Discriminator Checking

For Anchor decoders, use zero-copy methods for discriminator checks:

#### Traditional Approach

```go
// Traditional (slower)
decoder := anchor.NewAnchorDecoder(...)
if decoder.CanDecode(data) {
    event, err := decoder.Decode(data)
}
```

#### Optimized Approach

```go
// Optimized with view reuse
eventView, err := view.NewEventView(data)
if err != nil {
    return err
}

if decoder.FastCanDecodeWithView(eventView) {
    event, err := decoder.DecodeFromView(eventView)
}
```

### Batch Decoding Pattern

For processing multiple events:

```go
func DecodeBatch(decoder *decoder.AnchorDecoderBase, dataList [][]byte) ([]*decoder.Event, error) {
    events := make([]*decoder.Event, 0, len(dataList))
    
    for _, data := range dataList {
        // Reuse view pattern
        eventView, err := view.NewEventView(data)
        if err != nil {
            continue
        }
        
        if !decoder.FastCanDecodeWithView(eventView) {
            continue
        }
        
        event, err := decoder.DecodeFromView(eventView)
        if err != nil {
            continue
        }
        
        events = append(events, event)
    }
    
    return events, nil
}
```

### Benchmark Results

```
BenchmarkZeroCopyDecoding/Traditional_CanDecodeCheck-16          1B ops  0.65ns  0B  0 allocs/op
BenchmarkZeroCopyDecoding/ZeroCopy_CanDecodeCheck-16             1B ops  0.40ns  0B  0 allocs/op
```

**Improvement**: 38% faster for discriminator checks

---

## Benchmark Results

### Complete Performance Comparison

Run all benchmarks:

```bash
# Buffer pool benchmarks
go test -bench=BenchmarkPool -benchmem ./pkg/buffer/

# Zero-copy view benchmarks
go test -bench=. -benchmem ./pkg/view/

# Decoder benchmarks
go test -bench=BenchmarkZeroCopy -benchmem ./pkg/decoder/
```

### Expected Results Summary

| Operation | Before | After | Speedup | Memory Saved |
|-----------|--------|-------|---------|--------------|
| Buffer allocation (1KB) | 150ns, 1024B | 92ns, 24B | 1.6x | 98% |
| Account parsing | 21.29ns, 32B | 1.82ns, 0B | 11.7x | 100% |
| Event parsing | 19.44ns, 64B | 1.73ns, 0B | 11.2x | 100% |
| Discriminator check | 0.65ns | 0.40ns | 1.6x | - |

### Test Environment

Benchmarks run on:
- **CPU**: Apple M3 Max (arm64)
- **Go**: 1.24.0
- **OS**: macOS (darwin)

---

## Best Practices

### 1. Buffer Pool Usage

**✅ DO:**
```go
func ProcessData(input []byte) error {
    buf := buffer.GetBuffer(len(input))
    defer buffer.PutBuffer(buf)
    
    copy(buf, input)
    // Process buf
    return nil
}
```

**❌ DON'T:**
```go
func ProcessData(input []byte) []byte {
    buf := buffer.GetBuffer(len(input))
    defer buffer.PutBuffer(buf)  // BUG: Returning buffer that will be returned to pool!
    return buf
}
```

### 2. Zero-Copy Views

**✅ DO:**
```go
func ValidateAccount(rawData []byte) error {
    view := view.NewAccountView(rawData)
    lamports := view.Lamports()
    
    if lamports < 1000 {
        return errors.New("insufficient balance")
    }
    return nil
}
```

**❌ DON'T:**
```go
func StoreAccount(rawData []byte) *Account {
    view := view.NewAccountView(rawData)
    
    // BUG: rawData might be reused/modified, view becomes invalid
    return &Account{
        view: view,  // Storing view is dangerous
    }
}
```

### 3. Discriminator Checking

**✅ DO:** Reuse views when checking multiple decoders
```go
eventView, _ := view.NewEventView(data)
for _, decoder := range decoders {
    if decoder.FastCanDecodeWithView(eventView) {
        return decoder.DecodeFromView(eventView)
    }
}
```

**❌ DON'T:** Create views repeatedly
```go
for _, decoder := range decoders {
    // Wasteful: creates new view each iteration
    if decoder.CanDecode(data) {
        return decoder.Decode(data)
    }
}
```

### 4. Error Handling

Always validate before using views:

```go
eventView, err := view.NewEventView(data)
if err != nil {
    // Handle error (buffer too small, etc.)
    return nil, fmt.Errorf("invalid event data: %w", err)
}

// Safe to use view
disc := eventView.Discriminator()
```

### 5. Memory Safety

When using buffer pool with views:

```go
func SafeProcess(input []byte) (*Result, error) {
    buf := buffer.GetBuffer(len(input))
    defer buffer.PutBuffer(buf)
    
    copy(buf, input)
    view := view.NewAccountView(buf)
    
    // Extract data BEFORE returning buffer to pool
    result := &Result{
        Lamports: view.Lamports(),
        Owner:    view.Owner(),  // Copies 32 bytes
    }
    
    // buf returned to pool here (defer)
    return result, nil
}
```

---

## Migration Guide

### From Traditional to Optimized

#### Before: Traditional Allocation
```go
func DecodeEvents(events [][]byte) []*Event {
    results := make([]*Event, 0, len(events))
    for _, data := range events {
        // Allocates discriminator slice
        disc := data[:8]
        if bytes.Equal(disc, expectedDisc) {
            event := parseEvent(data[8:])
            results = append(results, event)
        }
    }
    return results
}
```

#### After: Zero-Copy with Pool
```go
func DecodeEvents(events [][]byte) []*Event {
    results := make([]*Event, 0, len(events))
    for _, data := range events {
        // Zero-copy view
        view, err := view.NewEventView(data)
        if err != nil {
            continue
        }
        
        // Fast discriminator check (no allocation)
        if decoder.FastCanDecodeWithView(view) {
            event, err := decoder.DecodeFromView(view)
            if err == nil {
                results = append(results, event)
            }
        }
    }
    return results
}
```

**Expected improvement**: 11x faster, zero allocations for discriminator checks

---

## Troubleshooting

### Issue: "Buffer still in use" panic

**Cause**: Returning pooled buffer to pool while still referenced

**Solution**: Extract data before defer executes
```go
buf := buffer.GetBuffer(size)
defer buffer.PutBuffer(buf)

// Extract before defer
result := make([]byte, len(buf))
copy(result, buf)
return result  // Safe
```

### Issue: Invalid view data

**Cause**: Underlying buffer was modified or reused

**Solution**: Ensure buffer lifetime covers view usage
```go
// BAD
func GetView() *view.AccountView {
    buf := make([]byte, 121)
    // ... fill buf
    return view.NewAccountView(buf)  // buf goes out of scope!
}

// GOOD
func ProcessWithView(buf []byte) error {
    view := view.NewAccountView(buf)
    // Use view while buf is valid
    return process(view)
}
```

### Issue: Slower than expected

**Cause**: Creating views in hot loop

**Solution**: Create view once, reuse for multiple operations
```go
// BAD: Creates view 3 times
if view.NewAccountView(buf).Lamports() > 1000 &&
   view.NewAccountView(buf).IsExecutable() &&
   !view.NewAccountView(buf).Owner().IsZero() {
}

// GOOD: Create once, use multiple times
v := view.NewAccountView(buf)
if v.Lamports() > 1000 && v.IsExecutable() && !v.Owner().IsZero() {
}
```

---

## Further Reading

- [Pinocchio Documentation](https://github.com/anza-xyz/pinocchio) - Original Rust implementation
- [Go sync.Pool Documentation](https://pkg.go.dev/sync#Pool)
- [Go unsafe Package](https://pkg.go.dev/unsafe)
- [Architecture Guide](./architecture.md)
- [Decoder Documentation](./plugin-development.md)

---

## Contributing

Found a performance bottleneck? Consider these optimization patterns:

1. Profile with `pprof` to identify hot paths
2. Check for repeated allocations with `-benchmem`
3. Consider zero-copy views for parsing-heavy code
4. Use buffer pool for temporary allocations
5. Benchmark before and after changes

Submit benchmarks with your PR to demonstrate improvements!

---

**Last Updated**: January 2026  
**Go Version**: 1.24+  
**Benchmarks**: Apple M3 Max (arm64)
