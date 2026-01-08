package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/storage"
)

type PostgresRepository struct {
	pool             *pgxpool.Pool
	accountRepo      storage.AccountRepository
	transactionRepo  storage.TransactionRepository
	instructionRepo  storage.InstructionRepository
	eventRepo        storage.EventRepository
	tokenAccountRepo storage.TokenAccountRepository
}

func NewPostgresRepository(ctx context.Context, cfg *config.PostgresConfig) (*PostgresRepository, error) {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode,
	)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	if cfg.ConnMaxLifetime > 0 {
		poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetime) * time.Second
	}
	poolConfig.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &PostgresRepository{
		pool: pool,
	}

	repo.accountRepo = &postgresAccountRepository{pool: pool}
	repo.transactionRepo = &postgresTransactionRepository{pool: pool}
	repo.instructionRepo = &postgresInstructionRepository{pool: pool}
	repo.eventRepo = &postgresEventRepository{pool: pool}
	repo.tokenAccountRepo = &postgresTokenAccountRepository{pool: pool}

	migrator := NewMigrator(pool)
	if err := migrator.Up(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

func (r *PostgresRepository) Accounts() storage.AccountRepository {
	return r.accountRepo
}

func (r *PostgresRepository) Transactions() storage.TransactionRepository {
	return r.transactionRepo
}

func (r *PostgresRepository) Instructions() storage.InstructionRepository {
	return r.instructionRepo
}

func (r *PostgresRepository) Events() storage.EventRepository {
	return r.eventRepo
}

func (r *PostgresRepository) TokenAccounts() storage.TokenAccountRepository {
	return r.tokenAccountRepo
}

func (r *PostgresRepository) Close() error {
	if r.pool != nil {
		r.pool.Close()
	}
	return nil
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
