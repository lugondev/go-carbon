// Package rpc provides a RPC-based datasource for the carbon pipeline.
//
// This package implements the Datasource interface using Solana RPC calls
// to fetch account and transaction data. It supports polling-based updates
// for account monitoring and transaction fetching.
package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/metrics"
	"github.com/lugondev/go-carbon/pkg/types"
)

// DefaultPollInterval is the default interval for polling account updates.
const DefaultPollInterval = 1 * time.Second

// DefaultMaxRetries is the default number of retries for RPC calls.
const DefaultMaxRetries = 3

// DefaultRetryDelay is the default delay between retries.
const DefaultRetryDelay = 500 * time.Millisecond

// Config holds the configuration for the RPC datasource.
type Config struct {
	// RPCURL is the URL of the Solana RPC endpoint.
	RPCURL string

	// PollInterval is the interval for polling account updates.
	PollInterval time.Duration

	// MaxRetries is the maximum number of retries for RPC calls.
	MaxRetries int

	// RetryDelay is the delay between retries.
	RetryDelay time.Duration

	// CommitmentLevel is the commitment level for RPC calls.
	CommitmentLevel rpc.CommitmentType
}

// DefaultConfig returns a default configuration.
func DefaultConfig(rpcURL string) *Config {
	return &Config{
		RPCURL:          rpcURL,
		PollInterval:    DefaultPollInterval,
		MaxRetries:      DefaultMaxRetries,
		RetryDelay:      DefaultRetryDelay,
		CommitmentLevel: rpc.CommitmentConfirmed,
	}
}

// AccountMonitorDatasource monitors specific accounts for changes.
type AccountMonitorDatasource struct {
	config   *Config
	client   *rpc.Client
	accounts []solana.PublicKey
	logger   *slog.Logger

	// lastSlots tracks the last known slot for each account.
	lastSlots map[string]uint64
	mu        sync.RWMutex
}

// NewAccountMonitorDatasource creates a new AccountMonitorDatasource.
func NewAccountMonitorDatasource(config *Config, accounts []solana.PublicKey) *AccountMonitorDatasource {
	return &AccountMonitorDatasource{
		config:    config,
		client:    rpc.New(config.RPCURL),
		accounts:  accounts,
		logger:    slog.Default(),
		lastSlots: make(map[string]uint64),
	}
}

// WithLogger sets a custom logger.
func (d *AccountMonitorDatasource) WithLogger(logger *slog.Logger) *AccountMonitorDatasource {
	d.logger = logger
	return d
}

// AddAccount adds an account to monitor.
func (d *AccountMonitorDatasource) AddAccount(account solana.PublicKey) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.accounts = append(d.accounts, account)
}

// Consume starts consuming updates from the RPC endpoint.
func (d *AccountMonitorDatasource) Consume(
	ctx context.Context,
	id datasource.DatasourceID,
	updates chan<- datasource.UpdateWithSource,
	m *metrics.Collection,
) error {
	d.logger.Info("starting RPC account monitor datasource",
		"datasource_id", id.String(),
		"num_accounts", len(d.accounts),
		"poll_interval", d.config.PollInterval,
	)

	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	// Initial fetch
	if err := d.fetchAccounts(ctx, id, updates, m); err != nil {
		d.logger.Error("initial fetch failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("RPC datasource shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := d.fetchAccounts(ctx, id, updates, m); err != nil {
				d.logger.Error("failed to fetch accounts", "error", err)
				_ = m.IncrementCounter(ctx, "rpc_fetch_errors", 1)
			}
		}
	}
}

