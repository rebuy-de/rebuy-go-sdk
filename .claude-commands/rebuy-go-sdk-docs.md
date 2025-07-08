---
description: Reads the documentation for rebuy-go-sdk into the LLM context.
---

# General Advice

- The examples below might have a wrong import path that needs to be adjusted to the project go module.
- Always use `./buildutil` for compiling the project.
- Strings that are passed into dependency injections should have a dedicated type (`type FooParam string`) that gets
  converted back into a plain `string` in the `New*` functions.

# File main.go

The file `./main.go` should look exactly like in the example project:

```
package main

import (
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/cmdutil"
	"github.com/sirupsen/logrus"

	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/cmd"
)

func main() {
	defer cmdutil.HandleExit()
	if err := cmd.NewRootCommand().Execute(); err != nil {
		logrus.Fatal(err)
	}
}
```

# File tools.go

The file `./tools.go` should contain blank imports for go generate tools, like in this example:

```
//go:build tools
// +build tools

package main

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
import (
	_ "github.com/Khan/genqlient" // only when using graphql
	_ "github.com/a-h/templ/cmd/templ" // only when using templ
	_ "github.com/rebuy-de/rebuy-go-sdk/v9/cmd/buildutil" // always used
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc" // only when using a database with sqlc
	_ "honnef.co/go/tools/cmd/staticcheck" // always used
)
```

# File cmd/root.go

The file `./cmd/root.go` defines all subcommands for the project. Mandatory ones are `daemon` and `dev`, which start the server either in production mode or in dev mode for local testing.

The entry point is always NewRootCommand`, which looks like this:

```
func NewRootCommand() *cobra.Command {
	return cmdutil.New(
		"full-example", "A full example app for the rebuy-go-sdk.",
		cmdutil.WithLogVerboseFlag(),
		cmdutil.WithLogToGraylog(),
		cmdutil.WithVersionCommand(),
		cmdutil.WithVersionLog(logrus.DebugLevel),

		cmdutil.WithSubCommand(
			cmdutil.New(
				"prod", "Run the application as daemon",
				cmdutil.WithRunner(new(ProdRunner)),
			)),

		cmdutil.WithSubCommand(cmdutil.New(
			"dev", "Run the application in local dev mode",
			cmdutil.WithRunner(new(DevRunner)),
		)),
	)
}
```

It might contain additional commands, but `daemon` and `dev` are mandatory. The `cmdutil.With*` options are also mandatory.

A Runner looks like this:

```
type FooRunner struct {
    // contains fields that are targets fir binding command line flags in `Bind()`.
    myParameter string
}

func (r *FooRunner) Bind(cmd *cobra.Command) error {
    // binds flags
	cmd.PersistentFlags().StringVar(
		&r.myParameter, "my-parameter", "default",
		`This is an example flag to show how the binding works.`)

	return nil
}

