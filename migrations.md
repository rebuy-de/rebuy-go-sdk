# Migrations

This file contains a list of tasks that are either required or at least
strongly recommended to align projects using this SDK.

## 2024-07-19 Replace cdnmirror with yarn

### Reasoning

Our `cdnmirror` is quite limited and we have no means of running Dependabot on those dependencies. Also with Yarn we are able to minify our own JS and CSS files.

### Steps

#### 1. Copy existing files

* Move static assets to `web/src/www`.
  * eg `mkdir -p web/src && mv cmd/assets web/src/www && rm -rf web/src/www/cdnmirror`
* Custom style should go into `web/src/index.css`.
  * The file can be a composition of `@import "xxx";` statements.
* Custom style should go into `web/src/index.js`.
  * The file can be a composition of `import 'xxx';` statements.

#### 2. Create basic files

*web/web.go*

```go
package web

import (
	"embed"
	"io/fs"
)

//go:generate yarn install
//go:generate yarn build

//go:embed all:dist/*
var Dist embed.FS

func FS() fs.FS {
	result, err := fs.Sub(Dist, "dist")
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

#### 3. Download Yarn

There might be a better way.

```sh
mkdir -p web/.yarn/releases/
curl -sSL -o web/.yarn/releases/yarn-4.2.2.cjs https://repo.yarnpkg.com/4.2.2/packages/yarnpkg-cli/bin/yarn.js
```

#### 4. Run Yarn

```sh
go generate ./web
```

This should run without errors and generate the directories `web/dist` and `web/node_modules`.

#### 5. Remove cdnmirror

* remove from `tools.go`
* remove from all `go:generate` comments
* run `go mod tidy`

#### 6. Replace References in HTML

* everything in `src/index.js` is merged and minified and available with `/assets/index.js`
* everything in `src/index.css` is merged and minified and available with `/assets/index.css`

#### 6. Update directory references

* use `web.FS()` instead of `//go:embed`
* use `os.DirFS("web/dist")` instead of `os.DirFS("cmd/assets")` in dev command

#### 7. Configure Dependabot

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

#### 8. Add Yarn to Dockerfile

With Alpine it would look like this:

```
RUN apk add --no-cache git git-lfs openssl nodejs yarn
```

### Hints

#### Example index.css

```css
@import "@fortawesome/fontawesome-free/css/all.css";
@import "bulma/css/bulma.css";

@import './style/bulma-patch.css';
@import './style/ry.css';
```

#### Infer config from cdnmirror command

Most config can be infered from the go:generate configs. For example with this unkpg source:

```
//go:generate cdnmirror --source https://unpkg.com/bootstrap@5.1.3/dist/css/bootstrap.min.css --target bootstrap-5.1.3-min.css
```

The URL follows this pattern: `https://unpkg.com/{package}@{version}/{import}`, where:
* `{package}` and `{version}` should go into the `dependencies` config of `package.json`.
* For CSS, add a statement like this into the `index.css`: `@import "{package}/{import}";`

There is not guarantee that this work, tho.

## 2024-06-14 Remove all uses of `github.com/pkg/errors`

### Reasoning

[github.com/pkg/errors](https://github.com/pkg/errors)` is deprecated. Since the built-in `errors` package improved a bit in the recent Go versions, we should remove all uses of `github.com/pkg/errors` and replace it with the `errors` package.

### Hints

* Use the pattern `return fmt.Errorf("something happenend with %#v: %w", someID, err)`
* The stack trace feature gets lost. Therefore it is suggested to properly add error messages each time handling errors.


## 2024-07-05 Switch to Jet Templates

### Reasoning

The builtin template engine has some shortcommings like being able to pass multiple and optional parameters to a template fragment. For example this block definition is not possible to builtin template engine:

```
{{ block entityLink(label, id, tab, extraClasses) }}
  {{ e := findEntity(.) }}
  <a class="ry-nowrap ry-underline {{ extraClasses }}" href="/{{.}}{{if id}}/{{id}}{{end}}{{ if tab }}/{{tab}}{{end}}">
    <span class="icon">
      <i class="fa-solid {{ e.IconClasses }}"></i>
    </span>
    {{ if label }}
    <span>{{ label }}</span>
    {{ else if id }}
    <span>{{ id }}</span>
    {{ else }}
    <span>{{ e.Plural }}</span>
    {{ end }}
  </a>
{{ end }}
```

### Hints

* The handler function signatures changed from `func(*webutil.View, *http.Request) webutil.Response` to `func(*http.Request) webutil.Response`.
* Wrap function changes from `webutil.ViewHandler.Wrap` to `webutil.WrapView`.
* Since the `View` interface is not part of the function signature, it needs to be added to the struct where the handler function is attached to.
* Non-HTML responses are now created with functions directly from the `webutil` package. For example `return webutilext.ViewError(http.StatusInternalServerError, err)` instead of `return v.Error(http.StatusBadRequest, fmt.Errorf(`unknown value for "until"`))`. where `v` was passed to the handler.

### Examples

Jet Set for Development:

```go
jet.NewSet(
	jet.NewOSFileSystemLoader("./pkg/app/web/templates"),
	jet.InDevelopmentMode(),
)
```

Jet Set for Production:

```go
jet.NewSet(
	webutil.JetFSLoader{FS: templateFS},
)
```

Jet Viewer based on Jet Set:

```go
webutilext.NewJetViewer(
	js,
	webutil.JetVarOption("clusterName", cn),
	webutil.JetVarOption("assetPath", "/assets/"+prefix),
)
```
