package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tidwall/pretty"
)

func dumpJSON(data any) error {
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	b = pretty.Color(b, pretty.TerminalStyle)
	fmt.Fprintln(os.Stderr, string(b))

	return nil
}
