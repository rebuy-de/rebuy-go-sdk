package main

import (
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
	"github.com/sirupsen/logrus"

	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/minimal/cmd"
)

func main() {
	defer cmdutil.HandleExit()
	if err := cmd.NewRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}
