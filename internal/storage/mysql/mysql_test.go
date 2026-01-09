package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/storage"
)

func TestNewMySQLRepository(t *testing.T) {
	t.Skip("Requires MySQL database - run manually with docker")

	cfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "carbon",
		Password:        "carbon123",
		Database:        "carbon_test",
		SSLMode:         "false",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 60,
	}

	ctx := context.Background()
	repo, err := NewMySQLRepository(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	if err := repo.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	if repo.Accounts() == nil {
		t.Error("Accounts repository is nil")
	}
	if repo.Transactions() == nil {
		t.Error("Transactions repository is nil")
	}
	if repo.Instructions() == nil {
		t.Error("Instructions repository is nil")
	}
	if repo.Events() == nil {
		t.Error("Events repository is nil")
	}
	if repo.TokenAccounts() == nil {
		t.Error("TokenAccounts repository is nil")
	}
}

func TestAccountRepository_SaveAndFind(t *testing.T) {
	t.Skip("Requires MySQL database - run manually with docker")

	cfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "carbon",
		Password:        "carbon123",
		Database:        "carbon_test",
		SSLMode:         "false",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 60,
	}

	ctx := context.Background()
	repo, err := NewMySQLRepository(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	now := time.Now()
	account := &storage.AccountModel{
		ID:         "test-account-1",
		Pubkey:     "TestPubkey123456789",
		Lamports:   1000000,
		Data:       []byte{1, 2, 3, 4, 5},
		Owner:      "11111111111111111111111111111111",
		Executable: false,
		RentEpoch:  100,
		Slot:       12345678,
		UpdatedAt:  now,
		CreatedAt:  now,
	}

	if err := repo.Accounts().Save(ctx, account); err != nil {
		t.Fatalf("failed to save account: %v", err)
	}

	found, err := repo.Accounts().FindByPubkey(ctx, account.Pubkey)
	if err != nil {
		t.Fatalf("failed to find account: %v", err)
	}

	if found == nil {
		t.Fatal("account not found")
	}

	if found.Pubkey != account.Pubkey {
		t.Errorf("pubkey mismatch: got %s, want %s", found.Pubkey, account.Pubkey)
	}
	if found.Lamports != account.Lamports {
		t.Errorf("lamports mismatch: got %d, want %d", found.Lamports, account.Lamports)
	}
	if found.Owner != account.Owner {
		t.Errorf("owner mismatch: got %s, want %s", found.Owner, account.Owner)
	}
}

func TestAccountRepository_SaveBatch(t *testing.T) {
	t.Skip("Requires MySQL database - run manually with docker")

	cfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "carbon",
		Password:        "carbon123",
		Database:        "carbon_test",
		SSLMode:         "false",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 60,
	}

	ctx := context.Background()
	repo, err := NewMySQLRepository(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	now := time.Now()
	accounts := []*storage.AccountModel{
		{
			ID:         "batch-account-1",
			Pubkey:     "BatchPubkey1",
			Lamports:   1000000,
			Data:       []byte{1},
			Owner:      "11111111111111111111111111111111",
			Executable: false,
			RentEpoch:  100,
			Slot:       12345678,
			UpdatedAt:  now,
			CreatedAt:  now,
		},
		{
			ID:         "batch-account-2",
			Pubkey:     "BatchPubkey2",
			Lamports:   2000000,
			Data:       []byte{2},
			Owner:      "11111111111111111111111111111111",
			Executable: false,
			RentEpoch:  100,
			Slot:       12345679,
			UpdatedAt:  now,
			CreatedAt:  now,
		},
		{
			ID:         "batch-account-3",
			Pubkey:     "BatchPubkey3",
			Lamports:   3000000,
			Data:       []byte{3},
			Owner:      "11111111111111111111111111111111",
			Executable: false,
			RentEpoch:  100,
			Slot:       12345680,
			UpdatedAt:  now,
			CreatedAt:  now,
		},
	}

	if err := repo.Accounts().SaveBatch(ctx, accounts); err != nil {
		t.Fatalf("failed to save batch: %v", err)
	}

	for _, account := range accounts {
		found, err := repo.Accounts().FindByPubkey(ctx, account.Pubkey)
		if err != nil {
			t.Fatalf("failed to find account %s: %v", account.Pubkey, err)
		}
		if found == nil {
			t.Fatalf("account %s not found", account.Pubkey)
		}
		if found.Lamports != account.Lamports {
			t.Errorf("lamports mismatch for %s: got %d, want %d", account.Pubkey, found.Lamports, account.Lamports)
		}
	}
}

func TestAccountRepository_FindByOwner(t *testing.T) {
	t.Skip("Requires MySQL database - run manually with docker")

	cfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "carbon",
		Password:        "carbon123",
		Database:        "carbon_test",
		SSLMode:         "false",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 60,
	}

	ctx := context.Background()
	repo, err := NewMySQLRepository(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	owner := "11111111111111111111111111111111"
	accounts, err := repo.Accounts().FindByOwner(ctx, owner, 10, 0)
	if err != nil {
		t.Fatalf("failed to find accounts: %v", err)
	}

	t.Logf("Found %d accounts for owner %s", len(accounts), owner)

	for _, account := range accounts {
		if account.Owner != owner {
			t.Errorf("owner mismatch: got %s, want %s", account.Owner, owner)
		}
	}
}

func TestMigrator_Up(t *testing.T) {
	t.Skip("Requires MySQL database - run manually with docker")

	cfg := &config.MySQLConfig{
		Host:            "localhost",
		Port:            3306,
		User:            "carbon",
		Password:        "carbon123",
		Database:        "carbon_test_migration",
		SSLMode:         "false",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 60,
	}

	ctx := context.Background()
	repo, err := NewMySQLRepository(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create repository: %v", err)
	}
	defer repo.Close()

	if err := repo.Ping(ctx); err != nil {
		t.Fatalf("failed to ping after migrations: %v", err)
	}

	t.Log("Migrations completed successfully")
}
