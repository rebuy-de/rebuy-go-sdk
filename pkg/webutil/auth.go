package webutil

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

const (
	authStateCookie = `oauthstate`
	authSessionName = `oauth`
)

// AuthInfo contains data about the currently logged in user.
type AuthInfo struct {
	Username  string
	Name      string
	Roles     []string
	UpdatedAt time.Time
}

type IDTokenClaims struct {
	Username    string `json:"preferred_username"`
	Name        string `json:"name"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

// HasRole returns true, if the user has the given role. The role name needs to
// be allowlisted in the AuthMiddleware, otherwise it will return false even if
// the user is in the team.
func (i AuthInfo) HasRole(want string) bool {
	for _, have := range i.Roles {
		if have == want {
			return true
		}
	}

	return false
}

func init() {
	gob.Register(AuthInfo{})
}

type AuthConfig struct {
	ClientID     string
	ClientSecret string
	ConfigURL    string
	RedirectURL  string
	SigningAlgs  []string
}

// Middleware is an HTTP request middleware that adds login endpoints. The
// request makes use of sessions, therefore the SessionMiddleware is required.
//
// The teams argument contains a whitelist of team names, that are copied into
// the AuthInfo, if the user is in those teams. It is desirable to copy only
// the needed subset of teams into the AuthInfo, because this data is carried
// in the session cookie.
//
// Endpoint "/auth/login" initiates the user login and redirects them to the
// GitHub OAuth page.
//
// Endpoint "/auth/callback" gets called by the user after being redirected
// from GitHub after a successful login.
func (c AuthConfig) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return c.authMiddlewareFunc(next)
	}
}

func (c AuthConfig) authMiddlewareFunc(next http.Handler) http.Handler {
	provider, err := oidc.NewProvider(context.Background(), c.ConfigURL)
	if err != nil {
		logrus.Error(err)
	}

	oidcConfig := &oidc.Config{
		ClientID:             c.ClientID,
		SupportedSigningAlgs: c.SigningAlgs,
	}

	mw := authMiddleware{
		next:  next,
		teams: map[string]struct{}{},
		config: &oauth2.Config{
			ClientID:     c.ClientID,
			ClientSecret: c.ClientSecret,
			RedirectURL:  c.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile", "roles"},
			Endpoint:     provider.Endpoint(),
		},
		verifier: provider.Verifier(oidcConfig),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", mw.handleLogin)
	mux.HandleFunc("/auth/callback", mw.handleCallback)
	mux.HandleFunc("/", mw.handleDefault)

	return mux
}

type authMiddleware struct {
	next     http.Handler
	teams    map[string]struct{}
	config   *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func (mw *authMiddleware) handleDefault(w http.ResponseWriter, r *http.Request) {
	mw.next.ServeHTTP(w, r)
}

func (mw *authMiddleware) handleLogin(w http.ResponseWriter, r *http.Request) {
	oauthState := mw.generateCookie(w)
	u := mw.config.AuthCodeURL(oauthState)
	http.Redirect(w, r, u, http.StatusTemporaryRedirect)
}

func (mw *authMiddleware) generateCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(10 * time.Minute)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:    authStateCookie,
		Value:   state,
		Expires: expiration,
	}
	http.SetCookie(w, &cookie)

	return state
}

func (mw *authMiddleware) handleCallback(w http.ResponseWriter, r *http.Request) {
	oauthState, err := r.Cookie(authStateCookie)
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Error("failed get auth cookie")
		return
	}

	if r.FormValue("state") != oauthState.Value {
		logrus.Warn("invalid oauth state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	token, err := mw.config.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Error("failed to exchange token")
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		logrus.WithError(errors.WithStack(err)).Error("No id_token field in oauth2 token")
		return
	}

	idToken, err := mw.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Error("Failed to verify ID Token: " + err.Error())
		return
	}

	var claims IDTokenClaims
	err = idToken.Claims(&claims)
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Error("Failed to unmarshal claims: " + err.Error())
		return
	}

	refreshSessionData(w, r, &claims)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func refreshSessionData(w http.ResponseWriter, r *http.Request, idTokenClaims *IDTokenClaims) error {
	session, err := SessionFromRequest(r)
	if err != nil {
		return errors.WithStack(err)
	}

	info, ok := session.Values["auth-info"].(AuthInfo)
	if !ok {
		info = AuthInfo{}
	}

	info.Username = idTokenClaims.Username
	info.Name = idTokenClaims.Name
	info.UpdatedAt = time.Now()
	info.Roles = idTokenClaims.RealmAccess.Roles

	session.Values["auth-info"] = info
	err = session.Save(r, w)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

//go:embed templates/*
var templateFS embed.FS

// DevAuthMiddleware is a dummy auth middleware that does not any actual
// authentication. It is supposed to be used for local development.
// The roles parameter defines which roles can be selected in the dummy login
// form.
func DevAuthMiddleware(roles ...string) func(http.Handler) http.Handler {
	subFS, _ := fs.Sub(templateFS, "templates")
	vh := NewViewHandler(subFS)

	roleNames := map[string]string{}
	for _, r := range roles {
		roleNames[fmt.Sprintf("role-%s", r)] = r
	}

	return func(next http.Handler) http.Handler {
		router := chi.NewRouter()
		router.HandleFunc("/*", next.ServeHTTP)

		router.Get("/auth/login", vh.Wrap(func(v *View, r *http.Request) Response {
			return v.HTML(http.StatusOK, "dev-login.html", map[string]any{
				"username": "dummy@example.com",
				"name":     "John Doe",
				"roles":    roleNames,
			})
		}))

		router.Post("/auth/login", func(w http.ResponseWriter, r *http.Request) {
			var claims IDTokenClaims

			claims.Username = r.PostFormValue("username")
			claims.Name = r.PostFormValue("name")

			for name, role := range roleNames {
				value := r.PostFormValue(name)
				if strings.TrimSpace(value) != "" {
					claims.RealmAccess.Roles = append(claims.RealmAccess.Roles, role)
				}
			}

			refreshSessionData(w, r, &claims)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		})

		return router
	}
}

// AuthTemplateFunctions returns auth related template functions.  These can
// then directly be used in the templates without having to add the auth info
// manually to the template data.
//
// Function `func AuthIsAuthenticated() bool` returns true, if the user is
// logged in.
//
// Function `func AuthInfo() *AuthInfo` returns the AuthInfo, if the user is
// logged and `nil` otherwise.
//
// Example:
//
//	{{ if AuthIsAuthenticated }}
//	  <span class="navbar-text">Hello, <em>{{ AuthInfo.Name }}</em>!</span>
//	{{ else }}
//	  <a class="nav-link" href="/auth/login">Login</span></a>
//	{{ end }}
func AuthTemplateFunctions(r *http.Request) template.FuncMap {
	authenticated := true
	info, err := AuthInfoFromRequest(r)
	if err != nil {
		authenticated = false
	}

	return template.FuncMap{
		"AuthIsAuthenticated": func() bool {
			return authenticated
		},
		"AuthInfo": func() *AuthInfo {
			return info
		},
	}
}

// AuthInfoFromContext extracts the AuthInfo from the given context. The
// AuthInfo is injected into the request via the AuthMiddleware. Therefore it
// is required to use this middleware to be able to get the AuthInfo.
func AuthInfoFromContext(ctx context.Context) (*AuthInfo, error) {
	sess, err := SessionFromContext(ctx)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	info, ok := sess.Values["auth-info"].(AuthInfo)
	if !ok {
		return nil, errors.Errorf("auth info not found in context")
	}

	return &info, nil
}

// AuthInfoFromRequest extracts the AuthInfo from the context within the given
// request. The AuthInfo is injected into the request via the AuthMiddleware.
// Therefore it is required to use this middleware to be able to get the
// AuthInfo.
func AuthInfoFromRequest(r *http.Request) (*AuthInfo, error) {
	return AuthInfoFromContext(r.Context())
}
