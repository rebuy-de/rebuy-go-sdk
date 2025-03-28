# rebuy-go-sdk

[![GoDoc](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk?status.svg)](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk)
![Build Status](https://github.com/rebuy-de/rebuy-go-sdk/workflows/Golang/badge.svg?branch=main)

Library for our Golang projects

> **Development Status** *rebuy-go-sdk* is designed for internal use. Since it
> uses [Semantic Versioning](https://semver.org/) it is safe to use, but expect
> big changes between major version updates.

## Table of Contents
- [Application Layout](#application-layout)
- [Command Structure with cmdutil](#command-structure-with-cmdutil)
- [Runner Pattern](#runner-pattern)
- [HTTP Handlers with webutil](#http-handlers-with-webutil)
- [Worker Management with runutil](#worker-management-with-runutil)
- [Dependency Injection with digutil](#dependency-injection-with-digutil)
- [Major Release Notes](#major-release-notes)

## Application Layout

### General Directory Structure

Please take a look at the examples directory to see how it actually looks like.

```
/
├── cmd/[subcommand/]
│   ├── `root.go`
│   └── `...`
├── `pkg/`
│   ├── `app/...`
│   ├── `dal/...`
│   ├── `bll/...`
│   └── `...`
├── `buildutil`
├── `go.mod`
├── `go.sum`
├── `LICENSE`
├── `main.go`
├── `README.md`
└── `tools.go`
```

* `/buildutil` is a convenience wrapper to execute the `buildutil` command
  from the SDK. It ensures that the application gets built with a defined
  version of `buildutil`.
* `/main.go` is the entrypoint of the application. It's typically very minimal,
  containing just enough code to initialize the command framework and handle errors.
  Its primary responsibility is to set up the application with the SDK's `cmdutil` package
  and delegate execution to the Cobra command structure defined in `/cmd/root.go`.
* `/tools.go` forces dependency management of modules, that are not directly
  imported. This is important for `go run` and `go generate` that use external
  modules. See [wiki][tools-wiki] for more details.
* `/cmd/root.go` contains the definition for all [Cobra][Cobra] commands and
  the Runners (see below) of the application. This is where you define your command-line
  interface structure, options, and connect the commands to their implementations.

### Command Structure Organization

The separation of concerns in command files follows a clear pattern:

**main.go** - Minimal application entry point:
```go
func main() {
    defer cmdutil.HandleExit()

    if err := cmd.NewRootCommand().Execute(); err != nil {
        logrus.Fatal(err)
    }
}
```

**cmd/root.go** - Command definition and runner setup:
```go
func NewRootCommand() *cobra.Command {
    runner := new(Runner)

    cmd := cmdutil.New(
        "myapp", "github.com/org/myapp",
        cmdutil.WithLogVerboseFlag(),
        cmdutil.WithVersionCommand(),
        cmdutil.WithRunner(runner),
    )

    // Add additional subcommands if needed
    cmd.AddCommand(newSubCommand())

    return cmd
}

// Runner implementation follows...
```

**cmd/server.go** - Server configuration and setup:
```go
// RunServer configures and starts the application server with dependency injection
func RunServer(ctx context.Context, c *dig.Container) error {
    // Register core dependencies
    err := errors.Join(
        // Register template viewer
        c.Provide(func(templateFS fs.FS) *webutil.GoTemplateViewer {
            return webutil.NewGoTemplateViewer(templateFS,
                webutil.SimpleTemplateFuncMap("formatTime", FormatTimeFunction),
            )
        }),

        // Register HTTP handlers
        webutil.ProvideHandler(c, handlers.NewUserHandler),
        webutil.ProvideHandler(c, handlers.NewDashboardHandler),

        // Register background workers
        runutil.ProvideWorker(c, workers.NewSyncWorker),

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

This separation of concerns follows a clear pattern:

1. `/main.go` initializes the command framework and handles errors
2. `/cmd/root.go` defines the CLI structure and environment-specific runners
3. `/cmd/server.go` contains shared server setup code used by all environments

The environment-specific runners in root.go do initialization specific to their environment
(production, development, etc.) and then call the common `RunServer` function to set up
the application components that are environment-independent:

```go
// In cmd/root.go
func (r *Runner) Run(ctx context.Context) error {
    // Production environment setup
    c := dig.New()

    // Provide production-specific dependencies
    err := errors.Join(
        c.Provide(func() *redis.Client {
            return redis.NewClient(&redis.Options{
                Addr: r.redisAddress,
            })
        }),
        c.Provide(func() fs.FS {
            return templateFS
        }),
    )
    if err != nil {
        return err
    }

    // Call common server setup
    return server.RunServer(ctx, c)
}

func (r *Runner) Dev(ctx context.Context, cmd *cobra.Command, args []string) error {
    // Development environment setup
    c := dig.New()

    // Provide development-specific dependencies (like local Redis)
    podman, err := podutil.DevPodman(ctx)
    if err != nil {
        return err
    }

    redisContainer, err := podutil.StartDevcontainer(ctx, podman, "dev-redis",
        "docker.io/redis:latest")
    if err != nil {
        return err
    }

    err = errors.Join(
        c.Provide(func() *redis.Client {
            return redis.NewClient(&redis.Options{
                Addr: redisContainer.TCPHostPort(6379),
            })
        }),
        c.Provide(func() fs.FS {
            // Use local filesystem for templates in dev mode
            return os.DirFS("./templates")
        }),
    )
    if err != nil {
        return err
    }

    // Call common server setup
    return server.RunServer(ctx, c)
}
* `/pkg/app` contains separate components of the application. The `/pkg/app`
  directory serves basically the same purpose as the `/cmd`, but is separated
  into multiple sub-packages. This is useful when the `/cmd` directory grows
  too big and contains components that are mostly independent from each other.
  How the sub packages of `/pkg/app` are designed is highly dependent on the
  application. It could be split model-based (eg users, projects, ...) or it
  could be split purpose-based (eg web, controllers, ...).
* `/pkg/bll` stands for "business logic layer" and contains sub-packages that
  solve a specific use-case of the application.
* `/pkg/dal` stands for "data access layer" and contains sub-packages that
  serve as a wrapper for external services and APIs. The idea of grouping such
  packages is to make their purpose clear and to avoid mixing access to
  external services with actual business logic.

[Cobra]: https://github.com/spf13/cobra
[tools-wiki]: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

## Command Structure with cmdutil

The SDK provides a streamlined approach to creating command-line applications through the `cmdutil` package. Here's how to set up your application:

```go
func main() {
    defer cmdutil.HandleExit()

    cmd := cmdutil.New(
        "myapp",                          // Short app name
        "github.com/org/myapp",           // Full app name
        cmdutil.WithLogVerboseFlag(),     // Add -v flag for verbose logging
        cmdutil.WithLogToGraylog(),       // Add Graylog support
        cmdutil.WithVersionCommand(),     // Add version command
        cmdutil.WithVersionLog(logrus.DebugLevel),
        cmdutil.WithRunner(new(Runner)),  // Add main application runner
    )

    if err := cmd.Execute(); err != nil {
        logrus.Fatal(err)
    }
}
```

This approach provides a consistent interface for command-line applications with built-in support for logging, versioning, and other common capabilities.

## Runner Pattern

Runners are structs that define command line flags and prepare the application for launch.

### Basic Runner Structure

```go
type Runner struct {
    name string
    redisAddress string
    // Other configuration fields
}

// Bind defines command line flags
func (r *Runner) Bind(cmd *cobra.Command) error {
    cmd.PersistentFlags().StringVar(
        &r.name, "name", "World",
        `Your name.`)

    cmd.PersistentFlags().StringVar(
        &r.redisAddress, "redis-address", "localhost:6379",
        `Redis server address.`)

    return nil
}

// Run executes the main application logic
func (r *Runner) Run(ctx context.Context) error {
    // Application setup and launch
    return nil
}
```

### Environment-Specific Runners

You can create different environment configurations for your application:

```go
// Run for production environment
func (r *Runner) Run(ctx context.Context) error {
    redisClient := redis.NewClient(&redis.Options{
        Addr: r.redisAddress,
    })

    // Production setup
    return r.runServer(ctx, redisClient)
}

// Dev runs the server in development mode
func (r *Runner) Dev(ctx context.Context, cmd *cobra.Command, args []string) error {
    // Create a local test Redis instance
    podman, err := podutil.DevPodman(ctx)
    if err != nil {
        return err
    }

    keydbContainer, err := podutil.StartDevcontainer(ctx, podman, "app-dev-keydb",
        "docker.io/eqalpha/keydb:latest")
    if err != nil {
        return err
    }

    redisClient := redis.NewClient(&redis.Options{
        Addr: keydbContainer.TCPHostPort(6379),
    })

    // Development setup with hot reloading, etc.
    return r.runServer(ctx, redisClient)
}

// Shared server setup with environment-specific dependencies
func (r *Runner) runServer(ctx context.Context, redisClient *redis.Client) error {
    // Common server setup and run
}
```

The purpose of splitting the Runner and the actual application code is:
* to get initializing errors as fast as possible (eg if the Redis server is not available),
* to be able to execute environment-specific code without having to use conditionals all over the code-base,
* to be able to mock services for local development
* and to define a proper interface for the application launch, which is very helpful for e2e tests.

## HTTP Handlers with webutil

The SDK provides a `webutil` package that streamlines HTTP request handling across projects.

### Creating Handlers

Handlers should be organized as structs that follow the SDK's handler registration pattern:

```go
// Define a handler struct with dependencies
type MyHandler struct {
    // Dependencies injected via constructor
    store     *SomeStore
    viewer    *webutil.GoTemplateViewer
}

// Constructor that creates a new handler instance
func NewMyHandler(store *SomeStore, viewer *webutil.GoTemplateViewer) *MyHandler {
    return &MyHandler{
        store:  store,
        viewer: viewer,
    }
}

// Register routes on a chi Router
func (h *MyHandler) Register(router chi.Router) {
    router.Get("/api/resource", webutil.WrapView(h.handleGetResource))
    router.Post("/api/resource", webutil.WrapView(h.handleCreateResource))
}

// Handler functions return a webutil.Response (which is an http.HandlerFunc)
func (h *MyHandler) handleGetResource(r *http.Request) webutil.Response {
    // Get data from store
    data, err := h.store.Get(r.Context(), "some-id")
    if err != nil {
        return webutil.ViewError(http.StatusInternalServerError, err)
    }

    // Return appropriate response
    return h.viewer.HTML(http.StatusOK, "resource.html", data)
    // Or for API endpoints
    // return webutil.ViewJSON(http.StatusOK, data)
}
```

### Response Helpers

The `webutil` package provides several helper functions to generate HTTP responses:

```go
// HTML response using a template
return h.viewer.HTML(http.StatusOK, "template.html", data)

// JSON response
return webutil.ViewJSON(http.StatusOK, data)

// Error response
return webutil.ViewError(http.StatusInternalServerError, err)

// Formatted error
return webutil.ViewErrorf(http.StatusBadRequest, "invalid parameter: %s", param)

// Redirect response
return webutil.ViewRedirect(http.StatusSeeOther, "/new-location")

// Empty response
return webutil.ViewNoContent(http.StatusNoContent)

// Inline HTML (for HTMX partial updates)
return webutil.ViewInlineHTML(http.StatusOK, "<span>Updated %s</span>", item)
```

### Template Viewers

The SDK supports different template engines:

1. **GoTemplateViewer** - For standard Go HTML templates
   ```go
   viewer := webutil.NewGoTemplateViewer(templateFS,
       webutil.SimpleTemplateFuncMap("formatTime", FormatTimeFunction),
       webutil.SimpleTemplateFuncMaps(template.FuncMap{
           "truncate": TruncateFunction,
           "format": FormatFunction,
       }),
   )
   ```

2. **JetViewer** - For the Jet template engine (provided by extension packages)
   ```go
   // Create a Jet loader from an fs.FS
   loader := webutilext.JetFSLoader{FS: templateFS}
   jetSet := jet.NewSet(loader)

   // Create the viewer with functions
   viewer := webutilext.NewJetViewer(jetSet,
       webutilext.JetFunctionOption("formatTime", FormatTimeFunction),
       webutilext.JetFunctionMapOption(map[string]any{
           "truncate": TruncateFunction,
       }),
   )
   ```

3. **Templ** - For the [templ](https://github.com/a-h/templ) type-safe HTML template engine

   Templ can be integrated with the SDK's webutil framework by creating a custom viewer type that adapts templ components to return webutil.Response functions.

   ```go
   // suggested content for pkg/app/templates/view.go
   package templates

   import (
       "fmt"
       "net/http"

       "github.com/a-h/templ"
       "github.com/rebuy-de/rebuy-go-sdk/v8/pkg/logutil"
       "github.com/rebuy-de/rebuy-go-sdk/v8/pkg/webutil"
   )

   //go:generate go run github.com/a-h/templ/cmd/templ generate
   //go:generate go run github.com/a-h/templ/cmd/templ fmt .

   type Viewer struct {
       assetPathPrefix webutil.AssetPathPrefix
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

   ```go
   // Usage with webutil handler:
   func (h *Handler) handleHome(r *http.Request) webutil.Response {
       data := GetPageData()
       return templates.View(http.StatusOK, HomePage(data))
   }
   ```

### Handler Registration with Dependency Injection

The SDK uses the [dig](https://github.com/uber-go/dig) dependency injection container to manage and register HTTP handlers:

```go
func RunServer(ctx context.Context, c *dig.Container) error {
    // Register the template viewer
    err := c.Provide(func(templateFS fs.FS, assetPathPrefix webutil.AssetPathPrefix) *webutil.GoTemplateViewer {
        return webutil.NewGoTemplateViewer(templateFS,
            webutil.AuthTemplateFunctions,
            webutil.SimpleTemplateFuncMap("formatTime", FormatTimeFunction),
            // Add more template functions as needed
        )
    })
    if err != nil {
        return err
    }

    // Register handlers with the dig container
    err = errors.Join(
        // Each of these calls registers a handler that implements Handler interface
        webutil.ProvideHandler(c, handlers.NewUserHandler),
        webutil.ProvideHandler(c, handlers.NewResourceHandler),
        webutil.ProvideHandler(c, handlers.NewWebhookHandler),

        // Register server last
        runutil.ProvideWorker(c, webutil.NewServer),
    )
    if err != nil {
        return err
    }

    // Run all registered workers (including the web server)
    return runutil.RunProvidedWorkers(ctx, c)
}
```

### Server Setup

The SDK provides a `webutil.Server` type that manages HTTP server setup:

```go
// In your main Runner.Run method:
func (r *Runner) Run(ctx context.Context) error {
    // Set up the dig container
    c := dig.New()

    // Provide dependencies (databases, clients, configs)
    err := errors.Join(
        c.Provide(func() fs.FS { return templateFS }),
        c.Provide(webutil.AssetDefaultProd), // Provides AssetPathPrefix and AssetCacheDuration
        // ... other dependencies
    )
    if err != nil {
        return err
    }

    // Run the server
    return RunServer(ctx, c)
}
```

## Worker Management with runutil

The SDK provides a robust worker management system in the `runutil` package. This makes it easy to run and manage long-running services and one-off jobs.

### Worker Interface

```go
// Worker is a service that is supposed to run continuously until the context is cancelled
type Worker interface {
    Run(ctx context.Context) error
}

// Job is a function that runs once and exits afterwards
type Job interface {
    RunOnce(ctx context.Context) error
}
```

### Worker with Dependency Injection

The SDK integrates with the dig dependency injection library:

```go
func SetupWorkers(ctx context.Context, c *dig.Container) error {
    // Register workers with the dig container
    err := errors.Join(
        runutil.ProvideWorker(c, workers.NewDatabaseCleanupWorker),
        runutil.ProvideWorker(c, workers.NewDataFetchWorker),
        runutil.ProvideWorker(c, workers.NewEventWatcherWorker),
    )
    if err != nil {
        return err
    }

    // Run all provided workers
    return runutil.RunProvidedWorkers(ctx, c)
}
```

### Retry and Backoff

The `runutil` package provides utilities for retrying operations with backoff:

```go
func FetchData(ctx context.Context) error {
    return runutil.Retry(ctx, func(ctx context.Context) error {
        // Operation that might fail
        return apiClient.FetchData(ctx)
    },
        runutil.WithMaxAttempts(5),
        runutil.WithBackoff(runutil.ExponentialBackoff(time.Second, 30*time.Second)),
        runutil.WithRetryableErrors(ErrTemporary, ErrTimeout),
    )
}
```


## Dependency Injection with digutil

The SDK uses Uber's [dig](https://github.com/uber-go/dig) library for dependency injection and provides helpers in the `digutil` package.

### Using Parameter Objects for Optional Dependencies

The `digutil` package provides helpers for optional dependencies:

```go
// Define options for a service
type ServiceOptions struct {
    // Required options
    Database *sql.DB

    // Optional options with defaults
    CacheTTL time.Duration `optional:"true"`
    MaxConns int           `optional:"true"`
    Logger   *log.Logger   `optional:"true"`
}

// Service constructor using options
func NewService(options ServiceOptions) *Service {
    // Apply defaults for optional parameters
    if options.CacheTTL == 0 {
        options.CacheTTL = 5 * time.Minute
    }

    if options.MaxConns == 0 {
        options.MaxConns = 10
    }

    if options.Logger == nil {
        options.Logger = log.Default()
    }

    return &Service{
        db:       options.Database,
        cacheTTL: options.CacheTTL,
        maxConns: options.MaxConns,
        logger:   options.Logger,
    }
}
```


## Major Release Notes

Note: `vN` is the new release (eg `v3`) and `vP` is the previous one (eg `v2`).

1. Create a new branch `release-vN` to avoid breaking changes getting into the previous release.
2. Do your breaking changes in the branch.
3. Update the imports everywhere:
   * `find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vO#github.com/rebuy-de/rebuy-go-sdk/vP#g' {} +`
4. Merge your branch.
5. Add Release on GitHub.
