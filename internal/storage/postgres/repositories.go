package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/storage"
)

type postgresAccountRepository struct {
	pool *pgxpool.Pool
}

func (r *postgresAccountRepository) Save(ctx context.Context, account *storage.AccountModel) error {
	query := `
		INSERT INTO accounts (id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (pubkey) DO UPDATE SET
			lamports = $3, data = $4, owner = $5, executable = $6, rent_epoch = $7, slot = $8, updated_at = $9
	`
	_, err := r.pool.Exec(ctx, query,
		account.ID, account.Pubkey, account.Lamports, account.Data, account.Owner,
		account.Executable, account.RentEpoch, account.Slot, account.UpdatedAt, account.CreatedAt,
	)
	return err
}

func (r *postgresAccountRepository) SaveBatch(ctx context.Context, accounts []*storage.AccountModel) error {
	if len(accounts) == 0 {
		return nil
	}

	helper := storage.NewPostgresBatchHelper(r.pool)
	query := `
		INSERT INTO accounts (id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (pubkey) DO UPDATE SET
			lamports = $3, data = $4, owner = $5, executable = $6, rent_epoch = $7, slot = $8, updated_at = $9
	`

	return helper.BatchInsert(ctx, query, len(accounts), func(batch *pgx.Batch, i int) {
		account := accounts[i]
		batch.Queue(query,
			account.ID, account.Pubkey, account.Lamports, account.Data, account.Owner,
			account.Executable, account.RentEpoch, account.Slot, account.UpdatedAt, account.CreatedAt,
		)
	})
}

func (r *postgresAccountRepository) FindByPubkey(ctx context.Context, pubkey string) (*storage.AccountModel, error) {
	query := `SELECT id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at
		FROM accounts WHERE pubkey = $1`

	var account storage.AccountModel
	err := r.pool.QueryRow(ctx, query, pubkey).Scan(
		&account.ID, &account.Pubkey, &account.Lamports, &account.Data, &account.Owner,
		&account.Executable, &account.RentEpoch, &account.Slot, &account.UpdatedAt, &account.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *postgresAccountRepository) FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*storage.AccountModel, error) {
	query := `SELECT id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at
		FROM accounts WHERE owner = $1 ORDER BY slot DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, owner, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*storage.AccountModel
	for rows.Next() {
		var account storage.AccountModel
		if err := rows.Scan(
			&account.ID, &account.Pubkey, &account.Lamports, &account.Data, &account.Owner,
			&account.Executable, &account.RentEpoch, &account.Slot, &account.UpdatedAt, &account.CreatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, &account)
	}

	return accounts, rows.Err()
}

func (r *postgresAccountRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.AccountModel, error) {
	query := `SELECT id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at
		FROM accounts WHERE slot = $1 LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, slot, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*storage.AccountModel
	for rows.Next() {
		var account storage.AccountModel
		if err := rows.Scan(
			&account.ID, &account.Pubkey, &account.Lamports, &account.Data, &account.Owner,
			&account.Executable, &account.RentEpoch, &account.Slot, &account.UpdatedAt, &account.CreatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, &account)
	}

	return accounts, rows.Err()
}

func (r *postgresAccountRepository) Delete(ctx context.Context, pubkey string) error {
	query := `DELETE FROM accounts WHERE pubkey = $1`
	_, err := r.pool.Exec(ctx, query, pubkey)
	return err
}

type postgresTransactionRepository struct {
	pool *pgxpool.Pool
}

func (r *postgresTransactionRepository) Save(ctx context.Context, tx *storage.TransactionModel) error {
	query := `
		INSERT INTO transactions (id, signature, slot, block_time, fee, is_vote, success, error_message,
			account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (signature) DO UPDATE SET
			slot = $3, block_time = $4, fee = $5, is_vote = $6, success = $7, error_message = $8,
			account_keys = $9, num_instructions = $10, num_inner_instructions = $11,
			log_messages = $12, compute_units_consumed = $13
	`
	_, err := r.pool.Exec(ctx, query,
		tx.ID, tx.Signature, tx.Slot, tx.BlockTime, tx.Fee, tx.IsVote, tx.Success, tx.ErrorMessage,
		tx.AccountKeys, tx.NumInstructions, tx.NumInnerInstructions, tx.LogMessages, tx.ComputeUnitsConsumed, tx.CreatedAt,
	)
	return err
}

