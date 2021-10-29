package webutil

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type TemplateRendererOption func(*http.Request, *template.Template) *template.Template

func TemplateRendererFunc(name string, fn interface{}) TemplateRendererOption {
	return func(r *http.Request, t *template.Template) *template.Template {
		return t.Funcs(template.FuncMap{
			name: fn,
		})
	}
}

type TemplateRenderer struct {
	box  *embed.FS
	opts []TemplateRendererOption
}

func NewTemplateRenderer(box *embed.FS, opts ...TemplateRendererOption) *TemplateRenderer {
	stdopts := []TemplateRendererOption{
		TemplateRendererFunc("StringTitle", strings.Title),
		TemplateRendererFunc("PrettyTime", TemplateFuncPrettyTime),
	}

	renderer := TemplateRenderer{
		box:  box,
		opts: append(stdopts, opts...),
	}

	return &renderer
}

func (tr *TemplateRenderer) RespondHTML(writer http.ResponseWriter, request *http.Request, name string, data interface{}) {
	tpl := template.New("")

	for _, opt := range tr.opts {
		tpl = opt(request, tpl)
	}

	err := fs.WalkDir(tr.box, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		file, err := tr.box.ReadFile(path)
		if err != nil {
			return err
		}
		tpl = tpl.New(filepath.Base(path))
		tpl, err = tpl.Parse(string(file))
		return err
	})
	if RespondError(writer, err) {
		return
	}

	buf := new(bytes.Buffer)
	err = tpl.ExecuteTemplate(buf, name, data)
	if RespondError(writer, err) {
		return
	}

	writer.Header().Set("Content-Type", "text/html")
	buf.WriteTo(writer)
}

func TemplateFuncPrettyTime(value interface{}) (string, error) {
	tPtr, ok := value.(*time.Time)
	if ok {
		if tPtr == nil {
			return "N/A", nil
		}
		value = *tPtr
	}

	t, ok := value.(time.Time)
	if !ok {
		return "", errors.Errorf("unexpected type")
	}

	if t.IsZero() {
		return "N/A", nil
	}

	format := "Mon, 2 Jan 15:04:05"

	t = t.Local()

	return t.Format(format), nil
}
