---
id: M0009
title: Switch from logrus to slog
date: 2025-02-19
sdk_version: v10
type: major
---

# Switch from logrus to slog

## Reasoning

`logrus` is in maintenance mode and `log/slog` is now part of the Go standard library (since Go 1.21). Using `slog`
reduces external dependencies and aligns with the Go ecosystem's direction. The Graylog
integration is now handled via `samber/slog-graylog` with `samber/slog-multi` for fanout to multiple handlers (CLI +
Graylog).

## Breaking Changes

* `logutil.Get(ctx)` now returns `*slog.Logger` instead of `logrus.FieldLogger`.
* `cmdutil.WithVersionLog()` now takes `slog.Level` instead of `logrus.Level`.
* All `logrus.Infof(...)` / `logrus.Warnf(...)` / `logrus.Errorf(...)` calls must be replaced with slog equivalents.
* `logrus.Fatal(err)` in `main()` should be replaced with `slog.Error("...", "error", err)` + `os.Exit(1)`.

## Migration Steps

### 1. Update imports

Replace all `"github.com/sirupsen/logrus"` imports with `"log/slog"`.

### 2. Replace logrus calls

| logrus                                 | slog                                          |
| -------------------------------------- | --------------------------------------------- |
| `logrus.Info("msg")`                   | `slog.Info("msg")`                            |
| `logrus.Infof("msg %s", v)`            | `slog.Info("msg", "key", v)`                  |
| `logrus.WithField("k", v).Info("msg")` | `slog.Info("msg", "k", v)`                    |
| `logrus.WithError(err).Error("msg")`   | `slog.Error("msg", "error", err)`             |
| `logrus.Fatal(err)`                    | `slog.Error("msg", "error", err); os.Exit(1)` |

### 3. Replace logutil calls

| Before                                           | After                                         |
| ------------------------------------------------ | --------------------------------------------- |
| `logutil.Get(ctx).Infof("msg %s", v)`            | `logutil.Get(ctx).Info("msg", "key", v)`      |
| `logutil.Get(ctx).WithField("k", v).Info("msg")` | `logutil.Get(ctx).Info("msg", "k", v)`        |
| `logutil.Get(ctx).WithError(err).Error("msg")`   | `logutil.Get(ctx).Error("msg", "error", err)` |

### 4. Update WithVersionLog

```go
// Before
cmdutil.WithVersionLog(logrus.DebugLevel)

// After
cmdutil.WithVersionLog(slog.LevelDebug)
```

### 5. Update main.go pattern

```go
// Before
func main() {
    defer cmdutil.HandleExit()
    if err := cmd.NewRootCommand().Execute(); err != nil {
        logrus.Fatal(err)
    }
}

// After
func main() {
    defer cmdutil.HandleExit()
    if err := cmd.NewRootCommand().Execute(); err != nil {
        slog.Error("command failed", "error", err)
        os.Exit(1)
    }
}
```

### 6. Remove logrus dependency

After migrating all code, remove `github.com/sirupsen/logrus` from `go.mod` and run `go mod tidy`.

## Notes

* The `logutil.Start(ctx, "subsystem")` / `logutil.Get(ctx)` pattern remains the same.
* The `--gelf-address` and `--verbose` CLI flags remain the same.
* slog uses structured key-value pairs instead of format strings. Prefer `slog.Info("msg", "key", value)` over
  `fmt.Sprintf`.
