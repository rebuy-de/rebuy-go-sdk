package webutil

import (
	"html/template"
	"net/http"
)

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
