package main

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/lugondev/go-carbon/pkg/simd"
)

func main() {
	fmt.Println("=== SIMD Discriminator Matching Example ===")
	fmt.Println()

	eventNames := []string{
		"SwapExecuted",
		"PoolInitialized",
		"LiquidityAdded",
		"LiquidityRemoved",
		"FeesCollected",
	}

	discriminators := make([]simd.Discriminator, len(eventNames))
	eventMap := make(map[simd.Discriminator]string)

	for i, name := range eventNames {
		disc := computeAnchorDiscriminator(name)
		discriminators[i] = disc
		eventMap[disc] = name
		fmt.Printf("Event: %-20s Discriminator: %x\n", name, disc)
	}

	fmt.Println("\n--- Single Match Example ---")
	matcher := simd.NewDiscriminatorMatcher(discriminators, simd.StrategyAuto)

	swapDisc := computeAnchorDiscriminator("SwapExecuted")
	idx := matcher.Match(swapDisc)
	if idx >= 0 {
		fmt.Printf("Found 'SwapExecuted' at index: %d\n", idx)
	}

	unknownDisc := simd.Discriminator{0, 0, 0, 0, 0, 0, 0, 0}
	idx = matcher.Match(unknownDisc)
	fmt.Printf("Unknown discriminator index: %d (not found)\n", idx)

	fmt.Println("\n--- Batch Match Example ---")
	targets := []simd.Discriminator{
		computeAnchorDiscriminator("SwapExecuted"),
		computeAnchorDiscriminator("LiquidityAdded"),
		{0, 0, 0, 0, 0, 0, 0, 0},
		computeAnchorDiscriminator("PoolInitialized"),
	}

	results := matcher.MatchBatch(targets)
	for i, result := range results {
		if result >= 0 {
			eventName := eventNames[result]
			fmt.Printf("Target[%d]: Found '%s' at index %d\n", i, eventName, result)
		} else {
			fmt.Printf("Target[%d]: Not found\n", i)
		}
	}

	fmt.Println("\n--- Performance Comparison ---")
	benchmarkStrategies(discriminators, 1000)

	fmt.Println("\n--- Batch Size Analysis ---")
	analyzeBatchSizes(discriminators)
}

func computeAnchorDiscriminator(eventName string) simd.Discriminator {
	data := []byte(fmt.Sprintf("event:%s", eventName))
	hash := sha256.Sum256(data)
	var disc simd.Discriminator
	copy(disc[:], hash[:8])
	return disc
}

func benchmarkStrategies(discriminators []simd.Discriminator, iterations int) {
	targets := make([]simd.Discriminator, 100)
	for i := range targets {
		targets[i] = discriminators[i%len(discriminators)]
	}

	strategies := []struct {
		name     string
		strategy simd.MatcherStrategy
	}{
		{"Map", simd.StrategyMap},
		{"SIMD", simd.StrategySIMD},
		{"Auto", simd.StrategyAuto},
	}

	for _, strat := range strategies {
		matcher := simd.NewDiscriminatorMatcher(discriminators, strat.strategy)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			_ = matcher.MatchBatch(targets)
		}
		elapsed := time.Since(start)

		fmt.Printf("Strategy: %-6s  Time: %v  Avg: %v/op\n",
			strat.name,
			elapsed,
			elapsed/time.Duration(iterations),
		)
	}
}

func analyzeBatchSizes(discriminators []simd.Discriminator) {
	batchSizes := []int{1, 10, 50, 100, 500, 1000}
	iterations := 10000

	for _, size := range batchSizes {
		targets := make([]simd.Discriminator, size)
		for i := range targets {
			targets[i] = discriminators[i%len(discriminators)]
		}

		matcher := simd.NewDiscriminatorMatcher(discriminators, simd.StrategyAuto)

		start := time.Now()
		for i := 0; i < iterations; i++ {
			_ = matcher.MatchBatch(targets)
		}
		elapsed := time.Since(start)

		avgNs := elapsed.Nanoseconds() / int64(iterations)
		perItemNs := avgNs / int64(size)

		fmt.Printf("Batch: %4d  Total: %6d ns/op  Per-item: %4d ns\n",
			size, avgNs, perItemNs)
	}
}
