# PackageUtil

PackageUtil is a command-line tool for creating distribution packages from Go binaries built with buildutil. It handles the packaging functionality that was removed from buildutil in commit 79fc0d5.

## Usage

```bash
packageutil [flags]
```

### Global Flags

- `--compressed` - Creates .tgz artifacts for POSIX targets and .zip for windows
- `--rpm` - Creates .rpm artifacts for linux targets  
- `--deb` - Creates .deb artifacts for linux targets
- `--s3-url <url>` - S3 base URL for uploads (e.g., s3://bucket/path/)
- `-v, --verbose` - Enable verbose logging

### Discovery

PackageUtil automatically discovers binaries in the `dist/` directory by looking for files with the naming pattern:
```
name-version-os-arch[.exe]
```

For example:
- `buildutil-v9.3.0+dirty-linux-amd64`
- `myapp-v1.0.0-windows-amd64.exe`

### Package Formats

#### Compressed Archives
- **TGZ** (Linux/macOS): Creates `.tar.gz` files containing all binaries for the target system
- **ZIP** (Windows): Creates `.zip` files containing all binaries for Windows

#### System Packages  
- **RPM**: Creates `.rpm` packages for Red Hat-based Linux distributions
- **DEB**: Creates `.deb` packages for Debian-based Linux distributions

All system packages install binaries to `/usr/bin/` and include proper metadata with the maintainer set to "rebuy Platform Team <dl-scb-tech-platform@rebuy.com>".

### S3 Upload

When `--s3-url` is provided, packageutil uploads all created artifacts to S3 with appropriate tags:
- `System`: Target system (e.g., "linux/amd64")  
- `Kind`: Package type (e.g., "tgz", "rpm", "deb")

## Examples

```bash
# Create compressed archives only
./packageutil --compressed

# Create all package types for Linux
./packageutil --compressed --rpm --deb

# Create packages and upload to S3
./packageutil --compressed --rpm --s3-url s3://mybucket/releases/

# Verbose output
./packageutil --compressed --verbose
```

## Integration with buildutil

1. Use `buildutil` to build your Go binaries
2. Use `packageutil` to create distribution packages from those binaries
3. Upload packages to your distribution channels

```bash
# Build phase
./buildutil -x linux/amd64 -x windows/amd64

# Package phase  
./packageutil --compressed --rpm --deb --s3-url s3://releases/
```