func (r *postgresTransactionRepository) SaveBatch(ctx context.Context, transactions []*storage.TransactionModel) error {
	if len(transactions) == 0 {
		return nil
	}

	helper := storage.NewPostgresBatchHelper(r.pool)
	query := `
		INSERT INTO transactions (id, signature, slot, block_time, fee, is_vote, success, error_message,
			account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (signature) DO UPDATE SET
			slot = $3, block_time = $4, fee = $5, is_vote = $6, success = $7, error_message = $8,
			account_keys = $9, num_instructions = $10, num_inner_instructions = $11,
			log_messages = $12, compute_units_consumed = $13
	`

	return helper.BatchInsert(ctx, query, len(transactions), func(batch *pgx.Batch, i int) {
		tx := transactions[i]
		batch.Queue(query,
			tx.ID, tx.Signature, tx.Slot, tx.BlockTime, tx.Fee, tx.IsVote, tx.Success, tx.ErrorMessage,
			tx.AccountKeys, tx.NumInstructions, tx.NumInnerInstructions, tx.LogMessages, tx.ComputeUnitsConsumed, tx.CreatedAt,
		)
	})
}

func (r *postgresTransactionRepository) FindBySignature(ctx context.Context, signature string) (*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message,
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions WHERE signature = $1`

	var tx storage.TransactionModel
	err := r.pool.QueryRow(ctx, query, signature).Scan(
		&tx.ID, &tx.Signature, &tx.Slot, &tx.BlockTime, &tx.Fee, &tx.IsVote, &tx.Success, &tx.ErrorMessage,
		&tx.AccountKeys, &tx.NumInstructions, &tx.NumInnerInstructions, &tx.LogMessages, &tx.ComputeUnitsConsumed, &tx.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *postgresTransactionRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message,
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions WHERE slot = $1 LIMIT $2 OFFSET $3`

	return r.queryTransactions(ctx, query, slot, limit, offset)
}

func (r *postgresTransactionRepository) FindByAccountKey(ctx context.Context, accountKey string, limit int, offset int) ([]*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message,
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions WHERE $1 = ANY(account_keys) ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	return r.queryTransactions(ctx, query, accountKey, limit, offset)
}

func (r *postgresTransactionRepository) FindRecent(ctx context.Context, limit int) ([]*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message,
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions ORDER BY created_at DESC LIMIT $1`

	return r.queryTransactions(ctx, query, limit)
}

func (r *postgresTransactionRepository) queryTransactions(ctx context.Context, query string, args ...interface{}) ([]*storage.TransactionModel, error) {
	return QueryMany(r.pool, ctx, query, scanTransaction, args...)
}

func scanTransaction(rows pgx.Rows) (*storage.TransactionModel, error) {
	var tx storage.TransactionModel
	err := rows.Scan(
		&tx.ID, &tx.Signature, &tx.Slot, &tx.BlockTime, &tx.Fee, &tx.IsVote, &tx.Success, &tx.ErrorMessage,
		&tx.AccountKeys, &tx.NumInstructions, &tx.NumInnerInstructions, &tx.LogMessages, &tx.ComputeUnitsConsumed, &tx.CreatedAt,
	)
	return &tx, err
}

type postgresInstructionRepository struct {
	pool *pgxpool.Pool
}

func (r *postgresInstructionRepository) Save(ctx context.Context, instruction *storage.InstructionModel) error {
	query := `
		INSERT INTO instructions (id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		instruction.ID, instruction.Signature, instruction.InstructionIndex, instruction.ProgramID,
		instruction.Data, instruction.Accounts, instruction.IsInner, instruction.InnerIndex, instruction.CreatedAt,
	)
	return err
}

