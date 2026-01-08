package decoder

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/view"
)

// BenchmarkAnchorDiscriminatorCheck benchmarks discriminator matching
func BenchmarkAnchorDiscriminatorCheck(b *testing.B) {
	disc1 := AnchorDiscriminator{175, 175, 109, 31, 13, 152, 155, 237}
	disc2 := AnchorDiscriminator{175, 175, 109, 31, 13, 152, 155, 237}

	b.Run("Equals", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = disc1.Equals(disc2)
		}
	})

	b.Run("DirectCompare", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = disc1 == disc2
		}
	})
}

// BenchmarkNewAnchorDiscriminator benchmarks creating discriminators
func BenchmarkNewAnchorDiscriminator(b *testing.B) {
	data := make([]byte, 64)
	for i := range data {
		data[i] = byte(i)
	}

	b.Run("Copy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewAnchorDiscriminator(data)
		}
	})

	b.Run("ZeroCopy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Simulated zero-copy (unsafe would be faster in practice)
			var disc AnchorDiscriminator
			copy(disc[:], data[:8])
			_ = disc
		}
	})
}

// BenchmarkRegistryDecode benchmarks the registry decode operation
func BenchmarkRegistryDecode(b *testing.B) {
	registry := NewRegistry()
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	// Create test event data with discriminator
	testDiscriminator := computeTestDiscriminator("TestEvent")
	testData := append(testDiscriminator[:], []byte("test event data payload")...)

	// Register a test decoder
	decoder := NewAnchorDecoder(
		"TestEvent",
		programID,
		testDiscriminator,
		func(data []byte) (interface{}, error) {
			return map[string]interface{}{
				"value": string(data),
			}, nil
		},
	)
	registry.Register("test", decoder)

	b.Run("SingleDecoder", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := registry.Decode(testData, &programID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithoutProgramID", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := registry.Decode(testData, nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkRegistryDecodeAll benchmarks batch decoding
func BenchmarkRegistryDecodeAll(b *testing.B) {
	registry := NewRegistry()
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	// Create test decoders for different events
	events := []string{"SwapEvent", "TransferEvent", "MintEvent", "BurnEvent"}
	testDataList := make([][]byte, len(events))

	for i, eventName := range events {
		disc := computeTestDiscriminator(eventName)
		decoder := NewAnchorDecoder(
			eventName,
			programID,
			disc,
			func(data []byte) (interface{}, error) {
				return map[string]interface{}{"data": data}, nil
			},
		)
		registry.Register(eventName, decoder)

		// Create test data
		testDataList[i] = append(disc[:], []byte(fmt.Sprintf("payload_%d", i))...)
	}

	sizes := []int{1, 10, 100, 1000}
	for _, size := range sizes {
		dataList := make([][]byte, size)
		for i := 0; i < size; i++ {
			dataList[i] = testDataList[i%len(testDataList)]
		}

		b.Run(fmt.Sprintf("Count_%d", size), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				events, err := registry.DecodeAll(dataList, &programID)
				if err != nil {
					b.Fatal(err)
				}
				if len(events) != size {
					b.Fatalf("expected %d events, got %d", size, len(events))
				}
			}
		})
	}
}

// BenchmarkDecoderTypes compares different decoder types
func BenchmarkDecoderTypes(b *testing.B) {
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	testData := make([]byte, 64)
	for i := range testData {
		testData[i] = byte(i)
	}

	b.Run("SimpleDecoder", func(b *testing.B) {
		decoder := NewSimpleDecoder(
			"test",
			programID,
			8,
			func(data []byte) (interface{}, error) {
				return data, nil
			},
		)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := decoder.Decode(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AnchorDecoder", func(b *testing.B) {
		disc := computeTestDiscriminator("TestEvent")
		anchorData := append(disc[:], testData...)
		decoder := NewAnchorDecoder(
			"TestEvent",
			programID,
			disc,
			func(data []byte) (interface{}, error) {
				return data, nil
			},
		)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := decoder.Decode(anchorData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CompositeDecoder", func(b *testing.B) {
		decoder1 := NewSimpleDecoder("test1", programID, 8, func(data []byte) (interface{}, error) {
			return data, nil
		})
		decoder2 := NewSimpleDecoder("test2", programID, 8, func(data []byte) (interface{}, error) {
			return data, nil
		})
		composite := NewCompositeDecoder("composite", decoder1, decoder2)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := composite.Decode(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkUtilityFunctions benchmarks utility decode functions
func BenchmarkUtilityFunctions(b *testing.B) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	b.Run("DecodeU64LE", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := DecodeU64LE(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DecodeU32LE", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := DecodeU32LE(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DecodeU16LE", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := DecodeU16LE(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Helper function to compute test discriminators
func computeTestDiscriminator(eventName string) AnchorDiscriminator {
	data := []byte(fmt.Sprintf("event:%s", eventName))
	hash := sha256.Sum256(data)
	var disc AnchorDiscriminator
	copy(disc[:], hash[:8])
	return disc
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("Event_WithCopy", func(b *testing.B) {
		data := make([]byte, 1024)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Current pattern: copy raw data
			rawCopy := make([]byte, len(data))
			copy(rawCopy, data)
			event := &Event{
				Name:          "Test",
				Data:          map[string]interface{}{"key": "value"},
				RawData:       rawCopy,
				ProgramID:     solana.PublicKey{},
				Discriminator: []byte{1, 2, 3, 4, 5, 6, 7, 8},
			}
			_ = event
		}
	})

	b.Run("Event_WithoutCopy", func(b *testing.B) {
		data := make([]byte, 1024)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event := &Event{
				Name:          "Test",
				Data:          map[string]interface{}{"key": "value"},
				RawData:       data,
				ProgramID:     solana.PublicKey{},
				Discriminator: data[:8],
			}
			_ = event
		}
	})
}

func BenchmarkZeroCopyDecoding(b *testing.B) {
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	testEventName := "TestEvent"
	hash := sha256.Sum256([]byte(fmt.Sprintf("event:%s", testEventName)))
	discriminator := NewAnchorDiscriminator(hash[:8])

	decoder := NewAnchorDecoder(
		testEventName,
		programID,
		discriminator,
		func(data []byte) (interface{}, error) {
			return map[string]interface{}{
				"value": "test",
			}, nil
		},
	)

	data := make([]byte, 64)
	copy(data[:8], discriminator[:])
	for i := 8; i < len(data); i++ {
		data[i] = byte(i)
	}

	b.Run("Traditional_Decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event, err := decoder.Decode(data)
			if err != nil {
				b.Fatal(err)
			}
			_ = event
		}
	})

	b.Run("ZeroCopy_FastDecodeWithView", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event, err := decoder.FastDecodeWithView(data)
			if err != nil {
				b.Fatal(err)
			}
			_ = event
		}
	})

	b.Run("ZeroCopy_ReuseView", func(b *testing.B) {
		eventView, err := view.NewEventView(data)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			event, err := decoder.DecodeFromView(eventView)
			if err != nil {
				b.Fatal(err)
			}
			_ = event
		}
	})

	b.Run("ZeroCopy_CanDecodeCheck", func(b *testing.B) {
		eventView, err := view.NewEventView(data)
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			canDecode := decoder.FastCanDecodeWithView(eventView)
			_ = canDecode
		}
	})

	b.Run("Traditional_CanDecodeCheck", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			canDecode := decoder.CanDecode(data)
			_ = canDecode
		}
	})
}
