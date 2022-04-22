package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
	"github.com/tidwall/pretty"
)

func dumpJSON(data interface{}) {
	b, err := json.MarshalIndent(data, "", "    ")
	cmdutil.Must(err)

	b = pretty.Color(b, pretty.TerminalStyle)
	fmt.Fprintln(os.Stderr, string(b))
}
