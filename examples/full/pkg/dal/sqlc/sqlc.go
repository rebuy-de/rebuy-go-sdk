package sqlc

import (
	"embed"
)

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc generate

//go:embed migrations/*.sql
var MigrationsFS embed.FS
