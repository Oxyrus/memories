# Memories

Server-rendered photo album manager built with Go, Gin, and [templ](https://templ.guide/). The project is organised as a Go module rooted at the repository root (`github.com/Oxyrus/memories`) and follows the usual Go convention of placing binaries in `cmd/`, internal-only packages in `internal/`, and shared components under `pkg/`.

## Prerequisites

- Go 1.25.3 (matches the version pinned in `go.mod`).
- The templ CLI (`v0.3.960` or later) installed locally. Use `go install github.com/a-h/templ/cmd/templ@v0.3.960` and ensure `$GOPATH/bin` (or `$HOME/go/bin`) is on your `PATH`.

## Development Workflow

### Makefile Targets

A repo-level `Makefile` provides the common build and test commands:

- `make build` — regenerates templ components, verifies compilation with `go build ./...`, and drops the `memories` binary in `bin/`.
- `make run` — rebuilds via `make build` and executes `./bin/memories`.
- `make test` / `make race` / `make cover` — unit tests with optional race detector or coverage.
- `make fmt` — format all Go sources with `gofmt`.
- `make vet` — static analysis via `go vet`.
- `make tidy` — keep `go.mod` / `go.sum` tidy.

### Regenerating templ Components

UI views live in `web/` as `.templ` files and are compiled to Go via the templ CLI:

1. Edit `.templ` files (for example `web/pages/login.templ` or `web/components/layout.templ`).
2. Run `templ generate` from the repo root (or `$HOME/go/bin/templ generate` if `templ` is not on your `PATH`).
3. Run `gofmt` on the generated `*_templ.go` files if your editor does not do this automatically (`gofmt -w web/**/*.go`).

The main layout component lives in `web/components/layout.templ` and exposes `components.MainLayout`, which wraps page content and injects shared markup such as the document `<head>`. Pages compose it using templ's `@components.MainLayout("Title") { ... }` syntax.

### Authentication Flow

Admin authentication posts to `/login` and, on success, redirects to `/albums`. The handler sets an admin cookie valid for 14 days. The `/albums` route is currently a stub: it returns `501 Not Implemented` until the listing view is completed.

## Project Structure

- `cmd/memories/` — main binary entry point.
- `internal/` — application code; HTTP handlers live under `internal/http/handlers`, middleware under `internal/http/middleware`, etc.
- `web/` — templ templates and generated components.
- `public/` — static assets served directly (currently empty).
- `data/` — placeholder for persistent storage or fixtures.

Put reusable libraries in `pkg/`, seed data or migrations in `assets/`, and test fixtures in `testdata/` alongside the package that consumes them.

## Recommended Checks

- `make fmt` to ensure Go sources are formatted.
- `make vet` and `make test` (or `make race`) before committing.
- `go test -cover ./...` to validate coverage goals for new packages.

## Contributing Notes

- Document exported Go identifiers with GoDoc comments.
- Prefer table-driven tests with descriptive `Test<Name><Behavior>` names.
- Add new templ components alongside their generated outputs and keep them in sync via `templ generate`.
- Capture any database migrations, seed files, or static assets under `assets/` and document their purpose with a short README in that subdirectory.
