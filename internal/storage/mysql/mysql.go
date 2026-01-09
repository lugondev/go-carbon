package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/storage"
)

func init() {
	storage.RegisterMySQLFactory(func(ctx context.Context, cfg *config.MySQLConfig) (storage.Repository, error) {
		return NewMySQLRepository(ctx, cfg)
	})
}

type MySQLRepository struct {
	db               *sql.DB
	accountRepo      storage.AccountRepository
	transactionRepo  storage.TransactionRepository
	instructionRepo  storage.InstructionRepository
	eventRepo        storage.EventRepository
	tokenAccountRepo storage.TokenAccountRepository
}

func NewMySQLRepository(ctx context.Context, cfg *config.MySQLConfig) (*MySQLRepository, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
	)

	if cfg.SSLMode != "" && cfg.SSLMode != "false" && cfg.SSLMode != "disable" {
		dsn += fmt.Sprintf("&tls=%s", cfg.SSLMode)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	repo := &MySQLRepository{
		db: db,
	}

	repo.accountRepo = &mysqlAccountRepository{db: db}
	repo.transactionRepo = &mysqlTransactionRepository{db: db}
	repo.instructionRepo = &mysqlInstructionRepository{db: db}
	repo.eventRepo = &mysqlEventRepository{db: db}
	repo.tokenAccountRepo = &mysqlTokenAccountRepository{db: db}

	migrator := NewMigrator(db)
	if err := migrator.Up(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

func (r *MySQLRepository) Accounts() storage.AccountRepository {
	return r.accountRepo
}

func (r *MySQLRepository) Transactions() storage.TransactionRepository {
	return r.transactionRepo
}

func (r *MySQLRepository) Instructions() storage.InstructionRepository {
	return r.instructionRepo
}

func (r *MySQLRepository) Events() storage.EventRepository {
	return r.eventRepo
}

func (r *MySQLRepository) TokenAccounts() storage.TokenAccountRepository {
	return r.tokenAccountRepo
}

func (r *MySQLRepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

func (r *MySQLRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}
