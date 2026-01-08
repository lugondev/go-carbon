package storage

import (
	"context"
	"fmt"

	"github.com/lugondev/go-carbon/internal/config"
)

type DatabaseType string

const (
	DatabaseTypeMongoDB  DatabaseType = "mongodb"
	DatabaseTypePostgres DatabaseType = "postgres"
)

type ConnectionManager struct {
	config     *config.DatabaseConfig
	repository Repository
}

func NewConnectionManager(cfg *config.DatabaseConfig) (*ConnectionManager, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("database is not enabled in configuration")
	}

	return &ConnectionManager{
		config: cfg,
	}, nil
}

func (cm *ConnectionManager) Connect(ctx context.Context) (Repository, error) {
	if cm.repository != nil {
		return cm.repository, nil
	}

	var repo Repository
	var err error

	switch DatabaseType(cm.config.Type) {
	case DatabaseTypeMongoDB:
		repo, err = NewMongoRepositoryFromConfig(ctx, &cm.config.MongoDB)
	case DatabaseTypePostgres:
		repo, err = NewPostgresRepositoryFromConfig(ctx, &cm.config.Postgres)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cm.config.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := repo.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	cm.repository = repo
	return repo, nil
}

func (cm *ConnectionManager) GetRepository() (Repository, error) {
	if cm.repository == nil {
		return nil, fmt.Errorf("database connection not established")
	}
	return cm.repository, nil
}

func (cm *ConnectionManager) Close() error {
	if cm.repository != nil {
		return cm.repository.Close()
	}
	return nil
}
