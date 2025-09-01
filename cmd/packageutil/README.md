# PackageUtil

PackageUtil is a command-line tool for creating distribution packages from Go binaries built with buildutil. It handles the packaging functionality that was removed from buildutil in commit 79fc0d5.

## Usage

```bash
packageutil [flags] binary-file1 [binary-file2 ...]
```

### Global Flags

- `--compressed` - Creates .tgz artifacts for POSIX targets and .zip for windows
- `--rpm` - Creates .rpm artifacts for linux targets  
- `--deb` - Creates .deb artifacts for linux targets
- `--s3-url <url>` - S3 base URL for uploads (e.g., s3://bucket/path/)
- `-v, --verbose` - Enable verbose logging

### Binary File Arguments

PackageUtil requires you to specify the binary files to package as command-line arguments. Each file must follow the naming pattern:
```
name-version-os-arch[.exe]
```

For example:
- `dist/buildutil-v9.3.0+dirty-linux-amd64`
- `dist/myapp-v1.0.0-windows-amd64.exe`

**Symlink Support**: If a binary file is a symlink, packageutil will follow the symlink and use the target file's name for parsing version and architecture information.

### Package Formats

#### Compressed Archives
- **TGZ** (Linux/macOS): Creates `.tar.gz` files containing all binaries for the target system
- **ZIP** (Windows): Creates `.zip` files containing all binaries for Windows

#### System Packages  
- **RPM**: Creates `.rpm` packages for Red Hat-based Linux distributions
- **DEB**: Creates `.deb` packages for Debian-based Linux distributions

All system packages install binaries to `/usr/bin/` and include proper metadata with the maintainer set to "rebuy Platform Team <dl-scb-tech-platform@rebuy.com>".

**Important**: RPM and DEB packages can only be created from Linux binaries. If you specify `--rpm` or `--deb` flags but provide non-Linux binaries, packageutil will exit with a validation error.

### S3 Upload

When `--s3-url` is provided, packageutil uploads all created artifacts to S3 with appropriate tags:
- `System`: Target system (e.g., "linux/amd64")  
- `Kind`: Package type (e.g., "tgz", "rpm", "deb")

## Examples

```bash
# Create compressed archives for specific binaries
./packageutil --compressed dist/myapp-v1.0.0-linux-amd64 dist/myapp-v1.0.0-darwin-arm64

# Create all package types for Linux binaries
./packageutil --compressed --rpm --deb dist/myapp-v1.0.0-linux-amd64

# Create packages and upload to S3
./packageutil --compressed --rpm --s3-url s3://mybucket/releases/ dist/myapp-v1.0.0-linux-*

# Using with symlinks (useful for CI/CD)
ln -s myapp-v1.0.0-linux-amd64 dist/latest-linux
./packageutil --compressed dist/latest-linux

# Verbose output
./packageutil --compressed --verbose dist/myapp-v1.0.0-*
```

## Error Handling

PackageUtil performs validation to ensure compatibility between package formats and binary platforms:

### Common Validation Errors

```bash
# Trying to create RPM from non-Linux binary
$ ./packageutil --rpm dist/myapp-darwin-amd64
error: package format validation failed:
- RPM packages can only be created from Linux binaries, but no Linux binaries were provided

# Trying to create DEB from Windows binary  
$ ./packageutil --deb dist/myapp-windows-amd64.exe
error: package format validation failed:
- DEB packages can only be created from Linux binaries, but no Linux binaries were provided

# Multiple validation errors
$ ./packageutil --rpm --deb dist/myapp-darwin-amd64
error: package format validation failed:
- RPM packages can only be created from Linux binaries, but no Linux binaries were provided
- DEB packages can only be created from Linux binaries, but no Linux binaries were provided
```

### Solutions

- **For RPM/DEB packages**: Provide Linux binaries (`*-linux-*` files)
- **For compressed archives**: Any platform works (creates appropriate .tgz or .zip files)
- **Mixed platforms**: Specify the appropriate package formats for each platform

```bash
# Correct: Create RPM from Linux binary
./packageutil --rpm dist/myapp-v1.0.0-linux-amd64

# Correct: Create compressed archives from any platform
./packageutil --compressed dist/myapp-v1.0.0-darwin-amd64 dist/myapp-v1.0.0-windows-amd64.exe
```

## Integration with buildutil

1. Use `buildutil` to build your Go binaries
2. Use `packageutil` to create distribution packages from those binaries
3. Upload packages to your distribution channels

### Basic Workflow

```bash
# Build phase - create binaries for multiple platforms
./buildutil -x linux/amd64 -x windows/amd64 -x darwin/amd64

# Package phase - create appropriate packages for each platform
# Note: RPM/DEB only work with Linux binaries
./packageutil --compressed dist/myapp-v*                           # Archives for all platforms
./packageutil --rpm --deb dist/myapp-v*-linux-amd64              # System packages for Linux only
./packageutil --compressed --s3-url s3://releases/ dist/myapp-v*  # Upload all archives
```

### Advanced Workflow with Platform-Specific Packaging

```bash
# Build for multiple platforms
./buildutil -x linux/amd64 -x linux/arm64 -x darwin/amd64 -x windows/amd64

# Create all Linux packages (archives + system packages)
./packageutil --compressed --rpm --deb dist/myapp-v*-linux-*

# Create archives for non-Linux platforms
./packageutil --compressed dist/myapp-v*-darwin-* dist/myapp-v*-windows-*

# Or combine everything with appropriate validation
./packageutil --compressed dist/myapp-v*  # Works for all platforms
```

### CI/CD Integration

```bash
# Example CI/CD pipeline step
#!/bin/bash
set -e

# Build all target platforms
./buildutil -x linux/amd64 -x darwin/amd64 -x windows/amd64

# Package Linux binaries with system packages
./packageutil --compressed --rpm --deb --s3-url s3://releases/linux/ dist/*-linux-*

# Package other platforms with archives only  
./packageutil --compressed --s3-url s3://releases/darwin/ dist/*-darwin-*
./packageutil --compressed --s3-url s3://releases/windows/ dist/*-windows-*
```
