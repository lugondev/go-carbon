# SIMD Discriminator Matching Example

This example demonstrates high-performance discriminator matching using the SIMD package.

## Features

- Single discriminator matching
- Batch discriminator matching
- Strategy comparison (Map vs SIMD vs Auto)
- Batch size performance analysis
- Real-world event discriminator computation

## Running

```bash
go run main.go
```

## Output

```
=== SIMD Discriminator Matching Example ===

Event: SwapExecuted         Discriminator: 96a61ae11c59264f
Event: PoolInitialized      Discriminator: 6476ad570cc6fee5
Event: LiquidityAdded       Discriminator: 9a1add6cee40d9a1
Event: LiquidityRemoved     Discriminator: e169d8277c74a9bd
Event: FeesCollected        Discriminator: e91775e16bb2fe08

--- Single Match Example ---
Found 'SwapExecuted' at index: 0
Unknown discriminator index: -1 (not found)

--- Batch Match Example ---
Target[0]: Found 'SwapExecuted' at index 0
Target[1]: Found 'LiquidityAdded' at index 2
Target[2]: Not found
Target[3]: Found 'PoolInitialized' at index 1

--- Performance Comparison ---
Strategy: Map     Time: 702µs  Avg: 702ns/op
Strategy: SIMD    Time: 470µs  Avg: 470ns/op
Strategy: Auto    Time: 659µs  Avg: 659ns/op

--- Batch Size Analysis ---
Batch:    1  Total:     16 ns/op  Per-item:   16 ns
Batch:   10  Total:    106 ns/op  Per-item:   10 ns
Batch:   50  Total:    318 ns/op  Per-item:    6 ns
Batch:  100  Total:    575 ns/op  Per-item:    5 ns
Batch:  500  Total:   2864 ns/op  Per-item:    5 ns
Batch: 1000  Total:   5673 ns/op  Per-item:    5 ns
```

## Key Insights

### Performance Results

1. **Strategy Comparison**: SIMD (linear scan) can be faster for specific batch sizes
2. **Batch Efficiency**: Per-item cost decreases with larger batches (amortization)
3. **Strategy Selection**: Auto strategy intelligently chooses based on batch size

### Real-World Usage

```go
// Create matcher with event discriminators
matcher := simd.NewDiscriminatorMatcher(discriminators, simd.StrategyAuto)

// Single lookup - O(1)
idx := matcher.Match(discriminator)

// Batch lookup - optimized
indices := matcher.MatchBatch(discriminators)
```

### Integration with go-carbon

See `pkg/decoder/batch.go` for how to integrate SIMD matching with the decoder system.

## See Also

- [SIMD Package Documentation](../../pkg/simd/README.md)
- [Batch Decoder](../../pkg/decoder/batch.go)
- [Performance Guide](../../docs/performance.md)
