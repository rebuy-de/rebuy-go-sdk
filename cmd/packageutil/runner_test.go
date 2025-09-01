package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBinaryName(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     *BinaryInfo
		wantErr  string
	}{
		{
			name:     "basic linux binary",
			filename: "myapp-v1.0.0-linux-amd64",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "linux",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "windows binary with exe",
			filename: "myapp-v1.0.0-windows-amd64.exe",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "windows",
					Arch: "amd64",
					Ext:  ".exe",
				},
			},
		},
		{
			name:     "darwin arm64 binary",
			filename: "myapp-v2.1.3-darwin-arm64",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v2.1.3",
				System: SystemInfo{
					OS:   "darwin",
					Arch: "arm64",
					Ext:  "",
				},
			},
		},
		{
			name:     "complex version with dirty tag",
			filename: "buildutil-v9.3.0+dirty-linux-amd64",
			want: &BinaryInfo{
				Name:    "buildutil",
				Version: "v9.3.0+dirty",
				System: SystemInfo{
					OS:   "linux",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "version with rc tag",
			filename: "myapp-v1.2.3-rc1-linux-amd64",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.2.3-rc1",
				System: SystemInfo{
					OS:   "linux",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "freebsd binary",
			filename: "myapp-v1.0.0-freebsd-amd64",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "freebsd",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "openbsd binary",
			filename: "myapp-v1.0.0-openbsd-amd64",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "openbsd",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "netbsd binary",
			filename: "myapp-v1.0.0-netbsd-amd64",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "netbsd",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "arm architecture",
			filename: "myapp-v1.0.0-linux-arm",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "linux",
					Arch: "arm",
					Ext:  "",
				},
			},
		},
		{
			name:     "386 architecture",
			filename: "myapp-v1.0.0-windows-386.exe",
			want: &BinaryInfo{
				Name:    "myapp",
				Version: "v1.0.0",
				System: SystemInfo{
					OS:   "windows",
					Arch: "386",
					Ext:  ".exe",
				},
			},
		},
		{
			name:     "application with dashes and no version",
			filename: "keydb-inspector-linux-amd64",
			want: &BinaryInfo{
				Name:    "keydb-inspector",
				Version: "",
				System: SystemInfo{
					OS:   "linux",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		{
			name:     "application with dashes and complex version",
			filename: "keydb-inspector-v1.0.1+dirty.57.5b2c4cf-linux-amd64",
			want: &BinaryInfo{
				Name:    "keydb-inspector",
				Version: "v1.0.1+dirty.57.5b2c4cf",
				System: SystemInfo{
					OS:   "linux",
					Arch: "amd64",
					Ext:  "",
				},
			},
		},
		// Error cases
		{
			name:     "insufficient parts",
			filename: "myapp-v1.0.0",
			wantErr:  "insufficient parts in filename (expected at least 4, got 2)",
		},
		{
			name:     "insufficient parts with 3",
			filename: "myapp-v1.0.0-linux",
			wantErr:  "insufficient parts in filename (expected at least 4, got 3)",
		},
		{
			name:     "unrecognized OS",
			filename: "myapp-v1.0.0-unknown-amd64",
			wantErr:  "unrecognized OS: unknown",
		},
		{
			name:     "empty filename",
			filename: "",
			wantErr:  "insufficient parts in filename (expected at least 4, got 1)",
		},
		{
			name:     "no dashes",
			filename: "myapp",
			wantErr:  "insufficient parts in filename (expected at least 4, got 1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBinaryName(tt.filename)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want.Name, got.Name)
			assert.Equal(t, tt.want.Version, got.Version)
			assert.Equal(t, tt.want.System.OS, got.System.OS)
			assert.Equal(t, tt.want.System.Arch, got.System.Arch)
			assert.Equal(t, tt.want.System.Ext, got.System.Ext)
		})
	}
}

func TestIsValidOS(t *testing.T) {
	tests := []struct {
		name string
		os   string
		want bool
	}{
		{"linux", "linux", true},
		{"darwin", "darwin", true},
		{"windows", "windows", true},
		{"freebsd", "freebsd", true},
		{"openbsd", "openbsd", true},
		{"netbsd", "netbsd", true},
		{"unknown", "unknown", false},
		{"empty", "", false},
		{"solaris", "solaris", false},
		{"plan9", "plan9", false},
		{"Linux", "Linux", false},     // case sensitive
		{"WINDOWS", "WINDOWS", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidOS(tt.os)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveSymlink(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "packageutil_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	runner := &Runner{}

	t.Run("regular file", func(t *testing.T) {
		regularFile := filepath.Join(tempDir, "regular")
		err := os.WriteFile(regularFile, []byte("test"), 0755)
		require.NoError(t, err)

		resolved, err := runner.resolveSymlink(regularFile)
		require.NoError(t, err)
		assert.Equal(t, regularFile, resolved)
	})

	t.Run("symlink", func(t *testing.T) {
		targetFile := filepath.Join(tempDir, "target-v1.0.0-linux-amd64")
		err := os.WriteFile(targetFile, []byte("test"), 0755)
		require.NoError(t, err)

		symlinkFile := filepath.Join(tempDir, "symlink")
		err = os.Symlink(targetFile, symlinkFile)
		require.NoError(t, err)

		resolved, err := runner.resolveSymlink(symlinkFile)
		require.NoError(t, err)
		assert.Equal(t, targetFile, resolved)
	})

	t.Run("nonexistent file", func(t *testing.T) {
		nonexistent := filepath.Join(tempDir, "nonexistent")
		_, err := runner.resolveSymlink(nonexistent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lstat file")
	})

	t.Run("broken symlink", func(t *testing.T) {
		brokenSymlink := filepath.Join(tempDir, "broken")
		err := os.Symlink("/nonexistent/target", brokenSymlink)
		require.NoError(t, err)

		_, err = runner.resolveSymlink(brokenSymlink)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve symlink")
	})
}

func TestDiscoverBinariesWithArgs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "packageutil_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()
	runner := &Runner{}

	t.Run("valid binary files", func(t *testing.T) {
		binary1 := filepath.Join(tempDir, "myapp-v1.0.0-linux-amd64")
		binary2 := filepath.Join(tempDir, "myapp-v1.0.0-darwin-arm64")

		err := os.WriteFile(binary1, []byte("test"), 0755)
		require.NoError(t, err)
		err = os.WriteFile(binary2, []byte("test"), 0755)
		require.NoError(t, err)

		err = runner.discoverBinaries(ctx, []string{binary1, binary2})
		require.NoError(t, err)

		require.Len(t, runner.Binaries, 2)
		assert.Equal(t, "myapp", runner.Binaries[0].Name)
		assert.Equal(t, "v1.0.0", runner.Binaries[0].Version)
		assert.Equal(t, "linux", runner.Binaries[0].System.OS)
		assert.Equal(t, "amd64", runner.Binaries[0].System.Arch)
	})

	t.Run("with symlink", func(t *testing.T) {
		targetFile := filepath.Join(tempDir, "target-v2.0.0-windows-amd64.exe")
		symlinkFile := filepath.Join(tempDir, "symlink.exe")

		err := os.WriteFile(targetFile, []byte("test"), 0755)
		require.NoError(t, err)
		err = os.Symlink(targetFile, symlinkFile)
		require.NoError(t, err)

		runner = &Runner{} // Reset runner
		err = runner.discoverBinaries(ctx, []string{symlinkFile})
		require.NoError(t, err)

		require.Len(t, runner.Binaries, 1)
		assert.Equal(t, "target", runner.Binaries[0].Name)
		assert.Equal(t, "v2.0.0", runner.Binaries[0].Version)
		assert.Equal(t, "windows", runner.Binaries[0].System.OS)
		assert.Equal(t, "amd64", runner.Binaries[0].System.Arch)
		assert.Equal(t, ".exe", runner.Binaries[0].System.Ext)
		assert.Equal(t, targetFile, runner.Binaries[0].Path)
	})

	t.Run("no arguments", func(t *testing.T) {
		runner = &Runner{}
		err = runner.discoverBinaries(ctx, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no target binary files specified")
	})

	t.Run("invalid binary name", func(t *testing.T) {
		invalidFile := filepath.Join(tempDir, "invalid-name")
		err := os.WriteFile(invalidFile, []byte("test"), 0755)
		require.NoError(t, err)

		runner = &Runner{}
		err = runner.discoverBinaries(ctx, []string{invalidFile})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse binary name")
	})

	t.Run("non-executable file", func(t *testing.T) {
		nonExecFile := filepath.Join(tempDir, "nonexec-v1.0.0-linux-amd64")
		err := os.WriteFile(nonExecFile, []byte("test"), 0644) // Not executable
		require.NoError(t, err)

		runner = &Runner{}
		err = runner.discoverBinaries(ctx, []string{nonExecFile})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not executable")
	})
}

func TestValidatePackageCompatibility(t *testing.T) {
	tests := []struct {
		name      string
		binaries  []BinaryInfo
		createRPM bool
		createDEB bool
		wantErr   string
	}{
		{
			name: "valid linux binary for RPM",
			binaries: []BinaryInfo{
				{System: SystemInfo{OS: "linux", Arch: "amd64"}},
			},
			createRPM: true,
			wantErr:   "",
		},
		{
			name: "invalid darwin binary for RPM",
			binaries: []BinaryInfo{
				{System: SystemInfo{OS: "darwin", Arch: "amd64"}},
			},
			createRPM: true,
			wantErr:   "RPM packages can only be created from Linux binaries, but no Linux binaries were provided",
		},
		{
			name: "invalid windows binary for DEB",
			binaries: []BinaryInfo{
				{System: SystemInfo{OS: "windows", Arch: "amd64"}},
			},
			createDEB: true,
			wantErr:   "DEB packages can only be created from Linux binaries, but no Linux binaries were provided",
		},
		{
			name: "multiple errors",
			binaries: []BinaryInfo{
				{System: SystemInfo{OS: "darwin", Arch: "amd64"}},
			},
			createRPM: true,
			createDEB: true,
			wantErr:   "package format validation failed:\n- RPM packages can only be created from Linux binaries, but no Linux binaries were provided\n- DEB packages can only be created from Linux binaries, but no Linux binaries were provided",
		},
		{
			name: "no validation needed",
			binaries: []BinaryInfo{
				{System: SystemInfo{OS: "darwin", Arch: "amd64"}},
			},
			createRPM: false,
			createDEB: false,
			wantErr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &Runner{
				Parameters: PackageParameters{
					CreateRPM: tt.createRPM,
					CreateDEB: tt.createDEB,
				},
				Binaries: tt.binaries,
			}

			err := runner.validatePackageCompatibility()

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}
