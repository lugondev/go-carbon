package storage

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
)

// BatchProcessor defines the interface for batch operations.
type BatchProcessor[T any] interface {
	ProcessBatch(ctx context.Context, items []T) error
}

// MongoBatchHelper provides reusable batch operations for MongoDB.
type MongoBatchHelper[T any] struct {
	collection *mongo.Collection
}

// NewMongoBatchHelper creates a new MongoDB batch helper.
func NewMongoBatchHelper[T any](collection *mongo.Collection) *MongoBatchHelper[T] {
	return &MongoBatchHelper[T]{
		collection: collection,
	}
}

// InsertMany performs batch insert for MongoDB.
func (h *MongoBatchHelper[T]) InsertMany(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}

	docs := make([]interface{}, len(items))
	for i, item := range items {
		docs[i] = item
	}

	_, err := h.collection.InsertMany(ctx, docs)
	return err
}

// PostgresBatchHelper provides reusable batch operations for PostgreSQL.
type PostgresBatchHelper struct {
	pool *pgxpool.Pool
}

// NewPostgresBatchHelper creates a new PostgreSQL batch helper.
func NewPostgresBatchHelper(pool *pgxpool.Pool) *PostgresBatchHelper {
	return &PostgresBatchHelper{
		pool: pool,
	}
}

// BatchInsert performs batch insert with the given query and item processor.
func (h *PostgresBatchHelper) BatchInsert(
	ctx context.Context,
	query string,
	items int,
	queueFunc func(batch *pgx.Batch, index int),
) error {
	if items == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for i := 0; i < items; i++ {
		queueFunc(batch, i)
	}

	br := h.pool.SendBatch(ctx, batch)
	defer br.Close()

	for i := 0; i < items; i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return br.Close()
}

type MySQLBatchHelper struct {
	db *sql.DB
}

func NewMySQLBatchHelper(db *sql.DB) *MySQLBatchHelper {
	return &MySQLBatchHelper{
		db: db,
	}
}

func (h *MySQLBatchHelper) BatchInsert(
	ctx context.Context,
	query string,
	items int,
	prepareFunc func(stmt *sql.Stmt, index int) error,
) error {
	if items == 0 {
		return nil
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := 0; i < items; i++ {
		if err := prepareFunc(stmt, i); err != nil {
			return err
		}
	}

	return tx.Commit()
}
