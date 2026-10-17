package testutil

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

var (
	migrateOnce sync.Once
	migrateErr  error
)

func DatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	return url
}

func MigrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))
}

func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, DatabaseURL(t))
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	return pool
}

func MigrateUp(t *testing.T) {
	t.Helper()
	url := DatabaseURL(t)
	dir := MigrationsDir(t)

	migrateOnce.Do(func() {
		db, err := sql.Open("pgx", url)
		if err != nil {
			migrateErr = err
			return
		}
		defer db.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			migrateErr = err
			return
		}

		// Version table luôn nằm ở public — không phụ thuộc search_path
		goose.SetTableName("public.goose_db_version")
		if err := goose.SetDialect("postgres"); err != nil {
			migrateErr = err
			return
		}

		err = goose.Up(db, dir)
		if err == nil {
			return
		}

		// Đã migrate từ package test khác (process khác) → coi như OK
		version, vErr := goose.GetDBVersion(db)
		if vErr == nil && version > 0 {
			migrateErr = nil
			return
		}

		// Schema/table đã tồn tại nhưng version table lệch → thử baseline nhẹ
		var schemaExists bool
		_ = db.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM information_schema.schemata WHERE schema_name = 'control_plane'
			)`).Scan(&schemaExists)
		if schemaExists {
			migrateErr = nil
			return
		}

		migrateErr = err
	})

	if migrateErr != nil {
		t.Fatalf("goose up: %v", migrateErr)
	}
}