func (r *postgresInstructionRepository) SaveBatch(ctx context.Context, instructions []*storage.InstructionModel) error {
	if len(instructions) == 0 {
		return nil
	}

	helper := storage.NewPostgresBatchHelper(r.pool)
	query := `
		INSERT INTO instructions (id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	return helper.BatchInsert(ctx, query, len(instructions), func(batch *pgx.Batch, i int) {
		inst := instructions[i]
		batch.Queue(query,
			inst.ID, inst.Signature, inst.InstructionIndex, inst.ProgramID,
			inst.Data, inst.Accounts, inst.IsInner, inst.InnerIndex, inst.CreatedAt,
		)
	})
}

func (r *postgresInstructionRepository) FindBySignature(ctx context.Context, signature string) ([]*storage.InstructionModel, error) {
	query := `SELECT id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at
		FROM instructions WHERE signature = $1 ORDER BY instruction_index ASC`

	rows, err := r.pool.Query(ctx, query, signature)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instructions []*storage.InstructionModel
	for rows.Next() {
		var inst storage.InstructionModel
		if err := rows.Scan(
			&inst.ID, &inst.Signature, &inst.InstructionIndex, &inst.ProgramID,
			&inst.Data, &inst.Accounts, &inst.IsInner, &inst.InnerIndex, &inst.CreatedAt,
		); err != nil {
			return nil, err
		}
		instructions = append(instructions, &inst)
	}

	return instructions, rows.Err()
}

func (r *postgresInstructionRepository) FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*storage.InstructionModel, error) {
	query := `SELECT id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at
		FROM instructions WHERE program_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, programID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instructions []*storage.InstructionModel
	for rows.Next() {
		var inst storage.InstructionModel
		if err := rows.Scan(
			&inst.ID, &inst.Signature, &inst.InstructionIndex, &inst.ProgramID,
			&inst.Data, &inst.Accounts, &inst.IsInner, &inst.InnerIndex, &inst.CreatedAt,
		); err != nil {
			return nil, err
		}
		instructions = append(instructions, &inst)
	}

	return instructions, rows.Err()
}

type postgresEventRepository struct {
	pool *pgxpool.Pool
}

func (r *postgresEventRepository) Save(ctx context.Context, event *storage.EventModel) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO events (id, signature, program_id, event_name, data, slot, block_time, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = r.pool.Exec(ctx, query,
		event.ID, event.Signature, event.ProgramID, event.EventName,
		dataJSON, event.Slot, event.BlockTime, event.CreatedAt,
	)
	return err
}

func (r *postgresEventRepository) SaveBatch(ctx context.Context, events []*storage.EventModel) error {
	if len(events) == 0 {
		return nil
	}

	marshaledData := make([][]byte, len(events))
	for i, event := range events {
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal event data at index %d: %w", i, err)
		}
		marshaledData[i] = dataJSON
	}

	helper := storage.NewPostgresBatchHelper(r.pool)
	query := `
		INSERT INTO events (id, signature, program_id, event_name, data, slot, block_time, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	return helper.BatchInsert(ctx, query, len(events), func(batch *pgx.Batch, i int) {
		event := events[i]
		batch.Queue(query,
			event.ID, event.Signature, event.ProgramID, event.EventName,
			marshaledData[i], event.Slot, event.BlockTime, event.CreatedAt,
		)
	})
}

func (r *postgresEventRepository) FindBySignature(ctx context.Context, signature string) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE signature = $1`

	return r.queryEvents(ctx, query, signature)
}

func (r *postgresEventRepository) FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE program_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	return r.queryEvents(ctx, query, programID, limit, offset)
}

func (r *postgresEventRepository) FindByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE event_name = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	return r.queryEvents(ctx, query, eventName, limit, offset)
}

