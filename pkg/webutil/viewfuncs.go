package webutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func WrapView(fn func(*http.Request) Response) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(r)(w, r)
	}
}

func ViewError(status int, err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logrus.
			WithField("stacktrace", fmt.Sprintf("%+v", err)).
			WithError(errors.WithStack(err))

		if errors.Is(err, context.Canceled) {
			l.Debugf("request cancelled: %s", err)

			// The code is copied from nginx, where it means that the client
			// closed the connection. It is necessary to alter the status code,
			// because DataDog will report errors, if the code is >=500,
			// regardless of the connection state.
			status = 499
		} else if status >= 500 {
			l.Errorf("request failed: %s", err)
		} else {
			l.Warnf("request failed: %s", err)
		}

		w.WriteHeader(status)
		fmt.Fprint(w, err.Error())
	}
}

func ViewErrorf(status int, text string, a ...interface{}) http.HandlerFunc {
	return ViewError(status, fmt.Errorf(text, a...))
}

func ViewRedirect(status int, location string, args ...interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf(location, args...)
		http.Redirect(w, r, url, status)
	}
}

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

func ViewInlineHTML(status int, data string, a ...any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		fmt.Fprintf(w, data, a...)
	}
}

func ViewNoContent(status int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
	}
}
