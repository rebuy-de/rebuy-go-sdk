package main

import (
	"encoding/json"
	"fmt"

	"github.com/rebuy-de/rebuy-go-sdk/v2/cmdutil"
)

func dumpJSON(data interface{}) {
	b, err := json.MarshalIndent(data, "", "    ")
	cmdutil.Must(err)
	fmt.Println(string(b))
}
