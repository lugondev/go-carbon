package processor

import (
	"context"
	"fmt"
	"testing"

	"github.com/lugondev/go-carbon/internal/metrics"
)

type testData struct {
	ID   int
	Data []byte
}

func BenchmarkProcessorFunc(b *testing.B) {
	ctx := context.Background()
	m := metrics.NewCollection()

	data := testData{
		ID:   1,
		Data: make([]byte, 1024),
	}

	processor := ProcessorFunc[testData](func(ctx context.Context, d testData, m *metrics.Collection) error {
		return nil
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = processor.Process(ctx, data, m)
	}
}

func BenchmarkChainedProcessor(b *testing.B) {
	ctx := context.Background()
	m := metrics.NewCollection()

	data := testData{
		ID:   1,
		Data: make([]byte, 1024),
	}

	counts := []int{1, 5, 10, 20}

	for _, count := range counts {
		processors := make([]Processor[testData], count)
		for i := 0; i < count; i++ {
			processors[i] = ProcessorFunc[testData](func(ctx context.Context, d testData, m *metrics.Collection) error {
				return nil
			})
		}

		chained := NewChainedProcessor(processors...)

		b.Run(fmt.Sprintf("Processors_%d", count), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = chained.Process(ctx, data, m)
			}
		})
	}
}

func BenchmarkBatchProcessor(b *testing.B) {
	ctx := context.Background()
	m := metrics.NewCollection()

	batchProcessor := ProcessorFunc[[]testData](func(ctx context.Context, batch []testData, m *metrics.Collection) error {
		return nil
	})

	batchSizes := []int{10, 100, 1000}

	for _, batchSize := range batchSizes {
		bp := NewBatchProcessor(batchProcessor, batchSize)
		data := testData{
			ID:   1,
			Data: make([]byte, 1024),
		}

		b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = bp.Process(ctx, data, m)
				if (i+1)%batchSize == 0 {
					_ = bp.FlushBatch(ctx, m)
				}
			}
		})
	}
}

func BenchmarkConditionalProcessor(b *testing.B) {
	ctx := context.Background()
	m := metrics.NewCollection()

	processor := ProcessorFunc[testData](func(ctx context.Context, d testData, m *metrics.Collection) error {
		return nil
	})

	conditionTrue := func(d testData) bool { return true }
	conditionFalse := func(d testData) bool { return false }

	data := testData{
		ID:   1,
		Data: make([]byte, 1024),
	}

	b.Run("AlwaysTrue", func(b *testing.B) {
		cp := NewConditionalProcessor(processor, conditionTrue)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = cp.Process(ctx, data, m)
		}
	})

	b.Run("AlwaysFalse", func(b *testing.B) {
		cp := NewConditionalProcessor(processor, conditionFalse)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = cp.Process(ctx, data, m)
		}
	})
}

func BenchmarkProcessorAllocation(b *testing.B) {
	ctx := context.Background()
	m := metrics.NewCollection()

	b.Run("WithAllocation", func(b *testing.B) {
		processor := ProcessorFunc[testData](func(ctx context.Context, d testData, m *metrics.Collection) error {
			copied := make([]byte, len(d.Data))
			copy(copied, d.Data)
			return nil
		})

		data := testData{
			ID:   1,
			Data: make([]byte, 1024),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = processor.Process(ctx, data, m)
		}
	})

	b.Run("WithoutAllocation", func(b *testing.B) {
		processor := ProcessorFunc[testData](func(ctx context.Context, d testData, m *metrics.Collection) error {
			_ = d.Data
			return nil
		})

		data := testData{
			ID:   1,
			Data: make([]byte, 1024),
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = processor.Process(ctx, data, m)
		}
	})
}
