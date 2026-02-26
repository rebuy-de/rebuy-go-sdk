package webutil

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"log/slog"

	"github.com/a-h/templ"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/typeutil"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

const (
	authStateCookie = `oauthstate`
)

type authState struct {
	CsrfToken   string `json:"csrf_token"`
	RedirectURI string `json:"redirect_uri"`
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
	return slices.Contains(i.RealmAccess.Roles, want)
}

type AuthMiddleware func(http.Handler) http.Handler

type authMiddleware struct {
	getClaimFromRequest func(http.ResponseWriter, *http.Request) (*AuthInfo, error)
	handleCallback      func(http.ResponseWriter, *http.Request) (string, error)
	handleLogin         func(http.ResponseWriter, *http.Request)
	handleLogout        func(http.ResponseWriter, *http.Request)
}

func (m *authMiddleware) handler(next http.Handler) http.Handler {
	router := chi.NewRouter()
	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		claims, err := m.getClaimFromRequest(w, r)
		if err != nil {
			slog.Warn("auth middleware", "error", err)
		} else if claims != nil {
			ctx := r.Context()
			ctx = typeutil.ContextWithValueSingleton(ctx, claims)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})

	router.HandleFunc("/auth/login", m.handleLogin)
	router.HandleFunc("/auth/logout", m.handleLogout)

	router.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		redirectURI, err := m.handleCallback(w, r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "handle callback: %v", err.Error())
			return
		}

		http.Redirect(w, r, redirectURI, http.StatusSeeOther)
	})

	return router
}

