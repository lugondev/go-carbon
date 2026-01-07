package main

import (
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/pkg/types"
)

var TokenProgramID = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
var Token2022ProgramID = solana.MustPublicKeyFromBase58("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")

type TokenAccountState uint8

const (
	TokenAccountStateUninitialized TokenAccountState = iota
	TokenAccountStateInitialized
	TokenAccountStateFrozen
)

func (s TokenAccountState) String() string {
	switch s {
	case TokenAccountStateUninitialized:
		return "Uninitialized"
	case TokenAccountStateInitialized:
		return "Initialized"
	case TokenAccountStateFrozen:
		return "Frozen"
	default:
		return "Unknown"
	}
}

type TokenAccount struct {
	Mint            types.Pubkey
	Owner           types.Pubkey
	Amount          uint64
	Delegate        *types.Pubkey
	State           TokenAccountState
	IsNative        *uint64
	DelegatedAmount uint64
	CloseAuthority  *types.Pubkey
}

type TokenAccountDecoder struct {
	logger *slog.Logger
}

func NewTokenAccountDecoder(logger *slog.Logger) *TokenAccountDecoder {
	return &TokenAccountDecoder{logger: logger}
}

func (d *TokenAccountDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[TokenAccount] {
	if acc == nil {
		return nil
	}

	if acc.Owner != types.Pubkey(TokenProgramID) && acc.Owner != types.Pubkey(Token2022ProgramID) {
		if d.logger != nil {
			d.logger.Debug("Account not owned by token program",
				"owner", acc.Owner.String(),
			)
		}
		return nil
	}

	if len(acc.Data) < 165 {
		if d.logger != nil {
			d.logger.Debug("Account data too short for token account",
				"size", len(acc.Data),
			)
		}
		return nil
	}

	tokenData, err := decodeTokenAccount(acc.Data)
	if err != nil {
		if d.logger != nil {
			d.logger.Debug("Failed to decode token account",
				"error", err,
			)
		}
		return nil
	}

	return &account.DecodedAccount[TokenAccount]{
		Lamports:   acc.Lamports,
		Owner:      acc.Owner,
		Executable: acc.Executable,
		RentEpoch:  acc.RentEpoch,
		Data:       *tokenData,
	}
}

func decodeTokenAccount(data []byte) (*TokenAccount, error) {
	if len(data) < 165 {
		return nil, fmt.Errorf("data too short: %d bytes", len(data))
	}

	account := &TokenAccount{}

	copy(account.Mint[:], data[0:32])
	copy(account.Owner[:], data[32:64])
	account.Amount = binary.LittleEndian.Uint64(data[64:72])

	if binary.LittleEndian.Uint32(data[72:76]) == 1 {
		delegate := types.Pubkey{}
		copy(delegate[:], data[76:108])
		account.Delegate = &delegate
	}

	account.State = TokenAccountState(data[108])

	if binary.LittleEndian.Uint32(data[109:113]) == 1 {
		isNative := binary.LittleEndian.Uint64(data[113:121])
		account.IsNative = &isNative
	}

	account.DelegatedAmount = binary.LittleEndian.Uint64(data[121:129])

	if binary.LittleEndian.Uint32(data[129:133]) == 1 {
		closeAuth := types.Pubkey{}
		copy(closeAuth[:], data[133:165])
		account.CloseAuthority = &closeAuth
	}

	return account, nil
}

type TokenMint struct {
	MintAuthority   *types.Pubkey
	Supply          uint64
	Decimals        uint8
	IsInitialized   bool
	FreezeAuthority *types.Pubkey
}

type TokenMintDecoder struct{}

func NewTokenMintDecoder() *TokenMintDecoder {
	return &TokenMintDecoder{}
}

func (d *TokenMintDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[TokenMint] {
	if acc == nil {
		return nil
	}

	if acc.Owner != types.Pubkey(TokenProgramID) && acc.Owner != types.Pubkey(Token2022ProgramID) {
		return nil
	}

	if len(acc.Data) < 82 {
		return nil
	}

	mintData, err := decodeTokenMint(acc.Data)
	if err != nil {
		return nil
	}

	return &account.DecodedAccount[TokenMint]{
		Lamports:   acc.Lamports,
		Owner:      acc.Owner,
		Executable: acc.Executable,
		RentEpoch:  acc.RentEpoch,
		Data:       *mintData,
	}
}

func decodeTokenMint(data []byte) (*TokenMint, error) {
	if len(data) < 82 {
		return nil, fmt.Errorf("data too short: %d bytes", len(data))
	}

	mint := &TokenMint{}

	if binary.LittleEndian.Uint32(data[0:4]) == 1 {
		mintAuth := types.Pubkey{}
		copy(mintAuth[:], data[4:36])
		mint.MintAuthority = &mintAuth
	}

	mint.Supply = binary.LittleEndian.Uint64(data[36:44])
	mint.Decimals = data[44]
	mint.IsInitialized = data[45] == 1

	if binary.LittleEndian.Uint32(data[46:50]) == 1 {
		freezeAuth := types.Pubkey{}
		copy(freezeAuth[:], data[50:82])
		mint.FreezeAuthority = &freezeAuth
	}

	return mint, nil
}
