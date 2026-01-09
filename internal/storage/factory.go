package storage

import (
	"context"

	"github.com/lugondev/go-carbon/internal/config"
)

var (
	mongoFactory    func(context.Context, *config.MongoDBConfig) (Repository, error)
	postgresFactory func(context.Context, *config.PostgresConfig) (Repository, error)
	mysqlFactory    func(context.Context, *config.MySQLConfig) (Repository, error)
)

func RegisterMongoFactory(factory func(context.Context, *config.MongoDBConfig) (Repository, error)) {
	mongoFactory = factory
}

func RegisterPostgresFactory(factory func(context.Context, *config.PostgresConfig) (Repository, error)) {
	postgresFactory = factory
}

func RegisterMySQLFactory(factory func(context.Context, *config.MySQLConfig) (Repository, error)) {
	mysqlFactory = factory
}

func NewMongoRepositoryFromConfig(ctx context.Context, cfg *config.MongoDBConfig) (Repository, error) {
	if mongoFactory == nil {
		panic("mongo factory not registered - import _ \"github.com/lugondev/go-carbon/internal/storage/mongo\"")
	}
	return mongoFactory(ctx, cfg)
}

func NewPostgresRepositoryFromConfig(ctx context.Context, cfg *config.PostgresConfig) (Repository, error) {
	if postgresFactory == nil {
		panic("postgres factory not registered - import _ \"github.com/lugondev/go-carbon/internal/storage/postgres\"")
	}
	return postgresFactory(ctx, cfg)
}

func NewMySQLRepositoryFromConfig(ctx context.Context, cfg *config.MySQLConfig) (Repository, error) {
	if mysqlFactory == nil {
		panic("mysql factory not registered - import _ \"github.com/lugondev/go-carbon/internal/storage/mysql\"")
	}
	return mysqlFactory(ctx, cfg)
}
