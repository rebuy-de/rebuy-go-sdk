package pgutil

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrRepeatableMigrationFailed = errors.New("repeatable migration failed")
	ErrInvalidRepeatableFile     = errors.New("invalid repeatable file")
)

const (
	repeatableSQLExtension = ".sql"
	repeatablePrefix       = "R_"
	repeatableMigrationsTable = "schema_migrations_repeatable"
)

// runRepeatableMigrations processes all SQL files with prefix 'R_' in the embedded filesystem and
// applies them to the database. It tracks changes using a schema_migrations_repeatable
// table and only applies files that are new or have changed.
func runRepeatableMigrations(ctx context.Context, conn *pgx.Conn, schemaName string, fs embed.FS, dir string) error {
	err := initRepeatableMigrationsTable(ctx, conn, schemaName)
	if err != nil {
		return fmt.Errorf("failed to initialize repeatable migrations table: %w", err)
	}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read embedded files: %w", err)
	}

	filenames := filterRepeatableSQLFiles(entries)
	sort.Strings(filenames)

	for _, filename := range filenames {
		err := applyRepeatableMigration(ctx, conn, schemaName, fs, dir, filename)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrRepeatableMigrationFailed, filename, err)
		}
	}

	return nil
}

func filterRepeatableSQLFiles(entries []fs.DirEntry) []string {
	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() &&
			filepath.Ext(name) == repeatableSQLExtension &&
			strings.HasPrefix(name, repeatablePrefix) {
			filenames = append(filenames, name)
		}
	}
	return filenames
}

func initRepeatableMigrationsTable(ctx context.Context, conn *pgx.Conn, schemaName string) error {
	query := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s.%s (
            filename VARCHAR(255) NOT NULL PRIMARY KEY,
            hash VARCHAR(64) NOT NULL,
            executed_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
        );`, pgx.Identifier{schemaName}.Sanitize(), repeatableMigrationsTable)

	_, err := conn.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create repeatable migrations table in schema %s: %w", schemaName, err)
	}
	return nil
}

func calculateRepeatableFileHash(content []byte) string {
	hasher := sha256.New()
	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

func applyRepeatableMigration(ctx context.Context, conn *pgx.Conn, schemaName string, fs embed.FS, dir, filename string) error {
	filePath := path.Join(dir, filename)
	content, err := fs.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("%w: failed to read repeatable migration file %s: %v", ErrInvalidRepeatableFile, filePath, err)
	}

	hash := calculateRepeatableFileHash(content)

	var storedHash string
	err = conn.QueryRow(ctx,
		fmt.Sprintf("SELECT hash FROM %s.%s WHERE filename = $1", pgx.Identifier{schemaName}.Sanitize(), repeatableMigrationsTable),
		filename).Scan(&storedHash)

	if err != pgx.ErrNoRows && err != nil {
		return fmt.Errorf("failed to query repeatable migration status for %s in schema %s: %w", filename, schemaName, err)
	}

	// Execute migration if file is new or content has changed
	if err == pgx.ErrNoRows || storedHash != hash {
		return executeRepeatableMigration(ctx, conn, schemaName, filename, string(content), hash)
	}

	// Migration already up to date, skipping
	return nil
}

func executeRepeatableMigration(ctx context.Context, conn *pgx.Conn, schemaName, filename, content, hash string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction for repeatable migration %s: %w", filename, err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Rollback error occurred, but transaction might already be closed
		}
	}()

	// Execute the repeatable migration SQL
	_, err = tx.Exec(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to execute repeatable migration %s: %w", filename, err)
	}

	// Update the migration tracking table
	upsertQuery := fmt.Sprintf(`
        INSERT INTO %s.%s (filename, hash)
        VALUES ($1, $2)
        ON CONFLICT (filename) 
        DO UPDATE SET hash = $2, executed_at = CURRENT_TIMESTAMP`, pgx.Identifier{schemaName}.Sanitize(), repeatableMigrationsTable)

	_, err = tx.Exec(ctx, upsertQuery, filename, hash)
	if err != nil {
		return fmt.Errorf("failed to update repeatable migration tracking record for %s in schema %s: %w", filename, schemaName, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction for repeatable migration %s: %w", filename, err)
	}

	return nil
}
