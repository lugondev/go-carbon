// Package main demonstrates token tracking functionality using the go-carbon framework.
//
// This example shows how to:
// - Monitor SPL Token accounts for balance changes
// - Decode token account data
// - Track token transfers and movements
// - Send alerts for significant token events
package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/datasource/rpc"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/internal/pipeline"
	"github.com/lugondev/go-carbon/internal/processor"
	"github.com/lugondev/go-carbon/pkg/types"
)

// RPC endpoint
const rpcEndpoint = "https://api.devnet.solana.com"

// SPL Token Program ID
var TokenProgramID = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")

// Token2022 Program ID
var Token2022ProgramID = solana.MustPublicKeyFromBase58("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")

// TokenAccount represents decoded SPL Token account data.
// SPL Token account layout (165 bytes for standard token accounts):
// - mint: 32 bytes
// - owner: 32 bytes
// - amount: 8 bytes (u64)
// - delegate: 36 bytes (option: 4 bytes tag + 32 bytes pubkey)
// - state: 1 byte
// - is_native: 12 bytes (option: 4 bytes tag + 8 bytes u64)
// - delegated_amount: 8 bytes (u64)
// - close_authority: 36 bytes (option: 4 bytes tag + 32 bytes pubkey)
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

// TokenAccountState represents the state of a token account.
type TokenAccountState uint8

const (
	TokenAccountStateUninitialized TokenAccountState = iota
	TokenAccountStateInitialized
	TokenAccountStateFrozen
)

// String returns the string representation of the token account state.
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

// TokenAccountDecoder decodes SPL Token account data.
type TokenAccountDecoder struct{}

