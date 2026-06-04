# Repository Guidelines

## Project Overview

`nex-gen-cms` is a Go 1.25 server-rendered CMS for Avanti Fellows curriculum and admin workflows. It uses:

- Go `net/http` handlers under `cmd/`, `internal/`, `config/`, `di/`, and `utils/`.
- HTML templates in `web/html/`.
- Static JS in `web/static/js/`.
- Tailwind v4 generated from `input.css` into `web/static/css/output.css`.
- Playwright tests in `tests/`.
- Terraform deployment code in `terraform/`.

Read `README.md` for local setup and `CONTEXT.md` for current domain language, especially Curriculum Config Management and LMS Chapter Exam Config.

## Local Setup

The app expects a `.env` at the repository root. Use `.env.example` and `README.md` as references. Important local dependencies include:

- Go `>= 1.25`.
- Node dependencies from `npm install` or `npm ci`.
- db-service running locally or reachable remotely.
- Postgres access through `DATABASE_URL`.
- Google OAuth values, or `DEV_LOGIN_EMAIL` for local dev login.

Run the app with:

```bash
make run
```

This builds CSS first, then runs:

```bash
go run ./cmd
```

While editing templates or `input.css`, run this in a separate terminal:

```bash
make css-watch
```

## Build And Test Commands

Use focused commands when possible:

```bash
go test ./...
npm run build:css
npx playwright test
```

For Playwright, the server should be running at `http://localhost:8080`, CSS should be built, and `playwright/.auth/user.json` may need to be generated through:

```bash
npx playwright test tests/auth.setup.spec.ts
```

CI builds CSS, builds the Go server, starts it, waits for `/login`, generates Playwright auth state, then runs Playwright excluding `@auth-setup`.

## Generated And Ignored Files

Do not commit generated or local-only artifacts:

- `web/static/css/output.css`
- `node_modules/`
- `.env*`
- `test-results/`
- `playwright-report/`
- `blob-report/`
- `playwright/.auth/`
- Terraform state and local tfvars

`web/static/css/output.css` is intentionally generated from `input.css`.

## Code Organization

- `cmd/main.go` owns server startup and route registration.
- `di/app_component.go` wires handlers, repositories, services, auth, and static serving.
- `internal/handlers/` contains HTTP handlers.
- `internal/repositories/remote/` calls db-service APIs for CMS content.
- `internal/repositories/db/` performs direct Postgres access for CMS-owned data.
- `internal/views/` renders Go templates.
- `web/html/` contains server-rendered templates and HTMX fragments.
- `web/static/js/` contains browser behavior.

Prefer existing handler, repository, service, DTO, and template patterns before adding new abstractions.

## Go Conventions

- Run `gofmt` on edited Go files.
- Keep route registration readable in `cmd/main.go`; use existing role wrappers for editor/admin protected routes.
- Use `middleware.RequireHTMX` for endpoints intended only for HTMX partial requests.
- Keep direct database access limited to repository code.
- Keep db-service API calls in remote repository code.
- Preserve existing error semantics in handlers when extending a workflow.
- Add focused tests near the behavior being changed.

## UI And Template Conventions

Before adding or changing screens, read `docs/UI-Style-Guide.md`.

Important UI rules:

- Use the Warm Professional design language.
- Use Tailwind token classes from `input.css` such as `bg-accent`, `text-ink`, `bg-bg-card`, and `border-border`.
- Avoid hardcoded hex colors in templates.
- Use Font Awesome 6 icons where the current templates use icons.
- Keep server-rendered Go templates and HTMX partials consistent with existing files in `web/html/`.
- Rebuild CSS after changing templates or `input.css`.

## Domain Language

Use the terminology in `CONTEXT.md`.

Preferred terms:

- Curriculum Config Management
- LMS Chapter Exam Config
- Exam Track
- CMS Admin

Avoid overloaded alternatives such as "syllabus config", "chapter setup", "stream", or "superuser" unless quoting existing text.

For Curriculum Config Management:

- It is global and belongs to CMS Admins only.
- It writes live LMS Chapter Exam Config rows directly through Postgres.
- It does not mutate LMS Curriculum Logs or Chapter Completion records.
- Duplicate coverage order values should warn, not block.
- Removing an in-syllabus row is a dedicated confirmed action.
- CSV export uses active filters and ignores pagination.

## Working Notes

- The working tree may contain user changes. Do not revert unrelated changes.
- Prefer `rg` and `rg --files` for searching.
- Keep changes scoped to the requested workflow.
- If frontend behavior changes, verify both server-rendered output and browser behavior where practical.
