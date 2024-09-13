# Migrations

This file contains a list of tasks that are either required or at least
strongly recommended to align projects using this SDK.

## M0004 2024-09-13 Isolate HTTP handlers

### Reasoning

Our previous approach to have a single `Server` struct and to put all logic there gets messy pretty fast. Worse than
that it is hard to refactor this to separate this into multiple structs or packages. Dependency injection will help us
to separate those things in the very beginning of a project without manual wiring of dependencies. One step towards this
to split up all HTTP handlers into separate files.

### Hints

* The handlers should be moved into the `pkg/app/handlers` package.
* The handler struct should have a `New...` constructor with all required dependencies as parameters.
* The handler struct should implement `interface { Register(chi.Router) }`, which gets called once to set up the routes.
  This will later be used for dependency injection integration.

### Example

```go
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/platform-inventory/pkg/bll/webutilext"
	"github.com/rebuy-de/platform-inventory/pkg/dal/sqlc"
	"github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
)

type KubeEventHandler struct {
	sqlc   *sqlc.Queries
	viewer *webutilext.JetViewer
}

func NewKubeEventHandler(
    sqlc *sqlc.Queries,
    viewer *webutilext.JetViewer,
) *KubeEventHandler {
	return &KubeEventHandler{
		sqlc:   sqlc,
		viewer: viewer,
	}
}

func (h *KubeEventHandler) Register(router chi.Router) {
	router.Get("/kube/events", webutilext.WrapView(h.list))
	router.Get("/kube/events/table-fragment", webutilext.WrapView(h.listFragment))
}

func (h *KubeEventHandler) list(r *http.Request) webutil.Response {
	events, err := h.sqlc.ListKubeEvents(r.Context())
	if err != nil {
		return webutilext.ViewError(http.StatusInternalServerError, err)
	}

	return h.viewer.HTML(http.StatusOK, "kube_event_list.html", events)
}

func (h *KubeEventHandler) listFragment(r *http.Request) webutil.Response {
	events, err := h.sqlc.ListKubeEvents(r.Context())
	if err != nil {
		return webutilext.ViewError(http.StatusInternalServerError, err)
	}

	return h.viewer.HTML(http.StatusOK, "frames/kube_event_table.html", events)
}
```

## M0003 2024-08-16 Change viewer interfaces of webutil

### Reasoning

The previous interface is a bit awkward to use with dependency injection together with splitting the HTTP handlers into
multiple structs. Additionally there were cases where we wanted to use the `webutil.Response` type for convenience, but
did not actually need any HTML rendering.

Therefore the interfaces are changed this way:
* The new `webuitil.WrapView` function replaces the old `webuitl.ViewHandler.Wrap` function and does not require any
  template definitions.
* All `webutil.Response` functions, that do not need templates, are pure functions now (ie not attached to a type).
* The handler interface gets reduced to `func(*http.Request) Response`, so it does not contain the view parameter
  anymore. When using HTML, it is required to attach the `webutil.GoTemplateViewer` to the struct that implements the
  handler.


## M0002 2024-07-19 Replace cdnmirror with yarn

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

## M0001 2024-06-14 Remove all uses of `github.com/pkg/errors`

### Reasoning

[github.com/pkg/errors](https://github.com/pkg/errors)` is deprecated. Since the built-in `errors` package improved a bit in the recent Go versions, we should remove all uses of `github.com/pkg/errors` and replace it with the `errors` package.

### Hints

* Use the pattern `return fmt.Errorf("something happenend with %#v: %w", someID, err)`
* The stack trace feature gets lost. Therefore it is suggested to properly add error messages each time handling errors.
