// Package pgutil provides utilities for PostgreSQL database operations with SQLC integration.
//
// The pgutil package consolidates common database patterns used across rebuy projects,
// providing unified connection management, migration framework, transaction wrappers,
// URI construction helpers, and standard SQLC configuration templates.
//
// # Features
//
//   - Connection Management: Unified connection creation with optional DataDog tracing
//   - Migration Framework: Generic migration execution with embedded filesystems
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
//	    // Run migrations
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
// For a complete working example, see the examples/full directory which demonstrates
// all pgutil features in a real application.
package pgutil
