package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/storage"
	"github.com/lugondev/go-carbon/pkg/decoder"
)

type DatasourceProcessor struct {
	repo   storage.Repository
	logger *slog.Logger
}

func NewDatasourceProcessor(repo storage.Repository, logger *slog.Logger) *DatasourceProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	return &DatasourceProcessor{
		repo:   repo,
		logger: logger,
	}
}

func (p *DatasourceProcessor) ProcessAccountUpdate(ctx context.Context, update *datasource.AccountUpdate) error {
	model := storage.AccountUpdateToModel(update.Pubkey, update.Account, update.Slot)

	if err := p.repo.Accounts().Save(ctx, model); err != nil {
		p.logger.Error("failed to save account",
			"pubkey", update.Pubkey.String(),
			"slot", update.Slot,
			"error", err,
		)
		return fmt.Errorf("failed to save account: %w", err)
	}

	p.logger.Debug("account saved to database",
		"pubkey", update.Pubkey.String(),
		"slot", update.Slot,
	)

	return nil
}

func (p *DatasourceProcessor) ProcessTransactionUpdate(ctx context.Context, update *datasource.TransactionUpdate) error {
	model := storage.TransactionUpdateToModel(
		update.Signature,
		update.Transaction,
		update.Meta,
		update.IsVote,
		update.Slot,
		update.BlockTime,
	)

	if err := p.repo.Transactions().Save(ctx, model); err != nil {
		p.logger.Error("failed to save transaction",
			"signature", update.Signature.String(),
			"slot", update.Slot,
			"error", err,
		)
		return fmt.Errorf("failed to save transaction: %w", err)
	}

	p.logger.Debug("transaction saved to database",
		"signature", update.Signature.String(),
		"slot", update.Slot,
		"success", update.Meta.IsSuccess(),
	)

	return nil
}

func (p *DatasourceProcessor) ProcessEvent(ctx context.Context, event *decoder.Event, signature, slot string, blockTime *int64) error {
	data, ok := event.Data.(map[string]interface{})
	if !ok {
		data = map[string]interface{}{
			"raw": event.Data,
		}
	}

	slotNum := uint64(0)
	if slot != "" {
		fmt.Sscanf(slot, "%d", &slotNum)
	}

	model := &storage.EventModel{
		ID:        signature + "_" + event.Name,
		Signature: signature,
		ProgramID: event.ProgramID.String(),
		EventName: event.Name,
		Data:      data,
		Slot:      slotNum,
		BlockTime: blockTime,
	}

	if err := p.repo.Events().Save(ctx, model); err != nil {
		p.logger.Error("failed to save event",
			"signature", signature,
			"program_id", event.ProgramID.String(),
			"event_name", event.Name,
			"error", err,
		)
		return fmt.Errorf("failed to save event: %w", err)
	}

	p.logger.Debug("event saved to database",
		"signature", signature,
		"program_id", event.ProgramID.String(),
		"event_name", event.Name,
	)

	return nil
}

type BatchDatasourceProcessor struct {
	repo             storage.Repository
	logger           *slog.Logger
	accountBatch     []*storage.AccountModel
	txBatch          []*storage.TransactionModel
	eventBatch       []*storage.EventModel
	accountBatchSize int
	txBatchSize      int
	eventBatchSize   int
}

func NewBatchDatasourceProcessor(repo storage.Repository, logger *slog.Logger, batchSize int) *BatchDatasourceProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &BatchDatasourceProcessor{
		repo:             repo,
		logger:           logger,
		accountBatch:     make([]*storage.AccountModel, 0, batchSize),
		txBatch:          make([]*storage.TransactionModel, 0, batchSize/2),
		eventBatch:       make([]*storage.EventModel, 0, batchSize*2),
		accountBatchSize: batchSize,
		txBatchSize:      batchSize / 2,
		eventBatchSize:   batchSize * 2,
	}
}

func (p *BatchDatasourceProcessor) ProcessAccountUpdate(ctx context.Context, update *datasource.AccountUpdate) error {
	model := storage.AccountUpdateToModel(update.Pubkey, update.Account, update.Slot)
	p.accountBatch = append(p.accountBatch, model)

	if len(p.accountBatch) >= p.accountBatchSize {
		return p.FlushAccounts(ctx)
	}

	return nil
}

func (p *BatchDatasourceProcessor) ProcessTransactionUpdate(ctx context.Context, update *datasource.TransactionUpdate) error {
	model := storage.TransactionUpdateToModel(
		update.Signature,
		update.Transaction,
		update.Meta,
		update.IsVote,
		update.Slot,
		update.BlockTime,
	)
	p.txBatch = append(p.txBatch, model)

	if len(p.txBatch) >= p.txBatchSize {
		return p.FlushTransactions(ctx)
	}

	return nil
}

func (p *BatchDatasourceProcessor) FlushAccounts(ctx context.Context) error {
	if len(p.accountBatch) == 0 {
		return nil
	}

	if err := p.repo.Accounts().SaveBatch(ctx, p.accountBatch); err != nil {
		p.logger.Error("failed to save account batch",
			"count", len(p.accountBatch),
			"error", err,
		)
		return fmt.Errorf("failed to save account batch: %w", err)
	}

	p.logger.Info("account batch saved to database", "count", len(p.accountBatch))
	p.accountBatch = p.accountBatch[:0]
	return nil
}

func (p *BatchDatasourceProcessor) FlushTransactions(ctx context.Context) error {
	if len(p.txBatch) == 0 {
		return nil
	}

	if err := p.repo.Transactions().SaveBatch(ctx, p.txBatch); err != nil {
		p.logger.Error("failed to save transaction batch",
			"count", len(p.txBatch),
			"error", err,
		)
		return fmt.Errorf("failed to save transaction batch: %w", err)
	}

	p.logger.Info("transaction batch saved to database", "count", len(p.txBatch))
	p.txBatch = p.txBatch[:0]
	return nil
}

func (p *BatchDatasourceProcessor) FlushEvents(ctx context.Context) error {
	if len(p.eventBatch) == 0 {
		return nil
	}

	if err := p.repo.Events().SaveBatch(ctx, p.eventBatch); err != nil {
		p.logger.Error("failed to save event batch",
			"count", len(p.eventBatch),
			"error", err,
		)
		return fmt.Errorf("failed to save event batch: %w", err)
	}

	p.logger.Info("event batch saved to database", "count", len(p.eventBatch))
	p.eventBatch = p.eventBatch[:0]
	return nil
}

func (p *BatchDatasourceProcessor) FlushAll(ctx context.Context) error {
	if err := p.FlushAccounts(ctx); err != nil {
		return err
	}
	if err := p.FlushTransactions(ctx); err != nil {
		return err
	}
	return p.FlushEvents(ctx)
}

func TokenAccountToModel(address, mint, owner solana.PublicKey, amount uint64, decimals uint8, slot uint64) *storage.TokenAccountModel {
	return storage.TokenAccountToModel(address, mint, owner, amount, decimals, nil, 0, false, nil, slot)
}
