//go:build tools

package main

import (
	_ "github.com/a-h/templ/cmd/templ"
	_ "github.com/rebuy-de/rebuy-go-sdk/v9/cmd/buildutil"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
