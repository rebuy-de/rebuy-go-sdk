package webutil

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/pkg/errors"
)

type GoTemplateViewer struct {
	fs       fs.FS
	funcMaps []TemplateFuncMap
}

func NewGoTemplateViewer(fs fs.FS, fms ...TemplateFuncMap) *GoTemplateViewer {
	return &GoTemplateViewer{
		FS:       fs,
		FuncMaps: fms,
	}
}

func (v *GoTemplateViewer) HTML(status int, filename string, data any) http.HandlerFunc {
}

func (v *GoTemplateViewer) Render(filename string, r *http.Request, data any) (*bytes.Buffer, error) {
	t := template.New(filename)

	for _, fm := range h.FuncMaps {
		t = t.Funcs(fm(r))
	}

	t, err := t.ParseFS(h.FS, "*")
	if err != nil {
		return nil, errors.Wrap(err, "parsing template failed")
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, data)

	return buf, errors.Wrap(err, "executing template failed")
}
