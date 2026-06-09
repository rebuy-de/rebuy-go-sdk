package chutil

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/migrations/*.sql
var testMigrations embed.FS

func TestFilterRepeatableSQLFiles(t *testing.T) {
	entries, err := testMigrations.ReadDir("testdata/migrations")
	require.NoError(t, err)

	got := filterRepeatableSQLFiles(entries)

	// Only the R_ prefixed .sql file is repeatable; the versioned 0001 file is
	// left for golang-migrate to handle.
	assert.Equal(t, []string{"R_001_events_view.sql"}, got)
}

func TestCalculateRepeatableFileHash(t *testing.T) {
	a := calculateRepeatableFileHash([]byte("select 1"))
	again := calculateRepeatableFileHash([]byte("select 1"))
	different := calculateRepeatableFileHash([]byte("select 2"))

	assert.Equal(t, a, again, "hash must be stable for identical content")
	assert.NotEqual(t, a, different, "hash must change with content")
	assert.Len(t, a, 64, "sha256 hex digest is 64 chars")
}

func TestRepeatableNeedsApply(t *testing.T) {
	// Never applied before: must run regardless of the (empty) stored hash.
	assert.True(t, repeatableNeedsApply("", false, "abc"))
	// Applied, content unchanged: skip.
	assert.False(t, repeatableNeedsApply("abc", true, "abc"))
	// Applied, content changed: re-run.
	assert.True(t, repeatableNeedsApply("abc", true, "def"))
}

func TestQuoteIdentifier(t *testing.T) {
	assert.Equal(t, "`analytics`", quoteIdentifier("analytics"))
	// Embedded backticks are doubled so a name cannot break out of the quoting.
	assert.Equal(t, "`a``b`", quoteIdentifier("a`b"))
}
