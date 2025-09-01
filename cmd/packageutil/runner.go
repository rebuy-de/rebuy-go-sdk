package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb" // blank import to register the format
	"github.com/goreleaser/nfpm/v2/files"
	_ "github.com/goreleaser/nfpm/v2/rpm" // blank import to register the format
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type PackageParameters struct {
	TargetSystems  []string
	TargetPackages []string
	S3URL          string

	CreateCompressed bool
	CreateRPM        bool
	CreateDEB        bool
}

type SystemInfo struct {
	OS   string
	Arch string
	Ext  string
}

func (i SystemInfo) FileSuffix() string {
	return fmt.Sprintf("%s-%s%s", i.OS, i.Arch, i.Ext)
}

func (i SystemInfo) Name() string {
	return fmt.Sprintf("%s/%s", i.OS, i.Arch)
}

type BinaryInfo struct {
	Name     string
	Path     string
	System   SystemInfo
	Basename string
	Version  string
}

type ArtifactInfo struct {
	Kind       string
	Filename   string
	S3Location *S3URL
	Aliases    []string
	System     SystemInfo
}

type S3URL struct {
	Bucket string
	Key    string
}

func ParseS3URL(raw string) (*S3URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 URL: %w", err)
	}

	if u.Scheme != "s3" && u.Scheme != "" {
		return nil, fmt.Errorf("unknown scheme %s for the S3 URL", u.Scheme)
	}

	return &S3URL{
		Bucket: u.Host,
		Key:    strings.TrimPrefix(path.Clean(u.Path), "/"),
	}, nil
}

func (u S3URL) Subpath(p ...string) S3URL {
	p = append([]string{u.Key}, p...)
	u.Key = path.Join(p...)
	return u
}

func (u S3URL) String() string {
	return fmt.Sprintf("s3://%s/%s", u.Bucket, u.Key)
}

type Runner struct {
	Parameters PackageParameters
	Binaries   []BinaryInfo
	Artifacts  []ArtifactInfo
}

func (r *Runner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringSliceVarP(
		&r.Parameters.TargetSystems, "cross-compile", "x", []string{},
		"Targets for cross compilation (eg linux/amd64). Can be used multiple times.")
	cmd.PersistentFlags().StringSliceVarP(
		&r.Parameters.TargetPackages, "package", "p", []string{},
		"Packages to search for binaries. Can be used multiple times.")
	cmd.PersistentFlags().StringVar(
		&r.Parameters.S3URL, "s3-url", "",
		"S3 base URL for uploads (e.g., s3://bucket/path/).")
	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CreateCompressed, "compressed", false,
		"Creates .tgz artifacts for POSIX targets and .zip for windows.")
	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CreateRPM, "rpm", false,
		"Creates .rpm artifacts for linux targets.")
	cmd.PersistentFlags().BoolVar(
		&r.Parameters.CreateDEB, "deb", false,
		"Creates .deb artifacts for linux targets.")

	return nil
}

func (r *Runner) Run(ctx context.Context, args []string) error {
	return runSeq(ctx,
		r.discoverBinaries,
		r.createArtifacts,
		r.uploadArtifacts,
	)
}

