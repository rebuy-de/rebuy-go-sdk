package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
	"github.com/tidwall/pretty"
)

func dumpJSON(data interface{}) {
	b, err := json.MarshalIndent(data, "", "    ")
	cmdutil.Must(err)

	b = pretty.Color(b, pretty.TerminalStyle)
	fmt.Fprintln(os.Stderr, string(b))
}

func byteFormat(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