func (r *FooRunner) Run(ctx context.Context, _ []string) error {
	c := dig.New() // dig always gets initialized in the beginning

	err := errors.Join(
		c.Provide(web.ProdFS), // web.DevFS for dev command
		c.Provide(webutil.AssetDefaultProd), // webutil.AssedDefaultDev for dev command
		c.Provide(func() *redis.Client {
			return redis.NewClient(&redis.Options{
				Addr: r.redisAddress,
			})
		}),
        // more environment-specific dependencies might be provided
	)
	if err != nil {
		return err
	}

	return RunServer(ctx, c) // a Runner always calls RunServer in cmd/server.go
}
```

# File cmd/server.go

The file `./cmd/server.go` always contains the single function RunServer that registers dependencies which are the same for all environments, registers HTTP handlers, registers workers and finally runs all workers with `runutil.RunProvidedWorkers`.

It looks similar to the example below. It is useful to group all `webutil.ProvideHandler` functions and all `runutil.ProvideWorker` functions.

```
func RunServer(ctx context.Context, c *dig.Container) error {
	err := errors.Join(
		c.Provide(templates.New),

		// Register HTTP handlers
		webutil.ProvideHandler(c, handlers.NewIndexHandler),
		webutil.ProvideHandler(c, handlers.NewHealthHandler),
		webutil.ProvideHandler(c, handlers.NewUsersHandler),

		c.Provide(func(
			authMiddleware webutil.AuthMiddleware,
		) webutil.Middlewares {
			return webutil.Middlewares(append(
				webutil.DefaultMiddlewares(),
				authMiddleware,
			))
		}),

		// Register background workers
		runutil.ProvideWorker(c, func(redisClient *redis.Client) *workers.DataSyncWorker {
			return workers.NewDataSyncWorker(redisClient)
		}),
		runutil.ProvideWorker(c, workers.NewPeriodicTaskWorker),

		// Register the HTTP server itself
		runutil.ProvideWorker(c, webutil.NewServer),
	)
	if err != nil {
		return err
	}

	// Start all registered workers
	return runutil.RunProvidedWorkers(ctx, c)
}
```


# Package pkg/bll

The package `./pkg/bll` contains isolated packages, that do not need access to things like network or the OS. Also they are usually very well testable.

* A good example is `xff` that takes HTTP headers as input and outputs the real-ip.
* Another good example is `humanize` that takes an integer and returns a human readable version with K, M or G sufixes.
* A bad example is a redis client.


# Package pkg/dal

The package `./pkg/dal` contains wrapper packages that help accessing data from outside of the program. Usually that are HTTP clients.


# Package pkg/app

The package `./pkg/app` contains sub-packages that define the actual project logic. Most common ones are

- pkg/app/handlers — Contains all HTTP handlers.
- pkg/app/templates — Contains HTML templates.
- pkg/app/workers — Contains background workers.

There might be additional packages, but they need to be focused on a specific topic. For example something like `pkg/app/tasks`, which contains a bunch of different task implementations.

# Package pkg/app/workers

The package `./pkg/app/workers` contains all background workers. There is one worker perfile, but there might be subworkers in each worker.

The worker must be registered using `runutil.ProvideWorker` in `cmd/server.go`.

The worker must implement this interface:

```
type WorkerConfiger interface {
	Workers() []Worker
}
```

All files should follow this example:

```
package workers

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/runutil"
)

// DataSyncWorker is responsible for periodically syncing data
type DataSyncWorker struct {
	redisClient *redis.Client // this is an example dependency
}

// NewDataSyncWorker creates a new data sync worker
func NewDataSyncWorker(redisClient *redis.Client) *DataSyncWorker {
	return &DataSyncWorker{
		redisClient: redisClient,
	}
}

// Workers implements the runutil.WorkerConfiger interface
func (w *DataSyncWorker) Workers() []runutil.Worker {
	return []runutil.Worker{
		runutil.DeclarativeWorker{
			Name:   "DataSyncWorker",
			Worker: runutil.Repeat(5*time.Minute, runutil.JobFunc(w.syncData)),
		},
	}
}

// syncData performs the actual data synchronization
func (w *DataSyncWorker) syncData(ctx context.Context) error {
	logutil.Get(ctx).Info("Synchronizing data...")

	// Record the current time in Redis as our last sync
	_, err := w.redisClient.Set(ctx, "last_sync", time.Now().Format(time.RFC3339), 0).Result()
	if err != nil {
		return fmt.Errorf("failed to update last sync time: %w", err)
	}

	// Simulate some work
	time.Sleep(500 * time.Millisecond)

	// Update the counter in Redis
	_, err = w.redisClient.Incr(ctx, "sync_count").Result()
	if err != nil {
		return fmt.Errorf("failed to update sync counter: %w", err)
	}

	logutil.Get(ctx).Info("Data synchronization completed")
	return nil
}
```


# Package pkg/app/handlers

The package `./pkg/app/handlers` contains all HTTP handlers. There is one handler per file and one handler might handle multiple routes.

The handler must be registered using `webutil.ProvideHandler` in `cmd/server.go`.

The handler must implement this interface:

```
type Handler interface {
	Register(chi.Router)
}
```

All files should follow this example:

```
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rebuy-de/rebuy-go-sdk/v9/examples/full/pkg/app/templates"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
)

// IndexHandler handles the home page
type IndexHandler struct {
	viewer *templates.Viewer
}

// NewIndexHandler creates a new index handler
func NewIndexHandler(
	viewer *templates.Viewer,
) *IndexHandler {
	return &IndexHandler{
		viewer: viewer,
	}
}

// Register registers the handler's routes
func (h *IndexHandler) Register(r chi.Router) {
	r.Get("/", webutil.WrapView(h.handleIndex)) // the path is always the full path
    // might contain additional routes
}

func (h *IndexHandler) handleIndex(r *http.Request) webutil.Response {
	return templates.View(http.StatusOK, h.viewer.WithRequest(r).HomePage())
}
```

# Package pkg/app/templates

When using templ as template engine, the package `./pkg/app/templates` looks like described here.

The file `./pkg/app/templates/view.go` always looks like this:

```
package templates

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
)

