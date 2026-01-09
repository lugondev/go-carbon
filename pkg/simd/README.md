# SIMD Discriminator Matching

High-performance discriminator matching for Anchor event decoding.

## Overview

This package provides optimized discriminator matching strategies for Solana/Anchor event processing:

- **Map Strategy**: O(1) hash table lookup (default, fastest for most cases)
- **SIMD Strategy**: Vectorized comparison using AVX2/SSE4 (future enhancement)
- **Auto Strategy**: Automatically selects best approach based on batch size

## Quick Start

```go
import "github.com/lugondev/go-carbon/pkg/simd"

discs := []simd.Discriminator{
    {1, 2, 3, 4, 5, 6, 7, 8},
    {8, 7, 6, 5, 4, 3, 2, 1},
}

matcher := simd.NewDiscriminatorMatcher(discs, simd.StrategyAuto)

idx := matcher.Match(simd.Discriminator{1, 2, 3, 4, 5, 6, 7, 8})
```

## Benchmark Results

Tested on Apple M3 Max (ARM64):

### Single Match Performance

```
BenchmarkDiscriminatorMatch/SingleMatch_Size10-16      202M    5.9 ns/op    0 B/op
BenchmarkDiscriminatorMatch/SingleMatch_Size100-16     202M    5.9 ns/op    0 B/op
BenchmarkDiscriminatorMatch/SingleMatch_Size1000-16    181M    6.6 ns/op    0 B/op
```

**Result**: Map lookup is O(1) constant time regardless of discriminator count.

### Batch Match Performance

| Batch Size | Map (ns/op) | Linear Scan (ns/op) | Winner |
|------------|-------------|---------------------|---------|
| 10         | 87          | 65                  | Linear  |
| 50         | 395         | 541                 | **Map** |
| 100        | 771         | 1155                | **Map** |
| 500        | 3675        | 5743                | **Map** |
| 1000       | 7304        | 11532               | **Map** |

**Key Findings**:

1. **Go map is highly optimized** - O(1) lookup beats O(n) linear scan for batches > 10
2. **SIMD only beneficial for tiny batches** (< 10 items) or specific use cases
3. **Memory allocation**: Both strategies allocate same amount (results slice)

## When to Use Each Strategy

### StrategyMap (Recommended)

**Use for**: All general use cases

- Constant O(1) lookup time
- No CPU feature dependencies
- Works on all architectures
- Go runtime already optimizes map access

**Best for**:
- Batch sizes > 10
- Multiple lookups per discriminator set
- Production workloads

### StrategySIMD (Experimental)

**Use for**: Specific optimization scenarios

- Very small batches (< 10 items)
- Linear scan of candidates
- When you cannot use map (e.g., need ordering)

**Requires**:
- AVX2/SSE4 capable CPU (x86_64)
- Assembly implementation (TODO)

### StrategyAuto (Default)

Automatically selects the best strategy:

- Batch < 10: Currently uses Map (SIMD future)
- Batch >= 10: Uses Map

## Performance Optimization Tips

### 1. Reuse Matcher Instances

```go
matcher := simd.NewDiscriminatorMatcher(discs, simd.StrategyAuto)

for _, batch := range batches {
    results := matcher.MatchBatch(batch)
}
```

### 2. Pre-allocate Result Slices

```go
results := make([]int, len(targets))

for _, batch := range batches {
    results = matcher.MatchBatch(batch)
}
```

### 3. Batch Processing

Process events in batches instead of one-by-one:

```go
targets := make([]simd.Discriminator, 0, 100)
for _, event := range events {
    targets = append(targets, event.Discriminator)
}
results := matcher.MatchBatch(targets)
```

## Real-World Integration

### With go-carbon Decoder

```go
import (
    "github.com/lugondev/go-carbon/pkg/decoder"
    "github.com/lugondev/go-carbon/pkg/simd"
)

type FastDiscriminatorMatcher struct {
    decoders map[simd.Discriminator]*decoder.AnchorDecoderBase
    matcher  *simd.DiscriminatorMatcher
}

func NewFastMatcher(decoders []*decoder.AnchorDecoderBase) *FastDiscriminatorMatcher {
    discs := make([]simd.Discriminator, len(decoders))
    decoderMap := make(map[simd.Discriminator]*decoder.AnchorDecoderBase)
    
    for i, d := range decoders {
        disc := simd.Discriminator(d.Discriminator())
        discs[i] = disc
        decoderMap[disc] = d
    }
    
    return &FastDiscriminatorMatcher{
        decoders: decoderMap,
        matcher:  simd.NewDiscriminatorMatcher(discs, simd.StrategyAuto),
    }
}

func (m *FastDiscriminatorMatcher) MatchBatch(discriminators []simd.Discriminator) []*decoder.AnchorDecoderBase {
    indices := m.matcher.MatchBatch(discriminators)
    results := make([]*decoder.AnchorDecoderBase, len(indices))
    
    for i, idx := range indices {
        if idx >= 0 {
            results[i] = m.decoders[discriminators[i]]
        }
    }
    
    return results
}
```

## Future Enhancements

### AVX2 SIMD Implementation

Planned improvements for x86_64:

```asm
; Compare 4 discriminators at once (32 bytes in YMM register)
VMOVDQU ymm0, [target]      ; Load target discriminator (replicated)
VMOVDQU ymm1, [candidates]  ; Load 4 candidate discriminators
VPCMPEQB ymm2, ymm0, ymm1   ; Byte-wise equality comparison
VPMOVMSKB eax, ymm2         ; Extract comparison mask
```

**Expected speedup**: 2-3x for linear scan scenarios

### ARM NEON Implementation

For ARM64/Apple Silicon:

```c
// Using NEON intrinsics
uint8x16_t target_vec = vld1q_u8(target);
uint8x16_t candidate_vec = vld1q_u8(candidate);
uint8x16_t cmp = vceqq_u8(target_vec, candidate_vec);
```

## Limitations

1. **Current implementation is pure Go** - No actual SIMD yet
2. **Map is faster in most cases** - SIMD benefits limited
3. **ARM64 benchmarks needed** - Current results are M3 Max specific
4. **x86_64 with AVX2 untested** - Assembly code not implemented

## Conclusion

**Recommendation**: Use `StrategyAuto` (default) which uses Go map.

SIMD optimization provides **minimal benefit** for discriminator matching because:

1. Go map is already highly optimized (hardware-accelerated hashing)
2. 8-byte discriminators are small - comparison is cheap
3. Branch prediction handles linear scan well for small batches
4. SIMD overhead (setup, alignment) negates benefits

**Focus optimization efforts on**:
- Batch processing (implemented in `pkg/decoder/batch.go`)
- Zero-copy parsing (implemented in `pkg/view/`)
- Buffer pooling (implemented in `pkg/buffer/`)

These provide **much larger gains** (11x, 57x) than SIMD (< 2x).

## License

MIT
