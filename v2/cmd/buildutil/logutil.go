package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
	"github.com/tidwall/pretty"
	"golang.org/x/crypto/ssh/terminal"
)

func dumpJSON(data interface{}) {
	b, err := json.MarshalIndent(data, "", "    ")
	cmdutil.Must(err)

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		b = pretty.Color(b, pretty.TerminalStyle)
	}

	fmt.Fprintln(os.Stderr, string(b))
}
