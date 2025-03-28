# Full Example Application

This is a comprehensive example application that demonstrates various features of the rebuy-go-sdk.

## Features

- Command structure with cmdutil
- HTTP handlers with webutil
- Worker management with runutil
- Development vs. production environments
- Template rendering using templ
- Web assets management

## Getting Started

To start the development server:

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
  - `/web` - Web assets (CSS, JavaScript, etc.)

## Templates

This application uses [templ](https://github.com/a-h/templ) for HTML templating. Templ is a type-safe templating language for Go that generates Go code from template files.

To generate template code after making changes:

```bash
templ generate pkg/app/templates
```

## License

MIT
