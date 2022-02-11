package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/rebuy-de/rebuy-go-sdk/v4/pkg/cmdutil"
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
		cmdutil.WithRun(Generate),
	)
}

func Generate(ctx context.Context, cmd *cobra.Command, args []string) {
	for _, sourceURL := range args {
		source, err := url.Parse(sourceURL)
		cmdutil.Must(err)

		targetFile := filepath.FromSlash(path.Join(targetPathPrefix, source.Path))
		targetDirectory := filepath.Dir(targetFile)

		resp, err := http.Get(sourceURL)
		cmdutil.Must(err)
		defer resp.Body.Close()

		err = os.MkdirAll(targetDirectory, 0755)
		cmdutil.Must(err)

		f, err := os.Create(targetFile)
		cmdutil.Must(err)
		defer f.Close()

		_, err = io.Copy(f, resp.Body)
		cmdutil.Must(err)
	}
}