// fetchAccounts fetches all monitored accounts and sends updates.
func (d *AccountMonitorDatasource) fetchAccounts(
	ctx context.Context,
	id datasource.DatasourceID,
	updates chan<- datasource.UpdateWithSource,
	m *metrics.Collection,
) error {
	d.mu.RLock()
	accounts := make([]solana.PublicKey, len(d.accounts))
	copy(accounts, d.accounts)
	d.mu.RUnlock()

	for _, pubkey := range accounts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		accountInfo, err := d.getAccountInfoWithRetry(ctx, pubkey)
		if err != nil {
			d.logger.Warn("failed to get account info",
				"pubkey", pubkey.String(),
				"error", err,
			)
			continue
		}

		if accountInfo == nil || accountInfo.Value == nil {
			continue
		}

		// Check if the account has been updated
		d.mu.RLock()
		lastSlot := d.lastSlots[pubkey.String()]
		d.mu.RUnlock()

		currentSlot := accountInfo.Context.Slot
		if currentSlot <= lastSlot {
			continue // No update
		}

		// Update the last known slot
		d.mu.Lock()
		d.lastSlots[pubkey.String()] = currentSlot
		d.mu.Unlock()

		// Convert to carbon types
		account := convertAccount(accountInfo.Value)

		update := datasource.UpdateWithSource{
			DatasourceID: id,
			Update: datasource.NewAccountUpdate(&datasource.AccountUpdate{
				Pubkey:  types.Pubkey(pubkey),
				Account: account,
				Slot:    currentSlot,
			}),
		}

		select {
		case updates <- update:
			d.logger.Debug("sent account update",
				"pubkey", pubkey.String(),
				"slot", currentSlot,
			)
			_ = m.IncrementCounter(ctx, "rpc_account_updates", 1)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// getAccountInfoWithRetry gets account info with retry logic.
func (d *AccountMonitorDatasource) getAccountInfoWithRetry(
	ctx context.Context,
	pubkey solana.PublicKey,
) (*rpc.GetAccountInfoResult, error) {
	var lastErr error

	for i := 0; i < d.config.MaxRetries; i++ {
		result, err := d.client.GetAccountInfoWithOpts(ctx, pubkey, &rpc.GetAccountInfoOpts{
			Commitment: d.config.CommitmentLevel,
		})
		if err == nil {
			return result, nil
		}

		lastErr = err
		d.logger.Debug("RPC call failed, retrying",
			"attempt", i+1,
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(d.config.RetryDelay):
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", d.config.MaxRetries, lastErr)
}

// UpdateTypes returns the types of updates this datasource can provide.
func (d *AccountMonitorDatasource) UpdateTypes() []datasource.UpdateType {
	return []datasource.UpdateType{datasource.UpdateTypeAccount}
}

// TransactionFetcherDatasource fetches transactions for specific signatures.
type TransactionFetcherDatasource struct {
	config     *Config
	client     *rpc.Client
	signatures []solana.Signature
	logger     *slog.Logger
	mu         sync.Mutex
}

// NewTransactionFetcherDatasource creates a new TransactionFetcherDatasource.
func NewTransactionFetcherDatasource(config *Config) *TransactionFetcherDatasource {
	return &TransactionFetcherDatasource{
		config:     config,
		client:     rpc.New(config.RPCURL),
		signatures: make([]solana.Signature, 0),
		logger:     slog.Default(),
	}
}

// WithLogger sets a custom logger.
func (d *TransactionFetcherDatasource) WithLogger(logger *slog.Logger) *TransactionFetcherDatasource {
	d.logger = logger
	return d
}

// AddSignature adds a transaction signature to fetch.
func (d *TransactionFetcherDatasource) AddSignature(sig solana.Signature) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.signatures = append(d.signatures, sig)
}

// Consume starts consuming updates from the RPC endpoint.
func (d *TransactionFetcherDatasource) Consume(
	ctx context.Context,
	id datasource.DatasourceID,
	updates chan<- datasource.UpdateWithSource,
	m *metrics.Collection,
) error {
	d.logger.Info("starting RPC transaction fetcher datasource",
		"datasource_id", id.String(),
		"num_signatures", len(d.signatures),
	)

	d.mu.Lock()
	signatures := make([]solana.Signature, len(d.signatures))
	copy(signatures, d.signatures)
	d.mu.Unlock()

	for _, sig := range signatures {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		tx, err := d.getTransactionWithRetry(ctx, sig)
		if err != nil {
			d.logger.Warn("failed to get transaction",
				"signature", sig.String(),
				"error", err,
			)
			continue
		}

		if tx == nil {
			continue
		}

		// Convert to carbon update
		update, err := d.convertTransaction(tx, sig)
		if err != nil {
			d.logger.Warn("failed to convert transaction",
				"signature", sig.String(),
				"error", err,
			)
			continue
		}

		updateWithSource := datasource.UpdateWithSource{
			DatasourceID: id,
			Update:       *update,
		}

		select {
		case updates <- updateWithSource:
			d.logger.Debug("sent transaction update",
				"signature", sig.String(),
			)
			_ = m.IncrementCounter(ctx, "rpc_transaction_updates", 1)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// getTransactionWithRetry gets a transaction with retry logic.
func (d *TransactionFetcherDatasource) getTransactionWithRetry(
	ctx context.Context,
	sig solana.Signature,
) (*rpc.GetTransactionResult, error) {
	var lastErr error

	for i := 0; i < d.config.MaxRetries; i++ {
		maxVersion := uint64(0)
		result, err := d.client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
			Commitment:                     d.config.CommitmentLevel,
			MaxSupportedTransactionVersion: &maxVersion,
		})
		if err == nil {
			return result, nil
		}

		lastErr = err
		d.logger.Debug("RPC call failed, retrying",
			"attempt", i+1,
			"error", err,
		)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(d.config.RetryDelay):
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", d.config.MaxRetries, lastErr)
}

// convertTransaction converts an RPC transaction result to a carbon update.
func (d *TransactionFetcherDatasource) convertTransaction(
	result *rpc.GetTransactionResult,
	sig solana.Signature,
) (*datasource.Update, error) {
	if result.Transaction == nil {
		return nil, fmt.Errorf("transaction is nil")
	}

	tx, err := result.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	meta := convertTransactionMeta(result.Meta)

	var blockTime *int64
	if result.BlockTime != nil {
		bt := int64(*result.BlockTime)
		blockTime = &bt
	}

	update := datasource.NewTransactionUpdate(&datasource.TransactionUpdate{
		Signature:   types.Signature(sig),
		Transaction: tx,
		Meta:        meta,
		Slot:        result.Slot,
		BlockTime:   blockTime,
	})

	return &update, nil
}

// UpdateTypes returns the types of updates this datasource can provide.
func (d *TransactionFetcherDatasource) UpdateTypes() []datasource.UpdateType {
	return []datasource.UpdateType{datasource.UpdateTypeTransaction}
}

// SlotMonitorDatasource monitors for new slots/blocks.
type SlotMonitorDatasource struct {
	config   *Config
	client   *rpc.Client
	logger   *slog.Logger
	lastSlot uint64
	mu       sync.Mutex
}

// NewSlotMonitorDatasource creates a new SlotMonitorDatasource.
func NewSlotMonitorDatasource(config *Config) *SlotMonitorDatasource {
	return &SlotMonitorDatasource{
		config: config,
		client: rpc.New(config.RPCURL),
		logger: slog.Default(),
	}
}

// WithLogger sets a custom logger.
func (d *SlotMonitorDatasource) WithLogger(logger *slog.Logger) *SlotMonitorDatasource {
	d.logger = logger
	return d
}

// Consume starts consuming updates from the RPC endpoint.
func (d *SlotMonitorDatasource) Consume(
	ctx context.Context,
	id datasource.DatasourceID,
	updates chan<- datasource.UpdateWithSource,
	m *metrics.Collection,
) error {
	d.logger.Info("starting RPC slot monitor datasource",
		"datasource_id", id.String(),
		"poll_interval", d.config.PollInterval,
	)

	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("slot monitor datasource shutting down")
			return ctx.Err()
		case <-ticker.C:
			if err := d.checkSlot(ctx, id, updates, m); err != nil {
				d.logger.Error("failed to check slot", "error", err)
			}
		}
	}
}

// checkSlot checks for new slots and sends block details updates.
func (d *SlotMonitorDatasource) checkSlot(
	ctx context.Context,
	id datasource.DatasourceID,
	updates chan<- datasource.UpdateWithSource,
	m *metrics.Collection,
) error {
	slot, err := d.client.GetSlot(ctx, d.config.CommitmentLevel)
	if err != nil {
		return fmt.Errorf("failed to get slot: %w", err)
	}

	d.mu.Lock()
	if slot <= d.lastSlot {
		d.mu.Unlock()
		return nil
	}
	d.lastSlot = slot
	d.mu.Unlock()

	// Get block time
	blockTime, err := d.client.GetBlockTime(ctx, slot)
	if err != nil {
		d.logger.Debug("failed to get block time", "slot", slot, "error", err)
	}

	var blockTimeVal *int64
	if blockTime != nil {
		bt := int64(*blockTime)
		blockTimeVal = &bt
	}

	update := datasource.UpdateWithSource{
		DatasourceID: id,
		Update: datasource.NewBlockDetailsUpdate(&datasource.BlockDetails{
			Slot:        slot,
			BlockTime:   blockTimeVal,
			BlockHeight: &slot, // Simplified - in practice you might want to get actual block height
		}),
	}

	select {
	case updates <- update:
		d.logger.Debug("sent block details update", "slot", slot)
		_ = m.IncrementCounter(ctx, "rpc_block_updates", 1)
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// UpdateTypes returns the types of updates this datasource can provide.
func (d *SlotMonitorDatasource) UpdateTypes() []datasource.UpdateType {
	return []datasource.UpdateType{datasource.UpdateTypeBlockDetails}
}

// Helper functions

// convertAccount converts a solana-go Account to a carbon types.Account.
func convertAccount(acc *rpc.Account) types.Account {
	if acc == nil {
		return types.Account{}
	}

	var rentEpoch uint64
	if acc.RentEpoch != nil {
		rentEpoch = acc.RentEpoch.Uint64()
	}

	return types.Account{
		Lamports:   acc.Lamports,
		Data:       acc.Data.GetBinary(),
		Owner:      types.Pubkey(acc.Owner),
		Executable: acc.Executable,
		RentEpoch:  rentEpoch,
	}
}

// convertTransactionMeta converts RPC transaction meta to carbon types.
func convertTransactionMeta(meta *rpc.TransactionMeta) types.TransactionStatusMeta {
	if meta == nil {
		return types.TransactionStatusMeta{}
	}

	result := types.TransactionStatusMeta{
		Fee:               meta.Fee,
		PreBalances:       meta.PreBalances,
		PostBalances:      meta.PostBalances,
		LogMessages:       meta.LogMessages,
		InnerInstructions: make([]types.InnerInstructions, 0),
	}

	// Convert inner instructions
	if meta.InnerInstructions != nil {
		for _, inner := range meta.InnerInstructions {
			innerIxs := types.InnerInstructions{
				Index:        uint8(inner.Index),
				Instructions: make([]types.InnerInstruction, 0, len(inner.Instructions)),
			}
			for _, ix := range inner.Instructions {
				var stackHeight *uint32
				if ix.StackHeight != 0 {
					sh := uint32(ix.StackHeight)
					stackHeight = &sh
				}

				// Convert accounts from uint16 to uint8
				accountIndexes := make([]uint8, len(ix.Accounts))
				for i, acc := range ix.Accounts {
					accountIndexes[i] = uint8(acc)
				}

				innerIx := types.InnerInstruction{
					Instruction: types.CompiledInstruction{
						ProgramIDIndex: uint8(ix.ProgramIDIndex),
						AccountIndexes: accountIndexes,
						Data:           []byte(ix.Data),
					},
					StackHeight: stackHeight,
				}
				innerIxs.Instructions = append(innerIxs.Instructions, innerIx)
			}
			result.InnerInstructions = append(result.InnerInstructions, innerIxs)
		}
	}

	// Convert token balances
	if meta.PreTokenBalances != nil {
		result.PreTokenBalances = make([]types.TransactionTokenBalance, 0, len(meta.PreTokenBalances))
		for _, tb := range meta.PreTokenBalances {
			result.PreTokenBalances = append(result.PreTokenBalances, convertTokenBalance(tb))
		}
	}

	if meta.PostTokenBalances != nil {
		result.PostTokenBalances = make([]types.TransactionTokenBalance, 0, len(meta.PostTokenBalances))
		for _, tb := range meta.PostTokenBalances {
			result.PostTokenBalances = append(result.PostTokenBalances, convertTokenBalance(tb))
		}
	}

	return result
}

// convertTokenBalance converts an RPC token balance to a carbon types.TransactionTokenBalance.
func convertTokenBalance(tb rpc.TokenBalance) types.TransactionTokenBalance {
	result := types.TransactionTokenBalance{
		AccountIndex:  uint8(tb.AccountIndex),
		Mint:          tb.Mint.String(),
		UITokenAmount: convertUITokenAmount(tb.UiTokenAmount),
	}

	if tb.Owner != nil {
		result.Owner = tb.Owner.String()
	}

	if tb.ProgramId != nil {
		result.ProgramID = tb.ProgramId.String()
	}

	return result
}

// convertUITokenAmount converts an RPC UI token amount to a carbon types.UITokenAmount.
func convertUITokenAmount(amount *rpc.UiTokenAmount) types.UITokenAmount {
	if amount == nil {
		return types.UITokenAmount{}
	}

	return types.UITokenAmount{
		UIAmount:       amount.UiAmount,
		Decimals:       amount.Decimals,
		Amount:         amount.Amount,
		UIAmountString: amount.UiAmountString,
	}
}
