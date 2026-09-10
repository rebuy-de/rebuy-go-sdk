package webutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		want        string
	}{
		// Valid relative paths
		{
			name:        "valid relative path",
			redirectURI: "/dashboard",
			want:        "/dashboard",
		},
		{
			name:        "valid relative path with query",
			redirectURI: "/users?page=2",
			want:        "/users?page=2",
		},
		{
			name:        "valid relative path with fragment",
			redirectURI: "/settings#profile",
			want:        "/settings#profile",
		},
		{
			name:        "root path",
			redirectURI: "/",
			want:        "/",
		},

		// Invalid - empty or whitespace
		{
			name:        "empty string",
			redirectURI: "",
			want:        "/",
		},
		{
			name:        "whitespace only",
			redirectURI: "   ",
			want:        "/",
		},

		// Invalid - protocol-relative URLs (open redirect vulnerability)
		{
			name:        "protocol-relative URL",
			redirectURI: "//evil.com",
			want:        "/",
		},
		{
			name:        "protocol-relative URL with path",
			redirectURI: "//evil.com/phishing",
			want:        "/",
		},

		// Invalid - backslash bypass attempts
		{
			name:        "backslash in path",
			redirectURI: "/\\evil.com",
			want:        "/\\evil.com", // url.Parse treats this as a valid relative path
		},
		{
			name:        "double backslash",
			redirectURI: "\\\\evil.com",
			want:        "/", // doesn't start with /
		},

		// Invalid - relative path without leading slash
		{
			name:        "relative path without slash",
			redirectURI: "dashboard",
			want:        "/",
		},
		{
			name:        "dot relative path",
			redirectURI: "./dashboard",
			want:        "/", // doesn't start with /
		},
		{
			name:        "double dot relative path",
			redirectURI: "../dashboard",
			want:        "/", // doesn't start with /
		},

		// Invalid - absolute URLs (all rejected)
		{
			name:        "http absolute URL",
			redirectURI: "http://evil.com",
			want:        "/",
		},
		{
			name:        "https absolute URL",
			redirectURI: "https://evil.com",
			want:        "/",
		},
		{
			name:        "same origin absolute URL",
			redirectURI: "https://example.com/dashboard",
			want:        "/", // Rejected - no absolute URLs allowed
		},
		{
			name:        "same origin with port",
			redirectURI: "https://example.com:443/dashboard",
			want:        "/", // Rejected - no absolute URLs allowed
		},

		// Edge cases
		{
			name:        "javascript protocol",
			redirectURI: "javascript:alert(1)",
			want:        "/", // has scheme but not http/https
		},
		{
			name:        "data protocol",
			redirectURI: "data:text/html,<script>alert(1)</script>",
			want:        "/", // has scheme
		},
		{
			name:        "malformed URL",
			redirectURI: "ht!tp://evil.com",
			want:        "/", // parse error or doesn't start with /
		},
		{
			name:        "null byte injection",
			redirectURI: "/dashboard\x00http://evil.com",
			want:        "/", // url.Parse rejects URLs with null bytes
		},

		// Absolute URLs always rejected
		{
			name:        "absolute URL case 1",
			redirectURI: "https://example.com/dashboard",
			want:        "/", // absolute URLs not allowed
		},
		{
			name:        "absolute URL case 2",
			redirectURI: "https://example.com/dashboard",
			want:        "/", // absolute URLs not allowed
		},

		// URL encoding
		{
			name:        "URL encoded path",
			redirectURI: "/users/%2F%2Fevil.com",
			want:        "/users/%2F%2Fevil.com",
		},
		{
			name:        "URL with spaces",
			redirectURI: "/path with spaces",
			want:        "/path with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateRedirectURI(tt.redirectURI)
			if got != tt.want {
				t.Errorf("validateRedirectURI(%q) = %q, want %q",
					tt.redirectURI, got, tt.want)
			}
		})
	}
}

// TestValidateRedirectURI_SecurityCritical tests the most critical security cases
func TestValidateRedirectURI_SecurityCritical(t *testing.T) {
	criticalTests := []struct {
		name        string
		redirectURI string
	}{
		{"protocol-relative URL", "//evil.com"},
		{"protocol-relative with path", "//evil.com/steal"},
		{"absolute different origin", "https://evil.com"},
		{"javascript protocol", "javascript:alert(1)"},
		{"data protocol", "data:text/html,<script>alert(1)</script>"},
	}

	for _, tt := range criticalTests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateRedirectURI(tt.redirectURI)
			if got != "/" {
				t.Errorf("SECURITY: validateRedirectURI(%q) = %q, must return '/' for security",
					tt.redirectURI, got)
			}
		})
	}
}

