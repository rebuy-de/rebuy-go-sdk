# pgutil Package

The `pgutil` package provides utilities for PostgreSQL database operations with SQLC integration, consolidating common database patterns used across rebuy projects.

## Features

- **Connection Management**: Unified connection creation with optional DataDog tracing
- **Migration Framework**: Generic migration execution with embedded filesystems  
- **Transaction Wrappers**: Reusable transaction and connection hijacking utilities
- **URI Construction**: Helper functions for database URI manipulation
- **SQLC Templates**: Standard configuration templates for consistent project setup

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "embed"
    
    "github.com/rebuy-de/rebuy-go-sdk/v9/pkg/pgutil"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
    ctx := context.Background()
    uri := "postgres://user:pass@localhost:5432/mydb"
    
    // Run migrations
    err := pgutil.MigrateWithEmbeddedFS(ctx, uri, "my_app", migrationsFS, "migrations")
    if err != nil {
        panic(err)
    }
    
    // Create connection with tracing
    queries, err := pgutil.NewQueriesInterface(ctx, uri, pgutil.ConnectionOptions{
        EnableTracing: true,
        SchemaName:   "my_app",
    }, sqlc.New) // sqlc.New is your SQLC-generated constructor
    
    if err != nil {
        panic(err)
    }
    
    // Use queries...
}
```

### With Transactions

```go
// Execute in transaction using the simple Tx function
err := pgutil.Tx(ctx, pool, func(tx pgx.Tx) error {
    // Create queries instance with transaction
    qtx := queries.WithTx(tx)
    
    // All operations within this function are transactional
    user, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{...})
    if err != nil {
        return err
    }
    
    _, err = qtx.CreatePost(ctx, sqlc.CreatePostParams{
        UserID: user.ID,
        // ...
    })
    return err
})
```

### Connection Hijacking (Advanced)

```go
// Get dedicated connection for advisory locks, prepared statements, etc.
conn, closer, err := pgutil.Hijack(ctx, pool)
if err != nil {
    return err
}
defer closer()

// Create queries instance with dedicated connection
dedicatedQueries := sqlc.New(conn)
// Use dedicatedQueries with exclusive connection
```

## Migration from Existing Projects

### Before (duplicated across projects)

```go
// pkg/dal/sqlc/sqlc.go - repeated in every project
func NewQueries(ctx context.Context, uri string) (*Queries, error) {
    config, err := pgxpool.ParseConfig(uri) 
    if err != nil {
        return nil, fmt.Errorf("parse uri: %w", err)
    }
    
    db, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("connect: %w", err) 
    }
    
    return New(db), nil
}

func Migrate(ctx context.Context, uri string) error {
    // 50+ lines of identical migration boilerplate...
}

func URI(base, username, password string) (string, error) {
    // Identical URI construction...
}
```

### After (using pgutil)

```go
// pkg/dal/sqlc/sqlc.go - simplified with pgutil
func NewQueries(ctx context.Context, uri string) (*Queries, error) {
    return pgutil.NewQueriesInterface(ctx, uri, pgutil.ConnectionOptions{
        SchemaName: "my_app",
    }, New)
}

func Migrate(ctx context.Context, uri string) error {
    return pgutil.MigrateWithEmbeddedFS(ctx, uri, "my_app", migrationsFS, "migrations")
}

func URI(base, username, password string) (string, error) {
    return pgutil.BuildURI(base, username, password)
}
```

## Configuration Templates

Copy the standard SQLC configuration:

```bash
cp pkg/pgutil/templates/sqlc.yaml pkg/dal/sqlc/sqlc.yaml
```

The template includes:
- PostgreSQL with pgx/v5 driver
- JSON tags with camelCase style  
- Proper UUID and timestamp handling
- Null-safe type generation

## Benefits

- **Code Reduction**: ~200+ lines removed per project
- **Consistency**: Standardized SQL patterns across all rebuy projects  
- **Maintenance**: Single location for SQL best practices and bug fixes
- **Features**: Transaction support for all projects (previously missing in some)
- **Flexibility**: Optional tracing, configurable connection patterns

## Examples

See [examples/full](../../examples/full) for a complete working example demonstrating all pgutil features in a real application.
