---
name: apply-sdk-migrations
description: Walks a rebuy-go-sdk project through pending SDK update migrations (major version bumps and minor adoptions). Use when the user asks to migrate, upgrade rebuy-go-sdk, apply SDK migrations, or update to a new SDK major version.
allowed-tools: Read(${CLAUDE_SKILL_DIR}/../*)
---

# Apply rebuy-go-sdk Migrations

This skill drives the SDK update migrations defined in the sibling
`rebuy-go-sdk` skill. The migration notes are versioned and split into:

- **Major** migrations — required when bumping the SDK to a new major version.
- **Minor** migrations — optional adoptions of new patterns; should be done
  before the next major bump.

## Files

All migration content is preloaded via `@`-references below, so this skill
never needs to read files from outside the target project.

Index:
- @../rebuy-go-sdk/migrations/index.md

SDK package context referenced by some migrations:
- @../rebuy-go-sdk/docs.md

Individual migrations (v8 minor):
- ../rebuy-go-sdk/migrations/v8/M0001-remove-pkg-errors.md
- ../rebuy-go-sdk/migrations/v8/M0002-cdnmirror-to-yarn.md
- ../rebuy-go-sdk/migrations/v8/M0003-webutil-viewer-interfaces.md
- ../rebuy-go-sdk/migrations/v8/M0004-isolate-http-handlers.md
- ../rebuy-go-sdk/migrations/v8/M0005-streamline-templates-assets.md
- ../rebuy-go-sdk/migrations/v8/M0006-use-webutil-newserver.md
- ../rebuy-go-sdk/migrations/v8/M0007-dependency-injection.md

Individual migrations (v9 major):
- @../rebuy-go-sdk/migrations/v9/M0008-update-to-v9.md

Individual migrations (v10):
- @../rebuy-go-sdk/migrations/v10/M0009-logrus-to-slog.md
- @../rebuy-go-sdk/migrations/v10/M0010-riverutil-periodic-jobs.md

The target project's `go.mod` is the source of truth for the current SDK
major version. The target project's `./buildutil` script is used to verify
each step.

## Workflow

1. **Detect current SDK version.** Read the target project's `go.mod` and
   grep for `github.com/rebuy-de/rebuy-go-sdk/vN`. Record `N` as the current
   major.
2. **Decide scope** with the user:
   - *Major bump* — current major → target major. The plan includes every
     `Major:` migration of every major from `current+1` through `target`,
     plus all unadopted `Minor:` migrations of the current major (they must
     be done before the bump per the index notes).
   - *Minor sweep* — only the unadopted `Minor:` migrations of the current
     major.
3. **Build and present the ordered migration plan.** Read
   `migrations/index.md` to get the canonical order. Show the user the list
   of migration IDs and titles in the order they will run.
4. **Wait for confirmation** before applying anything.
5. **For a major bump only**: run the `sed` command from `index.md` to
   rewrite the import paths from `vOLD` to `vNEW`:
   ```
   find . -type f -exec sed -i 's#github.com/rebuy-de/rebuy-go-sdk/vOLD#github.com/rebuy-de/rebuy-go-sdk/vNEW#g' {} +
   ```
   Then run `go mod tidy` and commit the import rewrite as its own commit.
6. **For each migration in the plan**:
   1. Use the already-loaded content of the migration's `M####-*.md` file
      (preloaded via the `@`-references above) to apply the changes to the
      target project.
   2. Run `./buildutil` to verify the project still builds and tests pass.
   3. Run `goimports` on edited Go files (skip generated files).
   4. Commit the migration as its own commit, with a message referencing the
      migration ID (e.g. `M0009: switch from logrus to slog`).
7. **Stop on failure.** If `./buildutil` fails, fix the issue or surface it
   to the user — never skip past a broken build to the next migration.

## Rules

- One commit per migration so each step is reviewable in isolation.
- Major migrations of the target version are mandatory. Minor migrations are
  optional, but offer them — the index requires they be done before the next
  major bump anyway.
- Never edit generated files or run `goimports` on them.
- Follow the project's global Go preferences (separate `err :=` line, `any`
  over `interface{}`, etc.) when applying migration changes.
