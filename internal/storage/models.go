package storage

import (
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/pkg/types"
)

type AccountModel struct {
	ID         string    `json:"id" bson:"_id,omitempty" db:"id"`
	Pubkey     string    `json:"pubkey" bson:"pubkey" db:"pubkey"`
	Lamports   uint64    `json:"lamports" bson:"lamports" db:"lamports"`
	Data       []byte    `json:"data" bson:"data" db:"data"`
	Owner      string    `json:"owner" bson:"owner" db:"owner"`
	Executable bool      `json:"executable" bson:"executable" db:"executable"`
	RentEpoch  uint64    `json:"rent_epoch" bson:"rent_epoch" db:"rent_epoch"`
	Slot       uint64    `json:"slot" bson:"slot" db:"slot"`
	UpdatedAt  time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at" db:"created_at"`
}

type TransactionModel struct {
	ID                   string    `json:"id" bson:"_id,omitempty" db:"id"`
	Signature            string    `json:"signature" bson:"signature" db:"signature"`
	Slot                 uint64    `json:"slot" bson:"slot" db:"slot"`
	BlockTime            *int64    `json:"block_time,omitempty" bson:"block_time,omitempty" db:"block_time"`
	Fee                  uint64    `json:"fee" bson:"fee" db:"fee"`
	IsVote               bool      `json:"is_vote" bson:"is_vote" db:"is_vote"`
	Success              bool      `json:"success" bson:"success" db:"success"`
	ErrorMessage         string    `json:"error_message,omitempty" bson:"error_message,omitempty" db:"error_message"`
	AccountKeys          []string  `json:"account_keys" bson:"account_keys" db:"account_keys"`
	NumInstructions      int       `json:"num_instructions" bson:"num_instructions" db:"num_instructions"`
	NumInnerInstructions int       `json:"num_inner_instructions" bson:"num_inner_instructions" db:"num_inner_instructions"`
	LogMessages          []string  `json:"log_messages,omitempty" bson:"log_messages,omitempty" db:"log_messages"`
	ComputeUnitsConsumed *uint64   `json:"compute_units_consumed,omitempty" bson:"compute_units_consumed,omitempty" db:"compute_units_consumed"`
	CreatedAt            time.Time `json:"created_at" bson:"created_at" db:"created_at"`
}

type InstructionModel struct {
	ID               string    `json:"id" bson:"_id,omitempty" db:"id"`
	Signature        string    `json:"signature" bson:"signature" db:"signature"`
	InstructionIndex int       `json:"instruction_index" bson:"instruction_index" db:"instruction_index"`
	ProgramID        string    `json:"program_id" bson:"program_id" db:"program_id"`
	Data             []byte    `json:"data" bson:"data" db:"data"`
	Accounts         []string  `json:"accounts" bson:"accounts" db:"accounts"`
	IsInner          bool      `json:"is_inner" bson:"is_inner" db:"is_inner"`
	InnerIndex       *int      `json:"inner_index,omitempty" bson:"inner_index,omitempty" db:"inner_index"`
	CreatedAt        time.Time `json:"created_at" bson:"created_at" db:"created_at"`
}

type EventModel struct {
	ID        string                 `json:"id" bson:"_id,omitempty" db:"id"`
	Signature string                 `json:"signature" bson:"signature" db:"signature"`
	ProgramID string                 `json:"program_id" bson:"program_id" db:"program_id"`
	EventName string                 `json:"event_name" bson:"event_name" db:"event_name"`
	Data      map[string]interface{} `json:"data" bson:"data" db:"data"`
	Slot      uint64                 `json:"slot" bson:"slot" db:"slot"`
	BlockTime *int64                 `json:"block_time,omitempty" bson:"block_time,omitempty" db:"block_time"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at" db:"created_at"`
}

type TokenAccountModel struct {
	ID              string    `json:"id" bson:"_id,omitempty" db:"id"`
	Address         string    `json:"address" bson:"address" db:"address"`
	Mint            string    `json:"mint" bson:"mint" db:"mint"`
	Owner           string    `json:"owner" bson:"owner" db:"owner"`
	Amount          uint64    `json:"amount" bson:"amount" db:"amount"`
	Decimals        uint8     `json:"decimals" bson:"decimals" db:"decimals"`
	Delegate        *string   `json:"delegate,omitempty" bson:"delegate,omitempty" db:"delegate"`
	DelegatedAmount uint64    `json:"delegated_amount" bson:"delegated_amount" db:"delegated_amount"`
	IsNative        bool      `json:"is_native" bson:"is_native" db:"is_native"`
	CloseAuthority  *string   `json:"close_authority,omitempty" bson:"close_authority,omitempty" db:"close_authority"`
	Slot            uint64    `json:"slot" bson:"slot" db:"slot"`
	UpdatedAt       time.Time `json:"updated_at" bson:"updated_at" db:"updated_at"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at" db:"created_at"`
}

func AccountUpdateToModel(pubkey types.Pubkey, account types.Account, slot uint64) *AccountModel {
	now := time.Now()
	return &AccountModel{
		ID:         pubkey.String(),
		Pubkey:     pubkey.String(),
		Lamports:   account.Lamports,
		Data:       account.Data,
		Owner:      account.Owner.String(),
		Executable: account.Executable,
		RentEpoch:  account.RentEpoch,
		Slot:       slot,
		UpdatedAt:  now,
		CreatedAt:  now,
	}
}

func TransactionUpdateToModel(signature types.Signature, tx *solana.Transaction, meta types.TransactionStatusMeta, isVote bool, slot uint64, blockTime *int64) *TransactionModel {
	accountKeys := make([]string, 0, len(tx.Message.AccountKeys))
	for _, key := range tx.Message.AccountKeys {
		accountKeys = append(accountKeys, key.String())
	}

	errorMsg := ""
	if meta.Err != nil {
		errorMsg = meta.Err.Error()
	}

	numInner := 0
	for _, inner := range meta.InnerInstructions {
		numInner += len(inner.Instructions)
	}

	return &TransactionModel{
		ID:                   signature.String(),
		Signature:            signature.String(),
		Slot:                 slot,
		BlockTime:            blockTime,
		Fee:                  meta.Fee,
		IsVote:               isVote,
		Success:              meta.IsSuccess(),
		ErrorMessage:         errorMsg,
		AccountKeys:          accountKeys,
		NumInstructions:      len(tx.Message.Instructions),
		NumInnerInstructions: numInner,
		LogMessages:          meta.LogMessages,
		ComputeUnitsConsumed: meta.ComputeUnitsConsumed,
		CreatedAt:            time.Now(),
	}
}

func TokenAccountToModel(address, mint, owner solana.PublicKey, amount uint64, decimals uint8, delegate *solana.PublicKey, delegatedAmount uint64, isNative bool, closeAuthority *solana.PublicKey, slot uint64) *TokenAccountModel {
	now := time.Now()
	model := &TokenAccountModel{
		ID:              address.String(),
		Address:         address.String(),
		Mint:            mint.String(),
		Owner:           owner.String(),
		Amount:          amount,
		Decimals:        decimals,
		DelegatedAmount: delegatedAmount,
		IsNative:        isNative,
		Slot:            slot,
		UpdatedAt:       now,
		CreatedAt:       now,
	}

	if delegate != nil {
		delegateStr := delegate.String()
		model.Delegate = &delegateStr
	}

	if closeAuthority != nil {
		closeAuthorityStr := closeAuthority.String()
		model.CloseAuthority = &closeAuthorityStr
	}

	return model
}
