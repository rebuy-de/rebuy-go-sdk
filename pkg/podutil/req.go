package podutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/go-querystring/query"
)

type RequestOption func(*http.Request) error

func RequestMethod(m string) RequestOption {
	return func(r *http.Request) error {
		r.Method = m
		return nil
	}
}

func RequestJSONBody(data any) RequestOption {
	return func(r *http.Request) error {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(data)
		if err != nil {
			return fmt.Errorf("encode json body: %w", err)
		}

		r.Header.Set("Content-Type", "application/json")
		r.Body = io.NopCloser(buf)

		return nil
	}
}

func RequestQueryStruct(data any) RequestOption {
	return func(r *http.Request) error {
		v, err := query.Values(data)
		if err != nil {
			return fmt.Errorf("pull image: %w", err)
		}

		r.URL.RawQuery = v.Encode()

		return nil
	}
}

func RequestPath(path string, a ...any) RequestOption {
	return func(r *http.Request) error {
		r.URL.Path = r.URL.Path + fmt.Sprintf(path, a...)
		return nil
	}
}
