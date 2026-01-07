package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"github.com/lugondev/go-carbon/internal/account"
	"github.com/lugondev/go-carbon/internal/metrics"
)

type TokenMovement struct {
	Account      string
	Mint         string
	Owner        string
	PrevBalance  uint64
	NewBalance   uint64
	Change       int64
	ChangeType   string
	State        TokenAccountState
	Slot         uint64
	IsNewAccount bool
	StateChanged bool
	PrevState    TokenAccountState
}

func (m *TokenMovement) FormatAmount(decimals uint8) string {
	if decimals == 0 {
		return fmt.Sprintf("%d", m.NewBalance)
	}
	divisor := math.Pow10(int(decimals))
	amount := float64(m.NewBalance) / divisor
	return fmt.Sprintf("%.6f", amount)
}

func (m *TokenMovement) FormatChange(decimals uint8) string {
	if decimals == 0 {
		return fmt.Sprintf("%+d", m.Change)
	}
	divisor := math.Pow10(int(decimals))
	change := float64(m.Change) / divisor
	return fmt.Sprintf("%+.6f", change)
}

type TokenTracker struct {
	logger    *slog.Logger
	config    *AlertsConfig
	balances  map[string]uint64
	states    map[string]TokenAccountState
	mintCache map[string]*MintInfo
}

type MintInfo struct {
	Address  string
	Decimals uint8
	Supply   uint64
}

func NewTokenTracker(logger *slog.Logger, config *AlertsConfig) *TokenTracker {
	return &TokenTracker{
		logger:    logger,
		config:    config,
		balances:  make(map[string]uint64),
		states:    make(map[string]TokenAccountState),
		mintCache: make(map[string]*MintInfo),
	}
}

func (t *TokenTracker) Process(
	ctx context.Context,
	input account.AccountProcessorInput[TokenAccount],
	m *metrics.Collection,
) error {
	pubkey := input.Metadata.Pubkey.String()
	tokenData := input.DecodedAccount.Data
	newBalance := tokenData.Amount

	prevBalance, exists := t.balances[pubkey]
	prevState, stateExists := t.states[pubkey]

	t.balances[pubkey] = newBalance
	t.states[pubkey] = tokenData.State

	var change int64
	var changeType string
	isNewAccount := !exists
	stateChanged := stateExists && prevState != tokenData.State

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

	movement := &TokenMovement{
		Account:      pubkey,
		Mint:         tokenData.Mint.String(),
		Owner:        tokenData.Owner.String(),
		PrevBalance:  prevBalance,
		NewBalance:   newBalance,
		Change:       change,
		ChangeType:   changeType,
		State:        tokenData.State,
		Slot:         input.Metadata.Slot,
		IsNewAccount: isNewAccount,
		StateChanged: stateChanged,
		PrevState:    prevState,
	}

	mintInfo := t.getMintInfo(tokenData.Mint.String())
	decimals := uint8(0)
	if mintInfo != nil {
		decimals = mintInfo.Decimals
	}

	t.logMovement(movement, decimals)
	t.checkAlerts(movement, decimals, m)

	_ = m.IncrementCounter(ctx, "token_updates", 1)
	_ = m.UpdateGauge(ctx, "token_balance_"+pubkey[:8], float64(newBalance))

	return nil
}

func (t *TokenTracker) logMovement(movement *TokenMovement, decimals uint8) {
	mintShort := movement.Mint
	if len(mintShort) > 8 {
		mintShort = mintShort[:8] + "..."
	}

	accountShort := movement.Account
	if len(accountShort) > 8 {
		accountShort = accountShort[:8] + "..."
	}

	ownerShort := movement.Owner
	if len(ownerShort) > 8 {
		ownerShort = ownerShort[:8] + "..."
	}

	logAttrs := []any{
		"account", accountShort,
		"mint", mintShort,
		"owner", ownerShort,
		"balance", movement.FormatAmount(decimals),
		"balance_raw", movement.NewBalance,
		"change_type", movement.ChangeType,
		"state", movement.State.String(),
		"slot", movement.Slot,
	}

	if movement.IsNewAccount {
		t.logger.Info("🆕 NEW TOKEN ACCOUNT DETECTED", logAttrs...)
	} else if movement.Change != 0 {
		direction := "➡️"
		if movement.Change > 0 {
			direction = "⬇️ RECEIVED"
		} else {
			direction = "⬆️ SENT"
		}

		logAttrs = append(logAttrs,
			"change", movement.FormatChange(decimals),
			"change_raw", movement.Change,
			"prev_balance", movement.FormatAmount(decimals),
		)

		t.logger.Info(fmt.Sprintf("%s TOKEN MOVEMENT", direction), logAttrs...)
	} else if movement.StateChanged {
		logAttrs = append(logAttrs,
			"prev_state", movement.PrevState.String(),
		)
		t.logger.Info("🔄 TOKEN ACCOUNT STATE CHANGED", logAttrs...)
	} else {
		t.logger.Debug("Token account unchanged", logAttrs...)
	}
}

