# Full Example Application

This is a comprehensive example application that demonstrates various features of the rebuy-go-sdk.

## Features

- Command structure with cmdutil
- HTTP handlers with webutil
- Worker management with runutil
- Database integration with pgutil and SQLC
- Development vs. production environments
- Template rendering using templ
- Web assets management

## Getting Started

### Prerequisites

- Go 1.24 or later
- PostgreSQL database
- Docker (optional, for running PostgreSQL locally)

### Database Setup

1. Start a PostgreSQL database:

```bash
# Using Docker
docker run -d --name postgres-full-example \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=full_example \
  -p 5432:5432 \
  postgres:16

# Or use your existing PostgreSQL instance and create the database:
# createdb full_example
```

2. The application will automatically run migrations on startup.

### Running the Application

```bash
./buildutil && ./dist/full dev
```

## Project Structure

- `/cmd` - Command-line commands
- `/pkg` - Package code
  - `/app` - Application code
    - `/handlers` - HTTP handlers
    - `/templates` - Templ templates
    - `/workers` - Background workers
  - `/dal` - Data Access Layer
    - `/sqlc` - Database queries and migrations
  - `/web` - Web assets (CSS, JavaScript, etc.)

## Database

This application demonstrates the rebuy-go-sdk's pgutil package for database operations:

- **Migrations**: Automatically managed using embedded SQL files in `pkg/dal/sqlc/migrations/`
- **Queries**: Type-safe database operations generated with [SQLC](https://sqlc.dev/)
- **Connection Management**: Handled by pgutil with optional DataDog tracing
- **Transactions**: Available through pgutil's transaction wrapper utilities

### Database Operations

Generate SQLC code after modifying queries:

```bash
CGO_ENABLED=0 go generate ./pkg/dal/sqlc
```

The Users page demonstrates real database CRUD operations, replacing the previous hardcoded data.

## Templates

This application uses [templ](https://github.com/a-h/templ) for HTML templating. Templ is a type-safe templating language for Go that generates Go code from template files.

To generate template code after making changes:

```bash
templ generate pkg/app/templates
```

## License

MIT