func runSeq(ctx context.Context, fns ...func(ctx context.Context) error) error {
	for _, fn := range fns {
		err := fn(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseBinaryName(filename string) (*BinaryInfo, error) {
	name := filename
	var ext string

	// Handle extension
	if strings.HasSuffix(name, ".exe") {
		ext = ".exe"
		name = strings.TrimSuffix(name, ".exe")
	}

	parts := strings.Split(name, "-")
	if len(parts) < 4 {
		return nil, fmt.Errorf("insufficient parts in filename (expected at least 4, got %d)", len(parts))
	}

	// Parse from the end: last two parts are arch and os
	arch := parts[len(parts)-1]
	os := parts[len(parts)-2]

	if !isValidOS(os) {
		return nil, fmt.Errorf("unrecognized OS: %s", os)
	}

	// Binary name is first part, version is everything between name and os-arch
	binaryName := parts[0]
	versionParts := parts[1 : len(parts)-2]
	version := strings.Join(versionParts, "-")

	return &BinaryInfo{
		Name:    binaryName,
		Version: version,
		System: SystemInfo{
			OS:   os,
			Arch: arch,
			Ext:  ext,
		},
	}, nil
}

func isValidOS(os string) bool {
	validOSes := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
		"freebsd": true,
		"openbsd": true,
		"netbsd":  true,
	}
	return validOSes[os]
}

func (r *Runner) dist(parts ...string) string {
	allParts := append([]string{"dist"}, parts...)
	return path.Join(allParts...)
}

func (r *Runner) discoverBinaries(ctx context.Context) error {
	logrus.Info("Discovering binaries")

	distDir := "dist"
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		return fmt.Errorf("dist directory not found - run buildutil first")
	}

	files, err := filepath.Glob(filepath.Join(distDir, "*"))
	if err != nil {
		return fmt.Errorf("failed to list dist directory: %w", err)
	}

	seen := make(map[string]bool)

	logrus.Debugf("Found %d files in dist directory", len(files))

	for _, file := range files {
		basename := filepath.Base(file)
		logrus.Debugf("Processing file: %s", basename)

		// Use Lstat to check the file itself, not what it points to
		info, err := os.Lstat(file)
		if err != nil {
			logrus.Debugf("Skipping %s: lstat failed: %v", basename, err)
			continue
		}

		if info.IsDir() {
			logrus.Debugf("Skipping %s: is directory", basename)
			continue
		}

		// Skip symlinks - we want actual binaries
		if info.Mode()&os.ModeSymlink != 0 {
			logrus.Debugf("Skipping %s: is symlink", basename)
			continue
		}

		if info.Mode()&0111 == 0 {
			logrus.Debugf("Skipping %s: not executable", basename)
			continue
		}

		// Parse binary information from filename
		binary, err := parseBinaryName(basename)
		if err != nil {
			logrus.Debugf("Skipping %s: %v", basename, err)
			continue
		}

		binary.Path = file
		binary.Basename = basename

		// Only include if we have valid system info and haven't seen this combination
		key := fmt.Sprintf("%s-%s-%s", binary.Name, binary.System.OS, binary.System.Arch)
		logrus.Debugf("Final check for %s: OS='%s', Arch='%s', seen=%v", basename, binary.System.OS, binary.System.Arch, seen[key])
		if binary.System.OS != "" && binary.System.Arch != "" && !seen[key] {
			seen[key] = true
			r.Binaries = append(r.Binaries, *binary)
			logrus.Debugf("Found binary: %s (%s)", binary.Basename, binary.System.Name())
		}
	}

	if len(r.Binaries) == 0 {
		return fmt.Errorf("no binaries found in dist directory")
	}

	return nil
}

func (r *Runner) createArtifacts(ctx context.Context) error {
	if !r.Parameters.CreateCompressed && !r.Parameters.CreateRPM && !r.Parameters.CreateDEB {
		logrus.Info("No artifact formats selected, skipping artifact creation")
		return nil
	}

	logrus.Info("Creating artifacts")

	systemBinaries := make(map[string][]BinaryInfo)
	for _, binary := range r.Binaries {
		key := binary.System.Name()
		systemBinaries[key] = append(systemBinaries[key], binary)
	}

	for _, binaries := range systemBinaries {
		if len(binaries) == 0 {
			continue
		}

		system := binaries[0].System
		name := binaries[0].Name

		if r.Parameters.CreateCompressed {
			if system.OS == "windows" {
				artifact, err := r.createZipArtifact(name, system, binaries)
				if err != nil {
					return err
				}
				r.Artifacts = append(r.Artifacts, artifact)
			} else {
				artifact, err := r.createTgzArtifact(name, system, binaries)
				if err != nil {
					return err
				}
				r.Artifacts = append(r.Artifacts, artifact)
			}
		}

		if system.OS == "linux" {
			if r.Parameters.CreateRPM {
				artifact, err := r.createRPMArtifact(name, system, binaries)
				if err != nil {
					return err
				}
				r.Artifacts = append(r.Artifacts, artifact)
			}
			if r.Parameters.CreateDEB {
				artifact, err := r.createDEBArtifact(name, system, binaries)
				if err != nil {
					return err
				}
				r.Artifacts = append(r.Artifacts, artifact)
			}
		}
	}

	return nil
}

func (r *Runner) createTgzArtifact(name string, system SystemInfo, binaries []BinaryInfo) (ArtifactInfo, error) {
	filename := fmt.Sprintf("%s-%s.tar.gz", name, system.FileSuffix())
	logrus.Infof("Creating tgz artifact: %s", filename)

	dst, err := os.Create(r.dist(filename))
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to create tgz file: %w", err)
	}
	defer dst.Close()

	zw := gzip.NewWriter(dst)
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	for _, binary := range binaries {
		f, err := os.Open(binary.Path)
		if err != nil {
			return ArtifactInfo{}, fmt.Errorf("failed to open binary %s: %w", binary.Path, err)
		}

		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return ArtifactInfo{}, fmt.Errorf("failed to stat binary %s: %w", binary.Path, err)
		}

		hdr := &tar.Header{
			Name: binary.Name,
			Mode: 0755,
			Size: fi.Size(),
		}
		err = tw.WriteHeader(hdr)
		if err != nil {
			f.Close()
			return ArtifactInfo{}, fmt.Errorf("failed to write tar header for %s: %w", binary.Name, err)
		}

		_, err = io.Copy(tw, f)
		f.Close()
		if err != nil {
			return ArtifactInfo{}, fmt.Errorf("failed to copy binary %s to tar: %w", binary.Name, err)
		}
	}

	return ArtifactInfo{
		Kind:     "tgz",
		Filename: filename,
		System:   system,
	}, nil
}

