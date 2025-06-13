package webutil

import (
	"bytes"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
)

type GoTemplateViewer struct {
	fs       fs.FS
	funcMaps []TemplateFuncMap
}

func NewGoTemplateViewer(fs fs.FS, fms ...TemplateFuncMap) *GoTemplateViewer {
	return &GoTemplateViewer{
		fs:       fs,
		funcMaps: fms,
	}
}

func (v *GoTemplateViewer) prepare(filename string, r *http.Request) (*template.Template, error) {
	t := template.New(filename)

	for _, fm := range v.funcMaps {
		t = t.Funcs(fm(r))
	}

	return t.ParseFS(v.fs, "*")
}

func (v *GoTemplateViewer) HTML(status int, filename string, data any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		t, err := v.prepare(filename, r)
		if err != nil {
			ViewError(http.StatusInternalServerError, err)(w, r)
			return
		}

		w.WriteHeader(status)

		err = t.Execute(w, data)
		if err != nil {
			// It is possible that we already sent the header, but we try again anyways.
			w.WriteHeader(http.StatusInternalServerError)

			// We do not send the actual error to the client, since we don't know what we already sent.
			slog.Error("failed to render", "error", err)
		}
	}
}

func (v *GoTemplateViewer) Render(filename string, r *http.Request, data any) (*bytes.Buffer, error) {
	t, err := v.prepare(filename, r)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, data)
	if err != nil {
		return nil, err
	}

	return buf, nil
}
