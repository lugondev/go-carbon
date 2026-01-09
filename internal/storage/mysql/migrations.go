package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

var migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema",
		Up: `
		CREATE TABLE IF NOT EXISTS accounts (
			id VARCHAR(255) PRIMARY KEY,
			pubkey VARCHAR(255) UNIQUE NOT NULL,
			lamports BIGINT NOT NULL,
			data LONGBLOB,
			owner VARCHAR(255) NOT NULL,
			executable BOOLEAN NOT NULL,
			rent_epoch BIGINT NOT NULL,
			slot BIGINT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_accounts_owner (owner),
			INDEX idx_accounts_slot (slot DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

		CREATE TABLE IF NOT EXISTS transactions (
			id VARCHAR(255) PRIMARY KEY,
			signature VARCHAR(255) UNIQUE NOT NULL,
			slot BIGINT NOT NULL,
			block_time BIGINT,
			fee BIGINT NOT NULL,
			is_vote BOOLEAN NOT NULL,
			success BOOLEAN NOT NULL,
			error_message TEXT,
			account_keys JSON NOT NULL,
			num_instructions INT NOT NULL,
			num_inner_instructions INT NOT NULL,
			log_messages JSON,
			compute_units_consumed BIGINT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_transactions_slot (slot DESC),
			INDEX idx_transactions_success (success),
			INDEX idx_transactions_block_time (block_time DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

		CREATE TABLE IF NOT EXISTS instructions (
			id VARCHAR(255) PRIMARY KEY,
			signature VARCHAR(255) NOT NULL,
			instruction_index INT NOT NULL,
			program_id VARCHAR(255) NOT NULL,
			data LONGBLOB,
			accounts JSON NOT NULL,
			is_inner BOOLEAN NOT NULL,
			inner_index INT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_instructions_signature (signature),
			INDEX idx_instructions_program_id (program_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

		CREATE TABLE IF NOT EXISTS events (
			id VARCHAR(255) PRIMARY KEY,
			signature VARCHAR(255) NOT NULL,
			program_id VARCHAR(255) NOT NULL,
			event_name VARCHAR(255) NOT NULL,
			data JSON NOT NULL,
			slot BIGINT NOT NULL,
			block_time BIGINT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_events_signature (signature),
			INDEX idx_events_program_id (program_id),
			INDEX idx_events_event_name (event_name),
			INDEX idx_events_slot (slot DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

		CREATE TABLE IF NOT EXISTS token_accounts (
			id VARCHAR(255) PRIMARY KEY,
			address VARCHAR(255) UNIQUE NOT NULL,
			mint VARCHAR(255) NOT NULL,
			owner VARCHAR(255) NOT NULL,
			amount BIGINT NOT NULL,
			decimals SMALLINT NOT NULL,
			delegate VARCHAR(255),
			delegated_amount BIGINT NOT NULL,
			is_native BOOLEAN NOT NULL,
			close_authority VARCHAR(255),
			slot BIGINT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_token_accounts_owner (owner),
			INDEX idx_token_accounts_mint (mint),
			INDEX idx_token_accounts_slot (slot DESC)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			description VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
		`,
		Down: `
		DROP TABLE IF EXISTS token_accounts;
		DROP TABLE IF EXISTS events;
		DROP TABLE IF EXISTS instructions;
		DROP TABLE IF EXISTS transactions;
		DROP TABLE IF EXISTS accounts;
		DROP TABLE IF EXISTS schema_migrations;
		`,
	},
}

type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

func (m *Migrator) Up(ctx context.Context) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	for _, migration := range migrations {
		applied, err := m.isMigrationApplied(ctx, migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check if migration %d is applied: %w", migration.Version, err)
		}

		if applied {
			continue
		}

		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Description)
	}

	return nil
}

func (m *Migrator) Down(ctx context.Context, targetVersion int) error {
	for i := len(migrations) - 1; i >= 0; i-- {
		migration := migrations[i]
		if migration.Version <= targetVersion {
			break
		}

		applied, err := m.isMigrationApplied(ctx, migration.Version)
		if err != nil {
			return fmt.Errorf("failed to check if migration %d is applied: %w", migration.Version, err)
		}

		if !applied {
			continue
		}

		if err := m.revertMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to revert migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Reverted migration %d: %s\n", migration.Version, migration.Description)
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INT PRIMARY KEY,
		description VARCHAR(255) NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

func (m *Migrator) isMigrationApplied(ctx context.Context, version int) (bool, error) {
	query := `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`
	var count int
	err := m.db.QueryRowContext(ctx, query, version).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (m *Migrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
		return err
	}

	insertQuery := `INSERT INTO schema_migrations (version, description) VALUES (?, ?)`
	if _, err := tx.ExecContext(ctx, insertQuery, migration.Version, migration.Description); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *Migrator) revertMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, migration.Down); err != nil {
		return err
	}

	deleteQuery := `DELETE FROM schema_migrations WHERE version = ?`
	if _, err := tx.ExecContext(ctx, deleteQuery, migration.Version); err != nil {
		return err
	}

	return tx.Commit()
}
