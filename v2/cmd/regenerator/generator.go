package main

import (
	"bytes"
	"text/template"

	"github.com/gobuffalo/packr/v2"
	"github.com/pkg/errors"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
)

func generateWrapper(version string) (string, error) {
	if version == "" {
		version = cmdutil.Version
	}

	box := packr.New("files", "./files")
	templateWrapper, err := box.FindString("wrapper.sh")
	if err != nil {
		return "", errors.WithStack(err)
	}

	t, err := template.
		New("buildutilw").
		Parse(templateWrapper)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse template")
	}

	vars := map[string]interface{}{
		"Version": version,
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, vars)
	if err != nil {
		return "", errors.Wrap(err, "failed to render template")
	}

	return buf.String(), nil
}
