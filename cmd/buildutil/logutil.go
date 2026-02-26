package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tidwall/pretty"
	"golang.org/x/term"
)

func dumpJSON(data any) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	if term.IsTerminal(int(os.Stderr.Fd())) {
		b = pretty.Color(b, pretty.TerminalStyle)
	}
	fmt.Fprintln(os.Stderr, string(b))

	return nil
}
