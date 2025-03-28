//go:build tools

package main

import (
	_ "github.com/a-h/templ/cmd/templ"
	_ "github.com/rebuy-de/rebuy-go-sdk/v8/cmd/buildutil"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
