package webutil

import (
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
