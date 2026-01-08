package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
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
			id TEXT PRIMARY KEY,
			pubkey TEXT UNIQUE NOT NULL,
			lamports BIGINT NOT NULL,
			data BYTEA,
			owner TEXT NOT NULL,
			executable BOOLEAN NOT NULL,
			rent_epoch BIGINT NOT NULL,
			slot BIGINT NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_accounts_owner ON accounts(owner);
		CREATE INDEX IF NOT EXISTS idx_accounts_slot ON accounts(slot DESC);

		CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			signature TEXT UNIQUE NOT NULL,
			slot BIGINT NOT NULL,
			block_time BIGINT,
			fee BIGINT NOT NULL,
			is_vote BOOLEAN NOT NULL,
			success BOOLEAN NOT NULL,
			error_message TEXT,
			account_keys TEXT[] NOT NULL,
			num_instructions INT NOT NULL,
			num_inner_instructions INT NOT NULL,
			log_messages TEXT[],
			compute_units_consumed BIGINT,
			created_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_transactions_slot ON transactions(slot DESC);
		CREATE INDEX IF NOT EXISTS idx_transactions_success ON transactions(success);
		CREATE INDEX IF NOT EXISTS idx_transactions_block_time ON transactions(block_time DESC);

		CREATE TABLE IF NOT EXISTS instructions (
			id TEXT PRIMARY KEY,
			signature TEXT NOT NULL,
			instruction_index INT NOT NULL,
			program_id TEXT NOT NULL,
			data BYTEA,
			accounts TEXT[] NOT NULL,
			is_inner BOOLEAN NOT NULL,
			inner_index INT,
			created_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_instructions_signature ON instructions(signature);
		CREATE INDEX IF NOT EXISTS idx_instructions_program_id ON instructions(program_id);

		CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			signature TEXT NOT NULL,
			program_id TEXT NOT NULL,
			event_name TEXT NOT NULL,
			data JSONB NOT NULL,
			slot BIGINT NOT NULL,
			block_time BIGINT,
			created_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_events_signature ON events(signature);
		CREATE INDEX IF NOT EXISTS idx_events_program_id ON events(program_id);
		CREATE INDEX IF NOT EXISTS idx_events_event_name ON events(event_name);
		CREATE INDEX IF NOT EXISTS idx_events_slot ON events(slot DESC);

		CREATE TABLE IF NOT EXISTS token_accounts (
			id TEXT PRIMARY KEY,
			address TEXT UNIQUE NOT NULL,
			mint TEXT NOT NULL,
			owner TEXT NOT NULL,
			amount BIGINT NOT NULL,
			decimals SMALLINT NOT NULL,
			delegate TEXT,
			delegated_amount BIGINT NOT NULL,
			is_native BOOLEAN NOT NULL,
			close_authority TEXT,
			slot BIGINT NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_token_accounts_mint ON token_accounts(mint);
		CREATE INDEX IF NOT EXISTS idx_token_accounts_owner ON token_accounts(owner);
		CREATE INDEX IF NOT EXISTS idx_token_accounts_slot ON token_accounts(slot DESC);
		`,
		Down: `
		DROP TABLE IF EXISTS token_accounts;
		DROP TABLE IF EXISTS events;
		DROP TABLE IF EXISTS instructions;
		DROP TABLE IF EXISTS transactions;
		DROP TABLE IF EXISTS accounts;
		`,
	},
}

type Migrator struct {
	pool *pgxpool.Pool
}

func NewMigrator(pool *pgxpool.Pool) *Migrator {
	return &Migrator{pool: pool}
}

func (m *Migrator) createMigrationsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version INT PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`
	_, err := m.pool.Exec(ctx, query)
	return err
}

func (m *Migrator) getCurrentVersion(ctx context.Context) (int, error) {
	var version int
	err := m.pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func (m *Migrator) Up(ctx context.Context) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	applied := 0
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		if _, err := tx.Exec(ctx, migration.Up); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
			migration.Version, migration.Description,
		); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		applied++
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	if applied > 0 {
		fmt.Printf("Applied %d migration(s)\n", applied)
	}

	return nil
}

func (m *Migrator) Down(ctx context.Context, steps int) error {
	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	if currentVersion == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rolledBack := 0
	for i := len(migrations) - 1; i >= 0 && rolledBack < steps; i-- {
		migration := migrations[i]
		if migration.Version > currentVersion {
			continue
		}

		if _, err := tx.Exec(ctx, migration.Down); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
		}

		if _, err := tx.Exec(ctx,
			"DELETE FROM schema_migrations WHERE version = $1",
			migration.Version,
		); err != nil {
			return fmt.Errorf("failed to remove migration record %d: %w", migration.Version, err)
		}

		rolledBack++
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	fmt.Printf("Rolled back %d migration(s)\n", rolledBack)
	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	currentVersion, err := m.getCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Printf("Current schema version: %d\n", currentVersion)
	fmt.Printf("Available migrations:\n")
	for _, migration := range migrations {
		status := "pending"
		if migration.Version <= currentVersion {
			status = "applied"
		}
		fmt.Printf("  [%s] v%d: %s\n", status, migration.Version, migration.Description)
	}

	return nil
}
