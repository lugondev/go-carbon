package decoder

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/view"
)

func createTestDecoders(count int) ([]*AnchorDecoderBase, [][]byte) {
	decoders := make([]*AnchorDecoderBase, count)
	testData := make([][]byte, count*10)

	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	for i := 0; i < count; i++ {
		eventName := fmt.Sprintf("Event%d", i)
		hash := sha256.Sum256([]byte(fmt.Sprintf("event:%s", eventName)))
		discriminator := NewAnchorDiscriminator(hash[:8])

		decoders[i] = NewAnchorDecoder(
			eventName,
			programID,
			discriminator,
			func(data []byte) (interface{}, error) {
				return map[string]interface{}{"value": "test"}, nil
			},
		)

		for j := 0; j < 10; j++ {
			data := make([]byte, 64)
			copy(data[:8], discriminator[:])
			for k := 8; k < len(data); k++ {
				data[k] = byte(k)
			}
			testData[i*10+j] = data
		}
	}

	return decoders, testData
}

func BenchmarkBatchDecoding(b *testing.B) {
	decoders, testData := createTestDecoders(10)
	registry := NewRegistry()
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	for _, decoder := range decoders {
		registry.RegisterForProgram(programID, decoder)
	}

	batchDecoder := NewBatchDecoder(registry)

	b.Run("Sequential_Traditional", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			events, err := registry.DecodeAll(testData, &programID)
			if err != nil {
				b.Fatal(err)
			}
			if len(events) == 0 {
				b.Fatal("no events decoded")
			}
		}
	})

	b.Run("Sequential_ZeroCopy", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			events, err := batchDecoder.DecodeAllFast(testData, &programID)
			if err != nil {
				b.Fatal(err)
			}
			if len(events) == 0 {
				b.Fatal("no events decoded")
			}
		}
	})

	b.Run("Parallel_2Workers", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			events, err := batchDecoder.DecodeAllParallel(testData, &programID, 2)
			if err != nil {
				b.Fatal(err)
			}
			if len(events) == 0 {
				b.Fatal("no events decoded")
			}
		}
	})

	b.Run("Parallel_4Workers", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			events, err := batchDecoder.DecodeAllParallel(testData, &programID, 4)
			if err != nil {
				b.Fatal(err)
			}
			if len(events) == 0 {
				b.Fatal("no events decoded")
			}
		}
	})

	b.Run("Parallel_8Workers", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			events, err := batchDecoder.DecodeAllParallel(testData, &programID, 8)
			if err != nil {
				b.Fatal(err)
			}
			if len(events) == 0 {
				b.Fatal("no events decoded")
			}
		}
	})
}

func BenchmarkDiscriminatorMatcher(b *testing.B) {
	decoders, testData := createTestDecoders(100)
	matcher := NewDiscriminatorMatcher(decoders)

	discriminators := make([][8]byte, len(testData))
	for i, data := range testData {
		eventView, _ := view.NewEventView(data)
		discriminators[i] = eventView.Discriminator()
	}

	b.Run("SingleMatch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			disc := discriminators[i%len(discriminators)]
			_, _ = matcher.Match(disc)
		}
	})

	b.Run("BatchMatch", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = matcher.MatchBatch(discriminators)
		}
	})

	b.Run("Sequential_Loop", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, disc := range discriminators {
				for _, decoder := range decoders {
					if decoder.discriminator == disc {
						break
					}
				}
			}
		}
	})

	b.Run("Map_Lookup", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, disc := range discriminators {
				_, _ = matcher.Match(disc)
			}
		}
	})
}

func BenchmarkBatchSizes(b *testing.B) {
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		decoders, testData := createTestDecoders(10)

		actualData := make([][]byte, size)
		for i := 0; i < size; i++ {
			actualData[i] = testData[i%len(testData)]
		}

		registry := NewRegistry()
		for _, decoder := range decoders {
			registry.RegisterForProgram(programID, decoder)
		}
		batchDecoder := NewBatchDecoder(registry)

		b.Run(fmt.Sprintf("Size_%d/Traditional", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = registry.DecodeAll(actualData, &programID)
			}
		})

		b.Run(fmt.Sprintf("Size_%d/ZeroCopy", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = batchDecoder.DecodeAllFast(actualData, &programID)
			}
		})

		b.Run(fmt.Sprintf("Size_%d/Parallel", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = batchDecoder.DecodeAllParallel(actualData, &programID, 4)
			}
		})
	}
}
