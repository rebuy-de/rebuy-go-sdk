---
id: M0011
title: Use fully-qualified table names
date: 2026-08-26
sdk_version: v10
type: minor
---

# Use fully-qualified table names

## Reasoning

`pgutil.NewPool` pins `search_path` to the application schema on every connection, so unqualified table names resolve and everything appears to work. The cost is that the query text alone no longer says which table it reads. The same SQL means a different table depending on the connection that runs it, it cannot be pasted into `psql`, a Grafana panel or a migration without silently hitting a different schema, and it cannot join across schemas at all.

Naming the schema in the query removes that ambiguity. `search_path` stays — golang-migrate needs it for its own `schema_migrations` table — but nothing else may depend on it.

The SDK now does the same for River. `riverutil.NewRiverClient` sets `river.Config.Schema` and `riverutil.Migrate` sets `rivermigrate.Config.Schema`, both from the injected `pgutil.Schema`, and the `rebuy_go_sdk_river_failing_jobs` collector queries `<schema>.river_job` instead of `river_job`. River's tables were always created inside the app schema; the SDK now says so explicitly rather than inferring it from `search_path`.

## Hints

- Nothing fails loudly. An unqualified query keeps working because `search_path` is still set, so the sweep has to be done by reading the files, not by waiting for an error.
- SQLC builds its catalog from the migration files. A query may only name a schema that the migrations actually create, which is why the two halves have to move together.
- Generated model names change when a previously unqualified table becomes qualified: `User` becomes `MyAppUser`. Add `rename:` entries to `sqlc.yaml` to keep the old Go names and hold the diff to the `.sql` files.
- **Never edit an existing versioned migration.** Repeatable `R__*.sql` migrations are a different case — they are `create or replace` and re-applied by content hash, so qualifying them in place is safe.
- Index names stay unqualified. Postgres puts an index in the schema of its table.
- River tables (`river_job`, `river_leader`, `river_migration`, ...) live in the app schema, not a separate one. The `docs.md` claim that River uses its own schema was wrong and has been corrected.
- Projects that call `riverutil.NewDatabaseCollector` directly must pass the schema as a second argument. `NewRiverClient` and `Migrate` also gained a `pgutil.Schema` parameter, but both are resolved through dig, so `c.Provide(riverutil.NewRiverClient)` and `c.Invoke(riverutil.Migrate)` need no change.

## Steps

### 1. Check what the existing migrations look like

Read `pkg/dal/sqlc/migrations/*.up.sql`. Two cases, and they need different work:

- **Objects are already schema-qualified** (`create table my_app.users (...)`, preceded by `create schema if not exists my_app;`). SQLC's catalog already holds `my_app.users`, which means the queries must already be qualified too — otherwise `sqlc generate` would not have resolved them. Verify and go to step 3.
- **Objects are unqualified** (`create table users (...)`). SQLC's catalog holds them under `public`, so qualifying a query would break `sqlc generate`. Do step 2 first.

### 2. Move the tables into the schema for SQLC's catalog

Add a **new** versioned migration — do not touch the old ones — with one statement per object:

```sql
-- 0007_qualify_schema.up.sql
create schema if not exists my_app;

alter table if exists public.users set schema my_app;
alter table if exists public.posts set schema my_app;
alter view  if exists public.user_posts set schema my_app;
```

`if exists` is what makes this safe. Against a real database the tables already sit in `my_app` (golang-migrate created them there through `search_path`), so `public.users` does not exist and the statement is a no-op. SQLC applies the move to its static catalog regardless, which is the point. In the rare environment where the table really is in `public`, the statement fixes it.

### 3. Qualify the queries

Prefix every table, view and function reference in `pkg/dal/sqlc/query_*.sql`, then regenerate:

```
CGO_ENABLED=0 go generate ./pkg/dal/sqlc
```

Check the diff in `gen_models.go`. If model names gained a schema prefix, either accept the new names and fix the call sites, or add `rename:` entries to keep the old ones.

### 4. Qualify the repeatable migrations

Prefix the objects in `pkg/dal/sqlc/migrations/R__*.sql` in place. They are `create or replace` statements re-applied by content hash, so the changed hash simply replaces the same object.

### 5. Qualify hand-written SQL in Go

Grep for SQL that does not go through SQLC — `pool.Query`, `pool.QueryRow`, `conn.Exec` after `pgutil.Hijack`, Prometheus collectors. Build the name from the injected `pgutil.Schema` rather than hardcoding it:

```go
query := fmt.Sprintf(
	`SELECT count(*) FROM %s.orders WHERE state = 'pending';`,
	pgx.Identifier{string(schema)}.Sanitize(),
)
```

### 6. Bump the SDK

Take the SDK version that carries this change so River stops relying on `search_path`. No call-site change is needed for `NewRiverClient` or `Migrate`.

## Examples

Before — the query depends on `search_path`:

```sql
-- name: ListUsers :many
select * from users
order by created_at desc;
```

After:

```sql
-- name: ListUsers :many
select * from my_app.users
order by created_at desc;
```

Keeping the generated Go names unchanged, in `sqlc.yaml`:

```yaml
rename:
  my_app_user: User
  my_app_post: Post
```
