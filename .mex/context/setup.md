---
name: setup
description: Dev environment setup and commands. Load when setting up the project for the first time or when environment issues arise.
triggers:
  - "setup"
  - "install"
  - "environment"
  - "getting started"
  - "how do I run"
  - "local development"
edges:
  - target: context/stack.md
    condition: when specific technology versions or library details are needed
  - target: context/auth.md
    condition: when configuring Google OAuth, DEV_LOGIN_EMAIL, or sign-in fails
  - target: patterns/generate-pdf.md
    condition: when PDF generation fails locally (Chrome / chromedp)
  - target: context/deployment.md
    condition: when mapping local env/setup to how it runs on staging/prod
last_updated: 2026-06-26
---

# Setup

## Prerequisites

- **Go >= 1.25** (`go version`).
- **Node.js 22** — only for the Tailwind CSS build and Playwright (Tailwind v4 needs Node 22).
- **A running db-service instance** — the content API (`DB_SERVICE_ENDPOINT`). See db-service's INSTALLATION docs.
- **PostgreSQL access** to the DB hosting `cms_user_permission` (the same DB as db-service; local or staging RDS).
- **Google OAuth client** (authorized redirect `http://localhost:8080/auth/google/callback`) — or use the
  `DEV_LOGIN_EMAIL` bypass for local dev without Google.

## First-time Setup

1. `git clone https://github.com/avantifellows/nex-gen-cms.git`
2. `go mod tidy`
3. `npm install` (first time only — Tailwind CLI + Playwright)
4. `npm run build:css` (`output.css` is generated, not committed — no styles without this)
5. Create a `.env` at the project root with the variables below (a `.env` file **must** exist —
   `config.LoadEnv` is fatal if it can't load one).
6. `go run ./cmd` (or `make run`, which builds CSS first, then starts the server)
7. Open http://localhost:8080 (redirects to `/login`)

## Environment Variables

Required:
- `DB_SERVICE_ENDPOINT` — db-service base URL, e.g. `http://localhost:4000/api/`
- `DB_SERVICE_TOKEN` — Bearer token used for all db-service calls
- `DATABASE_URL` — Postgres DSN for `cms_user_permission` lookups (e.g. `postgres://…?sslmode=disable`)
- `SESSION_SECRET` — long random string that signs the session JWT (sessions can't be issued/read without it)

Conditionally required (real Google sign-in):
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `OAUTH_REDIRECT_URL` — needed for Google login. If any is
  unset, Google login is disabled (the login page shows it as unavailable); use the dev bypass instead.

Optional:
- `DEV_LOGIN_EMAIL` — local-only bypass; exposes a "Sign in as <email>" button / `POST /dev-login`. The
  named user must exist and be active in `cms_user_permission`. **Never set in production.**
- `APP_ENV` — set to `production` to require `Secure` (HTTPS-only) cookies. Leave unset locally (HTTP).

> Removed: `CMS_USERNAME` / `CMS_PASSWORD` (old basic auth) are no longer used — see `context/decisions.md`.

## Common Commands

- `make run` — build CSS, then run the server on `:8080`.
- `go run ./cmd` — run the server (assumes CSS already built).
- `make css-watch` (or `npm run dev:css`) — rebuild CSS on every change; run in a second terminal while editing.
- `npm run build:css` — one-off Tailwind build.
- `go test ./...` — Go unit tests (cmd, config, views).
- `npx playwright test` — E2E tests (server must run on `:8080` with `DEV_LOGIN_EMAIL` set).
- `make build` / `go build -o nex-gen-cms ./cmd` — compile the server binary.

## Common Issues

- **`Error loading .env file` (fatal at startup):** create a `.env` at the project root — `LoadEnv` is fatal.
- **No styles after a fresh clone:** `output.css` isn't built — run `npm run build:css` (or `make run`).
- **Startup fails with `DATABASE_URL is not set` / `ping postgres`:** auth deps are built at startup and
  fail fast — set `DATABASE_URL` and make sure Postgres is reachable.
- **Can't sign in / "account is not authorized":** your email must exist and be `is_active` in
  `cms_user_permission` (admin `pritam@avantifellows.org` is seeded). Locally, set `DEV_LOGIN_EMAIL`.
- **Port 8080 already in use:** `lsof -i :8080`, then `kill -9 <PID>`.
- **Playwright auth setup fails:** the running server needs `DEV_LOGIN_EMAIL` set and that user present/active.
