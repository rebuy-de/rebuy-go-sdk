package pgutil

import (
	"context"
	"embed"
	"fmt"

	pgxtrace "github.com/DataDog/dd-trace-go/contrib/jackc/pgx.v5/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/digutil"
)

// URI represents a PostgreSQL connection string for dependency injection
type URI string

// Schema represents a database schema name for dependency injection
type Schema string

// MigrationFS represents an embedded filesystem containing migration files for dependency injection
type MigrationFS embed.FS

// EnableTracing represents whether to enable DataDog tracing for dependency injection
type EnableTracing bool

// NewPool creates a new PostgreSQL connection pool using typed parameters
func NewPool(ctx context.Context, uri URI, schema Schema, enableTracing digutil.Optional[EnableTracing]) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(string(uri))
	if err != nil {
		return nil, fmt.Errorf("parse database URI: %w", err)
	}

	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["search_path"] = string(schema)

	if enableTracing.Value != nil && *enableTracing.Value {
		pool, err := pgxtrace.NewPoolWithConfig(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("connect to database with tracing: %w", err)
		}
		return pool, nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return pool, nil
}
