# AGENTS.md

Guidance for agentic coding tools working in this repository.

## Purpose

Filestash is a storage-agnostic file management platform with a modular Go backend and a vanilla JS frontend. The repo is plugin-driven and uses a hook system for extension.

## Project Structure

- `/cmd/main.go` - Application entry point.
- `/server/` - Backend Go code.
  - `/common/` - Core types, hooks, error helpers, logging.
  - `/ctrl/` - HTTP controllers.
  - `/plugin/` - Plugins, naming pattern `plg_{category}_{name}/`.
  - `/middleware/` - HTTP middleware.
  - `/model/` - Models and plugin management.
  - `/pkg/workflow/` - Workflow engine.
- `/public/` - Frontend vanilla JS (checked by TypeScript).
  - `/assets/` - Pages, components, CSS, vendor libs.
  - `/package.json` - Frontend scripts and ESLint rules.
- `/config/` - JSON configs.
- `/docker/` - Docker compose setup.

## Build, Lint, Test

Backend (Go, module at repo root):

- Initialize deps and generate code:
  - `make init`
  - Runs `go get ./...` and `go generate -x ./server/...`
- Build binary with FTS5 and CGO:
  - `make build`
  - Runs `CGO_ENABLED=1 go build --tags "fts5" -o dist/filestash cmd/main.go`
- Build all:
  - `make`

Frontend (from `public/`):

- Type check JS with TS:
  - `npm run check`
- Lint:
  - `npm run lint`
- Tests:
  - `npm run test`
- Coverage:
  - `npm run coverage`
- Compress assets:
  - `make compress`
- Clean compressed assets:
  - `make clean`

Docker:

- Start stack with WOPI server:
  - `cd docker && docker compose down && docker compose up -d`
- Logs:
  - `cd docker && docker compose logs -f app`

## Run a Single Test

Backend (Go):

- Run a single test by name:
  - `go test ./... -run TestName`
- Run a single package:
  - `go test ./server/common -run TestName`
- If code relies on FTS5/CGO, add tags:
  - `go test -tags "fts5" ./... -run TestName`

Frontend (Vitest):

- Run a single file:
  - `cd public && npx vitest path/to/file.test.js`
- Run by test name:
  - `cd public && npx vitest -t "test name"`
- Run in watch mode:
  - `cd public && npx vitest --watch`

## Code Style Guidelines

### Go (backend)

Formatting:

- Use `gofmt` (tabs for indentation, standard Go formatting).
- Keep files consistent with existing import grouping (stdlib, third-party, local).

Imports:

- Let `gofmt` order imports; avoid manual ordering.
- Avoid unused imports; build will fail.

Types and naming:

- Exported types and functions use `PascalCase`.
- Unexported identifiers use `camelCase`.
- Interfaces are named by behavior (example: `IBackend`, `IAuthentication`).
- Plugin directories follow `plg_{category}_{name}`.

Error handling:

- Use `common.NewError` / `common.AppError` for HTTP-facing errors.
- Reuse common errors from `server/common/error.go` (ex: `ErrNotFound`).
- Use `common.HTTPError(err)` to map raw errors to HTTP-friendly ones when needed.
- Send API errors via `common.SendErrorResult` where applicable.

Logging:

- Use `common.Log` (Info, Warning, Error, Debug) rather than fmt.Print.
- Avoid logging secrets or credentials.

Concurrency and cleanup:

- Follow existing patterns in `server/common` for context usage and cleanup.
- Use `sync.WaitGroup` and context cancellation where long-lived goroutines are involved.

### JavaScript (frontend)

Formatting and linting:

- ESLint config is in `public/package.json` (`eslintConfig`).
- Indent with 4 spaces.
- Use double quotes; template literals are allowed.
- Always include semicolons.
- No space before function parens (`space-before-function-paren: never`).

Modules and syntax:

- Code uses ESM (`type: module` in `public/package.json`).
- Prefer named exports and explicit imports.
- Keep imports grouped logically; avoid unused imports.

Types and checks:

- TypeScript checks JS via `tsconfig.json` with `checkJs: true` and `strict: true`.
- `noImplicitAny` is false, but most other strict checks are enabled.
- Avoid unused locals/params (TS checks will error).

Tests:

- Vitest with jsdom and setup file `public/vite.setup.js`.
- Do not put test files under `assets/lib/vendor` (lint ignores vendor paths).

### JSON and configs

- Preserve existing formatting; JSON is commonly pretty-printed.
- Keep config keys consistent with existing files in `config/`.

## Plugin Patterns

- Plugins self-register in `init()` using hooks or registries, ex:
  - `Backend.Register("ftp", Ftp{})`
- Plugin hooks live in `server/common/plugin.go`.
- Prefer context-aware operations via `*common.App` where provided.
- Cache connections via `AppCache` where appropriate for backends.

## Frontend/Backend Integration

- API responses are JSON with 4-space indentation (`common.IndentSize`).
- Use `common.SendSuccessResult` / `SendSuccessResults` for responses.
- Keep response shapes consistent (`status: "ok"` or `status: "error"`).

## Cursor/Copilot Rules

- No `.cursor/rules/`, `.cursorrules`, or `.github/copilot-instructions.md` found in this repo.

## Notes and Gotchas

- Go version: `go 1.24.11` in `go.mod`.
- Build uses CGO and image libraries (libjpeg/libpng/libvips/etc).
- FTS5 tag is required for SQLite full-text search in builds.
- Plugins are loaded via blank imports in `server/plugin/index.go`.
- Do not edit vendored files under `public/assets/lib/vendor`.
