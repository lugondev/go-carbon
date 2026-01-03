// Package types provides base Solana types and structures used throughout the carbon framework.
// It wraps and extends the solana-go library types for consistency and convenience.
package types

import (
	"github.com/gagliardetto/solana-go"
)

// Pubkey is a Solana public key (32 bytes).
type Pubkey = solana.PublicKey

// Signature is a Solana transaction signature (64 bytes).
type Signature = solana.Signature

// Hash is a Solana hash (32 bytes), typically used for blockhashes.
type Hash = solana.Hash

// Account represents a Solana account with its data and metadata.
type Account struct {
	// Lamports is the number of lamports owned by this account.
	Lamports uint64 `json:"lamports"`

	// Data is the data held in this account.
	Data []byte `json:"data"`

	// Owner is the program that owns this account.
	Owner Pubkey `json:"owner"`

	// Executable indicates if the account contains a program.
	Executable bool `json:"executable"`

	// RentEpoch is the epoch at which this account will next owe rent.
	RentEpoch uint64 `json:"rent_epoch"`
}

// AccountMeta describes a single account involved in an instruction.
type AccountMeta struct {
	// Pubkey is the public key of the account.
	Pubkey Pubkey `json:"pubkey"`

	// IsSigner indicates if the account is a signer.
	IsSigner bool `json:"is_signer"`

	// IsWritable indicates if the account is writable.
	IsWritable bool `json:"is_writable"`
}

// ToSolanaAccountMeta converts to solana-go AccountMeta.
func (am *AccountMeta) ToSolanaAccountMeta() *solana.AccountMeta {
	return &solana.AccountMeta{
		PublicKey:  am.Pubkey,
		IsSigner:   am.IsSigner,
		IsWritable: am.IsWritable,
	}
}

// FromSolanaAccountMeta creates AccountMeta from solana-go AccountMeta.
func FromSolanaAccountMeta(meta *solana.AccountMeta) AccountMeta {
	return AccountMeta{
		Pubkey:     meta.PublicKey,
		IsSigner:   meta.IsSigner,
		IsWritable: meta.IsWritable,
	}
}

// Instruction represents a Solana instruction.
type Instruction struct {
	// ProgramID is the program that will process this instruction.
	ProgramID Pubkey `json:"program_id"`

	// Accounts is the list of accounts to pass to the program.
	Accounts []AccountMeta `json:"accounts"`

	// Data is the instruction data.
	Data []byte `json:"data"`
}

// CompiledInstruction represents a compiled instruction within a transaction.
type CompiledInstruction struct {
	// ProgramIDIndex is the index of the program ID in the account keys.
	ProgramIDIndex uint8 `json:"program_id_index"`

	// AccountIndexes is the list of indexes into the account keys.
	AccountIndexes []uint8 `json:"accounts"`

	// Data is the instruction data.
	Data []byte `json:"data"`
}

// InnerInstruction represents an inner instruction executed during a transaction.
type InnerInstruction struct {
	// Instruction is the compiled instruction.
	Instruction CompiledInstruction `json:"instruction"`

	// StackHeight is the call stack depth of this instruction.
	StackHeight *uint32 `json:"stack_height,omitempty"`
}

// InnerInstructions represents a group of inner instructions for a given instruction index.
type InnerInstructions struct {
	// Index is the index of the outer instruction in the transaction.
	Index uint8 `json:"index"`

	// Instructions is the list of inner instructions.
	Instructions []InnerInstruction `json:"instructions"`
}

// TransactionTokenBalance represents token balance information for an account.
type TransactionTokenBalance struct {
	// AccountIndex is the index of the account in the transaction's account keys.
	AccountIndex uint8 `json:"account_index"`

	// Mint is the token mint address.
	Mint string `json:"mint"`

	// Owner is the owner of the token account.
	Owner string `json:"owner"`

	// ProgramID is the token program ID.
	ProgramID string `json:"program_id"`

	// UITokenAmount contains the token amount information.
	UITokenAmount UITokenAmount `json:"ui_token_amount"`
}

