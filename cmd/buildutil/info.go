package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
)

type Version struct {
	Major, Minor, Patch int

	Kind   string
	Suffix string
}

func ParseVersion(s string) (Version, error) {
	var (
		v   Version
		err error

		reCore       = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)([\-\+].+)?$`)
		rePreRelease = regexp.MustCompile(`^[\-\+](alpha|beta|rc)\.?[0-9]+$`)
		reDescribe   = regexp.MustCompile(`\-([0-9]+)-g?([0-9a-f]+)(-dirty)?$`)
	)

	matchGroup := reCore.FindStringSubmatch(s)
	if matchGroup == nil {
		return Version{Kind: "unknown", Suffix: s}, nil
	}

	var (
		mMajor  = matchGroup[1]
		mMinor  = matchGroup[2]
		mPatch  = matchGroup[3]
		mSuffix = matchGroup[4]
	)

	v.Major, err = strconv.Atoi(mMajor)
	if err != nil {
		// Should not happen because of the regex.
		panic(err)
	}

	v.Minor, err = strconv.Atoi(mMinor)
	if err != nil {
		// Should not happen because of the regex.
		panic(err)
	}

	v.Patch, err = strconv.Atoi(mPatch)
	if err != nil {
		// Should not happen because of the regex.
		panic(err)
	}

	if mSuffix == "" {
		v.Kind = "release"
		return v, nil
	}

	if rePreRelease.MatchString(mSuffix) {
		v.Kind = "prerelease"
		v.Suffix = mSuffix[1:]
		return v, nil
	}

	matchGroupDescribe := reDescribe.FindStringSubmatch(mSuffix)
	if matchGroupDescribe != nil {
		var (
			mDistance = matchGroupDescribe[1]
			mCommit   = matchGroupDescribe[2]
			mDirty    = matchGroupDescribe[3]
		)

		v.Suffix = fmt.Sprintf("%s.%s", mDistance, mCommit)
		v.Kind = "snapshot"
		if mDirty != "" {
			v.Kind = "dirty"
		}

		return v, nil
	}

	if mSuffix == "-dirty" {
		v.Suffix = "dirty"
		return v, nil
	}

	v.Suffix = "unknown"
	return v, nil
}

func (v Version) String() string {
	s, r := v.StringRelease()

	if r == "" {
		return s
	}

	return fmt.Sprintf("%s+%s", s, r)
}

func (v Version) StringRelease() (string, string) {
	version := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)

	release := v.Suffix
	if v.Suffix != "" && v.Kind != "prerelease" && v.Kind != "" {
		release = fmt.Sprintf("%s.%s", v.Kind, release)
	}

	return version, release
}

func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

type SystemInfo struct {
	OS   string
	Arch string
	Ext  string `json:",omitempty"`
}

func (i SystemInfo) FileSuffix() string {
	return fmt.Sprintf("%s-%s%s", i.OS, i.Arch, i.Ext)
}

func (i SystemInfo) Name() string {
	return fmt.Sprintf("%s/%s", i.OS, i.Arch)
}

type BuildInfo struct {
	BuildDate  string
	System     SystemInfo
	Version    Version
	SDKVersion Version

	Go struct {
		Name    string
		Module  string
		Dir     string
		Version string
	}

	Commit struct {
		Hash       string
		Branch     string
		Date       string
		DirtyFiles []string `json:",omitempty"`
	}

	Test struct {
		Packages []string
		Files    []string
	}

	Targets   []TargetInfo
	Artifacts []ArtifactInfo `json:",omitempty"`
}

type ArtifactInfo struct {
	Kind       string
	Filename   string
	S3Location *S3URL   `json:",omitempty"`
	Aliases    []string `json:",omitempty"`
	System     SystemInfo
}

func (i *BuildInfo) NewArtifactInfo(kind string, name string, system SystemInfo, ext string) ArtifactInfo {
	if ext == "" {
		ext = system.Ext
	}

	suffix := fmt.Sprintf("%s-%s%s", system.OS, system.Arch, ext)
	filename := fmt.Sprintf("%s-%s-%s", name, i.Version.String(), suffix)
	aliases := []string{
		fmt.Sprintf("%s-%s", name, suffix),
	}

	if system.OS == i.System.OS && system.Arch == i.System.Arch {
		aliases = append(aliases, fmt.Sprintf("%s%s", name, ext))
	}

	return ArtifactInfo{
		Kind:     kind,
		Filename: filename,
		Aliases:  aliases,
		System:   system,
	}
}

type TargetInfo struct {
	Package string
	Name    string
	Outfile string
	System  SystemInfo
	CGO     bool
}

func CollectBuildInformation(ctx context.Context, p BuildParameters) (BuildInfo, error) {
	var (
		err  error
		info BuildInfo
	)

	os.Setenv("GOPATH", "")

	logrus.Info("Collecting build information")

	e := NewChainExecutor(ctx)

	info.BuildDate = time.Now().Format(time.RFC3339)
	info.Go.Module = e.OutputString(p.GoCommand, "list", "-m", "-mod=mod")
	info.Go.Dir = e.OutputString(p.GoCommand, "list", "-m", "-mod=mod", "-f", "{{.Dir}}")
	info.System.OS = e.OutputString(p.GoCommand, "env", "GOOS")
	info.System.Arch = e.OutputString(p.GoCommand, "env", "GOARCH")
	info.System.Ext = e.OutputString(p.GoCommand, "env", "GOEXE")
	info.Commit.Date = time.Unix(e.OutputInt64("git", "show", "-s", "--format=%ct"), 0).Format(time.RFC3339)
	info.Commit.Hash = e.OutputString("git", "rev-parse", "HEAD")
	info.Commit.Branch = e.OutputString("git", "rev-parse", "--abbrev-ref", "HEAD")

	info.SDKVersion, err = ParseVersion(e.OutputString(p.GoCommand, "list", "-mod=readonly", "-m", "-f", "{{.Version}}", "github.com/rebuy-de/rebuy-go-sdk/..."))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse sdk-version")
	}

	info.Version, err = ParseVersion(e.OutputString("git", "describe", "--always", "--dirty", "--tags"))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse version")
	}

	goVersionMatch := regexp.MustCompile(`(?m)go(\d.*) `).FindStringSubmatch(e.OutputString(p.GoCommand, "version"))
	if goVersionMatch == nil {
		info.Go.Version = "unknown version"
	} else {
		info.Go.Version = goVersionMatch[1]
	}

	status := strings.TrimSpace(e.OutputString("git", "status", "-s"))
	if status != "" {
		for _, file := range strings.Split(status, "\n") {
			info.Commit.DirtyFiles = append(info.Commit.DirtyFiles, strings.TrimSpace(file))
		}
	}

	nameMatch := regexp.MustCompile(`([^/]+)(/v\d+)?$`).FindStringSubmatch(info.Go.Module)
	if nameMatch == nil {
		info.Go.Name = path.Base(info.Go.Module)
	} else {
		info.Go.Name = nameMatch[1]
	}

	cmdutil.Must(e.Err())

	targetSystems := []SystemInfo{}
	for _, target := range p.TargetSystems {
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			logrus.Errorf("Invalid format for cross compiling target '%s'.", target)
			cmdutil.Exit(1)
		}

		tinfo := SystemInfo{}
		tinfo.OS = parts[0]
		tinfo.Arch = parts[1]
		if tinfo.OS == "windows" {
			tinfo.Ext = ".exe"
		}

		targetSystems = append(targetSystems, tinfo)
	}

	if len(targetSystems) == 0 {
		logrus.Info("No cross compiling targets specified. Using local machine.")
		targetSystems = append(targetSystems, info.System)
	}

	if len(p.TargetPackages) == 0 {
		logrus.Debug("No targets specified. Discovering all packages.")
		p.TargetPackages = []string{"./..."}
	}

	info.Targets = []TargetInfo{}
	for _, search := range p.TargetPackages {
		pkgs, err := packages.Load(&packages.Config{
			Context: ctx,
		}, search)
		cmdutil.Must(err)

		for _, pkg := range pkgs {
			if pkg.Name != "main" {
				continue
			}
			logrus.Debugf("Found Package %s", pkg.PkgPath)

			for _, targetSystem := range targetSystems {
				tinfo := TargetInfo{
					Package: pkg.PkgPath,
					Name:    path.Base(pkg.PkgPath),
					System:  targetSystem,
					CGO:     p.CGO,
				}

				if pkg.PkgPath == info.Go.Module {
					tinfo.Name = info.Go.Name
				}

				artifact := info.NewArtifactInfo("binary", tinfo.Name, targetSystem, "")
				tinfo.Outfile = artifact.Filename
				info.Artifacts = append(info.Artifacts, artifact)
				info.Targets = append(info.Targets, tinfo)
			}
		}
	}

	testPackages, err := packages.Load(nil, "./...")
	cmdutil.Must(err)

	info.Test.Packages = []string{}
	info.Test.Files = []string{}

	for _, pkg := range testPackages {
		info.Test.Packages = append(info.Test.Packages, pkg.PkgPath)
		info.Test.Files = append(info.Test.Files, pkg.GoFiles...)
	}

	for _, targetSystem := range targetSystems {
		if targetSystem.OS == "windows" && p.CreateCompressed {
			info.Artifacts = append(info.Artifacts, info.NewArtifactInfo(
				"zip", info.Go.Name, targetSystem, ".zip"))
		}

		if targetSystem.OS != "windows" && p.CreateCompressed {
			if p.CreateCompressed {
				info.Artifacts = append(info.Artifacts, info.NewArtifactInfo(
					"tgz", info.Go.Name, targetSystem, ".tar.gz"))
			}
		}

		if targetSystem.OS == "linux" {
			if p.CreateDEB {
				info.Artifacts = append(info.Artifacts, info.NewArtifactInfo(
					"deb", info.Go.Name, targetSystem, ".deb"))
			}
			if p.CreateRPM {
				info.Artifacts = append(info.Artifacts, info.NewArtifactInfo(
					"rpm", info.Go.Name, targetSystem, ".rpm"))
			}
		}
	}

	if p.S3URL != "" {
		base, err := ParseS3URL(p.S3URL)
		if err != nil {
			return info, errors.WithStack(err)
		}

		for i, a := range info.Artifacts {
			u := base.Subpath(a.Filename)
			info.Artifacts[i].S3Location = &u
		}
	}

	return info, nil
}

type S3URL struct {
	Bucket string
	Key    string
}

func ParseS3URL(raw string) (*S3URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse S3 URL")
	}

	if u.Scheme != "s3" && u.Scheme != "" {
		return nil, errors.Errorf("Unknown scheme %s for the S3 URL", u.Scheme)
	}

	return &S3URL{
		Bucket: u.Host,
		Key:    strings.TrimPrefix(path.Clean(u.Path), "/"),
	}, nil
}

func (u S3URL) Subpath(p ...string) S3URL {
	p = append([]string{u.Key}, p...)
	u.Key = path.Join(p...)
	return u // This is actually a copy, since we do not use pointers.
}

func (u S3URL) String() string {
	return fmt.Sprintf("s3://%s/%s", u.Bucket, u.Key)
}

func (u S3URL) MarshalJSON() ([]byte, error) {
	s := u.String()
	return json.Marshal(s)
}