func (r *Runner) createZipArtifact(name string, system SystemInfo, binaries []BinaryInfo) (ArtifactInfo, error) {
	filename := fmt.Sprintf("%s-%s.zip", name, system.FileSuffix())
	logrus.Infof("Creating zip artifact: %s", filename)

	dst, err := os.Create(r.dist(filename))
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to create zip file: %w", err)
	}
	defer dst.Close()

	zw := zip.NewWriter(dst)
	defer zw.Close()

	for _, binary := range binaries {
		f, err := os.Open(binary.Path)
		if err != nil {
			return ArtifactInfo{}, fmt.Errorf("failed to open binary %s: %w", binary.Path, err)
		}

		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return ArtifactInfo{}, fmt.Errorf("failed to stat binary %s: %w", binary.Path, err)
		}

		header, err := zip.FileInfoHeader(fi)
		if err != nil {
			f.Close()
			return ArtifactInfo{}, fmt.Errorf("failed to create zip header for %s: %w", binary.Name, err)
		}

		header.Name = binary.Name + system.Ext
		header.Method = zip.Deflate

		writer, err := zw.CreateHeader(header)
		if err != nil {
			f.Close()
			return ArtifactInfo{}, fmt.Errorf("failed to create zip entry for %s: %w", binary.Name, err)
		}

		_, err = io.Copy(writer, f)
		f.Close()
		if err != nil {
			return ArtifactInfo{}, fmt.Errorf("failed to copy binary %s to zip: %w", binary.Name, err)
		}
	}

	return ArtifactInfo{
		Kind:     "zip",
		Filename: filename,
		System:   system,
	}, nil
}

