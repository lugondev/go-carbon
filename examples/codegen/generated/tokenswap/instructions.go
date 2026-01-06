package token_swap

import (
	"github.com/gagliardetto/solana-go"
)

// Initialize a new swap pool
var InitializeDiscriminator = [8]byte{0xaf, 0xaf, 0x6d, 0x1f, 0x0d, 0x98, 0x9b, 0xed}

type InitializeInstruction struct {
	Fee  uint64 `json:"fee" borsh:"fee"`
	Bump uint8  `json:"bump" borsh:"bump"`
}

type InitializeAccounts struct {
	Pool          solana.PublicKey
	Authority     solana.PublicKey
	TokenA        solana.PublicKey
	TokenB        solana.PublicKey
	SystemProgram solana.PublicKey
}

// Execute a token swap
var SwapDiscriminator = [8]byte{0xf8, 0xc6, 0x9e, 0x91, 0xe1, 0x75, 0x87, 0xc8}

type SwapInstruction struct {
	AmountIn     uint64 `json:"amount_in" borsh:"amount_in"`
	MinAmountOut uint64 `json:"min_amount_out" borsh:"min_amount_out"`
}

type SwapAccounts struct {
	Pool         solana.PublicKey
	User         solana.PublicKey
	UserTokenIn  solana.PublicKey
	UserTokenOut solana.PublicKey
	PoolTokenIn  solana.PublicKey
	PoolTokenOut solana.PublicKey
	TokenProgram solana.PublicKey
}
