package webutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
type ViewHandler struct {
	FS       fs.FS
	FuncMaps []TemplateFuncMap
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func NewViewHandler(fs fs.FS, fms ...TemplateFuncMap) *ViewHandler {
	v := &ViewHandler{
		FS:       fs,
		FuncMaps: fms,
	}

	return v
}

type ResponseHandlerFunc func(*View, *http.Request) Response

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (h *ViewHandler) Wrap(fn ResponseHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(&View{handler: h}, r)(w, r)
	}
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (h *ViewHandler) Render(filename string, r *http.Request, d interface{}) (*bytes.Buffer, error) {
	t := template.New(filename)

	for _, fm := range h.FuncMaps {
		t = t.Funcs(fm(r))
	}

	t, err := t.ParseFS(h.FS, "*")
	if err != nil {
		return nil, errors.Wrap(err, "parsing template failed")
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, d)

	return buf, errors.Wrap(err, "executing template failed")
}

type TemplateFuncMap func(*http.Request) template.FuncMap

func SimpleTemplateFuncMap(name string, fn interface{}) TemplateFuncMap {
	return func(_ *http.Request) template.FuncMap {
		return template.FuncMap{
			name: fn,
		}
	}
}

func SimpleTemplateFuncMaps(fm template.FuncMap) TemplateFuncMap {
	return func(_ *http.Request) template.FuncMap {
		return fm
	}
}

type Response = http.HandlerFunc

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
type View struct {
	handler *ViewHandler
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (v *View) Error(status int, err error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logrus.
			WithField("stacktrace", fmt.Sprintf("%+v", err)).
			WithError(errors.WithStack(err))

		if status >= 500 {
			l.Errorf("request failed: %s", err)
		} else {
			l.Warnf("request failed: %s", err)
		}

		w.WriteHeader(status)
		fmt.Fprint(w, err.Error())
	}
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (v *View) Errorf(status int, text string, a ...interface{}) http.HandlerFunc {
	return v.Error(status, fmt.Errorf(text, a...))
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (v *View) Redirect(status int, location string, args ...interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := fmt.Sprintf(location, args...)
		http.Redirect(w, r, url, status)
	}
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (v *View) JSON(status int, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "    ")

		err := enc.Encode(data)
		if err != nil {
			v.Error(http.StatusInternalServerError, err)(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		buf.WriteTo(w)
	}
}

// Deprecated: M0003 Use GoTemplateViewer and View* functions instead.
func (v *View) HTML(status int, filename string, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, err := v.handler.Render(filename, r, data)
		if err != nil {
			v.Error(http.StatusInternalServerError, err)(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
		buf.WriteTo(w)
	}
}
