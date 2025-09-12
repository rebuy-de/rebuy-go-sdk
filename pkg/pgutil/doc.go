// Package pgutil provides utilities for PostgreSQL database operations with SQLC integration.
//
// The pgutil package consolidates common database patterns used across rebuy projects,
// providing unified connection management, migration framework, transaction wrappers,
// URI construction helpers, and standard SQLC configuration templates.
//
// # Features
//
//   - Connection Management: Unified connection creation with optional DataDog tracing
//   - Migration Framework: Generic migration execution with embedded filesystems (both normal and repeatable)
//   - Transaction Wrappers: Reusable transaction and connection hijacking utilities
//   - URI Construction: Helper functions for database URI manipulation
//   - SQLC Templates: Standard configuration templates for consistent project setup
//
// # Quick Start
//
// Basic usage example:
//
//	package main
//
//	import (
//	    "context"
//	    "embed"
//
//	    "github.com/rebuy-de/rebuy-go-sdk/v9/pkg/pgutil"
//	)
//
//	//go:embed migrations/*.sql
//	var migrationsFS embed.FS
//
//	func main() {
//	    ctx := context.Background()
//	    uri := "postgres://user:pass@localhost:5432/mydb"
//
//	    // Run migrations (both normal and repeatable)
//	    err := pgutil.MigrateWithEmbeddedFS(ctx, uri, "my_app", migrationsFS, "migrations")
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    // Create connection with tracing
//	    queries, err := pgutil.NewQueriesInterface(ctx, uri, pgutil.ConnectionOptions{
//	        EnableTracing: true,
//	        SchemaName:   "my_app",
//	    }, sqlc.New) // sqlc.New is your SQLC-generated constructor
//
//	    if err != nil {
//	        panic(err)
//	    }
//
//	    // Use queries...
//	}
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
// # Configuration Templates
//
// The package provides standard SQLC configuration templates. Copy the template to your project:
//
//	cp pkg/pgutil/templates/sqlc.yaml pkg/dal/sqlc/sqlc.yaml
//
// The template includes:
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
//	CREATE OR REPLACE VIEW user_stats AS
//	SELECT 
//	    user_id,
//	    COUNT(*) as total_orders,
//	    SUM(amount) as total_spent
//	FROM orders 
//	GROUP BY user_id;
//
// ## Usage with MigrateWithEmbeddedFS
//
// No changes to your existing code are needed. The function automatically handles both types:
//
//	//go:embed migrations/*.sql
//	var migrationsFS embed.FS
//
//	err := pgutil.MigrateWithEmbeddedFS(ctx, uri, "my_app", migrationsFS, "migrations")
//	// This will run normal migrations first, then repeatable migrations
//
// For a complete working example, see the examples/full directory which demonstrates
// all pgutil features in a real application.
package pgutil
