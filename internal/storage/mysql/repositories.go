package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/lugondev/go-carbon/internal/storage"
)

type mysqlAccountRepository struct {
	db *sql.DB
}

func (r *mysqlAccountRepository) Save(ctx context.Context, account *storage.AccountModel) error {
	query := `
		INSERT INTO accounts (id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			lamports = VALUES(lamports),
			data = VALUES(data),
			owner = VALUES(owner),
			executable = VALUES(executable),
			rent_epoch = VALUES(rent_epoch),
			slot = VALUES(slot),
			updated_at = VALUES(updated_at)
	`
	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.Pubkey, account.Lamports, account.Data, account.Owner,
		account.Executable, account.RentEpoch, account.Slot, account.UpdatedAt, account.CreatedAt,
	)
	return err
}

func (r *mysqlAccountRepository) SaveBatch(ctx context.Context, accounts []*storage.AccountModel) error {
	if len(accounts) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO accounts (id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			lamports = VALUES(lamports),
			data = VALUES(data),
			owner = VALUES(owner),
			executable = VALUES(executable),
			rent_epoch = VALUES(rent_epoch),
			slot = VALUES(slot),
			updated_at = VALUES(updated_at)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, account := range accounts {
		if _, err := stmt.ExecContext(ctx,
			account.ID, account.Pubkey, account.Lamports, account.Data, account.Owner,
			account.Executable, account.RentEpoch, account.Slot, account.UpdatedAt, account.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlAccountRepository) FindByPubkey(ctx context.Context, pubkey string) (*storage.AccountModel, error) {
	query := `SELECT id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at
		FROM accounts WHERE pubkey = ?`

	var account storage.AccountModel
	err := r.db.QueryRowContext(ctx, query, pubkey).Scan(
		&account.ID, &account.Pubkey, &account.Lamports, &account.Data, &account.Owner,
		&account.Executable, &account.RentEpoch, &account.Slot, &account.UpdatedAt, &account.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *mysqlAccountRepository) FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*storage.AccountModel, error) {
	query := `SELECT id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at
		FROM accounts WHERE owner = ? ORDER BY slot DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, owner, limit, offset)
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

func (r *mysqlAccountRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.AccountModel, error) {
	query := `SELECT id, pubkey, lamports, data, owner, executable, rent_epoch, slot, updated_at, created_at
		FROM accounts WHERE slot = ? LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, slot, limit, offset)
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

func (r *mysqlAccountRepository) Delete(ctx context.Context, pubkey string) error {
	query := `DELETE FROM accounts WHERE pubkey = ?`
	_, err := r.db.ExecContext(ctx, query, pubkey)
	return err
}

type mysqlTransactionRepository struct {
	db *sql.DB
}

func (r *mysqlTransactionRepository) Save(ctx context.Context, tx *storage.TransactionModel) error {
	accountKeysJSON, err := json.Marshal(tx.AccountKeys)
	if err != nil {
		return err
	}

	var logMessagesJSON []byte
	if tx.LogMessages != nil {
		logMessagesJSON, err = json.Marshal(tx.LogMessages)
		if err != nil {
			return err
		}
	}

	query := `
		INSERT INTO transactions (id, signature, slot, block_time, fee, is_vote, success, error_message, 
			account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			block_time = VALUES(block_time),
			fee = VALUES(fee),
			is_vote = VALUES(is_vote),
			success = VALUES(success),
			error_message = VALUES(error_message),
			account_keys = VALUES(account_keys),
			num_instructions = VALUES(num_instructions),
			num_inner_instructions = VALUES(num_inner_instructions),
			log_messages = VALUES(log_messages),
			compute_units_consumed = VALUES(compute_units_consumed)
	`
	_, err = r.db.ExecContext(ctx, query,
		tx.ID, tx.Signature, tx.Slot, tx.BlockTime, tx.Fee, tx.IsVote, tx.Success, tx.ErrorMessage,
		accountKeysJSON, tx.NumInstructions, tx.NumInnerInstructions, logMessagesJSON, tx.ComputeUnitsConsumed, tx.CreatedAt,
	)
	return err
}

func (r *mysqlTransactionRepository) SaveBatch(ctx context.Context, transactions []*storage.TransactionModel) error {
	if len(transactions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO transactions (id, signature, slot, block_time, fee, is_vote, success, error_message, 
			account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			block_time = VALUES(block_time),
			fee = VALUES(fee),
			is_vote = VALUES(is_vote),
			success = VALUES(success),
			error_message = VALUES(error_message),
			account_keys = VALUES(account_keys),
			num_instructions = VALUES(num_instructions),
			num_inner_instructions = VALUES(num_inner_instructions),
			log_messages = VALUES(log_messages),
			compute_units_consumed = VALUES(compute_units_consumed)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, transaction := range transactions {
		accountKeysJSON, err := json.Marshal(transaction.AccountKeys)
		if err != nil {
			return err
		}

		var logMessagesJSON []byte
		if transaction.LogMessages != nil {
			logMessagesJSON, err = json.Marshal(transaction.LogMessages)
			if err != nil {
				return err
			}
		}

		if _, err := stmt.ExecContext(ctx,
			transaction.ID, transaction.Signature, transaction.Slot, transaction.BlockTime, transaction.Fee,
			transaction.IsVote, transaction.Success, transaction.ErrorMessage, accountKeysJSON,
			transaction.NumInstructions, transaction.NumInnerInstructions, logMessagesJSON,
			transaction.ComputeUnitsConsumed, transaction.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlTransactionRepository) FindBySignature(ctx context.Context, signature string) (*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message, 
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions WHERE signature = ?`

	var tx storage.TransactionModel
	var accountKeysJSON, logMessagesJSON []byte
	err := r.db.QueryRowContext(ctx, query, signature).Scan(
		&tx.ID, &tx.Signature, &tx.Slot, &tx.BlockTime, &tx.Fee, &tx.IsVote, &tx.Success, &tx.ErrorMessage,
		&accountKeysJSON, &tx.NumInstructions, &tx.NumInnerInstructions, &logMessagesJSON,
		&tx.ComputeUnitsConsumed, &tx.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(accountKeysJSON, &tx.AccountKeys); err != nil {
		return nil, err
	}

	if logMessagesJSON != nil {
		if err := json.Unmarshal(logMessagesJSON, &tx.LogMessages); err != nil {
			return nil, err
		}
	}

	return &tx, nil
}

func (r *mysqlTransactionRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message, 
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions WHERE slot = ? ORDER BY slot DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, slot, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *mysqlTransactionRepository) FindByAccountKey(ctx context.Context, accountKey string, limit int, offset int) ([]*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message, 
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions WHERE JSON_CONTAINS(account_keys, ?) ORDER BY slot DESC LIMIT ? OFFSET ?`

	accountKeyJSON, err := json.Marshal(accountKey)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, query, string(accountKeyJSON), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *mysqlTransactionRepository) FindRecent(ctx context.Context, limit int) ([]*storage.TransactionModel, error) {
	query := `SELECT id, signature, slot, block_time, fee, is_vote, success, error_message, 
		account_keys, num_instructions, num_inner_instructions, log_messages, compute_units_consumed, created_at
		FROM transactions ORDER BY slot DESC LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

func (r *mysqlTransactionRepository) scanTransactions(rows *sql.Rows) ([]*storage.TransactionModel, error) {
	var transactions []*storage.TransactionModel
	for rows.Next() {
		var tx storage.TransactionModel
		var accountKeysJSON, logMessagesJSON []byte
		if err := rows.Scan(
			&tx.ID, &tx.Signature, &tx.Slot, &tx.BlockTime, &tx.Fee, &tx.IsVote, &tx.Success, &tx.ErrorMessage,
			&accountKeysJSON, &tx.NumInstructions, &tx.NumInnerInstructions, &logMessagesJSON,
			&tx.ComputeUnitsConsumed, &tx.CreatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(accountKeysJSON, &tx.AccountKeys); err != nil {
			return nil, err
		}

		if logMessagesJSON != nil {
			if err := json.Unmarshal(logMessagesJSON, &tx.LogMessages); err != nil {
				return nil, err
			}
		}

		transactions = append(transactions, &tx)
	}

	return transactions, rows.Err()
}

type mysqlInstructionRepository struct {
	db *sql.DB
}

func (r *mysqlInstructionRepository) Save(ctx context.Context, instruction *storage.InstructionModel) error {
	accountsJSON, err := json.Marshal(instruction.Accounts)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO instructions (id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			program_id = VALUES(program_id),
			data = VALUES(data),
			accounts = VALUES(accounts),
			is_inner = VALUES(is_inner),
			inner_index = VALUES(inner_index)
	`
	_, err = r.db.ExecContext(ctx, query,
		instruction.ID, instruction.Signature, instruction.InstructionIndex, instruction.ProgramID,
		instruction.Data, accountsJSON, instruction.IsInner, instruction.InnerIndex, instruction.CreatedAt,
	)
	return err
}

func (r *mysqlInstructionRepository) SaveBatch(ctx context.Context, instructions []*storage.InstructionModel) error {
	if len(instructions) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO instructions (id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			program_id = VALUES(program_id),
			data = VALUES(data),
			accounts = VALUES(accounts),
			is_inner = VALUES(is_inner),
			inner_index = VALUES(inner_index)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, instruction := range instructions {
		accountsJSON, err := json.Marshal(instruction.Accounts)
		if err != nil {
			return err
		}

		if _, err := stmt.ExecContext(ctx,
			instruction.ID, instruction.Signature, instruction.InstructionIndex, instruction.ProgramID,
			instruction.Data, accountsJSON, instruction.IsInner, instruction.InnerIndex, instruction.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlInstructionRepository) FindBySignature(ctx context.Context, signature string) ([]*storage.InstructionModel, error) {
	query := `SELECT id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at
		FROM instructions WHERE signature = ? ORDER BY instruction_index`

	rows, err := r.db.QueryContext(ctx, query, signature)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanInstructions(rows)
}

func (r *mysqlInstructionRepository) FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*storage.InstructionModel, error) {
	query := `SELECT id, signature, instruction_index, program_id, data, accounts, is_inner, inner_index, created_at
		FROM instructions WHERE program_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, programID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanInstructions(rows)
}

func (r *mysqlInstructionRepository) scanInstructions(rows *sql.Rows) ([]*storage.InstructionModel, error) {
	var instructions []*storage.InstructionModel
	for rows.Next() {
		var instruction storage.InstructionModel
		var accountsJSON []byte
		if err := rows.Scan(
			&instruction.ID, &instruction.Signature, &instruction.InstructionIndex, &instruction.ProgramID,
			&instruction.Data, &accountsJSON, &instruction.IsInner, &instruction.InnerIndex, &instruction.CreatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(accountsJSON, &instruction.Accounts); err != nil {
			return nil, err
		}

		instructions = append(instructions, &instruction)
	}

	return instructions, rows.Err()
}

type mysqlEventRepository struct {
	db *sql.DB
}

func (r *mysqlEventRepository) Save(ctx context.Context, event *storage.EventModel) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO events (id, signature, program_id, event_name, data, slot, block_time, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			program_id = VALUES(program_id),
			event_name = VALUES(event_name),
			data = VALUES(data),
			slot = VALUES(slot),
			block_time = VALUES(block_time)
	`
	_, err = r.db.ExecContext(ctx, query,
		event.ID, event.Signature, event.ProgramID, event.EventName, dataJSON, event.Slot, event.BlockTime, event.CreatedAt,
	)
	return err
}

func (r *mysqlEventRepository) SaveBatch(ctx context.Context, events []*storage.EventModel) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO events (id, signature, program_id, event_name, data, slot, block_time, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			program_id = VALUES(program_id),
			event_name = VALUES(event_name),
			data = VALUES(data),
			slot = VALUES(slot),
			block_time = VALUES(block_time)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, event := range events {
		dataJSON, err := json.Marshal(event.Data)
		if err != nil {
			return err
		}

		if _, err := stmt.ExecContext(ctx,
			event.ID, event.Signature, event.ProgramID, event.EventName, dataJSON, event.Slot, event.BlockTime, event.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlEventRepository) FindBySignature(ctx context.Context, signature string) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE signature = ?`

	rows, err := r.db.QueryContext(ctx, query, signature)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *mysqlEventRepository) FindByProgramID(ctx context.Context, programID string, limit int, offset int) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE program_id = ? ORDER BY slot DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, programID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *mysqlEventRepository) FindByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE event_name = ? ORDER BY slot DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, eventName, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *mysqlEventRepository) FindBySlot(ctx context.Context, slot uint64, limit int, offset int) ([]*storage.EventModel, error) {
	query := `SELECT id, signature, program_id, event_name, data, slot, block_time, created_at
		FROM events WHERE slot = ? LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, slot, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanEvents(rows)
}

func (r *mysqlEventRepository) scanEvents(rows *sql.Rows) ([]*storage.EventModel, error) {
	var events []*storage.EventModel
	for rows.Next() {
		var event storage.EventModel
		var dataJSON []byte
		if err := rows.Scan(
			&event.ID, &event.Signature, &event.ProgramID, &event.EventName, &dataJSON, &event.Slot, &event.BlockTime, &event.CreatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(dataJSON, &event.Data); err != nil {
			return nil, err
		}

		events = append(events, &event)
	}

	return events, rows.Err()
}

type mysqlTokenAccountRepository struct {
	db *sql.DB
}

func (r *mysqlTokenAccountRepository) Save(ctx context.Context, tokenAccount *storage.TokenAccountModel) error {
	query := `
		INSERT INTO token_accounts (id, address, mint, owner, amount, decimals, delegate, delegated_amount, 
			is_native, close_authority, slot, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			mint = VALUES(mint),
			owner = VALUES(owner),
			amount = VALUES(amount),
			decimals = VALUES(decimals),
			delegate = VALUES(delegate),
			delegated_amount = VALUES(delegated_amount),
			is_native = VALUES(is_native),
			close_authority = VALUES(close_authority),
			slot = VALUES(slot),
			updated_at = VALUES(updated_at)
	`
	_, err := r.db.ExecContext(ctx, query,
		tokenAccount.ID, tokenAccount.Address, tokenAccount.Mint, tokenAccount.Owner, tokenAccount.Amount,
		tokenAccount.Decimals, tokenAccount.Delegate, tokenAccount.DelegatedAmount, tokenAccount.IsNative,
		tokenAccount.CloseAuthority, tokenAccount.Slot, tokenAccount.UpdatedAt, tokenAccount.CreatedAt,
	)
	return err
}

func (r *mysqlTokenAccountRepository) SaveBatch(ctx context.Context, tokenAccounts []*storage.TokenAccountModel) error {
	if len(tokenAccounts) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO token_accounts (id, address, mint, owner, amount, decimals, delegate, delegated_amount, 
			is_native, close_authority, slot, updated_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			mint = VALUES(mint),
			owner = VALUES(owner),
			amount = VALUES(amount),
			decimals = VALUES(decimals),
			delegate = VALUES(delegate),
			delegated_amount = VALUES(delegated_amount),
			is_native = VALUES(is_native),
			close_authority = VALUES(close_authority),
			slot = VALUES(slot),
			updated_at = VALUES(updated_at)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tokenAccount := range tokenAccounts {
		if _, err := stmt.ExecContext(ctx,
			tokenAccount.ID, tokenAccount.Address, tokenAccount.Mint, tokenAccount.Owner, tokenAccount.Amount,
			tokenAccount.Decimals, tokenAccount.Delegate, tokenAccount.DelegatedAmount, tokenAccount.IsNative,
			tokenAccount.CloseAuthority, tokenAccount.Slot, tokenAccount.UpdatedAt, tokenAccount.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *mysqlTokenAccountRepository) FindByAddress(ctx context.Context, address string) (*storage.TokenAccountModel, error) {
	query := `SELECT id, address, mint, owner, amount, decimals, delegate, delegated_amount, 
		is_native, close_authority, slot, updated_at, created_at
		FROM token_accounts WHERE address = ?`

	var tokenAccount storage.TokenAccountModel
	err := r.db.QueryRowContext(ctx, query, address).Scan(
		&tokenAccount.ID, &tokenAccount.Address, &tokenAccount.Mint, &tokenAccount.Owner, &tokenAccount.Amount,
		&tokenAccount.Decimals, &tokenAccount.Delegate, &tokenAccount.DelegatedAmount, &tokenAccount.IsNative,
		&tokenAccount.CloseAuthority, &tokenAccount.Slot, &tokenAccount.UpdatedAt, &tokenAccount.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &tokenAccount, nil
}

func (r *mysqlTokenAccountRepository) FindByOwner(ctx context.Context, owner string, limit int, offset int) ([]*storage.TokenAccountModel, error) {
	query := `SELECT id, address, mint, owner, amount, decimals, delegate, delegated_amount, 
		is_native, close_authority, slot, updated_at, created_at
		FROM token_accounts WHERE owner = ? ORDER BY slot DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, owner, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTokenAccounts(rows)
}

func (r *mysqlTokenAccountRepository) FindByMint(ctx context.Context, mint string, limit int, offset int) ([]*storage.TokenAccountModel, error) {
	query := `SELECT id, address, mint, owner, amount, decimals, delegate, delegated_amount, 
		is_native, close_authority, slot, updated_at, created_at
		FROM token_accounts WHERE mint = ? ORDER BY slot DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, mint, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTokenAccounts(rows)
}

func (r *mysqlTokenAccountRepository) scanTokenAccounts(rows *sql.Rows) ([]*storage.TokenAccountModel, error) {
	var tokenAccounts []*storage.TokenAccountModel
	for rows.Next() {
		var tokenAccount storage.TokenAccountModel
		if err := rows.Scan(
			&tokenAccount.ID, &tokenAccount.Address, &tokenAccount.Mint, &tokenAccount.Owner, &tokenAccount.Amount,
			&tokenAccount.Decimals, &tokenAccount.Delegate, &tokenAccount.DelegatedAmount, &tokenAccount.IsNative,
			&tokenAccount.CloseAuthority, &tokenAccount.Slot, &tokenAccount.UpdatedAt, &tokenAccount.CreatedAt,
		); err != nil {
			return nil, err
		}
		tokenAccounts = append(tokenAccounts, &tokenAccount)
	}

	return tokenAccounts, rows.Err()
}
