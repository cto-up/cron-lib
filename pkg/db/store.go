package db

import (
	"context"
	"fmt"

	"github.com/cto-up/cron-lib/pkg/db/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Provides all function to execute db queries and transactions
type Store struct {
	*repository.Queries
	ConnPool *pgxpool.Pool
}

func NewStore(connPool *pgxpool.Pool) *Store {
	return &Store{
		Queries:  repository.New(connPool),
		ConnPool: connPool,
	}
}

func (s *Store) ExecTx(ctx context.Context, fn func(*repository.Queries) error) error {
	tx, err := s.ConnPool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted, // or RepeatableRead
	})
	if err != nil {
		return err
	}

	qtx := s.Queries.WithTx(tx)

	if err := fn(qtx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit error: %v", err)
	}

	return nil
}
