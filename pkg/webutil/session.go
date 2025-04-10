package webutil

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/cmdutil"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type sessionContextKeyType int

const (
	sessionContextKey sessionContextKeyType = 0
)

type SessionSecret []byte

// SessionSecretSourceVolatile creates a session secret that is stored in
// memory only. This implies, that all session data is lost after an
// application restart and an application cannot have more than one replicas.
func SessionSecretSourceVolatile() []byte {
	return securecookie.GenerateRandomKey(32)
}

type RedisSessioner interface {
	Get(context.Context, string) *redis.StringCmd
	Set(context.Context, string, any, time.Duration) *redis.StatusCmd
}

// SessionSecretSourceRedis stores the session secrets in Redis. If the key
// does not exist yet, it will create a new one.
func SessionSecretSourceRedis(ctx context.Context, client RedisSessioner, prefix string) ([]byte, error) {
	key := path.Join(prefix, "session-secret")

	secret, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		secret := SessionSecretSourceVolatile()
		err := client.Set(ctx, key, secret, 24*30*time.Hour).Err()
		return secret, errors.Wrap(err, "failed to set new secret")
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to read secret")
	}

	return []byte(secret), nil
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
func SessionMiddleware(secret SessionSecret, opts ...SessionMiddlewareOption) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return sessionMiddlewareFunc(next, secret, opts...)
	}
}

type sessionMiddlewareConfig struct {
	name  string
	store *sessions.CookieStore
}

type SessionMiddlewareOption func(c *sessionMiddlewareConfig)

func SessionMiddlewareCookieDomain(domain string) SessionMiddlewareOption {
	return func(c *sessionMiddlewareConfig) {
		c.store.Options.Domain = domain
	}
}

func SessionMiddlewareCookieUnsecure() SessionMiddlewareOption {
	return func(c *sessionMiddlewareConfig) {
		c.store.Options.Secure = false
	}
}

func sessionMiddlewareFunc(next http.Handler, secret SessionSecret, opts ...SessionMiddlewareOption) http.Handler {
	config := sessionMiddlewareConfig{
		name:  fmt.Sprintf("%s-session", cmdutil.Name),
		store: sessions.NewCookieStore(secret),
	}

	config.store.Options.HttpOnly = true
	config.store.Options.Secure = true

	for _, o := range opts {
		o(&config)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := config.store.Get(r, config.name)
		if err != nil {
			logrus.WithError(err).Warn("failed to restore session; creating new one")
			session.Save(r, w)
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