func (r *postgresEventRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE slot = $1 LIMIT $2 OFFSET $3`

	return r.queryEvents(ctx, query, slot, limit, offset)
}

func (r *postgresEventRepository) queryEvents(ctx context.Context, query string, args ...interface{}) ([]*storage.EventModel, error) {
	return QueryMany(r.pool, ctx, query, scanEvent, args...)
}

func scanEvent(rows pgx.Rows) (*storage.EventModel, error) {
	var event storage.EventModel
	var dataJSON []byte

	err := rows.Scan(
		&event.ID, &event.Signature, &event.ProgramID, &event.EventName,
		&dataJSON, &event.Slot, &event.BlockTime, &event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return &event, nil
}

type postgresTokenAccountRepository struct {
	pool *pgxpool.Pool
}

func (r *postgresTokenAccountRepository) Save(ctx context.Context, tokenAccount *storage.TokenAccountModel) error {
	query := `
		INSERT INTO token_accounts (id, address, mint, owner, amount, decimals, delegate, delegated_amount, is_native, close_authority, slot, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (address) DO UPDATE SET
			mint = $3, owner = $4, amount = $5, decimals = $6, delegate = $7,
			delegated_amount = $8, is_native = $9, close_authority = $10, slot = $11, updated_at = $12
	`
	_, err := r.pool.Exec(ctx, query,
		tokenAccount.ID, tokenAccount.Address, tokenAccount.Mint, tokenAccount.Owner,
		tokenAccount.Amount, tokenAccount.Decimals, tokenAccount.Delegate, tokenAccount.DelegatedAmount,
		tokenAccount.IsNative, tokenAccount.CloseAuthority, tokenAccount.Slot,
		tokenAccount.UpdatedAt, tokenAccount.CreatedAt,
	)
	return err
}

func (r *postgresTokenAccountRepository) SaveBatch(ctx context.Context, tokenAccounts []*storage.TokenAccountModel) error {
	if len(tokenAccounts) == 0 {
		return nil
	}

	helper := storage.NewPostgresBatchHelper(r.pool)
	query := `
		INSERT INTO token_accounts (id, address, mint, owner, amount, decimals, delegate, delegated_amount, is_native, close_authority, slot, updated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (address) DO UPDATE SET
			mint = $3, owner = $4, amount = $5, decimals = $6, delegate = $7,
			delegated_amount = $8, is_native = $9, close_authority = $10, slot = $11, updated_at = $12
	`

	return helper.BatchInsert(ctx, query, len(tokenAccounts), func(batch *pgx.Batch, i int) {
		ta := tokenAccounts[i]
		batch.Queue(query,
			ta.ID, ta.Address, ta.Mint, ta.Owner,
			ta.Amount, ta.Decimals, ta.Delegate, ta.DelegatedAmount,
			ta.IsNative, ta.CloseAuthority, ta.Slot,
			ta.UpdatedAt, ta.CreatedAt,
		)
	})
}

func (r *postgresTokenAccountRepository) FindByAddress(ctx context.Context, address string) (*storage.TokenAccountModel, error) {
	query := `SELECT id, address, mint, owner, amount, decimals, delegate, delegated_amount, is_native, close_authority, slot, updated_at, created_at
		FROM token_accounts WHERE address = $1`

	var ta storage.TokenAccountModel
	err := r.pool.QueryRow(ctx, query, address).Scan(
		&ta.ID, &ta.Address, &ta.Mint, &ta.Owner,
		&ta.Amount, &ta.Decimals, &ta.Delegate, &ta.DelegatedAmount,
		&ta.IsNative, &ta.CloseAuthority, &ta.Slot,
		&ta.UpdatedAt, &ta.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ta, nil
}

func (r *postgresTokenAccountRepository) FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*storage.TokenAccountModel, error) {
	query := `SELECT id, address, mint, owner, amount, decimals, delegate, delegated_amount, is_native, close_authority, slot, updated_at, created_at
		FROM token_accounts WHERE owner = $1 LIMIT $2 OFFSET $3`

	return r.queryTokenAccounts(ctx, query, owner, limit, offset)
}

func (r *postgresTokenAccountRepository) FindByMint(ctx context.Context, mint string, limit int, offset int) ([]*storage.TokenAccountModel, error) {
	query := `SELECT id, address, mint, owner, amount, decimals, delegate, delegated_amount, is_native, close_authority, slot, updated_at, created_at
		FROM token_accounts WHERE mint = $1 LIMIT $2 OFFSET $3`

	return r.queryTokenAccounts(ctx, query, mint, limit, offset)
}

func (r *postgresTokenAccountRepository) queryTokenAccounts(ctx context.Context, query string, args ...interface{}) ([]*storage.TokenAccountModel, error) {
	return QueryMany(r.pool, ctx, query, scanTokenAccount, args...)
}

func scanTokenAccount(rows pgx.Rows) (*storage.TokenAccountModel, error) {
	var ta storage.TokenAccountModel
	err := rows.Scan(
		&ta.ID, &ta.Address, &ta.Mint, &ta.Owner,
		&ta.Amount, &ta.Decimals, &ta.Delegate, &ta.DelegatedAmount,
		&ta.IsNative, &ta.CloseAuthority, &ta.Slot,
		&ta.UpdatedAt, &ta.CreatedAt,
	)
	return &ta, err
}

func init() {
	storage.RegisterPostgresFactory(func(ctx context.Context, cfg *config.PostgresConfig) (storage.Repository, error) {
		repo, err := NewPostgresRepository(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create postgres repository: %w", err)
		}
		return repo, nil
	})
}
