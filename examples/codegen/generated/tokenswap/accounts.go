package token_swap

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// Pool account storing swap configuration
var PoolDiscriminator = [8]byte{0xf1, 0x9a, 0x6d, 0x04, 0x11, 0xb1, 0x6d, 0xbc}

type Pool struct {
	Authority   solana.PublicKey `json:"authority" borsh:"authority"`
	TokenMintA  solana.PublicKey `json:"token_mint_a" borsh:"token_mint_a"`
	TokenMintB  solana.PublicKey `json:"token_mint_b" borsh:"token_mint_b"`
	TokenVaultA solana.PublicKey `json:"token_vault_a" borsh:"token_vault_a"`
	TokenVaultB solana.PublicKey `json:"token_vault_b" borsh:"token_vault_b"`
	Fee         uint64           `json:"fee" borsh:"fee"`
	Bump        uint8            `json:"bump" borsh:"bump"`
	TotalSwaps  uint64           `json:"total_swaps" borsh:"total_swaps"`
}

func (a *Pool) Discriminator() [8]byte {
	return PoolDiscriminator
}

func DecodePool(data []byte) (*Pool, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for Pool account")
	}

	var disc [8]byte
	copy(disc[:], data[:8])
	if disc != PoolDiscriminator {
		return nil, fmt.Errorf("invalid discriminator for Pool")
	}

	account := &Pool{}
	// TODO: Implement Borsh deserialization
	_ = data[8:]
	return account, nil
}
