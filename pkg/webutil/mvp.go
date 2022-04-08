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

// Model should be used by the Presenter and its purpose is to provide an
// interface for data generation that is used by templates. This has the
// advantage that we can reuse models for multiple views (eg JSON and HTML) and
// that the data generation is isolated from representation.
type Model func(*http.Request) (interface{}, int, error)

// View should be used by with the Presenter and its puropose is to avoid
// having to implement the Golang template rendering for the gazillionth time.
// This package contains some ready-to-use views.
type View func(http.ResponseWriter, *http.Request, interface{}, int, error)

// Presenter (from Model-view-presenter [1]) acts as a middleman between Model
// and View.
// [1]: https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93presenter
func Presenter(m Model, v View) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, code, err := m(r)
		if err != nil {
			logrus.
				WithField("stacktrace", fmt.Sprintf("%+v", err)).
				WithError(errors.WithStack(err)).
				Errorf("response failed: %s", err)
		}

		switch code {
		case http.StatusMovedPermanently:
			fallthrough
		case http.StatusFound:
			fallthrough
		case http.StatusSeeOther:
			fallthrough
		case http.StatusTemporaryRedirect:
			fallthrough
		case http.StatusPermanentRedirect:
			url := data.(string)
			http.Redirect(w, r, url, code)
			return
		}

		v(w, r, data, code, err)
	}
}

// NilModel is a Model that contains no data. Useful for rendering templates
// that do not need any data.
func NilModel(*http.Request) (interface{}, int, error) {
	return nil, http.StatusOK, nil
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

// HTMLTemplateView provides a View that renders the Model with html/template.
type HTMLTemplateView struct {
	FS       fs.FS
	FuncMaps []TemplateFuncMap
}

func NewHTMLTemplateView(fs fs.FS, fms ...TemplateFuncMap) *HTMLTemplateView {
	v := &HTMLTemplateView{
		FS:       fs,
		FuncMaps: fms,
	}

	return v
}

func (v *HTMLTemplateView) Render(filename string, r *http.Request, d interface{}) (*bytes.Buffer, error) {
	t := template.New(filename)

	for _, fm := range v.FuncMaps {
		t = t.Funcs(fm(r))
	}

	t, err := t.ParseFS(v.FS, "*")
	if err != nil {
		return nil, errors.Wrap(err, "parsing template failed")
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, d)

	return buf, errors.Wrap(err, "executing template failed")
}

// View returns a View that can be used by a Presenter.
//
// Usage:
//     html := &HTMLTemplateView{ FS: server.TemplateFS }
//     router.GET("/", Presenter(server.indexModel, html.View("index.html")))
func (v *HTMLTemplateView) View(filename string) View {
	return func(w http.ResponseWriter, r *http.Request, d interface{}, s int, err error) {
		if err != nil {
			w.WriteHeader(s)
			fmt.Fprint(w, err)
			return
		}

		buf, err := v.Render(filename, r, d)
		if err != nil {
			logrus.WithError(errors.WithStack(err)).Errorf("rendering template failed")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(s)
		w.Header().Set("Content-Type", "text/html")
		buf.WriteTo(w)
	}
}

// JSONView is a View that renders the Model as JSON.
func JSONView(w http.ResponseWriter, r *http.Request, d interface{}, s int, err error) {
	if err != nil {
		d = err
	}

	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "    ")
	err = enc.Encode(d)
	if err != nil {
		return
	}

	w.WriteHeader(s)
	w.Header().Set("Content-Type", "application/json")
	buf.WriteTo(w)

}
