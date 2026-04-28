---
name: apply-sdk-migrations
description: Walks a rebuy-go-sdk project through pending SDK update migrations (major version bumps and minor adoptions). Use when the user asks to migrate, upgrade rebuy-go-sdk, apply SDK migrations, or update to a new SDK major version.
allowed-tools: Read(${CLAUDE_SKILL_DIR}/**) Skill(rebuy-go-sdk:rebuy-go-sdk)
---

# Apply rebuy-go-sdk Migrations

This skill drives the SDK update migrations bundled alongside it. The migration
notes are versioned and split into:

- **Major** migrations — required when bumping the SDK to a new major version.
- **Minor** migrations — optional adoptions of new patterns; should be done
  before the next major bump.

SDK package context referenced by some migrations:

- Skill /rebuy-go-sdk:rebuy-go-sdk

The target project's `go.mod` is the source of truth for the current SDK major
version. The target project's `./buildutil` script is used to verify each step.

## Migrations

Migrations are grouped by the major SDK version they belong to. Within each
version, **major** migrations are required when bumping the SDK to that
version; **minor** migrations are new features or recommended patterns that
can be adopted incrementally.

### v10

**Major:**

- [M0009 — Switch from logrus to slog](migrations/v10/M0009-logrus-to-slog.md) · 2025-02-19

**Minor:**

- [M0010 — Adopt riverutil for periodic and background jobs](migrations/v10/M0010-riverutil-periodic-jobs.md) · 2026-04-27

### v9

**Major:**

- [M0008 — Update to v9](migrations/v9/M0008-update-to-v9.md) · 2024-04-22

**Minor:**

- _none_

### v8

**Major:**

- _none documented_

**Minor:**

- [M0007 — Switch to Dependency Injection](migrations/v8/M0007-dependency-injection.md) · 2024-11-08
- [M0006 — Use webutil.NewServer](migrations/v8/M0006-use-webutil-newserver.md) · 2024-09-20
- [M0005 — Streamline templates and assets](migrations/v8/M0005-streamline-templates-assets.md) · 2024-09-20
- [M0004 — Isolate HTTP handlers](migrations/v8/M0004-isolate-http-handlers.md) · 2024-09-13
- [M0003 — Change viewer interfaces of webutil](migrations/v8/M0003-webutil-viewer-interfaces.md) · 2024-08-16
- [M0002 — Replace cdnmirror with yarn](migrations/v8/M0002-cdnmirror-to-yarn.md) · 2024-07-19
- [M0001 — Remove uses of github.com/pkg/errors](migrations/v8/M0001-remove-pkg-errors.md) · 2024-06-14

## Major vs Minor

- Major migrations need to be done during a major upgrade of the SDK.
- Minor migrations should be done as soon as possible, but may be delayed until
  the next major upgrade. They must be done before the next major upgrade.

## Major version sed rewrite

When bumping to a new major version, rewrite the import paths first:

```
find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vOLD#github.com/rebuy-de/rebuy-go-sdk/vNEW#g' {} +
```

## Workflow

1. **Detect current SDK version.** Read the target project's `go.mod` and
   grep for `github.com/rebuy-de/rebuy-go-sdk/vN`. Record `N` as the current
   major.
2. **Decide scope** with the user:
   - *Major bump* — current major → target major. The plan includes every
     `Major:` migration of every major from `current+1` through `target`,
     plus all unadopted `Minor:` migrations of the current major (they must
     be done before the bump per the *Major vs Minor* note).
   - *Minor sweep* — only the unadopted `Minor:` migrations of the current
     major.
3. **Build and present the ordered migration plan.** Use the catalog above to
   get the canonical order. Show the user the list of migration IDs and titles
   in the order they will run.
4. **Wait for confirmation** before applying anything.
5. **For a major bump only**: run the `sed` command from the *Major version
   sed rewrite* section above to rewrite the import paths from `vOLD` to
   `vNEW`. Then run `go mod tidy` and commit the import rewrite as its own
   commit.
6. **For each migration in the plan**:
   1. Read the migration's `migrations/vN/M####-*.md` file from the skill
      bundle and apply the changes to the target project.
   2. Run `./buildutil` to verify the project still builds and tests pass.
   3. Run `goimports` on edited Go files (skip generated files).
   4. Commit the migration as its own commit, with a message referencing the
      migration ID (e.g. `M0009: switch from logrus to slog`).
7. **Stop on failure.** If `./buildutil` fails, fix the issue or surface it
   to the user — never skip past a broken build to the next migration.

## Rules

- One commit per migration so each step is reviewable in isolation.
- Major migrations of the target version are mandatory. Minor migrations are
  optional, but offer them — they must be done before the next major bump
  anyway.
- Never edit generated files or run `goimports` on them.
- Follow the project's global Go preferences (separate `err :=` line, `any`
  over `interface{}`, etc.) when applying migration changes.
