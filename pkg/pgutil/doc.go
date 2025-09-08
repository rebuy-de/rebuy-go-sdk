// Package pgutil provides utilities for PostgreSQL database operations with SQLC integration.
//
// This package abstracts common database patterns used across rebuy projects:
//   - Connection management with optional DataDog tracing
//   - Database migration execution with embedded FS
//   - Transaction wrapper utilities
//   - URI construction helpers
//   - Standard SQLC configuration templates
//
// Usage patterns:
//
//	// Basic connection
//	pool, err := pgutil.NewPool(ctx, uri, pgutil.ConnectionOptions{})
//
//	// With tracing and schema
//	pool, err := pgutil.NewPool(ctx, uri, pgutil.ConnectionOptions{
//		EnableTracing: true,
//		SchemaName:   "my_app",
//	})
//
//	// Migration
//	err := pgutil.MigrateWithEmbeddedFS(ctx, uri, "my_app", migrationsFS, "migrations")
package pgutil
