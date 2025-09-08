package pgutil

import (
	"fmt"
	"net/url"
)

// BuildURI constructs a PostgreSQL connection URI with credentials.
// This replaces the identical URI() function duplicated across all projects.
//
// Parameters:
//   - base: Base connection URI (e.g., "postgres://localhost:5432/mydb")
//   - username: Database username
//   - password: Database password
//
// Returns a complete URI with embedded credentials.
//
// Example usage:
//
//	uri, err := sqlutil.BuildURI("postgres://localhost:5432/mydb", "user", "pass")
//	// Returns: "postgres://user:pass@localhost:5432/mydb"
func BuildURI(base, username, password string) (string, error) {
	dbURI, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse base URI: %w", err)
	}

	dbURI.User = url.UserPassword(username, password)
	return dbURI.String(), nil
}

// ParseCredentials extracts username and password from a PostgreSQL URI.
// This is useful for separating credentials from connection details.
//
// Returns empty strings if no credentials are present in the URI.
func ParseCredentials(uri string) (username, password string, err error) {
	parsedURI, err := url.Parse(uri)
	if err != nil {
		return "", "", fmt.Errorf("parse URI: %w", err)
	}

	if parsedURI.User == nil {
		return "", "", nil
	}

	username = parsedURI.User.Username()
	password, _ = parsedURI.User.Password()
	return username, password, nil
}
