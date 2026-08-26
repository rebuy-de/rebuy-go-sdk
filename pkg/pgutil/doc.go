// Package pgutil provides utilities for PostgreSQL database operations with SQLC integration.
//
// The pgutil package consolidates common database patterns used across rebuy projects,
// providing unified connection management, migration framework, transaction wrappers,
// and URI construction helpers.
//
// # Features
//
//   - Connection Management: Unified connection creation with optional DataDog tracing
//   - Migration Framework: Generic migration execution with embedded filesystems (both normal and repeatable)
//   - Transaction Wrappers: Reusable transaction and connection hijacking utilities
//   - URI Construction: Helper functions for database URI manipulation
//
// # Quick Start
//
// Basic usage example:
//
//	package main
//
//	import (
//	    "context"
//
//	    "github.com/rebuy-de/rebuy-go-sdk/v10/pkg/digutil"
//	    "github.com/rebuy-de/rebuy-go-sdk/v10/pkg/pgutil"
//	    "github.com/myorg/myapp/pkg/dal/sqlc"
//	)
//
//	func main() {
//	    ctx := context.Background()
//	    uri := pgutil.URI("postgres://user:pass@localhost:5432/mydb")
//	    schema := pgutil.Schema("my_app")
//
//	    // Run migrations (both versioned and repeatable)
//	    err := pgutil.Migrate(ctx, uri, schema, pgutil.MigrationFS(sqlc.MigrationsFS))
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    pool, err := pgutil.NewPool(ctx, uri, schema, digutil.Optional[pgutil.EnableTracing]{})
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    queries := sqlc.New(pool) // sqlc.New is your SQLC-generated constructor
//
//	    // Use queries...
//	}
//
// In an application the wiring is done through dig instead, see below.
//
// # Dependency Injection with Dig
//
// Integration with uber-go/dig for dependency injection:
//
//	err := errors.Join(
//	    // Database configuration
//	    digutil.ProvideValue[pgutil.URI](c, "postgres://user:pass@localhost/mydb"),
//	    digutil.ProvideValue[pgutil.EnableTracing](c, true),
//	    digutil.ProvideValue[pgutil.Schema](c, "my_app"),
//	    digutil.ProvideValue[pgutil.MigrationFS](c, pgutil.MigrationFS(sqlc.MigrationsFS)),
//
//	    // Providers
//	    c.Provide(pgutil.NewPool, dig.As(new(sqlc.DBTX))),
//	    c.Provide(sqlc.New),
//
//	    // Run migrations
//	    c.Invoke(pgutil.Migrate),
//	)
//
// # Transaction Support
//
// Execute database operations within transactions:
//
//	// Execute in transaction using the simple Tx function
//	err := pgutil.Tx(ctx, pool, func(tx pgx.Tx) error {
//	    // Create queries instance with transaction
//	    qtx := queries.WithTx(tx)
//
//	    // All operations within this function are transactional
//	    user, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{...})
//	    if err != nil {
//	        return err
//	    }
//
//	    _, err = qtx.CreatePost(ctx, sqlc.CreatePostParams{
//	        UserID: user.ID,
//	        // ...
//	    })
//	    return err
//	})
//
// # Connection Hijacking
//
// For advanced use cases requiring dedicated connections (advisory locks, prepared statements):
//
//	// Get dedicated connection for advisory locks, prepared statements, etc.
//	conn, closer, err := pgutil.Hijack(ctx, pool)
//	if err != nil {
//	    return err
//	}
//	defer closer()
//
//	// Create queries instance with dedicated connection
//	dedicatedQueries := sqlc.New(conn)
//	// Use dedicatedQueries with exclusive connection
//
// # Fully-qualified table names
//
// Every table, view and function reference must carry the application schema, both
// in SQLC queries and in migration files:
//
//	select * from my_app.orders where id = $1;
//
// NewPool pins search_path to the Schema value, but that is only there so
// golang-migrate finds its own bookkeeping table. Queries must never depend on
// it: an unqualified name means a different table depending on which connection
// runs it, and it cannot be pasted into psql, Grafana or a migration unchanged.
//
// # SQLC Configuration
//
// The canonical sqlc.yaml lives in examples/full/pkg/dal/sqlc/sqlc.yaml. It sets up:
//   - PostgreSQL with pgx/v5 driver
//   - JSON tags with camelCase style
//   - Proper UUID and timestamp handling
//   - Null-safe type generation
//
// # Repeatable Migrations
//
// The pgutil package supports repeatable migrations alongside traditional versioned migrations.
// Repeatable migrations are ideal for views, functions, procedures, and reference data that may
// need to be recreated or updated when the underlying logic changes.
//
// ## File Naming Convention
//
// Repeatable migration files must follow the naming pattern: R_<description>.sql
// Examples:
//   - R_001_user_view.sql
//   - R_002_lookup_data.sql
//   - R_003_reporting_functions.sql
//
// ## Migration Process
//
// 1. Normal migrations run first (using golang-migrate/migrate library)
// 2. Repeatable migrations run after normal migrations complete successfully
// 3. Each repeatable migration is tracked by filename and SHA256 hash in schema_migrations_repeatable table
// 4. Files are only re-executed if their content has changed (detected via hash comparison)
// 5. All repeatable migrations execute within individual transactions for safety
//
// ## Example Directory Structure
//
//	migrations/
//	├── 000001_initial_schema.up.sql    # Normal migration
//	├── 000001_initial_schema.down.sql  # Normal migration
//	├── 000002_add_users_table.up.sql   # Normal migration
//	├── 000002_add_users_table.down.sql # Normal migration
//	├── R_001_user_stats_view.sql       # Repeatable migration
//	└── R_002_seed_lookup_data.sql      # Repeatable migration
//
// ## Example Repeatable Migration Content
//
//	-- R_001_user_stats_view.sql
//	CREATE OR REPLACE VIEW my_app.user_stats AS
//	SELECT
//	    user_id,
//	    COUNT(*) as total_orders,
//	    SUM(amount) as total_spent
//	FROM my_app.orders
//	GROUP BY user_id;
//
// ## Usage with Migrate
//
// No changes to your existing code are needed. Migrate automatically handles both types:
//
//	err := pgutil.Migrate(ctx, uri, schema, pgutil.MigrationFS(sqlc.MigrationsFS))
//	// This will run versioned migrations first, then repeatable migrations
//
// For a complete working example, see the examples/full directory which demonstrates
// all pgutil features in a real application.
package pgutil
