# Repository Guidelines

## Project Structure & Module Organization
- `go.mod` pins the module `github.com/Oxyrus/memories` for Go 1.25.3; treat the repository root as the module root.
- Add runnable entry points under `cmd/memories/` (or additional `cmd/*` directories per binary) and keep domain packages private under `internal/`.
- Reusable packages that may be shared across binaries belong in `pkg/`. Fixtures should live in `testdata/` beside the package that consumes them.
- Store static assets (templates, seed data, migrations) in `assets/` and document their purpose with a short README to keep new contributors oriented.

## Build, Test, and Development Commands
- `make build` regenerates templ components, verifies module compilation, and writes the binary to `bin/memories`.
- `make run` builds and executes the compiled binary (`./bin/memories`).
- `make test`, `make race`, and `make cover` exercise the test suite with optional race detection and coverage.
- `make fmt` and `make vet` run `gofmt` and `go vet` across the module, while `make tidy` keeps module files clean.
- `go tool templ generate` (or `make generate`) regenerates Go code from `.templ` files after template changes.

### Runtime configuration
- `ADMIN_PASSWORD` must be provided before the server will start successfully.
- `MEMORIES_DB_PATH` defaults to `data/memories.db` and is created automatically by the SQLite storage layer.
- `MEMORIES_UPLOADS_PATH` defaults to `public/uploads`; uploads are written beneath `<slug>/`.
- The `config.Config` type documents the remaining env vars: `MEMORIES_ADDR`, `MEMORIES_LOG_LEVEL`, and `MEMORIES_ADMIN_COOKIE`.

## Coding Style & Naming Conventions
- Trust `gofmt`; do not hand-format files. The standard tool enforces tabs and canonical spacing.
- Package names stay short, all lowercase, and free of underscores. Exported identifiers use CamelCase; keep unexported ones scoped and descriptive.
- Group related files by feature and use explicit suffixes such as `_service.go` or `_handler.go` when they improve discoverability.
- Document every exported type or function with a GoDoc comment that begins with the identifier name so documentation tooling renders cleanly.
- Shared HTML layout and styling live in `web/components/layout.templ`; pages compose that component and are regenerated via `go tool templ generate`.

## Testing Guidelines
- Write table-driven tests and name them `Test<Thing><Behavior>` for clarity. Keep subtests focused with `t.Run`.
- Store mocks and fixtures in `testdata/` and guard slower or external-integration tests with `//go:build integration`.
- Run `go test -cover ./...` before submitting changes; target at least 80 % coverage for new packages and add regression tests for every bug fix. Use `make test`/`make cover` for convenience.

## Authentication & Routing
- Admin routes under `/albums`, `/albums/new`, `/albums/:slug/edit`, and `/albums/:slug` are protected via `middleware.RequireAdmin`, which checks the configured admin cookie and redirects to `/login?next=...` when missing.
- Public album viewers are available at `/a/:slug`. They render a hero image, thumbnail carousel, and a fullscreen lightbox managed by inline JS in `web/pages/albums_public.templ`.
- Successful logins set the admin cookie for 14 days and respect the `next` parameter before defaulting to `/albums`.

### Storage & uploads
- Storage lives in `internal/storage/sqlite`; `Albums` and `Photos` repositories auto-create tables when the first connection is opened. No migration tool is in place yet.
- Uploaded photos are saved under `public/uploads/<album-slug>/<generated-filename>` using `AlbumHandler.UploadPhoto`.
- After saving, `sanitizePhoto` re-encodes JPEGs to strip EXIF (including GPS) and applies orientation based on EXIF data. Non-JPEG uploads pass through untouched.
- `toAlbumPhoto` builds URLs with `/uploads/<slug>/<file>`; keep this helper in sync if upload paths change.

## Commit & Pull Request Guidelines
- Use imperative, present-tense subjects around 50 characters (for example, `Add photo upload metadata`). Follow with a brief body explaining rationale and validation (`go test ./...`, lint results, manual checks).
- Reference any tracking issues with keywords such as `Fixes #123` when relevant.
- Open pull requests only after tests pass, add screenshots or sample payloads for user-facing changes, and call out configuration updates or migrations in the description so reviewers and deployers can prepare.
