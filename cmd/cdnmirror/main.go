package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/cmdutil"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const targetPathPrefix = `assets/cdnmirror`

func main() {
	defer cmdutil.HandleExit()
	if err := NewRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"cdnmirror", "Downloads assets from CDNs so the server can serve them directly.",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithRunner(new(Generate)),
	)
}

type Generate struct {
	Source string
	Target string
	Minify string
}

func (g *Generate) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&g.Source, "source", "", `URL to the original CDN.`)
	cmd.PersistentFlags().StringVar(
		&g.Target, "target", "", `Name of the target file in assets/cdnmirror`)
	cmd.PersistentFlags().StringVar(
		&g.Minify, "minify", "", `Minify file with given type; allowed values: js`)
	return nil
}

func (g *Generate) Run(ctx context.Context) error {
	err := os.MkdirAll(targetPathPrefix, 0755)
	if err != nil {
		return err
	}

	err = writeGitignore()
	if err != nil {
		return err
	}

	return g.download()
}

func writeGitignore() error {
	filename := path.Join(targetPathPrefix, ".gitignore")

	buf := new(bytes.Buffer)
	fmt.Fprintln(buf, "*")
	fmt.Fprintln(buf, "!.gitignore")

	return os.WriteFile(filename, buf.Bytes(), 0644)
}

func (g *Generate) download() error {
	targetFile := filepath.FromSlash(path.Join(targetPathPrefix, g.Target))

	resp, err := http.Get(g.Source)
	if err != nil {
		return fmt.Errorf("request source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	var code string

	switch g.Minify {
	case "js":
		result := api.Transform(string(body), api.TransformOptions{
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
		})
		if len(result.Errors) != 0 {
			return errors.Errorf("%#v", result.Errors)
		}
		code = string(result.Code)

	case "":
		code = string(body)
	default:
		return fmt.Errorf("invalid minify option %q", g.Minify)
	}

	err = os.MkdirAll(path.Dir(targetFile), 0755)
	if err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	f, err := os.Create(targetFile)
	if err != nil {
		return fmt.Errorf("create target file: %w", err)
	}
	defer f.Close()

	_, err = io.WriteString(f, code)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
