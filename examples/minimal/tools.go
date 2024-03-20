//go:build tools
// +build tools

package main

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
import (
	_ "github.com/rebuy-de/rebuy-go-sdk/v8/cmd/buildutil"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
