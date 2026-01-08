package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/datasource"
	"github.com/lugondev/go-carbon/internal/processor/database"
	"github.com/lugondev/go-carbon/internal/storage"
	_ "github.com/lugondev/go-carbon/internal/storage/mongo"
	_ "github.com/lugondev/go-carbon/internal/storage/postgres"
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
			Type:    "postgres",
			Postgres: config.PostgresConfig{
				Host:            "localhost",
				Port:            5432,
				User:            "carbon",
				Password:        "carbon123",
				Database:        "carbon_db",
				SSLMode:         "disable",
				MaxOpenConns:    25,
				MaxIdleConns:    5,
				ConnMaxLifetime: 300,
			},
			MongoDB: config.MongoDBConfig{
				URI:            "mongodb://localhost:27017",
				Database:       "carbon_db",
				MaxPoolSize:    100,
				MinPoolSize:    10,
				ConnectTimeout: 10,
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

	logger.Info("connected to database", "type", cfg.Database.Type)

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
		log.Fatalf("failed to process account update: %v", err)
	}

	logger.Info("account saved successfully", "pubkey", accountUpdate.Pubkey.String())

	batchProcessor := database.NewBatchDatasourceProcessor(repo, logger, 100)

	for i := 0; i < 5; i++ {
		pubkey := solana.NewWallet().PublicKey()
		update := &datasource.AccountUpdate{
			Pubkey: pubkey,
			Account: types.Account{
				Lamports:   uint64(1000000 * (i + 1)),
				Data:       []byte{byte(i)},
				Owner:      solana.SystemProgramID,
				Executable: false,
				RentEpoch:  100,
			},
			Slot: 12345678 + uint64(i),
		}

		if err := batchProcessor.ProcessAccountUpdate(ctx, update); err != nil {
			log.Fatalf("failed to add to batch: %v", err)
		}
	}

	if err := batchProcessor.FlushAll(ctx); err != nil {
		log.Fatalf("failed to flush batch: %v", err)
	}

	logger.Info("batch saved successfully")

	accounts, err := repo.Accounts().FindByOwner(ctx, solana.SystemProgramID.String(), 0, 10)
	if err != nil {
		log.Fatalf("failed to query accounts: %v", err)
	}

	logger.Info("query results", "count", len(accounts))
	for _, acc := range accounts {
		fmt.Printf("Account: %s, Lamports: %d, Slot: %d\n", acc.Pubkey, acc.Lamports, acc.Slot)
	}

	time.Sleep(1 * time.Second)
	logger.Info("example completed successfully")
}
