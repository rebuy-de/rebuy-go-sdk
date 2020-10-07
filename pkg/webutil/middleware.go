package webutil

import (
	"net/http"
)

// Middleware is a function that wraps an http.Handler. The function is
// supposed to call the next handler to continue to execute the process.
// This type is used by the MiddlewareChain to allow an easy usage of multiple
// middlewares which makes it easy to see the proper calling order.
type Middleware func(next http.Handler) http.Handler

// NewMiddlewareChain initializes an empty MiddlewareChain.
func NewMiddlewareChain() MiddlewareChain {
	return MiddlewareChain{}
}

// MiddlewareChain is a builder for an http.Handler out of multiple middlewares.
type MiddlewareChain []Middleware

// Then defines the next middleware to call.
func (c MiddlewareChain) Then(mw Middleware) MiddlewareChain {
	return MiddlewareChain(append(c, mw))
}

// Finally builds the http.Handler that is used by the HTTP server.
func (c MiddlewareChain) Finally(h http.Handler) http.Handler {
	for i := len(c) - 1; i >= 0; i-- {
		middleware := c[i]
		h = middleware(h)
	}
	return h
}
