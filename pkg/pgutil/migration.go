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

// Migrate runs database migrations using an embedded filesystem.
//
// Parameters:
//   - ctx: Context for cancellation
//   - uri: PostgreSQL connection URI
//   - schemaName: Name of the schema to create (e.g., "llm_gateway", "knowledge_base_bot")
//   - migrationsFS: Embedded filesystem containing migration files
//   - migrationsDir: Directory path within the embedded FS (e.g., "migrations")
func Migrate(ctx context.Context, uri URI, schema Schema, migrationsFS MigrationFS) error {
	return migrateWithEmbeddedFS(ctx, string(uri), string(schema), embed.FS(migrationsFS), "migrations")
}

func migrateWithEmbeddedFS(ctx context.Context, uri string, schemaName string, migrationsFS embed.FS, migrationsDir string) error {
	config, err := pgx.ParseConfig(uri)
	if err != nil {
		return fmt.Errorf("parse database URI: %w", err)
	}

	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["search_path"] = schemaName

	db := stdlib.OpenDB(*config)
	defer db.Close()

	err = createSchemaIfNotExists(ctx, db, schemaName)
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	// Run normal migrations first
	err = runNormalMigrations(ctx, db, schemaName, migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to run normal migrations: %w", err)
	}

	// Run repeatable migrations after normal migrations
	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create pgx connection for repeatable migrations: %w", err)
	}
	defer conn.Close(ctx)

	err = runRepeatableMigrations(ctx, conn, schemaName, migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to run repeatable migrations: %w", err)
	}

	return nil
}

// runNormalMigrations handles the traditional versioned migrations using golang-migrate
func runNormalMigrations(ctx context.Context, db *sql.DB, schemaName string, migrationsFS embed.FS, migrationsDir string) error {
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

// createSchemaIfNotExists creates the specified schema if it doesn't exist
func createSchemaIfNotExists(ctx context.Context, db *sql.DB, schemaName string) error {
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schemaName)
	_, err := db.ExecContext(ctx, query)
	return err
}
