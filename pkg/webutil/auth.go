package webutil

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"html/template"
	"net/http"
	"path"
	"time"

	"github.com/google/go-github/v50/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	oauth2_github "golang.org/x/oauth2/github"
)

const (
	authStateCookie = `oauthstate`
	authSessionName = `oauth`
)

// AuthInfo contains data about the currently logged in user.
type AuthInfo struct {
	GitHubToken string
	UpdatedAt   time.Time

	Login string
	Name  string
	Teams []string
}

// InTeam returns true, if the user is in the given team. The team name needs to
// be whitelisted in the AuthMiddleware, otherwise it will return false even if
// the user is in the team.
func (i AuthInfo) InTeam(want string) bool {
	for _, have := range i.Teams {
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
}

// AuthMiddleware is an HTTP request middleware that adds login endpoints. The
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
func AuthMiddleware(creds AuthConfig, teams ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return authMiddlewareFunc(next, creds, teams...)
	}
}

func authMiddlewareFunc(next http.Handler, creds AuthConfig, teams ...string) http.Handler {
	mw := authMiddleware{
		next:  next,
		teams: map[string]struct{}{},
		config: &oauth2.Config{
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			Scopes:       []string{"user"},
			Endpoint:     oauth2_github.Endpoint,
		},
	}

	for _, team := range teams {
		mw.teams[team] = struct{}{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", mw.handleLogin)
	mux.HandleFunc("/auth/callback", mw.handleCallback)
	mux.HandleFunc("/", mw.handleDefault)

	return mux
}

type authMiddleware struct {
	next   http.Handler
	teams  map[string]struct{}
	config *oauth2.Config
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
	var expiration = time.Now().Add(365 * 24 * time.Hour)

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
		logrus.Warn("invalid oauth google state")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	token, err := mw.config.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Error("failed to exchange token")
		return
	}

	err = mw.refreshSessionData(w, r, &token.AccessToken)
	if err != nil {
		logrus.WithError(errors.WithStack(err)).Error("failed to refresh session data")
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func (mw *authMiddleware) refreshSessionData(w http.ResponseWriter, r *http.Request, optionalToken *string) error {
	session, err := SessionFromRequest(r)
	if err != nil {
		return errors.WithStack(err)
	}

	info, ok := session.Values["auth-info"].(AuthInfo)
	if !ok {
		info = AuthInfo{}
	}
	if optionalToken != nil {
		info.GitHubToken = *optionalToken
	}
	if info.GitHubToken == "" {
		return errors.Errorf("GitHub token not found")
	}

	client := github.NewClient(
		oauth2.NewClient(r.Context(), oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: info.GitHubToken},
		)),
	)

	user, _, err := client.Users.Get(r.Context(), "")
	if err != nil {
		return errors.WithStack(err)
	}

	ghTeams, _, err := client.Teams.ListUserTeams(r.Context(), nil)
	if err != nil {
		return errors.WithStack(err)
	}

	cachedTeams := []string{}
	for _, team := range ghTeams {
		name := path.Join(team.GetOrganization().GetLogin(), team.GetSlug())
		_, ok := mw.teams[name]
		if ok {
			cachedTeams = append(cachedTeams, name)
		}
	}

	if len(cachedTeams) == 0 {
		return errors.Errorf("login failed: user is not part of any required team")
	}

	info.Login = user.GetLogin()
	info.Name = user.GetName()
	info.UpdatedAt = time.Now()
	info.Teams = cachedTeams

	session.Values["auth-info"] = info
	err = session.Save(r, w)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
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
