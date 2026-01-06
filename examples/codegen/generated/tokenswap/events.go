package token_swap

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/decoder/anchor"
	"github.com/lugondev/go-carbon/pkg/decoder"
)

var _ = binary.LittleEndian

var SwapExecutedEventDiscriminator = [8]byte{0x40, 0xc6, 0xcd, 0xe8, 0x26, 0x08, 0x71, 0xe2}

type SwapExecutedEvent struct {
	User      solana.PublicKey `json:"user" borsh:"user"`
	TokenIn   solana.PublicKey `json:"token_in" borsh:"token_in"`
	TokenOut  solana.PublicKey `json:"token_out" borsh:"token_out"`
	AmountIn  uint64           `json:"amount_in" borsh:"amount_in"`
	AmountOut uint64           `json:"amount_out" borsh:"amount_out"`
	Timestamp int64            `json:"timestamp" borsh:"timestamp"`
}

func (e *SwapExecutedEvent) Discriminator() [8]byte {
	return SwapExecutedEventDiscriminator
}

func DecodeSwapExecutedEvent(data []byte) (*SwapExecutedEvent, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for SwapExecuted event")
	}

	event := &SwapExecutedEvent{}
	offset := 0
	copy(event.User[:], data[offset:offset+32])
	offset += 32
	copy(event.TokenIn[:], data[offset:offset+32])
	offset += 32
	copy(event.TokenOut[:], data[offset:offset+32])
	offset += 32
	event.AmountIn = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	event.AmountOut = binary.LittleEndian.Uint64(data[offset:])
	offset += 8
	event.Timestamp = int64(binary.LittleEndian.Uint64(data[offset:]))
	offset += 8

	_ = offset
	return event, nil
}

var PoolInitializedEventDiscriminator = [8]byte{0x70, 0x5f, 0x17, 0xbe, 0xea, 0x2c, 0x57, 0x0c}

type PoolInitializedEvent struct {
	Pool       solana.PublicKey `json:"pool" borsh:"pool"`
	Authority  solana.PublicKey `json:"authority" borsh:"authority"`
	TokenMintA solana.PublicKey `json:"token_mint_a" borsh:"token_mint_a"`
	TokenMintB solana.PublicKey `json:"token_mint_b" borsh:"token_mint_b"`
	Fee        uint64           `json:"fee" borsh:"fee"`
}

func (e *PoolInitializedEvent) Discriminator() [8]byte {
	return PoolInitializedEventDiscriminator
}

func DecodePoolInitializedEvent(data []byte) (*PoolInitializedEvent, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for PoolInitialized event")
	}

	event := &PoolInitializedEvent{}
	offset := 0
	copy(event.Pool[:], data[offset:offset+32])
	offset += 32
	copy(event.Authority[:], data[offset:offset+32])
	offset += 32
	copy(event.TokenMintA[:], data[offset:offset+32])
	offset += 32
	copy(event.TokenMintB[:], data[offset:offset+32])
	offset += 32
	event.Fee = binary.LittleEndian.Uint64(data[offset:])
	offset += 8

	_ = offset
	return event, nil
}

func NewEventDecoders(programID solana.PublicKey) []decoder.Decoder {
	return []decoder.Decoder{
		NewSwapExecutedDecoder(programID),
		NewPoolInitializedDecoder(programID),
	}
}

func NewSwapExecutedDecoder(programID solana.PublicKey) decoder.Decoder {
	return anchor.NewAnchorEventDecoder(
		"SwapExecuted",
		programID,
		decoder.NewAnchorDiscriminator(SwapExecutedEventDiscriminator[:]),
		func(data []byte) (interface{}, error) {
			return DecodeSwapExecutedEvent(data)
		},
	)
}

func NewPoolInitializedDecoder(programID solana.PublicKey) decoder.Decoder {
	return anchor.NewAnchorEventDecoder(
		"PoolInitialized",
		programID,
		decoder.NewAnchorDiscriminator(PoolInitializedEventDiscriminator[:]),
		func(data []byte) (interface{}, error) {
			return DecodePoolInitializedEvent(data)
		},
	)
}
