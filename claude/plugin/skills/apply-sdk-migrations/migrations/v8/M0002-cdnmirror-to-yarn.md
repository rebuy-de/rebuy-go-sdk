---
id: M0002
title: Replace cdnmirror with yarn
date: 2024-07-19
sdk_version: v8
type: minor
---

# Replace cdnmirror with yarn

## Reasoning

Our `cdnmirror` is quite limited and we have no means of running Dependabot on those dependencies. Also with Yarn we are able to minify our own JS and CSS files.

## Steps

### 1. Copy existing files

* Move static assets to `web/src/www`.
  * eg `mkdir -p web/src && mv cmd/assets web/src/www && rm -rf web/src/www/cdnmirror`
* Custom style should go into `web/src/index.css`.
  * The file can be a composition of `@import "xxx";` statements.
* Custom style should go into `web/src/index.js`.
  * The file can be a composition of `import 'xxx';` statements.

### 2. Create basic files

*web/web.go*

```go
package web

import (
	"embed"
	"io/fs"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
)

//go:generate yarn install
//go:generate yarn build

//go:embed all:dist/*
var embedded embed.FS

func DevFS() webutil.AssetFS {
	return os.DirFS("web/dist")
}

func ProdFS() webutil.AssetFS {
	result, err := fs.Sub(embedded, "dist")
	if err != nil {
		panic(err)
	}

	return result
}
```

*web/package.json*

* Set project name (kebab case).
* Adjust dependencies.

```json
{
  "name": "__PROJECT_NAME__",
  "version": "1.0.0",
  "packageManager": "yarn@4.2.2",
  "private": true,
  "dependencies": {
    "@fortawesome/fontawesome-free": "^6.5.2",
    "bulma": "^1.0.1",
    "htmx.org": "^1.9.12",
    "idiomorph": "^0.3.0"
  },
  "devDependencies": {
    "esbuild": "^0.23.0",
    "nodemon": "^3.1.4"
  },
  "scripts": {
    "build": "node esbuild.config.mjs"
  }
}
```

*web/esbuild.config.mjs*

* Remove files from `entryPoints` if not needed. Remove the whole `esbuild.build` command, if the `entryPoints` are empty.
* Remove `fs.cpSync`, if there are not static assets, like favicons.

```js
import * as esbuild from 'esbuild'
import fs from 'node:fs'

await esbuild.build({
  entryPoints: [
     'src/index.js', 'src/index.css',
  ],
  bundle: true,
  minify: true,
  sourcemap: true,
  outdir: 'dist/',
  format: 'esm',
  loader: {
    '.woff2': 'file',
    '.ttf': 'file'
  },
})

fs.cpSync('src/www', 'dist', {recursive: true});

// The HTMX stuff does not deal well with ESM bundling. It is not needed tho,
// therefore we copy the assets manually and link them directly in the <head>.
const scripts = [
  'hyperscript.org/dist/_hyperscript.min.js',
  'hyperscript.org/dist/template.js',
  'htmx.org/dist/htmx.min.js',
  'idiomorph/dist/idiomorph-ext.min.js',
];

scripts.forEach((file) => {
    fs.cpSync(`node_modules/${file}`, `dist/${file}`, {recursive: true});
});
```

*web/.gitattributes`

```
.yarn/releases/* filter=lfs diff=lfs merge=lfs -text
```

*web/.gitignore*

```
/node_modules
/dist/*
!/dist/.keepdir

.pnp.*
.yarn/*
!.yarn/patches
!.yarn/plugins
!.yarn/releases
!.yarn/sdks
!.yarn/versions
```

*web/.yarnrc.yml*

```yaml
yarnPath: .yarn/releases/yarn-4.2.2.cjs
enableGlobalCache: false
nmMode: hardlinks-global
nodeLinker: node-modules
```

### 3. Download Yarn

There might be a better way.

```sh
mkdir -p web/.yarn/releases/
curl -sSL -o web/.yarn/releases/yarn-4.2.2.cjs https://repo.yarnpkg.com/4.2.2/packages/yarnpkg-cli/bin/yarn.js
```

### 4. Run Yarn

```sh
go generate ./web
```

This should run without errors and generate the directories `web/dist` and `web/node_modules`.

### 5. Remove cdnmirror

* remove from `tools.go`
* remove from all `go:generate` comments
* run `go mod tidy`

### 6. Replace References in HTML

* everything in `src/index.js` is merged and minified and available with `/assets/index.js`
* everything in `src/index.css` is merged and minified and available with `/assets/index.css`

### 6. Update directory references

* use `web.ProdFS()` instead of `//go:embed`
* use `web.DevFS()` instead of `os.DirFS("cmd/assets")` in dev command

### 7. Configure Dependabot

Configure `.github/dependabot.yml`:

```yaml
  - package-ecosystem: "npm"
    directory: "/web"
    schedule:
      interval: "weekly"
      day: "tuesday"
      time: "10:00"
      timezone: "Europe/Berlin"
    groups:
      yarn:
        patterns:
          - "*"
```

### 8. Add Yarn to Dockerfile

With Alpine it would look like this:

```
RUN apk add --no-cache git git-lfs openssl nodejs yarn
```

## Hints

### Example index.css

```css
@import "@fortawesome/fontawesome-free/css/all.css";
@import "bulma/css/bulma.css";

@import './style/bulma-patch.css';
@import './style/ry.css';
```

### Infer config from cdnmirror command

Most config can be infered from the go:generate configs. For example with this unkpg source:

```
//go:generate cdnmirror --source https://unpkg.com/bootstrap@5.1.3/dist/css/bootstrap.min.css --target bootstrap-5.1.3-min.css
```

The URL follows this pattern: `https://unpkg.com/{package}@{version}/{import}`, where:
* `{package}` and `{version}` should go into the `dependencies` config of `package.json`.
* For CSS, add a statement like this into the `index.css`: `@import "{package}/{import}";`

There is not guarantee that this work, tho.
