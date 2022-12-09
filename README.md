# rebuy-go-sdk

[![GoDoc](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk?status.svg)](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk)
![Build Status](https://github.com/rebuy-de/rebuy-go-sdk/workflows/Golang/badge.svg?branch=main)

Library for our Golang projects

> **Development Status** *rebuy-go-sdk* is designed for internal use. Since it
> uses [Semantic Versioning](https://semver.org/) it is safe to use, but expect
> big changes between major version updates.


## Application Layout

### General Directory Structure

Please take a look at the examples directory to see how it actually looks like.

```
/
├── cmd/[subcommand/]
│   ├── `root.go`
│   └── `...`
├── `pkg/`
│   ├── `dal/...`
│   ├── `bll/...`
│   └── `...`
├── `buildutil`
├── `Dockerfile`
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
* `/main.go` is the entrypoint of the application and delegates the execution
  to the [Cobra][Cobra] specific functions in `/cmd`.
* `/tools.go` forces dependency management of modules, that are not directly
  imported. This is important for `go run` and `go generate` that use external
  modules. See [wiki][tools-wiki] for more details.
* `/cmd/root.go` contains the definition for all [Cobra][Cobra] commands and
  the Runners (see below) of the application.
* `/pkg/bll` stands for "business logic layer" and contains sub-packages that
  solve a specific use-case of the application.
* `/pkg/dal` stands for "data access layer" and contains sub-packages that
  serve as a wrapper for external services and APIs. The idea of grouping such
  packages is to make their purpose clear and to avoid mixing access to
  external services with actual business logic.

[Cobra]: https://github.com/spf13/cobra
[tools-wiki]: https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module


### Runners

Runners are structs that defines command line flags and prepare that application for launch.

The definition of the flags happen in a function typically called `Bind` with
the signature `func(cmd *cobra.Command) error`:

```go
func (r *Runner) Bind(cmd *cobra.Command) error {
	cmd.PersistentFlags().StringVar(
		&r.name, "name", "World",
		`Your name.`)
	return nil
}
```

The launch preparation happens in functions with the signature `func(ctx
context.Context, cmd *cobra.Command, args []string`. There can be multiple
function for different environments (eg `Default`, `Dev`, `Staging`). The
advantage of having multiple functions is that it is easy to setup services in
a environment specific way. For example while the `default` settings use the
normal Redis client and use files from `go:embed`, the `dev` settings could
setup a [miniredis][miniredis] instance and open files directly from the file
system to avoid having to restart the server on HTML template changes.

See [`examples/full/cmd/root.go`][full-example-root] for a typical
implementation.

[full-example-root]: https://github.com/rebuy-de/rebuy-go-sdk/blob/master/examples/full/cmd/root.go
[miniredis]: https://github.com/alicebob/miniredis

The purpose of splitting the Runner and the actual application code is:

* to get initializing errors as fast as possible (eg if the Redis server is not available),
* to be able to execute environment-specific code without having to use conditionals all over the code-base,
* to be able to mock services for local development
* and to define a proper interface for the application launch, which is very helpful for e2e tests.


## Major Release Notes

Note: `vN` is the new release (eg `v3`) and `vP` is the previous one (eg `v2`).

1. Create a new branch `release-vN` to avoid breaking changes getting into the previous release.
2. Do your breaking changes in the branch.
3. Update the imports everywhere:
   * `find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vO#github.com/rebuy-de/rebuy-go-sdk/vP#g' {} +`
4. Merge your branch.
5. Add Release on GitHub.
