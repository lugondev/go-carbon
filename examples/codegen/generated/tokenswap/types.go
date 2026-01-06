package token_swap

import (
	"github.com/gagliardetto/solana-go"
)

var _ = solana.PublicKey{}

type SwapDirection uint8

const (
	SwapDirectionAtoB SwapDirection = iota
	SwapDirectionBtoA
)

type PoolConfig struct {
	Fee         uint64 `json:"fee" borsh:"fee"`
	MaxSlippage uint16 `json:"max_slippage" borsh:"max_slippage"`
	Paused      bool   `json:"paused" borsh:"paused"`
}
