package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type MintScanner struct {
	client     *rpc.Client
	logger     *slog.Logger
	maxResults int
}

func NewMintScanner(endpoint string, logger *slog.Logger, maxResults int) *MintScanner {
	return &MintScanner{
		client:     rpc.New(endpoint),
		logger:     logger,
		maxResults: maxResults,
	}
}

func (s *MintScanner) GetTokenAccountsByMint(
	ctx context.Context,
	mint solana.PublicKey,
) ([]solana.PublicKey, error) {
	s.logger.Info("🔍 Scanning for token accounts by mint...",
		"mint", mint.String()[:8]+"...",
	)

	filters := []rpc.RPCFilter{
		{
			Memcmp: &rpc.RPCFilterMemcmp{
				Offset: 0,
				Bytes:  solana.Base58(mint[:]),
			},
		},
		{
			DataSize: 165,
		},
	}

	opts := &rpc.GetProgramAccountsOpts{
		Encoding:   solana.EncodingBase64,
		Filters:    filters,
		Commitment: rpc.CommitmentConfirmed,
	}

	s.logger.Debug("Calling getProgramAccounts RPC...",
		"program", TokenProgramID.String()[:8]+"...",
	)

	result, err := s.client.GetProgramAccountsWithOpts(
		ctx,
		TokenProgramID,
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get program accounts: %w", err)
	}

	s.logger.Debug("RPC response received",
		"account_count", len(result),
	)

	var accounts []solana.PublicKey
	for _, acc := range result {
		accounts = append(accounts, acc.Pubkey)

		if len(accounts) >= s.maxResults {
			s.logger.Warn("⚠️  Reached max account limit",
				"limit", s.maxResults,
				"mint", mint.String()[:8]+"...",
			)
			break
		}
	}

	s.logger.Info("✅ Found token accounts",
		"count", len(accounts),
		"mint", mint.String()[:8]+"...",
	)

	return accounts, nil
}

func (s *MintScanner) GetTokenAccountsByMints(
	ctx context.Context,
	mints []solana.PublicKey,
) ([]solana.PublicKey, error) {
	var allAccounts []solana.PublicKey

	for _, mint := range mints {
		accounts, err := s.GetTokenAccountsByMint(ctx, mint)
		if err != nil {
			s.logger.Error("Failed to scan mint",
				"mint", mint.String(),
				"error", err,
			)
			continue
		}

		allAccounts = append(allAccounts, accounts...)
	}

	return allAccounts, nil
}

func (s *MintScanner) GetMintDecimals(
	ctx context.Context,
	mint solana.PublicKey,
) (uint8, error) {
	acc, err := s.client.GetAccountInfo(ctx, mint)
	if err != nil {
		return 0, fmt.Errorf("failed to get mint account: %w", err)
	}

	if acc.Value == nil {
		return 0, fmt.Errorf("mint account not found")
	}

	data := acc.Value.Data.GetBinary()

	if len(data) < 45 {
		return 0, fmt.Errorf("invalid mint data length: %d", len(data))
	}

	decimals := data[44]

	s.logger.Debug("Mint decimals retrieved",
		"mint", mint.String()[:8]+"...",
		"decimals", decimals,
	)

	return decimals, nil
}
