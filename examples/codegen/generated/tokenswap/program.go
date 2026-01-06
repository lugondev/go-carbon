package token_swap

import (
	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/decoder/anchor"
	"github.com/lugondev/go-carbon/pkg/decoder"
	"github.com/lugondev/go-carbon/pkg/plugin"
)

const ProgramName = "token_swap"
const ProgramVersion = "0.1.0"

var ProgramID = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

func NewTokenSwapPlugin(programID solana.PublicKey) plugin.Plugin {
	decoders := NewEventDecoders(programID)
	return anchor.NewAnchorEventPlugin(
		ProgramName,
		programID,
		decoders,
	)
}

func GetDecoderRegistry(programID solana.PublicKey) *decoder.Registry {
	registry := decoder.NewRegistry()
	for _, d := range NewEventDecoders(programID) {
		registry.Register(d.GetName(), d)
	}
	return registry
}