func (r *Runner) createRPMArtifact(name string, system SystemInfo, binaries []BinaryInfo) (ArtifactInfo, error) {
	filename := fmt.Sprintf("%s-%s.rpm", name, system.FileSuffix())
	logrus.Infof("Creating rpm artifact: %s", filename)

	bindir := "/usr/bin"
	contents := files.Contents{}

	for _, binary := range binaries {
		content := files.Content{
			Source:      binary.Path,
			Destination: path.Join(bindir, binary.Name),
		}
		contents = append(contents, &content)
	}

	info := &nfpm.Info{
		Name:       name,
		Arch:       system.Arch,
		Platform:   system.OS,
		Version:    binaries[0].Version,
		Release:    "1",
		Maintainer: "rebuy Platform Team <dl-scb-tech-platform@rebuy.com>",
		Overridables: nfpm.Overridables{
			Contents: contents,
		},
	}

	err := nfpm.Validate(info)
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to validate nfpm info: %w", err)
	}

	packager, err := nfpm.Get("rpm")
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to get rpm packager: %w", err)
	}

	w, err := os.Create(r.dist(filename))
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to create rpm file: %w", err)
	}
	defer w.Close()

	err = packager.Package(nfpm.WithDefaults(info), w)
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to package rpm: %w", err)
	}

	return ArtifactInfo{
		Kind:     "rpm",
		Filename: filename,
		System:   system,
	}, nil
}

func (r *Runner) createDEBArtifact(name string, system SystemInfo, binaries []BinaryInfo) (ArtifactInfo, error) {
	filename := fmt.Sprintf("%s-%s.deb", name, system.FileSuffix())
	logrus.Infof("Creating deb artifact: %s", filename)

	bindir := "/usr/bin"
	contents := files.Contents{}

	for _, binary := range binaries {
		content := files.Content{
			Source:      binary.Path,
			Destination: path.Join(bindir, binary.Name),
		}
		contents = append(contents, &content)
	}

	info := &nfpm.Info{
		Name:       name,
		Arch:       system.Arch,
		Platform:   system.OS,
		Version:    binaries[0].Version,
		Release:    "1",
		Maintainer: "rebuy Platform Team <dl-scb-tech-platform@rebuy.com>",
		Overridables: nfpm.Overridables{
			Contents: contents,
		},
	}

	err := nfpm.Validate(info)
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to validate nfpm info: %w", err)
	}

	packager, err := nfpm.Get("deb")
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to get deb packager: %w", err)
	}

	w, err := os.Create(r.dist(filename))
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to create deb file: %w", err)
	}
	defer w.Close()

	err = packager.Package(nfpm.WithDefaults(info), w)
	if err != nil {
		return ArtifactInfo{}, fmt.Errorf("failed to package deb: %w", err)
	}

	return ArtifactInfo{
		Kind:     "deb",
		Filename: filename,
		System:   system,
	}, nil
}

func (r *Runner) uploadArtifacts(ctx context.Context) error {
	if r.Parameters.S3URL == "" {
		logrus.Info("No S3 URL specified, skipping upload")
		return nil
	}

	if len(r.Artifacts) == 0 {
		logrus.Info("No artifacts to upload")
		return nil
	}

	logrus.Info("Uploading artifacts to S3")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithDefaultRegion("eu-west-1"))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	base, err := ParseS3URL(r.Parameters.S3URL)
	if err != nil {
		return fmt.Errorf("failed to parse S3 URL: %w", err)
	}

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))

	for _, artifact := range r.Artifacts {
		s3Location := base.Subpath(artifact.Filename)

		logrus.Infof("Uploading %s", s3Location.String())

		f, err := os.Open(r.dist(artifact.Filename))
		if err != nil {
			return fmt.Errorf("failed to open artifact %s: %w", artifact.Filename, err)
		}

		tags := url.Values{}
		tags.Set("System", artifact.System.Name())
		tags.Set("Kind", artifact.Kind)

		_, err = uploader.Upload(ctx, &s3.PutObjectInput{
			Bucket:  &s3Location.Bucket,
			Key:     &s3Location.Key,
			Tagging: aws.String(tags.Encode()),
			Body:    f,
		})
		if err != nil {
			f.Close()
			return fmt.Errorf("failed to upload %s: %w", artifact.Filename, err)
		}

		f.Close()
	}

	return nil
}
