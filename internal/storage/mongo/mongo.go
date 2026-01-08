package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/storage"
)

type MongoRepository struct {
	client           *mongo.Client
	database         *mongo.Database
	accounts         *mongo.Collection
	transactions     *mongo.Collection
	instructions     *mongo.Collection
	events           *mongo.Collection
	tokenAccounts    *mongo.Collection
	accountRepo      storage.AccountRepository
	transactionRepo  storage.TransactionRepository
	instructionRepo  storage.InstructionRepository
	eventRepo        storage.EventRepository
	tokenAccountRepo storage.TokenAccountRepository
}

func NewMongoRepository(ctx context.Context, cfg *config.MongoDBConfig) (*MongoRepository, error) {
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetConnectTimeout(time.Duration(cfg.ConnectTimeout) * time.Second).
		SetRetryWrites(true).
		SetRetryReads(true)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.Database)

	repo := &MongoRepository{
		client:        client,
		database:      database,
		accounts:      database.Collection("accounts"),
		transactions:  database.Collection("transactions"),
		instructions:  database.Collection("instructions"),
		events:        database.Collection("events"),
		tokenAccounts: database.Collection("token_accounts"),
	}

	repo.accountRepo = &mongoAccountRepository{collection: repo.accounts}
	repo.transactionRepo = &mongoTransactionRepository{collection: repo.transactions}
	repo.instructionRepo = &mongoInstructionRepository{collection: repo.instructions}
	repo.eventRepo = &mongoEventRepository{collection: repo.events}
	repo.tokenAccountRepo = &mongoTokenAccountRepository{collection: repo.tokenAccounts}

	if err := repo.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return repo, nil
}

func (r *MongoRepository) createIndexes(ctx context.Context) error {
	indexes := []struct {
		collection *mongo.Collection
		models     []mongo.IndexModel
	}{
		{
			collection: r.accounts,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "pubkey", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "owner", Value: 1}}},
				{Keys: bson.D{{Key: "slot", Value: -1}}},
			},
		},
		{
			collection: r.transactions,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "signature", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "slot", Value: -1}}},
				{Keys: bson.D{{Key: "account_keys", Value: 1}}},
				{Keys: bson.D{{Key: "created_at", Value: -1}}},
			},
		},
		{
			collection: r.instructions,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "signature", Value: 1}, {Key: "instruction_index", Value: 1}}},
				{Keys: bson.D{{Key: "program_id", Value: 1}}},
			},
		},
		{
			collection: r.events,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "signature", Value: 1}}},
				{Keys: bson.D{{Key: "program_id", Value: 1}}},
				{Keys: bson.D{{Key: "event_name", Value: 1}}},
				{Keys: bson.D{{Key: "slot", Value: -1}}},
			},
		},
		{
			collection: r.tokenAccounts,
			models: []mongo.IndexModel{
				{Keys: bson.D{{Key: "address", Value: 1}}, Options: options.Index().SetUnique(true)},
				{Keys: bson.D{{Key: "owner", Value: 1}}},
				{Keys: bson.D{{Key: "mint", Value: 1}}},
			},
		},
	}

	for _, idx := range indexes {
		if _, err := idx.collection.Indexes().CreateMany(ctx, idx.models); err != nil {
			return err
		}
	}

	return nil
}

func (r *MongoRepository) Accounts() storage.AccountRepository {
	return r.accountRepo
}

func (r *MongoRepository) Transactions() storage.TransactionRepository {
	return r.transactionRepo
}

func (r *MongoRepository) Instructions() storage.InstructionRepository {
	return r.instructionRepo
}

func (r *MongoRepository) Events() storage.EventRepository {
	return r.eventRepo
}

func (r *MongoRepository) TokenAccounts() storage.TokenAccountRepository {
	return r.tokenAccountRepo
}

func (r *MongoRepository) Close() error {
	if r.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return r.client.Disconnect(ctx)
	}
	return nil
}

func (r *MongoRepository) Ping(ctx context.Context) error {
	return r.client.Ping(ctx, readpref.Primary())
}
