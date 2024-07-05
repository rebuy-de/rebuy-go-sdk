package webutil

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"reflect"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type JetViewer struct {
	views       *jet.Set
	htmlOptions []JetViewerHTMLOption
}

type JetOption func(*JetViewer)

func WithJetViewerHTMLOption(o JetViewerHTMLOption) JetOption {
	return func(jet *JetViewer) {
		jet.htmlOptions = append(jet.htmlOptions, o)
	}
}

func JetFunctionOption(name string, fn any) JetOption {
	return func(jet *JetViewer) {
		jet.views.AddGlobal(name, fn)
	}
}

// deprecated: use JetFunctionOption
func JetFunctionMapOption(funcs map[string]any) JetOption {
	return func(jet *JetViewer) {
		for name, fn := range funcs {
			jet.views.AddGlobal(name, fn)
		}
	}
}

func JetVarOption(key string, value any) JetOption {
	return func(jet *JetViewer) {
		jet.views.AddGlobal(key, value)
	}
}

func NewJetViewer(js *jet.Set, options ...JetOption) *JetViewer {
	jv := &JetViewer{
		views: js,
	}

	jv.views.AddGlobal("contains", strings.Contains)

	jv.views.AddGlobalFunc("deref", func(a jet.Arguments) reflect.Value {
		a.RequireNumOfArguments("pointer", 1, 1)
		v := a.Get(0)
		if v.Kind() == reflect.Ptr {
			return v.Elem()
		}

		return v
	})

	jv.apply(options...)

	return jv
}

func (j *JetViewer) apply(options ...JetOption) {
	for _, option := range options {
		option(j)
	}
}

type JetViewerHTMLOption func(*http.Request, *jet.VarMap)

func WithVar(name string, value any) JetViewerHTMLOption {
	return func(_ *http.Request, vars *jet.VarMap) {
		vars.Set(name, value)
	}
}

func WithRequestVar(name string, fn func(*http.Request) any) JetViewerHTMLOption {
	return func(r *http.Request, vars *jet.VarMap) {
		vars.Set(name, fn(r))
	}
}

func WithVarf(name string, s string, a ...any) JetViewerHTMLOption {
	return WithVar(name, fmt.Sprintf(s, a...))
}

func (j *JetViewer) HTML(status int, filename string, data any, opts ...JetViewerHTMLOption) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span, ctx := tracer.StartSpanFromContext(
			r.Context(), "render",
			tracer.Tag(ext.ResourceName, filename),
			tracer.Tag(ext.SpanKind, ext.SpanKindInternal),
		)
		r = r.WithContext(ctx)
		defer span.Finish()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		view, err := j.views.GetTemplate(filename)
		if err != nil {
			ViewError(http.StatusInternalServerError, err)(w, r)
			return
		}

		vars := make(jet.VarMap)
		vars.Set("currentURLPath", r.URL.Path)

		for _, o := range j.htmlOptions {
			o(r, &vars)
		}

		for _, o := range opts {
			o(r, &vars)
		}

		err = view.Execute(w, vars, data)
		if err != nil {
			ViewError(http.StatusInternalServerError, err)(w, r)
			return
		}
	}
}

type JetFSLoader struct {
	fs.FS
}

func (l JetFSLoader) Exists(path string) bool {
	f, err := l.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func (l JetFSLoader) Open(path string) (io.ReadCloser, error) {
	path = strings.TrimLeft(path, "/")
	return l.FS.Open(path)
}
