package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScanFunc[T any] func(rows pgx.Rows) (*T, error)

func QueryMany[T any](
	pool *pgxpool.Pool,
	ctx context.Context,
	query string,
	scanFunc ScanFunc[T],
	args ...interface{},
) ([]*T, error) {
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*T
	for rows.Next() {
		item, err := scanFunc(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	return results, rows.Err()
}

func QueryOne[T any](
	pool *pgxpool.Pool,
	ctx context.Context,
	query string,
	scanFunc func(row pgx.Row) (*T, error),
	args ...interface{},
) (*T, error) {
	row := pool.QueryRow(ctx, query, args...)
	return scanFunc(row)
}
