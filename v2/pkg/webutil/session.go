package webutil

import (
	"context"
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type sessionContextKeyType int

const (
	sessionName = `rebuy-go-sdk`

	sessionContextKey sessionContextKeyType = 0
)

type SessionSecret []byte

// SessionSecretSourceVolatile creates a session secret that is stored in
// memory only. This implies, that all session data is lost after an
// application restart and an application cannot have more than one replicas.
func SessionSecretSourceVolatile() []byte {
	return securecookie.GenerateRandomKey(32)
}

// SessionFromContext extracts the Session store from the given context. The
// session store is injected into the request via the SessionMiddleware.
// Therefore it is required to use this middleware to be able to get the store.
func SessionFromContext(ctx context.Context) (*sessions.Session, error) {
	sess, ok := ctx.Value(sessionContextKey).(*sessions.Session)
	if !ok {
		return nil, errors.Errorf("session not found in context")
	}
	return sess, nil
}

// SessionFromRequest returns the results of SessionFromContext for the context
// of the given request.
func SessionFromRequest(r *http.Request) (*sessions.Session, error) {
	return SessionFromContext(r.Context())
}

// SessionMiddleware inizializes the session store and injects it into the
// context of the requests.
func SessionMiddleware(secret SessionSecret) Middleware {
	return func(next http.Handler) http.Handler {
		return sessionMiddlewareFunc(next, secret)
	}
}

func sessionMiddlewareFunc(next http.Handler, secret SessionSecret) http.Handler {
	store := sessions.NewCookieStore(secret)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, sessionName)
		if err != nil {
			logrus.WithError(err).Warn("failed to restore session; creating new one")
			session.Save(r, w)
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
