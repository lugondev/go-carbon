package decoder_test

import (
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/decoder"
)

func ExampleBatchDecoder_basic() {
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	registry := decoder.NewRegistry()

	eventName := "SwapExecuted"
	hash := sha256.Sum256([]byte(fmt.Sprintf("event:%s", eventName)))
	disc := decoder.NewAnchorDiscriminator(hash[:8])

	swapDecoder := decoder.NewAnchorDecoder(
		eventName,
		programID,
		disc,
		func(data []byte) (interface{}, error) {
			return map[string]interface{}{
				"user":      "user123",
				"amountIn":  1000000,
				"amountOut": 950000,
			}, nil
		},
	)

	registry.RegisterForProgram(programID, swapDecoder)

	batchDecoder := decoder.NewBatchDecoder(registry)

	testData := make([][]byte, 100)
	for i := range testData {
		data := make([]byte, 64)
		copy(data[:8], disc[:])
		testData[i] = data
	}

	events, err := batchDecoder.DecodeAllFast(testData, &programID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded %d events\n", len(events))
}

func ExampleBatchDecoder_parallel() {
	programID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

	registry := decoder.NewRegistry()
	batchDecoder := decoder.NewBatchDecoder(registry)

	var largeDataset [][]byte

	events, err := batchDecoder.DecodeAllParallel(largeDataset, &programID, 4)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded %d events with 4 workers\n", len(events))
}

func ExampleDiscriminatorMatcher() {
	decoders := []*decoder.AnchorDecoderBase{}

	matcher := decoder.NewDiscriminatorMatcher(decoders)

	var discriminator [8]byte
	copy(discriminator[:], []byte{1, 2, 3, 4, 5, 6, 7, 8})

	foundDecoder, exists := matcher.Match(discriminator)
	if exists {
		fmt.Printf("Found decoder: %s\n", foundDecoder.GetName())
	}
}
