package chutil

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrRepeatableMigrationFailed = errors.New("repeatable migration failed")
	ErrInvalidRepeatableFile     = errors.New("invalid repeatable file")
)

const (
	repeatableSQLExtension    = ".sql"
	repeatablePrefix          = "R_"
	repeatableMigrationsTable = "schema_migrations_repeatable"
)

// runRepeatableMigrations processes all SQL files with prefix 'R_' in the
// embedded filesystem and applies them to ClickHouse. It tracks applied files in
// a schema_migrations_repeatable table and only re-applies files that are new or
// whose content hash has changed.
//
// Unlike the Postgres counterpart in pgutil, each R_*.sql file must contain a
// single statement: ClickHouse has no multi-statement transactions and the std
// driver's ExecContext runs one statement per call. A single
// `CREATE OR REPLACE VIEW`, which ClickHouse supports, is the common case.
func runRepeatableMigrations(ctx context.Context, db *sql.DB, database string, fsys embed.FS) error {
	err := initRepeatableMigrationsTable(ctx, db, database)
	if err != nil {
		return fmt.Errorf("failed to initialize repeatable migrations table: %w", err)
	}

	entries, err := fsys.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read embedded files: %w", err)
	}

	filenames := filterRepeatableSQLFiles(entries)
	sort.Strings(filenames)

	for _, filename := range filenames {
		err := applyRepeatableMigration(ctx, db, database, fsys, filename)
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

// initRepeatableMigrationsTable creates the tracking table. ClickHouse has no
// primary-key uniqueness, so a ReplacingMergeTree ordered by filename collapses
// to the newest row per file (highest executed_at), which is read back with
// FINAL in applyRepeatableMigration.
func initRepeatableMigrationsTable(ctx context.Context, db *sql.DB, database string) error {
	query := fmt.Sprintf(`
        CREATE TABLE IF NOT EXISTS %s.%s (
            filename    String,
            hash        String,
            executed_at DateTime DEFAULT now()
        ) ENGINE = ReplacingMergeTree(executed_at)
        ORDER BY filename`, quoteIdentifier(database), repeatableMigrationsTable)

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create repeatable migrations table in database %s: %w", database, err)
	}
	return nil
}

// repeatableNeedsApply reports whether a repeatable file must run: either it was
// never applied (found is false) or its content hash has changed since the last
// run.
func repeatableNeedsApply(storedHash string, found bool, newHash string) bool {
	return !found || storedHash != newHash
}

func calculateRepeatableFileHash(content []byte) string {
	hasher := sha256.New()
	hasher.Write(content)
	return hex.EncodeToString(hasher.Sum(nil))
}

func applyRepeatableMigration(ctx context.Context, db *sql.DB, database string, fsys embed.FS, filename string) error {
	filePath := path.Join(migrationsDir, filename)
	content, err := fsys.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("%w: failed to read repeatable migration file %s: %v", ErrInvalidRepeatableFile, filePath, err)
	}

	hash := calculateRepeatableFileHash(content)

	// FINAL forces ReplacingMergeTree to resolve to the newest row for the file,
	// so we compare against the last-applied hash even before a background merge.
	var storedHash string
	query := fmt.Sprintf("SELECT hash FROM %s.%s FINAL WHERE filename = ?",
		quoteIdentifier(database), repeatableMigrationsTable)
	err = db.QueryRowContext(ctx, query, filename).Scan(&storedHash)

	found := true
	switch {
	case errors.Is(err, sql.ErrNoRows):
		found = false
	case err != nil:
		return fmt.Errorf("failed to query repeatable migration status for %s in database %s: %w", filename, database, err)
	}

	if !repeatableNeedsApply(storedHash, found, hash) {
		// Already applied and unchanged; skip.
		return nil
	}

	return executeRepeatableMigration(ctx, db, database, filename, string(content), hash)
}

// executeRepeatableMigration runs the migration body and records its hash. There
// is no surrounding transaction (ClickHouse does not support DDL transactions),
// and the tracking insert relies on the ReplacingMergeTree to supersede any
// previous hash for the same filename rather than an upsert.
func executeRepeatableMigration(ctx context.Context, db *sql.DB, database, filename, content, hash string) error {
	_, err := db.ExecContext(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to execute repeatable migration %s: %w", filename, err)
	}

	insert := fmt.Sprintf("INSERT INTO %s.%s (filename, hash) VALUES (?, ?)",
		quoteIdentifier(database), repeatableMigrationsTable)
	_, err = db.ExecContext(ctx, insert, filename, hash)
	if err != nil {
		return fmt.Errorf("failed to update repeatable migration tracking record for %s in database %s: %w", filename, database, err)
	}

	return nil
}
