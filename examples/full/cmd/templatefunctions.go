package cmd

import (
	"time"

	"github.com/pkg/errors"
)

func PrettyTimeTemplateFunction(value interface{}) (string, error) {
	tPtr, ok := value.(*time.Time)
	if ok {
		if tPtr == nil {
			return "N/A", nil
		}
		value = *tPtr
	}

	t, ok := value.(time.Time)
	if !ok {
		return "", errors.Errorf("unexpected type %T", value)
	}

	if t.IsZero() {
		return "N/A", nil
	}

	format := "Mon, 2 Jan 15:04:05"

	t = t.Local()

	return t.Format(format), nil
}
