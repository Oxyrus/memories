# Memories

Server-rendered photo album manager built with Go, Gin, and [templ](https://templ.guide/). It lets you curate private albums, upload photos, and share a minimalist public gallery per album.

## Features

- **SQLite-backed storage** – albums and photos are stored in a single SQLite database (`data/memories.db`). Tables are created on demand by the storage layer; no external migrations are required yet.
- **Photo uploads with sanitisation** – photos are uploaded to `public/uploads/<album-slug>/`. JPEG uploads are re-encoded on the server with EXIF data (including GPS coordinates) stripped and orientation applied so files are safe to share.
- **Admin workflow** – authenticated admins can list, create, edit, and upload photos for albums under `/albums`. Logins set a 14-day admin cookie.
- **Public sharing** – every album is viewable at `/a/{slug}` with a full-bleed hero image, thumbnail carousel, and fullscreen viewer.
- **templ-powered UI** – layout and pages are authored with templ components (`web/components` and `web/pages`), keeping markup and styling alongside Go logic.

## Prerequisites

- Go 1.25.3 (matches `go.mod`).
- templ CLI `v0.3.960` or later: `go install github.com/a-h/templ/cmd/templ@v0.3.960`.

## Configuration

Environment variables (via `.env` or shell) control runtime behaviour:

| Variable | Purpose | Default |
| --- | --- | --- |
| `ADMIN_PASSWORD` | Password required to log in | _required_ |
| `MEMORIES_ADDR` | Listen address | `:8080` |
| `MEMORIES_DB_PATH` | SQLite database path | `data/memories.db` |
| `MEMORIES_UPLOADS_PATH` | Directory for uploaded photos | `public/uploads` |
| `MEMORIES_LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `MEMORIES_ADMIN_COOKIE` | Cookie name for admin auth | `memories_admin` |

Ensure the uploads directory exists and is writable by the process (`make run` will create it as needed).

## Development Workflow

### Makefile Targets

- `make build` — regenerates templ views and builds the binary to `bin/memories`.
- `make run` — builds then launches the server (`./bin/memories`).
- `make test`, `make race`, `make cover` — run the test suite with optional race detector or coverage report.
- `make fmt`, `make vet`, `make tidy` — format, vet, and tidy dependencies.
- `make generate` — shortcut for `go tool templ generate`.

### Working With templ

1. Edit `.templ` files in `web/components` or `web/pages`.
2. Run `go tool templ generate` (or `make generate`) from the repo root.
3. `gofmt` the generated `*_templ.go` files if your editor does not do it automatically.

`components.MainLayout` defines the shared typography and monochrome styling; all pages render inside it for a consistent look.

### Running Locally

```bash
export ADMIN_PASSWORD=change-me
make run
```

Visit `http://localhost:8080/login`, authenticate with the password above, and start managing albums from `/albums`. Public viewers are available at `/a/<slug>`.

## Project Structure

- `cmd/memories/` — main binary entry point.
- `internal/config` — environment-driven config loader.
- `internal/http/handlers` — Gin handlers for albums, auth, uploads, and the public viewer.
- `internal/storage` — SQLite implementations for albums and photos (auto-creates tables).
- `web/components`, `web/pages` — templ components plus generated Go.
- `public/uploads` — uploaded photo assets served directly.
- `data/` — default location for the SQLite database file.

Reusable packages belong in `pkg/`, shared assets in `assets/`, and fixtures in `testdata/` near their consumers.

## Recommended Checks

- `make fmt`
- `make vet`
- `make test` (or `make race`)
- `go test -cover ./...`

## Contributing Notes

- Document exported Go identifiers with GoDoc comments.
- Prefer table-driven tests (`Test<Thing><Behavior>`).
- Regenerate templ output and include it in commits whenever templates change.
- Highlight schema changes, seed data updates, or new assets in PR descriptions so deployers can prepare.
