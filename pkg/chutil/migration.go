package chutil

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	chgo "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/golang-migrate/migrate/v4"
	chmigrate "github.com/golang-migrate/migrate/v4/database/clickhouse"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// migrationsDir is the top-level directory inside a MigrationFS that holds both
// versioned (`*.up.sql`) and repeatable (`R_*.sql`) migration files.
const migrationsDir = "migrations"

// MigrationFS is an embedded filesystem containing migration files, for
// dependency injection. It is the ClickHouse counterpart of pgutil.MigrationFS.
type MigrationFS embed.FS

// Migrate applies database migrations to ClickHouse. It is the counterpart of
// pgutil.Migrate, adapted to ClickHouse semantics: it creates the target
// database (ClickHouse's namespace, analogous to a Postgres schema) if missing,
// runs versioned migrations via golang-migrate, then applies repeatable
// (R_*.sql) migrations. The database name is taken from auth.Database.
//
// Migrations run over a database/sql connection (clickhouse-go's std driver),
// not the native batcher connection, because golang-migrate needs a *sql.DB.
func Migrate(ctx context.Context, addr Addr, auth Auth, migrationsFS MigrationFS) error {
	// The target database may not exist yet, so bootstrap on the default
	// database to create it before opening a connection scoped to it.
	bootstrap := openMigrationDB(addr, auth, "")
	defer bootstrap.Close()

	if err := bootstrap.PingContext(ctx); err != nil {
		slog.WarnContext(ctx, "skipping clickhouse migrations: connection unavailable", "error", err)
		return nil
	}

	err := createDatabaseIfNotExists(ctx, bootstrap, auth.Database)
	if err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	db := openMigrationDB(addr, auth, auth.Database)
	defer db.Close()

	err = runNormalMigrations(db, auth.Database, embed.FS(migrationsFS))
	if err != nil {
		return fmt.Errorf("failed to run normal migrations: %w", err)
	}

	err = runRepeatableMigrations(ctx, db, auth.Database, embed.FS(migrationsFS))
	if err != nil {
		return fmt.Errorf("failed to run repeatable migrations: %w", err)
	}

	return nil
}

// openMigrationDB opens a database/sql handle scoped to the given database. An
// empty database connects to the server's default, which is needed to create
// the target database before it exists.
func openMigrationDB(addr Addr, auth Auth, database string) *sql.DB {
	return chgo.OpenDB(&chgo.Options{
		Addr: []string{string(addr)},
		Auth: chgo.Auth{
			Database: database,
			Username: auth.Username,
			Password: auth.Password,
		},
		Compression: &chgo.Compression{Method: chgo.CompressionLZ4},
	})
}

// createDatabaseIfNotExists creates the target database if it does not exist.
func createDatabaseIfNotExists(ctx context.Context, db *sql.DB, database string) error {
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", quoteIdentifier(database))
	_, err := db.ExecContext(ctx, query)
	return err
}

// runNormalMigrations applies the versioned migrations using golang-migrate's
// clickhouse driver.
func runNormalMigrations(db *sql.DB, database string, migrationsFS embed.FS) error {
	sourceDriver, err := iofs.New(migrationsFS, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load migration source driver: %w", err)
	}

	databaseDriver, err := chmigrate.WithInstance(db, &chmigrate.Config{
		DatabaseName:    database,
		MigrationsTable: "schema_migrations",
		// MergeTree replicates well; the driver appends an ORDER BY for any
		// "*Tree" engine, so the migration-tracking table is created correctly.
		MigrationsTableEngine: "MergeTree",
		// Allow more than one statement per *.up.sql file, matching the
		// multi-statement behaviour the Postgres driver provides by default.
		MultiStatementEnabled: true,
	})
	if err != nil {
		return fmt.Errorf("failed to load database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "clickhouse", databaseDriver)
	if err != nil {
		return fmt.Errorf("failed to setup migration: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// quoteIdentifier wraps a ClickHouse identifier in backticks, escaping embedded
// backticks, so a database or table name cannot break out of the quoting.
// ClickHouse has no pgx.Identifier equivalent.
func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}
