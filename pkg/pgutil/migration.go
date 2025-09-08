package pgutil

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// MigrateWithEmbeddedFS runs database migrations using an embedded filesystem.
// This is a generic replacement for the migration code duplicated across all projects.
//
// Parameters:
//   - ctx: Context for cancellation
//   - uri: PostgreSQL connection URI
//   - schemaName: Name of the schema to create (e.g., "llm_gateway", "knowledge_base_bot")
//   - migrationsFS: Embedded filesystem containing migration files
//   - migrationsDir: Directory path within the embedded FS (e.g., "migrations")
//
// Example usage:
//
//	//go:embed migrations/*.sql
//	var migrationsFS embed.FS
//
//	err := sqlutil.MigrateWithEmbeddedFS(ctx, uri, "my_app", migrationsFS, "migrations")
func MigrateWithEmbeddedFS(ctx context.Context, uri string, schemaName string, migrationsFS embed.FS, migrationsDir string) error {
	config, err := pgx.ParseConfig(uri)
	if err != nil {
		return fmt.Errorf("parse database URI: %w", err)
	}

	db := stdlib.OpenDB(*config)
	defer db.Close()

	err = createSchemaIfNotExists(ctx, db, schemaName)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load migration source driver: %w", err)
	}

	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{
		SchemaName: schemaName,
	})
	if err != nil {
		return fmt.Errorf("failed to load database driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		"postgres", databaseDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to setup migration: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Migrate runs database migrations using dependency injection types
func Migrate(ctx context.Context, uri URI, schema Schema, migrationsFS MigrationFS) error {
	return MigrateWithEmbeddedFS(ctx, string(uri), string(schema), embed.FS(migrationsFS), "migrations")
}

// createSchemaIfNotExists creates the specified schema if it doesn't exist
func createSchemaIfNotExists(ctx context.Context, db *sql.DB, schemaName string) error {
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schemaName)
	_, err := db.ExecContext(ctx, query)
	return err
}
