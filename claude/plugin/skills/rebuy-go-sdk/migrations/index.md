# SDK Migrations Index

Migrations are grouped by the major SDK version they belong to. Within each
version, **major** migrations are required when bumping the SDK to that
version; **minor** migrations are new features or recommended patterns that
can be adopted incrementally.

## Notes

### Use sed for migrating to a major version.

```
find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vOLD#github.com/rebuy-de/rebuy-go-sdk/vNEW#g' {} +`
```

### Major vs Minor

- Major migrations need to be done during a major upgrade of the SDK.
- Minor migrations should be done as soon as possible, but may be delayed until the next major upgrade. The must be done
  before the next major upgrade.

## Versions

## v10

**Major:**

- [M0009 — Switch from logrus to slog](v10/M0009-logrus-to-slog.md) · 2025-02-19

**Minor:**

- [M0010 — Adopt riverutil for periodic and background jobs](v10/M0010-riverutil-periodic-jobs.md) · 2026-04-27

## v9

**Major:**

- [M0008 — Update to v9](v9/M0008-update-to-v9.md) · 2024-04-22

**Minor:**

- _none_

## v8

**Major:**

- _none documented_

**Minor:**

- [M0007 — Switch to Dependency Injection](v8/M0007-dependency-injection.md) · 2024-11-08
- [M0006 — Use webutil.NewServer](v8/M0006-use-webutil-newserver.md) · 2024-09-20
- [M0005 — Streamline templates and assets](v8/M0005-streamline-templates-assets.md) · 2024-09-20
- [M0004 — Isolate HTTP handlers](v8/M0004-isolate-http-handlers.md) · 2024-09-13
- [M0003 — Change viewer interfaces of webutil](v8/M0003-webutil-viewer-interfaces.md) · 2024-08-16
- [M0002 — Replace cdnmirror with yarn](v8/M0002-cdnmirror-to-yarn.md) · 2024-07-19
- [M0001 — Remove uses of github.com/pkg/errors](v8/M0001-remove-pkg-errors.md) · 2024-06-14
