package storage

import (
	"context"
)

type AccountRepository interface {
	Save(ctx context.Context, account *AccountModel) error
	SaveBatch(ctx context.Context, accounts []*AccountModel) error
	FindByPubkey(ctx context.Context, pubkey string) (*AccountModel, error)
	FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*AccountModel, error)
	FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*AccountModel, error)
	Delete(ctx context.Context, pubkey string) error
}

type TransactionRepository interface {
	Save(ctx context.Context, tx *TransactionModel) error
	SaveBatch(ctx context.Context, transactions []*TransactionModel) error
	FindBySignature(ctx context.Context, signature string) (*TransactionModel, error)
	FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*TransactionModel, error)
	FindByAccountKey(ctx context.Context, accountKey string, limit int, offset int) ([]*TransactionModel, error)
	FindRecent(ctx context.Context, limit int) ([]*TransactionModel, error)
}

type InstructionRepository interface {
	Save(ctx context.Context, instruction *InstructionModel) error
	SaveBatch(ctx context.Context, instructions []*InstructionModel) error
	FindBySignature(ctx context.Context, signature string) ([]*InstructionModel, error)
	FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*InstructionModel, error)
}

type EventRepository interface {
	Save(ctx context.Context, event *EventModel) error
	SaveBatch(ctx context.Context, events []*EventModel) error
	FindBySignature(ctx context.Context, signature string) ([]*EventModel, error)
	FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*EventModel, error)
	FindByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*EventModel, error)
	FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*EventModel, error)
}

type TokenAccountRepository interface {
	Save(ctx context.Context, tokenAccount *TokenAccountModel) error
	SaveBatch(ctx context.Context, tokenAccounts []*TokenAccountModel) error
	FindByAddress(ctx context.Context, address string) (*TokenAccountModel, error)
	FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*TokenAccountModel, error)
	FindByMint(ctx context.Context, mint string, limit int, offset int) ([]*TokenAccountModel, error)
}

type Repository interface {
	Accounts() AccountRepository
	Transactions() TransactionRepository
	Instructions() InstructionRepository
	Events() EventRepository
	TokenAccounts() TokenAccountRepository
	Close() error
	Ping(ctx context.Context) error
}