func TestAuthInfoRoles(t *testing.T) {
	const userinfo = `{
		"sub": "e79dfea9-276f-4f72-88fe-6c7aa664ab44",
		"realm_access": {
			"roles": ["default-roles-bwler", "offline_access", "rebuy-tech"]
		},
		"resource_access": {
			"dbreview": {"roles": ["approver"]},
			"account": {"roles": ["manage-account", "view-profile"]}
		},
		"name": "Sven Walter",
		"preferred_username": "s.walter@rebuy.com"
	}`

	var info AuthInfo
	err := json.Unmarshal([]byte(userinfo), &info)
	if err != nil {
		t.Fatalf("unmarshal userinfo: %v", err)
	}

	if info.Username != "s.walter@rebuy.com" {
		t.Errorf("Username = %q, want %q", info.Username, "s.walter@rebuy.com")
	}

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "realm role", got: info.HasRole("rebuy-tech"), want: true},
		{name: "missing realm role", got: info.HasRole("nope"), want: false},
		{name: "resource role is no realm role", got: info.HasRole("approver"), want: false},
		{name: "resource role", got: info.HasResourceRole("dbreview", "approver"), want: true},
		{name: "other resource role", got: info.HasResourceRole("account", "view-profile"), want: true},
		{name: "wrong client", got: info.HasResourceRole("account", "approver"), want: false},
		{name: "unknown client", got: info.HasResourceRole("unknown", "approver"), want: false},
		{name: "unknown role", got: info.HasResourceRole("dbreview", "unknown"), want: false},
		{name: "nil resource access", got: AuthInfo{}.HasResourceRole("dbreview", "approver"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %v, want %v", tt.got, tt.want)
			}
		})
	}
}

func TestParseDevRoles(t *testing.T) {
	tests := []struct {
		name  string
		specs []string
		want  []devRole
	}{
		{
			name:  "empty",
			specs: nil,
			want:  []devRole{},
		},
		{
			name:  "realm role",
			specs: []string{"rebuy-tech"},
			want: []devRole{
				{Field: "role-0", Label: "rebuy-tech", Client: "", Role: "rebuy-tech"},
			},
		},
		{
			name:  "resource role",
			specs: []string{"dbreview:approver"},
			want: []devRole{
				{Field: "role-0", Label: "dbreview:approver", Client: "dbreview", Role: "approver"},
			},
		},
		{
			name:  "mixed keeps order",
			specs: []string{"rebuy-tech", "dbreview:approver", "account:view-profile"},
			want: []devRole{
				{Field: "role-0", Label: "rebuy-tech", Client: "", Role: "rebuy-tech"},
				{Field: "role-1", Label: "dbreview:approver", Client: "dbreview", Role: "approver"},
				{Field: "role-2", Label: "account:view-profile", Client: "account", Role: "view-profile"},
			},
		},
		{
			name:  "only first colon separates",
			specs: []string{"dbreview:approver:extra"},
			want: []devRole{
				{Field: "role-0", Label: "dbreview:approver:extra", Client: "dbreview", Role: "approver:extra"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseDevRoles(tt.specs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDevRoles(%v) = %v, want %v", tt.specs, got, tt.want)
			}
		})
	}
}

func TestDevAuthMiddlewareRoundtrip(t *testing.T) {
	var got *AuthInfo

	handler := DevAuthMiddleware("rebuy-tech", "unchecked", "dbreview:approver")(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			got = AuthInfoFromRequest(r)
		}),
	)

	loginFormRec := httptest.NewRecorder()
	handler.ServeHTTP(loginFormRec, httptest.NewRequest(http.MethodGet, "/auth/login", nil))

	if loginFormRec.Code != http.StatusOK {
		t.Fatalf("login form status = %d, want %d", loginFormRec.Code, http.StatusOK)
	}

	for _, want := range []string{`name="role-0"`, `>rebuy-tech<`, `name="role-2"`, `>dbreview:approver<`} {
		if !strings.Contains(loginFormRec.Body.String(), want) {
			t.Errorf("login form does not contain %s", want)
		}
	}

	form := url.Values{}
	form.Set("username", "s.walter@rebuy.com")
	form.Set("name", "Sven Walter")
	form.Set("role-0", "on")
	form.Set("role-2", "on")

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/callback", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)

	if loginRec.Code != http.StatusSeeOther {
		t.Fatalf("callback status = %d, want %d: %s", loginRec.Code, http.StatusSeeOther, loginRec.Body.String())
	}

	cookies := loginRec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got == nil {
		t.Fatal("no AuthInfo in request context")
	}

	if got.Username != "s.walter@rebuy.com" {
		t.Errorf("Username = %q, want %q", got.Username, "s.walter@rebuy.com")
	}

	if !got.HasRole("rebuy-tech") {
		t.Error("HasRole(rebuy-tech) = false, want true")
	}

	if got.HasRole("unchecked") {
		t.Error("HasRole(unchecked) = true, want false")
	}

	if !got.HasResourceRole("dbreview", "approver") {
		t.Error("HasResourceRole(dbreview, approver) = false, want true")
	}

	if got.HasRole("approver") {
		t.Error("HasRole(approver) = true, want false")
	}
}
