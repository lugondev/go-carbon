package pipeline

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/types"
)

// compiledToInstruction converts a compiled instruction to a full instruction.
// It handles both solana-go CompiledInstruction and custom types.CompiledInstruction.
func (p *Pipeline) compiledToInstruction(
	compiled interface{},
	accountKeys []types.Pubkey,
) *types.Instruction {
	// Handle solana-go CompiledInstruction
	if compiledIx, ok := compiled.(solana.CompiledInstruction); ok {
		return p.compileFromSolanaInstruction(compiledIx, accountKeys)
	}

	// Handle custom CompiledInstruction type
	if compiledIx, ok := compiled.(types.CompiledInstruction); ok {
		return p.compileFromTypesInstruction(compiledIx, accountKeys)
	}

	p.Logger.Warn("unknown compiled instruction type", "type", fmt.Sprintf("%T", compiled))
	return nil
}

// compileFromSolanaInstruction compiles a solana-go CompiledInstruction.
func (p *Pipeline) compileFromSolanaInstruction(
	compiledIx solana.CompiledInstruction,
	accountKeys []types.Pubkey,
) *types.Instruction {
	// Validate program ID index
	if int(compiledIx.ProgramIDIndex) >= len(accountKeys) {
		p.Logger.Warn("invalid program ID index",
			"index", compiledIx.ProgramIDIndex,
			"account_keys_len", len(accountKeys),
		)
		return nil
	}

	// Resolve program ID
	programID := accountKeys[compiledIx.ProgramIDIndex]

	// Resolve accounts
	accounts := make([]types.AccountMeta, 0, len(compiledIx.Accounts))
	for _, accountIndex := range compiledIx.Accounts {
		if int(accountIndex) >= len(accountKeys) {
			p.Logger.Warn("invalid account index",
				"index", accountIndex,
				"account_keys_len", len(accountKeys),
			)
			continue
		}

		accounts = append(accounts, types.AccountMeta{
			Pubkey:     accountKeys[accountIndex],
			IsSigner:   false, // Will be determined by transaction metadata
			IsWritable: false, // Will be determined by transaction metadata
		})
	}

	return &types.Instruction{
		ProgramID: programID,
		Accounts:  accounts,
		Data:      compiledIx.Data,
	}
}

// compileFromTypesInstruction compiles a custom types.CompiledInstruction.
func (p *Pipeline) compileFromTypesInstruction(
	compiledIx types.CompiledInstruction,
	accountKeys []types.Pubkey,
) *types.Instruction {
	// Validate program ID index
	if int(compiledIx.ProgramIDIndex) >= len(accountKeys) {
		p.Logger.Warn("invalid program ID index",
			"index", compiledIx.ProgramIDIndex,
			"account_keys_len", len(accountKeys),
		)
		return nil
	}

	// Resolve program ID
	programID := accountKeys[compiledIx.ProgramIDIndex]

	// Resolve accounts
	accounts := make([]types.AccountMeta, 0, len(compiledIx.AccountIndexes))
	for _, accountIndex := range compiledIx.AccountIndexes {
		if int(accountIndex) >= len(accountKeys) {
			p.Logger.Warn("invalid account index",
				"index", accountIndex,
				"account_keys_len", len(accountKeys),
			)
			continue
		}

		accounts = append(accounts, types.AccountMeta{
			Pubkey:     accountKeys[accountIndex],
			IsSigner:   false,
			IsWritable: false,
		})
	}

	return &types.Instruction{
		ProgramID: programID,
		Accounts:  accounts,
		Data:      compiledIx.Data,
	}
}

// compiledInnerToInstruction converts a compiled inner instruction to a full instruction.
func (p *Pipeline) compiledInnerToInstruction(
	inner types.InnerInstruction,
	accountKeys []types.Pubkey,
) *types.Instruction {
	compiled := inner.Instruction

	// Validate program ID index
	if int(compiled.ProgramIDIndex) >= len(accountKeys) {
		p.Logger.Warn("invalid inner instruction program ID index",
			"index", compiled.ProgramIDIndex,
			"account_keys_len", len(accountKeys),
		)
		return nil
	}

	// Resolve program ID
	programID := accountKeys[compiled.ProgramIDIndex]

	// Resolve accounts
	accounts := make([]types.AccountMeta, 0, len(compiled.AccountIndexes))
	for _, accountIndex := range compiled.AccountIndexes {
		if int(accountIndex) >= len(accountKeys) {
			p.Logger.Warn("invalid inner instruction account index",
				"index", accountIndex,
				"account_keys_len", len(accountKeys),
			)
			continue
		}

		accounts = append(accounts, types.AccountMeta{
			Pubkey:     accountKeys[accountIndex],
			IsSigner:   false,
			IsWritable: false,
		})
	}

	return &types.Instruction{
		ProgramID: programID,
		Accounts:  accounts,
		Data:      compiled.Data,
	}
}
