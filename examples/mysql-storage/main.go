package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/processor/database"
	"github.com/lugondev/go-carbon/internal/storage"
	_ "github.com/lugondev/go-carbon/internal/storage/mysql"
	"github.com/lugondev/go-carbon/pkg/types"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("shutting down gracefully...")
		cancel()
	}()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Enabled: true,
			Type:    "mysql",
			MySQL: config.MySQLConfig{
				Host:            "localhost",
				Port:            3306,
				User:            "carbon",
				Password:        "carbon123",
				Database:        "carbon_db",
				SSLMode:         "false",
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
		},
	}

	connMgr, err := storage.NewConnectionManager(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to create connection manager: %v", err)
	}

	repo, err := connMgr.Connect(ctx)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer connMgr.Close()

	logger.Info("connected to MySQL database successfully")

	if err := demonstrateSingleOperations(ctx, repo, logger); err != nil {
		log.Fatalf("single operations failed: %v", err)
	}

	if err := demonstrateBatchOperations(ctx, repo, logger); err != nil {
		log.Fatalf("batch operations failed: %v", err)
	}

	if err := demonstrateQueries(ctx, repo, logger); err != nil {
		log.Fatalf("queries failed: %v", err)
	}

	logger.Info("MySQL storage example completed successfully")
}

func demonstrateSingleOperations(ctx context.Context, repo storage.Repository, logger *slog.Logger) error {
	logger.Info("=== Demonstrating Single Operations ===")

	dbProcessor := database.NewDatasourceProcessor(repo, logger)

	accountUpdate := &datasource.AccountUpdate{
		Pubkey: solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"),
		Account: types.Account{
			Lamports:   1000000,
			Data:       []byte{1, 2, 3, 4, 5},
			Owner:      solana.SystemProgramID,
			Executable: false,
			RentEpoch:  100,
		},
		Slot: 12345678,
	}

	if err := dbProcessor.ProcessAccountUpdate(ctx, accountUpdate); err != nil {
		return fmt.Errorf("failed to process account update: %w", err)
	}

	logger.Info("account saved successfully",
		"pubkey", accountUpdate.Pubkey.String(),
		"lamports", accountUpdate.Account.Lamports,
		"slot", accountUpdate.Slot,
	)

	return nil
}

func demonstrateBatchOperations(ctx context.Context, repo storage.Repository, logger *slog.Logger) error {
	logger.Info("=== Demonstrating Batch Operations ===")

	batchProcessor := database.NewBatchDatasourceProcessor(repo, logger, 100)

	systemOwner := solana.SystemProgramID
	for i := 0; i < 5; i++ {
		pubkey := solana.NewWallet().PublicKey()
		accountUpdate := &datasource.AccountUpdate{
			Pubkey: pubkey,
			Account: types.Account{
				Lamports:   uint64(1000000 * (i + 1)),
				Data:       []byte{byte(i)},
				Owner:      systemOwner,
				Executable: false,
				RentEpoch:  100,
			},
			Slot: uint64(12345678 + i),
		}
		if err := batchProcessor.ProcessAccountUpdate(ctx, accountUpdate); err != nil {
			return fmt.Errorf("failed to add to batch: %w", err)
		}
	}

	if err := batchProcessor.FlushAll(ctx); err != nil {
		return fmt.Errorf("failed to flush batch: %w", err)
	}

	logger.Info("batch operations completed", "count", 5)
	return nil
}

func demonstrateQueries(ctx context.Context, repo storage.Repository, logger *slog.Logger) error {
	logger.Info("=== Demonstrating Queries ===")

	systemOwner := solana.SystemProgramID.String()
	accounts, err := repo.Accounts().FindByOwner(ctx, systemOwner, 10, 0)
	if err != nil {
		return fmt.Errorf("failed to query accounts: %w", err)
	}

	logger.Info("query results", "count", len(accounts), "owner", systemOwner)

	for i, account := range accounts {
		fmt.Printf("Account %d: %s, Lamports: %d, Slot: %d\n",
			i+1, account.Pubkey, account.Lamports, account.Slot)
	}

	tokenAccount := "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"
	account, err := repo.Accounts().FindByPubkey(ctx, tokenAccount)
	if err != nil {
		return fmt.Errorf("failed to find account by pubkey: %w", err)
	}

	if account != nil {
		logger.Info("found specific account",
			"pubkey", account.Pubkey,
			"lamports", account.Lamports,
			"slot", account.Slot,
		)
	}

	return nil
}