// DecodeAccount implements the AccountDecoder interface.
func (d *TokenAccountDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[TokenAccount] {
	if acc == nil {
		return nil
	}

	// Check if this is a token account (owned by Token Program or Token-2022)
	if acc.Owner != types.Pubkey(TokenProgramID) && acc.Owner != types.Pubkey(Token2022ProgramID) {
		return nil
	}

	// Token accounts are 165 bytes
	if len(acc.Data) < 165 {
		return nil
	}

	tokenData, err := decodeTokenAccount(acc.Data)
	if err != nil {
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

// decodeTokenAccount decodes raw bytes into TokenAccount.
func decodeTokenAccount(data []byte) (*TokenAccount, error) {
	if len(data) < 165 {
		return nil, fmt.Errorf("data too short: %d bytes", len(data))
	}

	account := &TokenAccount{}

	// Mint (0-32)
	copy(account.Mint[:], data[0:32])

	// Owner (32-64)
	copy(account.Owner[:], data[32:64])

	// Amount (64-72)
	account.Amount = binary.LittleEndian.Uint64(data[64:72])

	// Delegate (72-108) - COption<Pubkey>
	if binary.LittleEndian.Uint32(data[72:76]) == 1 {
		delegate := types.Pubkey{}
		copy(delegate[:], data[76:108])
		account.Delegate = &delegate
	}

	// State (108)
	account.State = TokenAccountState(data[108])

	// IsNative (109-121) - COption<u64>
	if binary.LittleEndian.Uint32(data[109:113]) == 1 {
		isNative := binary.LittleEndian.Uint64(data[113:121])
		account.IsNative = &isNative
	}

	// DelegatedAmount (121-129)
	account.DelegatedAmount = binary.LittleEndian.Uint64(data[121:129])

	// CloseAuthority (129-165) - COption<Pubkey>
	if binary.LittleEndian.Uint32(data[129:133]) == 1 {
		closeAuth := types.Pubkey{}
		copy(closeAuth[:], data[133:165])
		account.CloseAuthority = &closeAuth
	}

	return account, nil
}

// TokenTracker tracks token accounts and their balances.
type TokenTracker struct {
	logger   *slog.Logger
	balances map[string]uint64 // pubkey -> last known balance
}

// NewTokenTracker creates a new TokenTracker.
func NewTokenTracker(logger *slog.Logger) *TokenTracker {
	return &TokenTracker{
		logger:   logger,
		balances: make(map[string]uint64),
	}
}

// Process implements the Processor interface.
func (t *TokenTracker) Process(
	ctx context.Context,
	input account.AccountProcessorInput[TokenAccount],
	m *metrics.Collection,
) error {
	pubkey := input.Metadata.Pubkey.String()
	tokenData := input.DecodedAccount.Data
	newBalance := tokenData.Amount

	// Get previous balance
	prevBalance, exists := t.balances[pubkey]
	t.balances[pubkey] = newBalance

	// Calculate change
	var change int64
	var changeType string
	if exists {
		change = int64(newBalance) - int64(prevBalance)
		if change > 0 {
			changeType = "RECEIVED"
		} else if change < 0 {
			changeType = "SENT"
		} else {
			changeType = "NO_CHANGE"
		}
	} else {
		changeType = "NEW_ACCOUNT"
	}

	t.logger.Info("Token account update",
		"account", pubkey,
		"mint", tokenData.Mint.String(),
		"owner", tokenData.Owner.String(),
		"balance", newBalance,
		"change", change,
		"change_type", changeType,
		"state", tokenData.State.String(),
		"slot", input.Metadata.Slot,
	)

	// Record metrics
	_ = m.IncrementCounter(ctx, "token_updates", 1)
	_ = m.UpdateGauge(ctx, "token_balance_"+pubkey[:8], float64(newBalance))

	// Alert on significant changes (more than 1 million tokens raw amount)
	if exists && (change > 1_000_000 || change < -1_000_000) {
		t.logger.Warn("ALERT: Significant token movement detected!",
			"account", pubkey,
			"mint", tokenData.Mint.String(),
			"change", change,
		)
		_ = m.IncrementCounter(ctx, "significant_transfers", 1)
	}

	return nil
}

// TokenMint represents decoded SPL Token Mint data.
// SPL Token Mint layout (82 bytes):
// - mint_authority: 36 bytes (option: 4 bytes tag + 32 bytes pubkey)
// - supply: 8 bytes (u64)
// - decimals: 1 byte
// - is_initialized: 1 byte (bool)
// - freeze_authority: 36 bytes (option: 4 bytes tag + 32 bytes pubkey)
type TokenMint struct {
	MintAuthority   *types.Pubkey
	Supply          uint64
	Decimals        uint8
	IsInitialized   bool
	FreezeAuthority *types.Pubkey
}

// TokenMintDecoder decodes SPL Token Mint data.
type TokenMintDecoder struct{}

// DecodeAccount implements the AccountDecoder interface.
func (d *TokenMintDecoder) DecodeAccount(acc *types.Account) *account.DecodedAccount[TokenMint] {
	if acc == nil {
		return nil
	}

	// Check if this is a mint account (owned by Token Program or Token-2022)
	if acc.Owner != types.Pubkey(TokenProgramID) && acc.Owner != types.Pubkey(Token2022ProgramID) {
		return nil
	}

	// Mint accounts are 82 bytes (or larger for Token-2022 with extensions)
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

// decodeTokenMint decodes raw bytes into TokenMint.
func decodeTokenMint(data []byte) (*TokenMint, error) {
	if len(data) < 82 {
		return nil, fmt.Errorf("data too short: %d bytes", len(data))
	}

	mint := &TokenMint{}

	// MintAuthority (0-36) - COption<Pubkey>
	if binary.LittleEndian.Uint32(data[0:4]) == 1 {
		mintAuth := types.Pubkey{}
		copy(mintAuth[:], data[4:36])
		mint.MintAuthority = &mintAuth
	}

	// Supply (36-44)
	mint.Supply = binary.LittleEndian.Uint64(data[36:44])

	// Decimals (44)
	mint.Decimals = data[44]

	// IsInitialized (45)
	mint.IsInitialized = data[45] == 1

	// FreezeAuthority (46-82) - COption<Pubkey>
	if binary.LittleEndian.Uint32(data[46:50]) == 1 {
		freezeAuth := types.Pubkey{}
		copy(freezeAuth[:], data[50:82])
		mint.FreezeAuthority = &freezeAuth
	}

	return mint, nil
}

// MintTracker tracks mint accounts and supply changes.
type MintTracker struct {
	logger   *slog.Logger
	supplies map[string]uint64 // mint address -> last known supply
}

// NewMintTracker creates a new MintTracker.
func NewMintTracker(logger *slog.Logger) *MintTracker {
	return &MintTracker{
		logger:   logger,
		supplies: make(map[string]uint64),
	}
}

// Process implements the Processor interface.
func (t *MintTracker) Process(
	ctx context.Context,
	input account.AccountProcessorInput[TokenMint],
	m *metrics.Collection,
) error {
	pubkey := input.Metadata.Pubkey.String()
	mintData := input.DecodedAccount.Data
	newSupply := mintData.Supply

	// Get previous supply
	prevSupply, exists := t.supplies[pubkey]
	t.supplies[pubkey] = newSupply

	// Calculate change
	var change int64
	var changeType string
	if exists {
		change = int64(newSupply) - int64(prevSupply)
		if change > 0 {
			changeType = "MINTED"
		} else if change < 0 {
			changeType = "BURNED"
		} else {
			changeType = "NO_CHANGE"
		}
	} else {
		changeType = "NEW_MINT"
	}

	t.logger.Info("Token mint update",
		"mint", pubkey,
		"supply", newSupply,
		"change", change,
		"change_type", changeType,
		"decimals", mintData.Decimals,
		"is_initialized", mintData.IsInitialized,
		"slot", input.Metadata.Slot,
	)

	// Record metrics
	_ = m.IncrementCounter(ctx, "mint_updates", 1)

	return nil
}

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting go-carbon token tracker example")

	// Create RPC config
	rpcConfig := rpc.DefaultConfig(rpcEndpoint)
	rpcConfig.PollInterval = 3 * time.Second

	// Example token accounts to monitor (devnet)
	// You can replace these with actual token accounts you want to track
	accountsToMonitor := []solana.PublicKey{
		// Add token account addresses here
		// Example: solana.MustPublicKeyFromBase58("your-token-account-address"),
	}

	// Create the RPC datasource
	rpcDatasource := rpc.NewAccountMonitorDatasource(rpcConfig, accountsToMonitor)
	rpcDatasource.WithLogger(logger)

	// Create token account decoder and tracker
	tokenDecoder := &TokenAccountDecoder{}
	tokenTracker := NewTokenTracker(logger)
	tokenPipe := account.NewAccountPipe(tokenDecoder, tokenTracker)
	tokenPipe.WithLogger(logger)

	// Create mint decoder and tracker
	mintDecoder := &TokenMintDecoder{}
	mintTracker := NewMintTracker(logger)
	mintPipe := account.NewAccountPipe(mintDecoder, mintTracker)
	mintPipe.WithLogger(logger)

	// Create metrics collection
	metricsCollection := metrics.NewCollection(
		metrics.NewLogMetrics(logger),
	)

	// Build the pipeline with multiple account pipes
	p := pipeline.Builder().
		Datasource(datasource.NewNamedDatasourceID("rpc-devnet"), rpcDatasource).
		AccountPipe(tokenPipe).
		AccountPipe(mintPipe).
		Metrics(metricsCollection).
		MetricsFlushInterval(15 * time.Second).
		ChannelBufferSize(500).
		WithGracefulShutdown().
		Logger(logger).
		Build()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run pipeline in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- p.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", "signal", sig)
		cancel()
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			logger.Error("Pipeline error", "error", err)
			os.Exit(1)
		}
	}

	logger.Info("Token tracker stopped")
}

// ExampleFilterByMint demonstrates filtering token accounts by specific mints.
func ExampleFilterByMint() {
	logger := slog.Default()

	// Target mint addresses
	targetMints := map[string]bool{
		"So11111111111111111111111111111111111111112": true, // Wrapped SOL
		// Add more mints as needed
	}

	// Processor that filters by mint
	filteredProcessor := processor.NewConditionalProcessor(
		processor.ProcessorFunc[account.AccountProcessorInput[TokenAccount]](
			func(ctx context.Context, input account.AccountProcessorInput[TokenAccount], m *metrics.Collection) error {
				logger.Info("Tracked token account updated",
					"mint", input.DecodedAccount.Data.Mint.String(),
					"balance", input.DecodedAccount.Data.Amount,
				)
				return nil
			},
		),
		func(input account.AccountProcessorInput[TokenAccount]) bool {
			mintStr := input.DecodedAccount.Data.Mint.String()
			return targetMints[mintStr]
		},
	)

	tokenDecoder := &TokenAccountDecoder{}
	tokenPipe := account.NewAccountPipe(tokenDecoder, filteredProcessor)
	tokenPipe.WithLogger(logger)

	fmt.Println("Created filtered token pipe")
}
