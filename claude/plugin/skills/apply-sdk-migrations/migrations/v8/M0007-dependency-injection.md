---
id: M0007
title: Switch to Dependency Injection
date: 2024-11-08
sdk_version: v8
type: minor
---

# Switch to Dependency Injection

## Reasoning

Our previous approach to have a single `Server` struct and to put all logic there gets messy pretty fast. Worse than
that it is hard to refactor this to separate this into multiple structs or packages. Dependency injection will help us
to separate those things in the very beginning of a project without manual wiring of dependencies.

## Hints

* The primary server.Run function should be replaced with a standalone `RunServer(ctx context.Context, c *dig.Container) error`
* The `cmdutil.Runner` should setup environment specific dependencies (eg Redis vs Miniredis) while `RunServer` should
  setup the independent ones (eg services, workers and handlers).

## Examples

The `RunServer` function could look like this:

```go
func RunServer(ctx context.Context, c *dig.Container) error {
	return errors.Join(
		// define services and repos
		c.Provide(...),

		// define HTTP handlers
		webutil.ProvideHandler(c, handlers.New...),

		// define workers
		runutil.ProvideWorker(c, workers.New...),

		// start all workers
		runutil.RunProvidedWorkers(ctx, c),
	)
}
```