func (t *TokenTracker) checkAlerts(movement *TokenMovement, decimals uint8, m *metrics.Collection) {
	if !t.config.Enabled {
		return
	}

	ctx := context.Background()

	if t.config.AlertNewAccounts && movement.IsNewAccount {
		t.logger.Warn("🚨 ALERT: New token account detected!",
			"account", movement.Account,
			"mint", movement.Mint,
			"owner", movement.Owner,
			"initial_balance", movement.FormatAmount(decimals),
			"initial_balance_raw", movement.NewBalance,
		)
		_ = m.IncrementCounter(ctx, "alert_new_accounts", 1)
	}

	if t.config.AlertStateChanges && movement.StateChanged {
		t.logger.Warn("🚨 ALERT: Token account state changed!",
			"account", movement.Account,
			"mint", movement.Mint,
			"prev_state", movement.PrevState.String(),
			"new_state", movement.State.String(),
		)
		_ = m.IncrementCounter(ctx, "alert_state_changes", 1)

		if movement.State == TokenAccountStateFrozen {
			t.logger.Error("❄️ CRITICAL: Token account is FROZEN!",
				"account", movement.Account,
				"mint", movement.Mint,
			)
		}
	}

	if movement.IsNewAccount {
		return
	}

	absChange := movement.Change
	if absChange < 0 {
		absChange = -absChange
	}

	if absChange > t.config.Threshold {
		direction := "sent from"
		emoji := "📤"
		if movement.Change > 0 {
			direction = "received to"
			emoji = "📥"
		}

		t.logger.Warn(fmt.Sprintf("🚨 ALERT: Significant token movement %s", direction),
			"emoji", emoji,
			"account", movement.Account,
			"mint", movement.Mint,
			"owner", movement.Owner,
			"amount", movement.FormatChange(decimals),
			"amount_raw", movement.Change,
			"prev_balance", movement.FormatAmount(decimals),
			"new_balance", movement.FormatAmount(decimals),
			"threshold", t.config.Threshold,
		)
		_ = m.IncrementCounter(ctx, "alert_significant_transfers", 1)
	}
}

func (t *TokenTracker) getMintInfo(mint string) *MintInfo {
	return t.mintCache[mint]
}

func (t *TokenTracker) RegisterMintInfo(mint string, decimals uint8, supply uint64) {
	t.mintCache[mint] = &MintInfo{
		Address:  mint,
		Decimals: decimals,
		Supply:   supply,
	}
}

type MintTracker struct {
	logger       *slog.Logger
	supplies     map[string]uint64
	tokenTracker *TokenTracker
}

func NewMintTracker(logger *slog.Logger, tokenTracker *TokenTracker) *MintTracker {
	return &MintTracker{
		logger:       logger,
		supplies:     make(map[string]uint64),
		tokenTracker: tokenTracker,
	}
}

func (t *MintTracker) Process(
	ctx context.Context,
	input account.AccountProcessorInput[TokenMint],
	m *metrics.Collection,
) error {
	pubkey := input.Metadata.Pubkey.String()
	mintData := input.DecodedAccount.Data
	newSupply := mintData.Supply

	if t.tokenTracker != nil {
		t.tokenTracker.RegisterMintInfo(pubkey, mintData.Decimals, newSupply)
	}

	prevSupply, exists := t.supplies[pubkey]
	t.supplies[pubkey] = newSupply

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

	mintShort := pubkey
	if len(mintShort) > 8 {
		mintShort = mintShort[:8] + "..."
	}

	divisor := math.Pow10(int(mintData.Decimals))
	supplyFormatted := float64(newSupply) / divisor

	logAttrs := []any{
		"mint", mintShort,
		"supply", fmt.Sprintf("%.6f", supplyFormatted),
		"supply_raw", newSupply,
		"decimals", mintData.Decimals,
		"change_type", changeType,
		"is_initialized", mintData.IsInitialized,
		"slot", input.Metadata.Slot,
	}

	if change != 0 {
		changeFormatted := float64(change) / divisor
		logAttrs = append(logAttrs,
			"change", fmt.Sprintf("%+.6f", changeFormatted),
			"change_raw", change,
		)
	}

	if changeType == "NEW_MINT" {
		t.logger.Info("🪙 NEW TOKEN MINT DISCOVERED", logAttrs...)
	} else if change != 0 {
		emoji := "🔥"
		if change > 0 {
			emoji = "⚡"
		}
		t.logger.Info(fmt.Sprintf("%s TOKEN SUPPLY CHANGED", emoji), logAttrs...)
	} else {
		t.logger.Debug("Token mint unchanged", logAttrs...)
	}

	_ = m.IncrementCounter(ctx, "mint_updates", 1)

	return nil
}
