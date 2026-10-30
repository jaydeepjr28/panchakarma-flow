package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/config"
	"github.com/jaydeepjr28/panchakarma-flow/pkg/logger"
)

// DB wraps standard database/sql.DB with connection pooling and health checks.
type DB struct {
	*sql.DB
	cfg *config.DatabaseConfig
}

// Connect initializes a PostgreSQL connection pool with configured limits and verifies connectivity.
func Connect(ctx context.Context, cfg *config.DatabaseConfig) (*DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database handle: %w", err)
	}

	// Apply connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Ping database with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database at %s:%d/%s: %w", cfg.Host, cfg.Port, cfg.DBName, err)
	}

	logger.Info("Connected to PostgreSQL successfully",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("db_name", cfg.DBName),
		slog.Int("max_open_conns", cfg.MaxOpenConns),
	)

	return &DB{DB: db, cfg: cfg}, nil
}

// NewTestDB creates a database.DB wrapper for unit/mock testing.
func NewTestDB(sqlDB *sql.DB) *DB {
	return &DB{DB: sqlDB, cfg: &config.DatabaseConfig{DBName: "test_db"}}
}

// PingCheck validates database connectivity for health checks.
func (d *DB) PingCheck(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return d.PingContext(pingCtx)
}

// WithTx executes a function within a database transaction, rolling back on error and committing on success.
func (d *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RunMigrations executes all *.up.sql files in migrationDir in alphanumeric order.
func (d *DB) RunMigrations(ctx context.Context, migrationDir string) error {
	// Create schema migrations tracking table if not exists
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := d.ExecContext(ctx, createTableQuery); err != nil {
		return fmt.Errorf("failed to initialize schema_migrations table: %w", err)
	}

	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		alt := filepath.Join("backend", migrationDir)
		if _, err := os.Stat(alt); err == nil {
			migrationDir = alt
		}
	}

	files, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migrationDir, err)
	}

	var upFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			upFiles = append(upFiles, f.Name())
		}
	}
	sort.Strings(upFiles)

	for _, filename := range upFiles {
		version := strings.TrimSuffix(filename, ".up.sql")

		var exists bool
		err := d.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", version, err)
		}

		if exists {
			continue // Already applied
		}

		filePath := filepath.Join(migrationDir, filename)
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
		}

		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin migration transaction for %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s failed: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to record migration version for %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration transaction for %s: %w", version, err)
		}

		logger.Info("Applied database migration successfully",
			slog.String("db", d.cfg.DBName),
			slog.String("version", version),
		)
	}

	return nil
}
