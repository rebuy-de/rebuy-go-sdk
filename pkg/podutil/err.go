package podutil

import (
	"encoding/json"
	"fmt"
	"io"
)

type Error struct {
	Cause    string `json:"cause"`
	Message  string `json:"message"`
	Response int    `json:"response"`
}

func decodeError(r io.Reader) error {
	var typed Error

	err := json.NewDecoder(r).Decode(&typed)
	if err != nil {
		return fmt.Errorf("%d", err)
	}

	return &typed
}

func (err Error) Error() string {
	return fmt.Sprintf("(%d) %s", err.Response, err.Message)
}
