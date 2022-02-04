#!/bin/sh

set -euo pipefail

cd $(dirname $0)

podman run -v "$(pwd):/cdn" -w /cdn --rm docker.io/library/node:17-alpine npm install

echo 'import * as Turbo from "@hotwired/turbo"' | \
    ./node_modules/.bin/esbuild --bundle --format=esm --minify --outfile=dist/@hotwired/turbo.js