// UITokenAmount represents token amount in various formats.
type UITokenAmount struct {
	// Amount is the raw token amount as a string.
	Amount string `json:"amount"`

	// Decimals is the number of decimals for the token.
	Decimals uint8 `json:"decimals"`

	// UIAmount is the token amount as a float (may be nil for large amounts).
	UIAmount *float64 `json:"ui_amount,omitempty"`

	// UIAmountString is the token amount as a formatted string.
	UIAmountString string `json:"ui_amount_string"`
}

// Reward represents a reward applied to an account.
type Reward struct {
	// Pubkey is the public key of the account that received the reward.
	Pubkey string `json:"pubkey"`

	// Lamports is the reward amount in lamports.
	Lamports int64 `json:"lamports"`

	// PostBalance is the account balance after the reward was applied.
	PostBalance uint64 `json:"post_balance"`

	// RewardType is the type of reward.
	RewardType *RewardType `json:"reward_type,omitempty"`

	// Commission is the vote account commission when the reward was credited.
	Commission *uint8 `json:"commission,omitempty"`
}

// RewardType represents the type of reward.
type RewardType string

const (
	RewardTypeFee     RewardType = "Fee"
	RewardTypeRent    RewardType = "Rent"
	RewardTypeStaking RewardType = "Staking"
	RewardTypeVoting  RewardType = "Voting"
)

// LoadedAddresses represents the addresses loaded from address lookup tables.
type LoadedAddresses struct {
	// Writable is the list of writable addresses loaded.
	Writable []Pubkey `json:"writable"`

	// Readonly is the list of readonly addresses loaded.
	Readonly []Pubkey `json:"readonly"`
}

// TransactionReturnData represents the return data from a transaction.
type TransactionReturnData struct {
	// ProgramID is the program that returned the data.
	ProgramID Pubkey `json:"program_id"`

	// Data is the returned data.
	Data []byte `json:"data"`
}

// TransactionStatusMeta contains metadata about a transaction's execution status.
type TransactionStatusMeta struct {
	// Err is the error if the transaction failed, nil if successful.
	Err error `json:"err,omitempty"`

	// Fee is the fee charged for this transaction.
	Fee uint64 `json:"fee"`

	// PreBalances is the list of account balances before the transaction.
	PreBalances []uint64 `json:"pre_balances"`

	// PostBalances is the list of account balances after the transaction.
	PostBalances []uint64 `json:"post_balances"`

	// InnerInstructions is the list of inner instructions executed.
	InnerInstructions []InnerInstructions `json:"inner_instructions,omitempty"`

	// LogMessages is the list of log messages produced during execution.
	LogMessages []string `json:"log_messages,omitempty"`

	// PreTokenBalances is the list of token balances before the transaction.
	PreTokenBalances []TransactionTokenBalance `json:"pre_token_balances,omitempty"`

	// PostTokenBalances is the list of token balances after the transaction.
	PostTokenBalances []TransactionTokenBalance `json:"post_token_balances,omitempty"`

	// Rewards is the list of rewards applied.
	Rewards []Reward `json:"rewards,omitempty"`

	// LoadedAddresses is the addresses loaded from lookup tables.
	LoadedAddresses LoadedAddresses `json:"loaded_addresses"`

	// ReturnData is the data returned by the transaction.
	ReturnData *TransactionReturnData `json:"return_data,omitempty"`

	// ComputeUnitsConsumed is the number of compute units consumed.
	ComputeUnitsConsumed *uint64 `json:"compute_units_consumed,omitempty"`
}

// IsSuccess returns true if the transaction was successful.
func (m *TransactionStatusMeta) IsSuccess() bool {
	return m.Err == nil
}

// LamportsPerSOL is the number of lamports per SOL.
const LamportsPerSOL uint64 = 1_000_000_000

// LamportsToSOL converts lamports to SOL.
func LamportsToSOL(lamports uint64) float64 {
	return float64(lamports) / float64(LamportsPerSOL)
}

// SOLToLamports converts SOL to lamports.
func SOLToLamports(sol float64) uint64 {
	return uint64(sol * float64(LamportsPerSOL))
}
