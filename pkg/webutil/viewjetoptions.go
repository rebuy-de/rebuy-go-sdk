package webutil

import (
	"fmt"
	"net/http"

	"github.com/CloudyKit/jet/v6"
)

type JetOption func(*http.Request, *jet.VarMap)

func JetOptions(options ...JetOption) JetOption {
	return func(r *http.Request, m *jet.VarMap) {
		for _, option := range options {
			option(r, m)
		}
	}
}

func WithVar(name string, value any) JetOption {
	return func(_ *http.Request, m *jet.VarMap) {
		m.Set(name, value)
	}
}

func WithVarf(name string, s string, a ...any) JetOption {
	return WithVar(name, fmt.Sprintf(s, a...))
}

// WithRequestVar is a JetOption that adds any variable or function to the
// template context, that is based on the *http.Request.
func WithRequestVar[O any](name string, fn func(*http.Request) O) JetOption {
	return func(r *http.Request, m *jet.VarMap) {
		m.Set(name, fn(r))
	}
}

// WithRequestVar1 is a shortcut to define a template function with one
// argument that depends on the *http.Request. It simply avoids nesting two
// `return func...` by merging their arguments.
func WithRequestVar1[I1, O any](name string, fn func(*http.Request, I1) O) JetOption {
	return WithRequestVar(name, func(r *http.Request) any {
		return func(v1 I1) any {
			return fn(r, v1)
		}
	})
}

// WithRequestVar2 is a shortcut to define a template function with two
// argument that depends on the *http.Request. It simply avoids nesting two
// `return func...` by merging their arguments.
func WithRequestVar2[I1, I2, O any](name string, fn func(*http.Request, I1, I2) O) JetOption {
	return WithRequestVar(name, func(r *http.Request) any {
		return func(v1 I1, v2 I2) any {
			return fn(r, v1, v2)
		}
	})
}

// WithRequestVar3 is a shortcut to define a template function with three
// argument that depends on the *http.Request. It simply avoids nesting two
// `return func...` by merging their arguments.
func WithRequestVar3[I1, I2, I3, O any](name string, fn func(*http.Request, I1, I2, I3) O) JetOption {
	return WithRequestVar(name, func(r *http.Request) any {
		return func(v1 I1, v2 I2, v3 I3) any {
			return fn(r, v1, v2, v3)
		}
	})
}
