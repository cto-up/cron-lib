package service

import (
	"context"
	"fmt"

	dbCore "ctoup.com/coreapp/pkg/core/db"
	"github.com/cto-up/cron-lib/pkg/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SeedService struct {
	store     *db.Store
	coreStore *dbCore.Store
}

func NewSeedService(connPool *pgxpool.Pool) *SeedService {
	return &SeedService{store: db.NewStore(connPool, true), coreStore: dbCore.NewStore(connPool)}
}

func (s *SeedService) SeedReferenceData(ctx context.Context, tenantID string) error {
	// Begin a transaction directly from the connection pool
	tx, err := s.store.ConnPool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return err
	}

	// Create query objects for both stores using the same transaction
	qtx := s.store.Queries.WithTx(tx)
	coreTx := s.coreStore.Queries.WithTx(tx)

	if err := s.seedRefenceData(ctx, qtx, coreTx, tenantID); err != nil {
		return err
	}
	err = s.seedCoreRefenceData(ctx, tenantID)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit error: %v", err)
	}
	return nil

}

func (s *SeedService) SeedSampleData(ctx context.Context, tenantID string) error {
	// Begin a transaction directly from the connection pool
	tx, err := s.store.ConnPool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return err
	}

	// Create query objects for both stores using the same transaction
	qtx := s.store.Queries.WithTx(tx)

	// Execute operations using both transaction objects
	if err := s.seedSampleData(ctx, qtx, tenantID); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit error: %v", err)
	}

	return nil
}