type AuthSecrets struct {
	ClientID     string `vault:"client_id"`
	ClientSecret string `vault:"client_secret"`
	SessionKey   string `vault:"session_key"`
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

	encrypter, err := newCookieEncrypter[oauth2.Token](config.Secrets.SessionKey)
	if err != nil {
		return nil, fmt.Errorf("create encrypter: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     config.Secrets.ClientID,
		ClientSecret: config.Secrets.ClientSecret,
		RedirectURL:  strings.TrimRight(config.BaseURL, "/") + "/auth/callback",
		Scopes:       []string{oidc.ScopeOpenID, "email", "roles"},
		Endpoint:     provider.Endpoint(),
	}

	m := authMiddleware{
		handleLogin: func(w http.ResponseWriter, r *http.Request) {
			redirectURI := r.URL.Query().Get("redirect")
			redirectURI = validateRedirectURI(redirectURI)

			oauthState, err := generateCookie(w, redirectURI)
			if err != nil {
				http.Error(w, fmt.Sprintf("generate cookie: %v", err), http.StatusInternalServerError)
				return
			}

			u := oauth2Config.AuthCodeURL(oauthState)
			http.Redirect(w, r, u, http.StatusTemporaryRedirect)
		},
		handleLogout: func(w http.ResponseWriter, r *http.Request) {
			cookie := &http.Cookie{
				Name:     encrypter.cookieName(),
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		},
		getClaimFromRequest: func(w http.ResponseWriter, r *http.Request) (*AuthInfo, error) {
			token, err := encrypter.ReadCookie(r)
			if err != nil {
				return nil, fmt.Errorf("get auth cookie: %w", err)
			}
			if token == nil {
				return nil, nil
			}

			tokenSource := oauth2Config.TokenSource(r.Context(), token)
			ui, err := provider.UserInfo(r.Context(), tokenSource)
			if err != nil {
				return nil, fmt.Errorf("get userinfo: %w", err)
			}

			freshToken, err := tokenSource.Token()
			if err != nil {
				return nil, fmt.Errorf("get fresh token: %w", err)
			}

			if freshToken.Expiry.After(token.Expiry) {
				// This means the token was automatically refreshed by the
				// oauth library when callind UserInfo(). We need to pass this
				// token down to the user.
				err = encrypter.WriteCookie(w, freshToken)
				if err != nil {
					return nil, fmt.Errorf("write refreshed token cookie: %w", err)
				}
			}

			var info AuthInfo
			err = ui.Claims(&info)
			if err != nil {
				return nil, fmt.Errorf("get claim from userinfo: %w", err)
			}

			return &info, nil
		},
		handleCallback: func(w http.ResponseWriter, r *http.Request) (string, error) {
			stateCookie, err := r.Cookie(authStateCookie)
			if err != nil {
				return "", fmt.Errorf("get auth cookie: %w", err)
			}

			// Decode the state cookie
			stateJSON, err := base64.URLEncoding.DecodeString(stateCookie.Value)
			if err != nil {
				return "", fmt.Errorf("decode state cookie: %w", err)
			}

			var state authState
			err = json.Unmarshal(stateJSON, &state)
			if err != nil {
				return "", fmt.Errorf("unmarshal state: %w", err)
			}

			// Verify CSRF token
			if r.FormValue("state") != state.CsrfToken {
				return "", fmt.Errorf("invalid oauth state cookie")
			}

			token, err := oauth2Config.Exchange(r.Context(), r.FormValue("code"))
			if err != nil {
				return "", fmt.Errorf("exchange token: %w", err)
			}

			err = encrypter.WriteCookie(w, token)
			if err != nil {
				return "", fmt.Errorf("write token cookie: %w", err)
			}

			return state.RedirectURI, nil
		},
	}

	return m.handler, nil
}

// validateRedirectURI validates and sanitizes a redirect URI.
// It only allows relative paths starting with "/".
// Returns "/" as default if the URI is invalid or empty.
func validateRedirectURI(redirectURI string) string {
	redirectURI = strings.TrimSpace(redirectURI)

	if redirectURI == "" {
		return "/"
	}

	// Parse the redirect URI
	u, err := url.Parse(redirectURI)
	if err != nil {
		// Invalid URL, use default
		return "/"
	}

	// Only allow relative URLs (no scheme and no host)
	if u.Scheme != "" || u.Host != "" {
		// Has scheme or host, reject it
		return "/"
	}

	// Must start with / to be a valid path
	if !strings.HasPrefix(u.Path, "/") {
		return "/"
	}

	// Prevent protocol-relative URLs like //evil.com
	if strings.HasPrefix(redirectURI, "//") {
		return "/"
	}

	return redirectURI
}

func generateCookie(w http.ResponseWriter, redirectURI string) (string, error) {
	var expiration = time.Now().Add(10 * time.Minute)

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	csrfToken := base64.URLEncoding.EncodeToString(b)

	state := authState{
		CsrfToken:   csrfToken,
		RedirectURI: redirectURI,
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("marshal state: %w", err)
	}

	stateEncoded := base64.URLEncoding.EncodeToString(stateJSON)

	cookie := http.Cookie{
		Name:     authStateCookie,
		Value:    stateEncoded,
		Path:     "/",
		Expires:  expiration,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)

	return csrfToken, nil
}

//go:embed templates/*
var templateFS embed.FS

// DevAuthMiddleware is a dummy auth middleware that does not any actual
// authentication. It is supposed to be used for local development.
// The roles parameter defines which roles can be selected in the dummy login
// form.
func DevAuthMiddleware(roles ...string) AuthMiddleware {
	subFS, _ := fs.Sub(templateFS, "templates")

	viewer := NewGoTemplateViewer(subFS)

	roleNames := map[string]string{}
	for _, r := range roles {
		roleNames[fmt.Sprintf("role-%s", r)] = r
	}

	m := authMiddleware{
		handleLogin: WrapView(func(r *http.Request) Response {
			redirectURI := r.URL.Query().Get("redirect")
			redirectURI = validateRedirectURI(redirectURI)

			return viewer.HTML(http.StatusOK, "dev-login.html", map[string]any{
				"username":    "dummy@example.com",
				"name":        "John Doe",
				"roles":       roleNames,
				"redirectURI": redirectURI,
			})
		}),
		handleLogout: func(w http.ResponseWriter, r *http.Request) {
			cookie := &http.Cookie{
				Name:     "rebuy-go-sdk-auth",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, cookie)
			http.Redirect(w, r, "/", http.StatusSeeOther)
		},
		getClaimFromRequest: func(_ http.ResponseWriter, r *http.Request) (*AuthInfo, error) {
			cookie, err := r.Cookie("rebuy-go-sdk-auth")
			if errors.Is(err, http.ErrNoCookie) {
				return nil, nil
			}
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
		handleCallback: func(w http.ResponseWriter, r *http.Request) (string, error) {
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

			http.SetCookie(w, &http.Cookie{
				Name:     "rebuy-go-sdk-auth",
				Value:    base64.RawURLEncoding.EncodeToString(jsonPayload),
				Path:     "/",
				Expires:  time.Now().Add(7 * 24 * time.Hour),
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			redirectURI := r.PostFormValue("redirect_uri")
			redirectURI = validateRedirectURI(redirectURI)

			return redirectURI, nil
		},
	}

	return m.handler
}

// AuthLoginURL generates a safe login URL with a redirect parameter pointing
// to the current request path. The redirect path is validated to ensure it's
// a safe relative path. If the path is invalid or would redirect to root,
// returns a login URL without a redirect parameter.
//
// This function is designed for use with Templ templates and returns a
// templ.SafeURL that can be used directly in href attributes without additional
// escaping.
//
// Example usage in Templ:
//
//	<a href={ webutil.AuthLoginURL(request) }>Login</a>
func AuthLoginURL(r *http.Request) templ.SafeURL {
	requestURI := r.URL.RequestURI()

	// Validate the redirect path
	validatedPath := validateRedirectURI(requestURI)

	// Build URL using url.URL for proper construction
	u := &url.URL{
		Path: "/auth/login",
	}

	// If validation returned "/" (default for invalid paths), don't add redirect param
	if validatedPath != "/" {
		q := url.Values{}
		q.Set("redirect", validatedPath)
		u.RawQuery = q.Encode()
	}

	return templ.SafeURL(u.String())
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

type cookieEncrypter[T any] struct {
	block cipher.Block
}

func newCookieEncrypter[T any](key string) (*cookieEncrypter[T], error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	return &cookieEncrypter[T]{
		block: block,
	}, nil
}

func (e cookieEncrypter[T]) cookieName() string {
	return cmdutil.Name + "-token"
}

func (e cookieEncrypter[T]) WriteCookie(w http.ResponseWriter, obj *T) error {
	cookieValue, err := e.Encrypt(obj)
	if err != nil {
		return fmt.Errorf("encrypt cookie: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     e.cookieName(),
		Value:    cookieValue,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

func (e cookieEncrypter[T]) ReadCookie(r *http.Request) (*T, error) {
	cookie, err := r.Cookie(e.cookieName())
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get auth cookie: %w", err)
	}

	token, err := e.Decrypt(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}

	return token, nil
}

func (e cookieEncrypter[T]) Encrypt(obj *T) (string, error) {
	payload, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	encrypted, err := e.EncryptBytes(payload)
	if err != nil {
		return "", err
	}

	return base64.RawStdEncoding.EncodeToString(encrypted), nil
}

func (e cookieEncrypter[T]) EncryptBytes(data []byte) ([]byte, error) {
	iv := make([]byte, aes.BlockSize)
	_, err := io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCTR(e.block, iv)
	cipherText := make([]byte, len(data))
	stream.XORKeyStream(cipherText, data)

	return append(iv, cipherText...), nil
}

func (e cookieEncrypter[T]) Decrypt(value string) (*T, error) {
	encrypted, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}

	payload, err := e.DecryptBytes(encrypted)
	if err != nil {
		return nil, err
	}

	var result T
	err = json.Unmarshal(payload, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (e cookieEncrypter[T]) DecryptBytes(encryptedData []byte) ([]byte, error) {
	if len(encryptedData) < aes.BlockSize {
		return nil, fmt.Errorf("encrypted data too short: got %d bytes, need at least %d", len(encryptedData), aes.BlockSize)
	}

	iv := encryptedData[:aes.BlockSize]
	stream := cipher.NewCTR(e.block, iv)

	plainText := make([]byte, len(encryptedData)-aes.BlockSize)
	stream.XORKeyStream(plainText, encryptedData[aes.BlockSize:])

	return plainText, nil
}
