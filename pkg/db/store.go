package db

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	sqlservice "ctoup.com/coreapp/pkg/shared/sql"
	"github.com/cto-up/cron-lib/pkg/db/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Provides all function to execute db queries and transactions
type Store struct {
	*repository.Queries
	ConnPool *pgxpool.Pool
}

func NewStore(connPool *pgxpool.Pool, skipMigration bool) *Store {
	if !skipMigration {
		migrate(connPool.Config().ConnString())
	}
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

var prefix = "cron"

// getMigrationPath returns the absolute path to the migration directory
func getMigrationPath() string {
	_, filename, _, _ := runtime.Caller(0)
	// Get the directory containing this file (store.go)
	dir := filepath.Dir(filename)
	// Construct path to migration directory
	migrationPath := filepath.Join(dir, "migration")
	return "file://" + migrationPath
}

var once = sync.Once{}

func migrate(dbConnection string) {
	once.Do(func() {
		path := getMigrationPath()
		sqlservice.MigrateUp(dbConnection, path, prefix)
	})
}

func MigrateUp(dbConnection string) error {
	path := getMigrationPath()
	return sqlservice.MigrateUp(dbConnection, path, prefix)
}

func MigrateDown(dbConnection string) error {
	path := getMigrationPath()
	return sqlservice.MigrateDown(dbConnection, path, prefix)
}
