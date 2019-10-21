package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
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
		rePreRelease = regexp.MustCompile(`^\+(alpha|beta|rc)\.[0-9]+$`)
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
		// Should not happend because of the regex.
		panic(err)
	}

	v.Minor, err = strconv.Atoi(mMinor)
	if err != nil {
		// Should not happend because of the regex.
		panic(err)
	}

	v.Patch, err = strconv.Atoi(mPatch)
	if err != nil {
		// Should not happend because of the regex.
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

	v.Suffix = "unknown"
	return v, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Suffix != "" {
		if v.Kind != "prerelease" {
			s = fmt.Sprintf("%s+%s.%s", s, v.Kind, v.Suffix)
		} else {
			s = fmt.Sprintf("%s+%s", s, v.Suffix)
		}
	}

	return s
}

type SystemInfo struct {
	OS   string
	Arch string
	Ext  string `json:",omitempty"`
}

func (i SystemInfo) FileSufix() string {
	return fmt.Sprintf("%s-%s%s", i.OS, i.Arch, i.Ext)
}

func (i SystemInfo) Name() string {
	return fmt.Sprintf("%s/%s", i.OS, i.Arch)
}

type BuildInfo struct {
	BuildDate string
	System    SystemInfo
	Version   Version

	Go struct {
		Module string
		Dir    string
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

	Targets []TargetInfo
}

type TargetInfo struct {
	Package string
	Name    string
	System  SystemInfo

	Outfile struct {
		Name    string
		Aliases []string
	}
}

func CollectBuildInformation(ctx context.Context, pkgArgs []string, targetSystemArgs []string) (BuildInfo, error) {
	var (
		err  error
		info BuildInfo
	)

	os.Setenv("GOPATH", "")

	logrus.Info("Collecting build information")

	e := NewExecutor(ctx)

	info.BuildDate = time.Now().Format(time.RFC3339)
	info.Go.Module = e.GetString("go", "list", "-m")
	info.Go.Dir = e.GetString("go", "list", "-m", "-f", "{{.Dir}}")
	info.System.OS = e.GetString("go", "env", "GOOS")
	info.System.Arch = e.GetString("go", "env", "GOARCH")
	info.System.Ext = e.GetString("go", "env", "GOEXE")
	info.Commit.Date = time.Unix(e.GetInt64("git", "show", "-s", "--format=%ct"), 0).Format(time.RFC3339)
	info.Commit.Hash = e.GetString("git", "rev-parse", "HEAD")
	info.Commit.Branch = e.GetString("git", "rev-parse", "--abbrev-ref", "HEAD")

	info.Version, err = ParseVersion(e.GetString("git", "describe", "--always", "--dirty", "--tags"))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse version")
	}

	status := strings.TrimSpace(e.GetString("git", "status", "-s"))
	if status != "" {
		info.Commit.DirtyFiles = strings.Split(status, "\n")
	}

	cmdutil.Must(e.Err())

	targetSystems := []SystemInfo{}
	for _, target := range targetSystemArgs {
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

	if len(pkgArgs) == 0 {
		logrus.Debug("No targets specified. Discovering all packages.")
		pkgArgs = []string{"./..."}
	}

	info.Targets = []TargetInfo{}
	for _, search := range pkgArgs {
		pkgs, err := packages.Load(nil, search)
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
				}

				tinfo.Outfile.Name = fmt.Sprintf("%s-%s-%s",
					tinfo.Name, info.Version.String(),
					tinfo.System.FileSufix())

				tinfo.Outfile.Aliases = []string{
					fmt.Sprintf("%s-%s", tinfo.Name, tinfo.System.FileSufix()),
				}

				if tinfo.System.OS == info.System.OS && tinfo.System.Arch == info.System.Arch {
					tinfo.Outfile.Aliases = append(tinfo.Outfile.Aliases, tinfo.Name)
				}

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

	return info, nil

}
