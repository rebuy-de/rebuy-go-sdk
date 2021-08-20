package webutil

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"
)

// Deprecated: Use MVP instead.
type TemplateRendererOption func(*http.Request, *template.Template) *template.Template

// Deprecated: Use MVP instead.
func TemplateRendererFunc(name string, fn interface{}) TemplateRendererOption {
	return func(r *http.Request, t *template.Template) *template.Template {
		return t.Funcs(template.FuncMap{
			name: fn,
		})
	}
}

// Deprecated: Use MVP instead.
type TemplateRenderer struct {
	box  *packr.Box
	opts []TemplateRendererOption
}

// Deprecated: Use MVP instead.
func NewTemplateRenderer(box *packr.Box, opts ...TemplateRendererOption) *TemplateRenderer {
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

// Deprecated: Use MVP instead.
func (tr *TemplateRenderer) RespondHTML(writer http.ResponseWriter, request *http.Request, name string, data interface{}) {
	tpl := template.New("")

	for _, opt := range tr.opts {
		tpl = opt(request, tpl)
	}

	err := tr.box.Walk(func(name string, file packr.File) error {
		var err error
		tpl = tpl.New(name)
		tpl, err = tpl.Parse(file.String())
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
