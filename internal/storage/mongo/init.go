package mongo

import (
	"context"
	"fmt"

	"github.com/lugondev/go-carbon/internal/config"
	"github.com/lugondev/go-carbon/internal/storage"
)

func init() {
	storage.RegisterMongoFactory(func(ctx context.Context, cfg *config.MongoDBConfig) (storage.Repository, error) {
		repo, err := NewMongoRepository(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create mongo repository: %w", err)
		}
		return repo, nil
	})
}
