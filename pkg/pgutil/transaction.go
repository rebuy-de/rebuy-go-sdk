package pgutil

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
)

// Tx executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
//
// This function is designed to work with any SQLC-generated queries struct that has a WithTx method.
//
// Example usage:
//
//	err := pgutil.Tx(ctx, pool, func(tx pgx.Tx) error {
//		qtx := queries.WithTx(tx)
//		// Use qtx for transactional operations
//		return nil
//	})
func Tx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	err = fn(tx)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Hijack acquires a dedicated connection from the pool for exclusive use.
// The returned closer function must be called to release the connection back to the pool.
//
// This is useful for operations that require connection-level state, such as:
//   - Advisory locks
//   - Prepared statements
//   - Connection-specific settings
//
// Example usage:
//
//	conn, closer, err := pgutil.Hijack(ctx, pool)
//	if err != nil {
//		return err
//	}
//	defer closer()
//	queries := sqlc.New(conn)
//	// Use queries with dedicated connection
func Hijack(ctx context.Context, pool *pgxpool.Pool) (*pgx.Conn, func(), error) {
	pconn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("acquire connection from pool: %w", err)
	}

	conn := pconn.Hijack()

	closer := func() {
		err := conn.Close(context.Background())
		if err != nil {
			logutil.Get(ctx).Error("failed to close hijacked connection", "error", err)
		}
	}

	return conn, closer, nil
}
