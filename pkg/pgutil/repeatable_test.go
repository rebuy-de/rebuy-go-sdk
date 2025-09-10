package pgutil

import (
	"io/fs"
	"testing"
)

func TestFilterRepeatableSQLFiles(t *testing.T) {
	cases := []struct {
		name     string
		entries  []fs.DirEntry
		expected []string
	}{
		{
			name: "filters repeatable files correctly",
			entries: []fs.DirEntry{
				mockDirEntry{name: "001_initial.sql", isDir: false},
				mockDirEntry{name: "R_001_create_view.sql", isDir: false},
				mockDirEntry{name: "R_002_seed_data.sql", isDir: false},
				mockDirEntry{name: "003_add_index.sql", isDir: false},
				mockDirEntry{name: "README.md", isDir: false},
				mockDirEntry{name: "migrations", isDir: true},
				mockDirEntry{name: "R_003_update_view.sql", isDir: false},
			},
			expected: []string{
				"R_001_create_view.sql",
				"R_002_seed_data.sql",
				"R_003_update_view.sql",
			},
		},
		{
			name:     "handles empty directory",
			entries:  []fs.DirEntry{},
			expected: []string{},
		},
		{
			name: "handles no repeatable files",
			entries: []fs.DirEntry{
				mockDirEntry{name: "001_initial.sql", isDir: false},
				mockDirEntry{name: "002_add_table.sql", isDir: false},
			},
			expected: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterRepeatableSQLFiles(tc.entries)

			if len(result) != len(tc.expected) {
				t.Fatalf("Expected %d files, got %d", len(tc.expected), len(result))
			}

			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("Expected file %s at index %d, got %s", expected, i, result[i])
				}
			}
		})
	}
}

func TestCalculateRepeatableFileHash(t *testing.T) {
	cases := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "empty file",
			content:  []byte(""),
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "simple content",
			content:  []byte("SELECT 1;"),
			expected: "17db4fd369edb9244b9f91d9aeed145c3d04ad8ba6e95d06247f07a63527d11a",
		},
		{
			name:     "multiline content",
			content:  []byte("CREATE VIEW test AS\nSELECT * FROM users;"),
			expected: "0fa3b2be6d08449a847089b9b8a4e504ae69004d1c50b68d79bc517ffff02432",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateRepeatableFileHash(tc.content)
			if result != tc.expected {
				t.Errorf("Expected hash %s, got %s", tc.expected, result)
			}
		})
	}
}

// mockDirEntry implements fs.DirEntry for testing
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode          { return 0 }
func (m mockDirEntry) Info() (fs.FileInfo, error) { return nil, nil }