//go:generate go run github.com/a-h/templ/cmd/templ generate
//go:generate go run github.com/a-h/templ/cmd/templ fmt .

type Viewer struct {
	assetPathPrefix webutil.AssetPathPrefix
    // All values that are needed by the templates and are provided by dig should go here.
}

type RequestAwareViewer struct {
	*Viewer
	request *http.Request
    // Should only contain fields that change between requests. Everything else should be injected into the Viewer.
}

func New(
	assetPathPrefix webutil.AssetPathPrefix,
) *Viewer {
	return &Viewer{
		assetPathPrefix: assetPathPrefix,
	}
}

func (v *Viewer) assetPath(path string) string {
	return fmt.Sprintf("/assets/%v%v", v.assetPathPrefix, path)
}

func (v *Viewer) WithRequest(r *http.Request) *RequestAwareViewer {
	return &RequestAwareViewer{
		Viewer:  v,
		request: r,
	}
}

func View(status int, node templ.Component) webutil.Response {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)

		err := node.Render(r.Context(), w)
		if err != nil {
			logutil.Get(r.Context()).Error(err)
		}
	}
}
```

The `RequestAwareViewer` is only needed, when a component accessed request data, like auth information. If that is not the case the `Viewer` is enough, but it is fine to always use the `RequestAwareViewer`.

The `RequestAwareViewer` can be called like this from a handler:

```
return templates.View(http.StatusOK,
	h.viewer.WithRequest(r).APIKeyPage(apikeys))
```

An example component could look like this:

```
templ (v *RequestAwareViewer) APIKeyPage(apikeys []sqlc.Apikey) {
	@v.page("API Keys") {
		<ul>
			for _, key := range apikeys {
				<li>{ key }</li>
			}
		</ul>
	}
}
```

It is advised to have the file `./pkg/app/templates/page.go` to have a base layout that looks like this:

```
templ (v *RequestAwareViewer) base(title string) {
	<!DOCTYPE html>
	<html lang="en">
		<head>
			<meta charset="UTF-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
			<title>{ title }</title>
			<link rel="icon" type="image/svg+xml" href={ v.assetPath("/favicon.svg") }/>
			<link rel="stylesheet" href={ v.assetPath("/index.css") }/>
			<script src={ v.assetPath("/index.js") }></script>
			<script src={ v.assetPath("/hyperscript.org/dist/_hyperscript.min.js") }></script>
			<script src={ v.assetPath("/hyperscript.org/dist/template.js") }></script>
			<script src={ v.assetPath("/htmx.org/dist/htmx.min.js") }></script>
			<script src={ v.assetPath("/idiomorph/dist/idiomorph-ext.min.js") }></script>
		</head>
		<body hx-ext="morph">
			<nav class="navbar" role="navigation" aria-label="main navigation">
				<div class="navbar-brand">
					<a class="navbar-item" href="/">
						<img src={ v.assetPath("/favicon.svg") } width="28" height="28" class="mr-3"/>
						<strong>LLM Gateway</strong>
					</a>
				</div>
				<div class="navbar-menu">
					<div class="navbar-start"></div>
					<div class="navbar-end">
						@v.authComponent()
						<div class="navbar-item">
							<button _="on click send ry:toggleTheme to <html/>">
								<i class="fa-solid fa-circle-half-stroke"></i>
							</button>
						</div>
					</div>
				</div>
			</nav>
			<section class="section">
				<div class="container-fluid">
					{ children... }
				</div>
			</section>
		</body>
	</html>
}

templ (v *RequestAwareViewer) page(title string) {
	// Store the title in the viewer
	// v.currentTitle = title - done in WithRequestPage
	@v.base(title) {
		{ children... }
	}
}
```

# Package pkg/dal/sqlc

The package `./pkg/app/sqlc` contains all SQL queries, when using SQLC.

SQL queries are stored in files with the name pattern `query_$table.sql`. SQL reads those files and writes Go code in `query_$table.sql`. The command for this is `go run github.com/sqlc-dev/sqlc/cmd/sqlc generate`, which gets executed by `go generate`.

The file `./pkg/dal/sqlc/sqlc.go` should always look close like this:

```
package sqlc

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:generate go run github.com/sqlc-dev/sqlc/cmd/sqlc generate

