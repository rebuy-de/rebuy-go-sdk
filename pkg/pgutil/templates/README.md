# SQLC Configuration Templates

This directory contains standard SQLC configuration templates for rebuy projects.

## Usage

Copy the `sqlc.yaml` template to your project's `pkg/dal/sqlc/` directory and customize as needed:

```bash
cp pkg/pgutil/templates/sqlc.yaml pkg/dal/sqlc/sqlc.yaml
```

## Standard Configuration

The template provides:

- PostgreSQL engine with pgx/v5 driver
- JSON tag generation with camelCase style
- Database tag generation for struct fields
- Proper UUID and timestamp type handling
- Null-safe type generation

## Customization

For projects needing custom field naming or additional type overrides, you can extend the configuration:

```yaml
# Add to sqlc.yaml for custom field naming
rename:
  commit_sha: "CommitSHA"
  event_uid: "EventUID"
  
# Add custom column overrides
overrides:
  - column: "my_table.array_field"
    go_type:
      type: "[]string"
```
