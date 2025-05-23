package webutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"
)

// WrapView converts a function that returns a Response into an http.HandlerFunc.
// This allows for a more functional approach to HTTP handling where the handler
// logic can be separated from the actual response writing.
func WrapView(fn func(*http.Request) Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(r)(w, r)
	}
}

// ViewError returns an http.HandlerFunc that writes an error response with the given HTTP status code.
// It also logs the error with appropriate severity based on the status code.
// If the error is a context.Canceled error, it changes the status code to 499 (client closed connection).
func ViewError(status int, err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stacktrace := fmt.Sprintf("%+v", err)
		wrappedErr := errors.WithStack(err)

		if errors.Is(err, context.Canceled) {
			slog.Debug("request cancelled", "error", err, "stacktrace", stacktrace)

			// The code is copied from nginx, where it means that the client
			// closed the connection. It is necessary to alter the status code,
			// because DataDog will report errors, if the code is >=500,
			// regardless of the connection state.
			status = 499
		} else if status >= 500 {
			slog.Error("request failed", "error", wrappedErr, "stacktrace", stacktrace)
		} else {
			slog.Warn("request failed", "error", wrappedErr, "stacktrace", stacktrace)
		}

		w.WriteHeader(status)
		fmt.Fprint(w, err.Error())
	}
}

// ViewErrorf returns an http.HandlerFunc that writes a formatted error response with the given HTTP status code.
// It uses fmt.Errorf to format the error message with the provided format string and arguments.
func ViewErrorf(status int, format string, args ...any) http.HandlerFunc {
	err := fmt.Errorf(format, args...)
	return ViewError(status, err)
}

// ViewRedirectf returns an http.HandlerFunc that redirects to the formatted location string.
// It uses fmt.Sprintf to format the location with the provided arguments.
func ViewRedirectf(status int, location string, args ...any) http.HandlerFunc {
	return ViewRedirect(status, fmt.Sprintf(location, args...))
}

// ViewRedirect returns an http.HandlerFunc that performs an HTTP redirect to the specified location.
// The status parameter should be an appropriate HTTP redirection status code (e.g., 301, 302, 303).
func ViewRedirect(status int, location string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, location, status)
	}
}

// ViewJSON returns an http.HandlerFunc that writes the provided data as indented JSON.
// It sets the Content-Type header to application/json and the provided status code.
// If encoding fails, it responds with an internal server error.
func ViewJSON(status int, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "    ")

		err := enc.Encode(data)
		if err != nil {
			ViewError(http.StatusInternalServerError, err)(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		buf.WriteTo(w)
	}
}

// ViewInlineHTML returns an http.HandlerFunc that writes the provided HTML string with formatting.
// It sets the Content-Type header to text/html and the specified status code.
// The format string and arguments are passed to fmt.Fprintf for rendering.
func ViewInlineHTML(status int, data string, a ...any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		fmt.Fprintf(w, data, a...)
	}
}

// ViewNoContent returns an http.HandlerFunc that writes only a status code with no response body.
// This is useful for endpoints that need to return a success or error status without content.
func ViewNoContent(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}
}
