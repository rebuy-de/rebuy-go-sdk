package riverutil

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rebuy-de/rebuy-go-sdk/v10/pkg/pgutil"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func Migrate(ctx context.Context, pool *pgxpool.Pool, schema pgutil.Schema) error {
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{
		Schema: string(schema),
	})
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}
