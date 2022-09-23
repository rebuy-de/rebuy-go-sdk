package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/pkg/errors"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/webutil"
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
		"cdnmirror SOURCE_NAME..", "Downloads assets from CDNs so the server can serve them directly.",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithRun(Generate),
	)
}

func Generate(ctx context.Context, cmd *cobra.Command, args []string) {
	err := os.MkdirAll(targetPathPrefix, 0755)
	cmdutil.Must(err)

	writeGitignore()

	for _, name := range args {
		source := resolve(name)
		download(source)
	}
}

func writeGitignore() {
	filename := path.Join(targetPathPrefix, ".gitignore")

	buf := new(bytes.Buffer)
	fmt.Fprintln(buf, "*")
	fmt.Fprintln(buf, "!.gitignore")

	err := ioutil.WriteFile(filename, buf.Bytes(), 0644)
	cmdutil.Must(err)
}

func resolve(name string) webutil.CDNMirrorSource {
	switch name {
	case "@hotwired/turbo":
		return webutil.CDNMirrorSourceHotwiredTurbo()
	case "bootstrap":
		return webutil.CDNMirrorSourceBootstrap()
	case "font-awesome-sprites":
		return webutil.CDNMirrorSourceFontAwesomeSprites()
	case "bulma":
		return webutil.CDNMirrorSourceBulma()
	default:
		cmdutil.Must(errors.Errorf("invalid source name"))
		return webutil.CDNMirrorSource{}
	}
}

func download(source webutil.CDNMirrorSource) {
	targetFile := filepath.FromSlash(path.Join(targetPathPrefix, source.Target))

	resp, err := http.Get(source.URL)
	cmdutil.Must(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cmdutil.Must(fmt.Errorf(resp.Status))
	}

	body, err := io.ReadAll(resp.Body)
	cmdutil.Must(err)

	var code string

	switch source.Minify {
	case webutil.CDNMirrorMinifyJS:
		result := api.Transform(string(body), api.TransformOptions{
			MinifyWhitespace:  true,
			MinifyIdentifiers: true,
			MinifySyntax:      true,
		})
		if len(result.Errors) != 0 {
			cmdutil.Must(errors.Errorf("%#v", result.Errors))
		}
		code = string(result.Code)

	default:
		code = string(body)
	}

	f, err := os.Create(targetFile)
	cmdutil.Must(err)
	defer f.Close()

	_, err = io.WriteString(f, code)
	cmdutil.Must(err)
}
