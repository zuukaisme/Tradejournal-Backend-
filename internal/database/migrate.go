package database

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(741852963)"); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create migration history: %w", err)
	}

	var applied bool
	if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = 1)").Scan(&applied); err != nil {
		return fmt.Errorf("check migration history: %w", err)
	}
	if !applied {
		schema, err := migrationFiles.ReadFile("migrations/000001_initial_schema.up.sql")
		if err != nil {
			return fmt.Errorf("read initial migration: %w", err)
		}
		if _, err := tx.Exec(ctx, string(schema)); err != nil {
			return fmt.Errorf("apply initial migration: %w", err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES (1)"); err != nil {
			return fmt.Errorf("record initial migration: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}
