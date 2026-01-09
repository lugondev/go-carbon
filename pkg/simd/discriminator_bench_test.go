package simd

import (
	"fmt"
	"testing"
)

func generateTestDiscriminators(count int) []Discriminator {
	discs := make([]Discriminator, count)
	for i := range discs {
		for j := 0; j < 8; j++ {
			discs[i][j] = byte((i*8 + j) % 256)
		}
	}
	return discs
}

func BenchmarkDiscriminatorMatch(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, size := range sizes {
		discs := generateTestDiscriminators(size)
		matcher := NewDiscriminatorMatcher(discs, StrategyAuto)
		target := discs[size/2]

		b.Run(fmt.Sprintf("SingleMatch_Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = matcher.Match(target)
			}
		})
	}
}

func BenchmarkDiscriminatorMatchBatch(b *testing.B) {
	batchSizes := []int{10, 50, 100, 500, 1000}
	candidateSize := 100

	for _, batchSize := range batchSizes {
		discs := generateTestDiscriminators(candidateSize)
		targets := generateTestDiscriminators(batchSize)

		for i := range targets {
			targets[i] = discs[i%candidateSize]
		}

		b.Run(fmt.Sprintf("Map_Batch%d_Candidates%d", batchSize, candidateSize), func(b *testing.B) {
			matcher := NewDiscriminatorMatcher(discs, StrategyMap)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = matcher.MatchBatch(targets)
			}
		})

		b.Run(fmt.Sprintf("SIMD_Batch%d_Candidates%d", batchSize, candidateSize), func(b *testing.B) {
			matcher := NewDiscriminatorMatcher(discs, StrategySIMD)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = matcher.MatchBatch(targets)
			}
		})

		b.Run(fmt.Sprintf("Auto_Batch%d_Candidates%d", batchSize, candidateSize), func(b *testing.B) {
			matcher := NewDiscriminatorMatcher(discs, StrategyAuto)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = matcher.MatchBatch(targets)
			}
		})
	}
}

func BenchmarkCompareDiscriminators(b *testing.B) {
	a := Discriminator{1, 2, 3, 4, 5, 6, 7, 8}
	b1 := Discriminator{1, 2, 3, 4, 5, 6, 7, 8}

	b.Run("Equal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = CompareDiscriminators(a, b1)
		}
	})

	b2 := Discriminator{1, 2, 3, 4, 5, 6, 7, 9}
	b.Run("NotEqual", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = CompareDiscriminators(a, b2)
		}
	})
}

func BenchmarkCompareDiscriminatorsBatch(b *testing.B) {
	sizes := []int{10, 100, 1000}
	target := Discriminator{1, 2, 3, 4, 5, 6, 7, 8}

	for _, size := range sizes {
		candidates := generateTestDiscriminators(size)
		candidates[size/2] = target

		b.Run(fmt.Sprintf("Size%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = CompareDiscriminatorsBatch(target, candidates)
			}
		})
	}
}
