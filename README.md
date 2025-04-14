# rebuy-go-sdk

[![GoDoc](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk?status.svg)](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk)
![Build Status](https://github.com/rebuy-de/rebuy-go-sdk/workflows/Golang/badge.svg?branch=main)

Library for our Golang projects

> **Development Status** *rebuy-go-sdk* is designed for internal use. Since it
> uses [Semantic Versioning](https://semver.org/) it is safe to use, but expect
> big changes between major version updates.

## Documentation

The complete SDK documentation is available via standard Go documentation tools. Use `go doc` or visit the [GoDoc site](https://godoc.org/github.com/rebuy-de/rebuy-go-sdk) to browse the documentation.

### Major topics:

- Application Layout - see `go doc github.com/rebuy-de/rebuy-go-sdk`
- Command Structure with cmdutil - see `go doc github.com/rebuy-de/rebuy-go-sdk/pkg/cmdutil`
- Runner Pattern - see `go doc github.com/rebuy-de/rebuy-go-sdk/pkg/cmdutil`
- HTTP Handlers with webutil - see `go doc github.com/rebuy-de/rebuy-go-sdk/pkg/webutil`
- Worker Management with runutil - see `go doc github.com/rebuy-de/rebuy-go-sdk/pkg/runutil`
- Dependency Injection with digutil - see `go doc github.com/rebuy-de/rebuy-go-sdk/pkg/digutil`

## Examples

For practical examples of using the SDK, check the `examples/` directory, which contains:

- `examples/minimal/` - A minimal application using the SDK
- `examples/full/` - A complete application with HTTP handlers, workers, and more

## Major Release Notes

Note: `vN` is the new release (eg `v3`) and `vP` is the previous one (eg `v2`).

1. Create a new branch `release-vN` to avoid breaking changes getting into the previous release.
2. Do your breaking changes in the branch.
3. Update the imports everywhere:
   * `find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vO#github.com/rebuy-de/rebuy-go-sdk/vP#g' {} +`
4. Merge your branch.
5. Add Release on GitHub.