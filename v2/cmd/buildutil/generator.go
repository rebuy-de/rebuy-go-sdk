package main

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"

	"github.com/rebuy-de/rebuy-go-sdk/v2/pkg/cmdutil"
)

const (
	templateWrapper = `#!/bin/bash

set -euo pipefail

cd $(dirname "$0")

VERSION="{{.Version}}"

get_arch() {
    ARCH=$(uname -m)
    case $ARCH in
        armv5*) ARCH="armv5";;
        armv6*) ARCH="armv6";;
        armv7*) ARCH="armv7";;
        aarch64) ARCH="arm64";;
        x86) ARCH="386";;
        x86_64) ARCH="amd64";;
        i686) ARCH="386";;
        i386) ARCH="386";;
    esac
    echo "$ARCH"
}

get_os() {
    echo $(uname) | tr '[:upper:]' '[:lower:]'
}

fname="buildutil-${VERSION}-$(get_os)-$(get_arch)"
cachedir="${HOME}/.rebuy/cache"
fpath="${cachedir}/${fname}"

encoded=$(echo $fname | sed "s/+/%2B/g")
url="https://rebuy-github-releases.s3-eu-west-1.amazonaws.com/rebuy-go-sdk/${encoded}"

if ! [ -f ${fpath} ]
then
	mkdir -p ${cachedir}
    curl --fail -sS -o ${fpath} ${url} || exit 1
    chmod +x ${fpath}
fi

exec ${fpath} "$@"
`
)

func generateWrapper(version string) (string, error) {
	if version == "" {
		version = cmdutil.Version
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
