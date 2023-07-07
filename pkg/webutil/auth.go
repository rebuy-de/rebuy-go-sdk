package webutil

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v6/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v6/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v6/pkg/typeutil"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	authStateCookie = `oauthstate`
	authSessionName = `oauth`
)

func cookieName() string {
	return cmdutil.Name + "-token"
}

type AuthInfo struct {
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
	for _, have := range i.RealmAccess.Roles {
		if have == want {
			return true
		}
	}

	return false
}

type authMiddleware struct {
	getClaimFromRequest     func(*http.Request) (*AuthInfo, error)
	createTokenFromCallback func(*http.Request) (string, error)
	handleLogin             func(http.ResponseWriter, *http.Request)
}

func (m *authMiddleware) handler(next http.Handler) http.Handler {
	router := chi.NewRouter()
	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		claims, err := m.getClaimFromRequest(r)
		if err != nil {
			logutil.Get(r.Context()).Warnf("auth middleware: %v", err.Error())
		} else {
			ctx := r.Context()
			ctx = typeutil.ContextWithValueSingleton(ctx, claims)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})

	router.HandleFunc("/auth/login", m.handleLogin)

	router.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := m.createTokenFromCallback(r)
		if err != nil {
			fmt.Fprintf(w, "handle callback: %v", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		cookie := http.Cookie{
			Name:     cookieName(),
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(7 * 24 * time.Hour), // TODO
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	return router
}

type AuthSecrets struct {
	ClientID     string `vault:"client_id"`
	ClientSecret string `vault:"client_secret"`
}

type AuthConfig struct {
	Secrets     AuthSecrets
	ConfigURL   string
	BaseURL     string
	SigningAlgs []string
}

func (c *AuthConfig) Bind(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(
		&c.ConfigURL, "oidc-config-url", "",
		`URL to retrieve the OpenID Provider Configuration Information.`)
	cmd.PersistentFlags().StringVar(
		&c.BaseURL, "oidc-base-url", "",
		`Public reachable URL of this application. Used for cookies and the callback URL.`)
}

func NewAuthMiddleware(ctx context.Context, config AuthConfig) (func(http.Handler) http.Handler, error) {
	provider, err := oidc.NewProvider(ctx, config.ConfigURL)
	if err != nil {
		return nil, fmt.Errorf("init OIDC provider: %w", err)
	}

	oidcConfig := &oidc.Config{
		ClientID:             config.Secrets.ClientID,
		SupportedSigningAlgs: config.SigningAlgs,
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.Secrets.ClientID,
		ClientSecret: config.Secrets.ClientSecret,
		RedirectURL:  strings.TrimRight(config.BaseURL, "/") + "/auth/callback",
		Scopes:       []string{oidc.ScopeOpenID, "email", "roles"},
		Endpoint:     provider.Endpoint(),
	}

	verifier := provider.Verifier(oidcConfig)

	m := authMiddleware{
		handleLogin: func(w http.ResponseWriter, r *http.Request) {
			oauthState := generateCookie(w)
			u := oauth2Config.AuthCodeURL(oauthState)
			http.Redirect(w, r, u, http.StatusTemporaryRedirect)
		},
		getClaimFromRequest: func(r *http.Request) (*AuthInfo, error) {
			cookie, err := r.Cookie(cookieName())
			if err != nil {
				return nil, fmt.Errorf("get auth cookie")
			}

			idToken, err := verifier.Verify(r.Context(), cookie.Value)
			if err != nil {
				return nil, fmt.Errorf("verify token: %w", err)
			}

			var authInfo AuthInfo
			err = idToken.Claims(&authInfo)
			if err != nil {
				return nil, fmt.Errorf("unmarshal claims: %w", err)
			}

			return &authInfo, nil
		},
		createTokenFromCallback: func(r *http.Request) (string, error) {
			oauthState, err := r.Cookie(authStateCookie)
			if err != nil {
				return "", fmt.Errorf("get auth cookie: %w", err)
			}

			if r.FormValue("state") != oauthState.Value {
				return "", fmt.Errorf("invalid oauth state cookie")
			}

			token, err := oauth2Config.Exchange(r.Context(), r.FormValue("code"))
			if err != nil {
				return "", fmt.Errorf("exchange token: %w", err)
			}

			rawIDToken, ok := token.Extra("id_token").(string)
			if !ok {
				return "", fmt.Errorf("no id_token field in oauth2 token")
			}

			idToken, err := verifier.Verify(r.Context(), rawIDToken)
			if err != nil {
				return "", fmt.Errorf("verify token: %w", err)
			}

			var claims AuthInfo
			err = idToken.Claims(&claims)
			if err != nil {
				return "", fmt.Errorf("unmarshal claims: %w", err)
			}

			return rawIDToken, nil
		},
	}

	return m.handler, nil
}

func generateCookie(w http.ResponseWriter) string {
	var expiration = time.Now().Add(10 * time.Minute)

	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{
		Name:     authStateCookie,
		Value:    state,
		Expires:  expiration,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)

	return state
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

	m := authMiddleware{
		handleLogin: vh.Wrap(func(v *View, r *http.Request) Response {
			return v.HTML(http.StatusOK, "dev-login.html", map[string]any{
				"username": "dummy@example.com",
				"name":     "John Doe",
				"roles":    roleNames,
			})
		}),
		getClaimFromRequest: func(r *http.Request) (*AuthInfo, error) {
			cookie, err := r.Cookie(cookieName())
			if err != nil {
				return nil, fmt.Errorf("get cookie: %w", err)
			}

			jsonPayload, err := base64.RawURLEncoding.DecodeString(cookie.Value)
			if err != nil {
				return nil, fmt.Errorf("b64 decode cookie: %w", err)
			}

			var claims AuthInfo
			err = json.Unmarshal(jsonPayload, &claims)
			if err != nil {
				return nil, fmt.Errorf("json decode cookie: %w", err)
			}

			return &claims, nil
		},
		createTokenFromCallback: func(r *http.Request) (string, error) {
			var claims AuthInfo

			claims.Username = r.PostFormValue("username")
			claims.Name = r.PostFormValue("name")

			for name, role := range roleNames {
				value := r.PostFormValue(name)
				if strings.TrimSpace(value) != "" {
					claims.RealmAccess.Roles = append(claims.RealmAccess.Roles, role)
				}
			}

			jsonPayload, err := json.Marshal(claims)
			if err != nil {
				return "", fmt.Errorf("marshal cookie: %v", err)
			}

			b64Payload := base64.RawURLEncoding.EncodeToString(jsonPayload)

			return string(b64Payload), nil
		},
	}

	return m.handler
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
	info := AuthInfoFromRequest(r)
	if info == nil {
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
func AuthInfoFromContext(ctx context.Context) *AuthInfo {
	return typeutil.FromContextSingleton[AuthInfo](ctx)
}

// AuthInfoFromRequest extracts the AuthInfo from the context within the given
// request. The AuthInfo is injected into the request via the AuthMiddleware.
// Therefore it is required to use this middleware to be able to get the
// AuthInfo.
func AuthInfoFromRequest(r *http.Request) *AuthInfo {
	return AuthInfoFromContext(r.Context())
}