func NewQueries(ctx context.Context, uri string) (*Queries, error) {
	config, err := pgxpool.ParseConfig(uri)
	if err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}

	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("parse uri: %w", err)
	}

	return New(db), nil
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(ctx context.Context, uri string) error {
	config, err := pgx.ParseConfig(uri)
	if err != nil {
		return fmt.Errorf("parse uri: %w", err)
	}

	db := stdlib.OpenDB(*config)
	defer db.Close()

	_, err = db.ExecContext(ctx, "create schema if not exists platform_inventory;")
	if err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to load source driver: %w", err)
	}

	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to load database driver: %w", err)
	}

	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		"postgres", databaseDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to setup migration: %w", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to do migration: %w", err)
	}

	return nil
}

func URI(base, username, password string) (string, error) {
	dbURI, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	dbURI.User = url.UserPassword(username, password)
	return dbURI.String(), nil
}
```

The file `./pkg/dal/sqlc/tx.go` should always look close like this:

```
package sqlc

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/logutil"
)

type WithTxFunc func(*Queries) error

type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func (q *Queries) Tx(ctx context.Context, fn WithTxFunc) error {
	db, ok := q.db.(beginner)
	if !ok {
		return fmt.Errorf("DB interface does not implement transactions: %T", q.db)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := q.WithTx(tx)

	err = fn(qtx)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (q *Queries) Hijack(ctx context.Context) (*Queries, func(), error) {
	pool := q.db.(*pgxpool.Pool)
	pconn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}

	conn := pconn.Hijack()

	closer := func() {
		err := conn.Close(context.Background())
		if err != nil {
			logutil.Get(ctx).Error(err)
		}
	}

	return New(conn), closer, nil
}
```

```
version: 2
sql:
  - engine: "postgresql"
    schema: "migrations/"
    queries: "."
    gen:
      go:
        package: "sqlc"
        out: "."
        sql_package: "pgx/v5"

        emit_json_tags: true
        emit_pointers_for_null_types: true

        output_db_file_name: gen_db.go
        output_models_file_name: gen_models.go

        json_tags_case_style: camel

        # rename contains an object with a mapping from postgres identifiers to Go identifiers.
        # The mapping is done by sqlc and only needs an entry here, if the auto generated one is wrong. This is mostly the case for wrong initialisms.
        rename:
          my_example_uid: MyExampleUID # example

        # overrides specifies to which Go type a database entry gets deserialized to.
        overrides:
          # UUIDs should always be deserialized the Google UUID package.
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              package: "uuid"
              type: "UUID"
          - db_type: "uuid"
            nullable: true
            go_type:
                import: "github.com/google/uuid"
                package: "uuid"
                type: "NullUUID"

          # Timestamps should always be deserialized to native Go times.
          - db_type: "timestamptz"
            go_type:
              import: "time"
              type: "Time"
          - db_type: "timestamptz"
            nullable: true
            go_type:
              import: "time"
              type: "Time"
              pointer: true

          # there might be other project-specific entries
```

The directory `./pkg/dal/sqlc/migrations` contains all migration scripts. They have the file pattern `DDDD_$title.up.sql`, where DDDD is a number with 0 padding and $title is short title of the migration step.

# Pakage web

The package `./web` contains all web assets that get delivered to the browser. It supports dependency management by Yarn.

The file `./web/web.go` is the interface to other Go packages and must look like this:

```
package web

import (
	"embed"
	"io/fs"
	"os"

	"github.com/rebuy-de/rebuy-go-sdk/v9/pkg/webutil"
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

The file `./web/esbuild.config.mjs` contains the build script and looks like this:

```
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

The `scripts` array only needs to contain files, that are actually used in any HTML `<head>`. The remaining code above should follow the example closely.

The file `./web/package.json` describes the needed dependencies and looks like this, where the actual dependencies might be different:

```
{
  "name": "project-name",
  "version": "1.0.0",
  "packageManager": "yarn@4.7.0",
  "private": true,
  "dependencies": {
    "@fortawesome/fontawesome-free": "^6.7.2",
    "bulma": "^1.0.4",
    "htmx.org": "^2.0.4",
    "hyperscript.org": "^0.9.14",
    "idiomorph": "^0.7.3"
  },
  "devDependencies": {
    "esbuild": "^0.25.4",
    "nodemon": "^3.1.10"
  },
  "scripts": {
    "build": "node esbuild.config.mjs"
  }
}
```
