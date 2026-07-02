---
name: stack
description: Technology stack, library choices, and the reasoning behind them. Load when working with specific technologies or making decisions about libraries and tools.
triggers:
  - "library"
  - "package"
  - "dependency"
  - "which tool"
  - "technology"
edges:
  - target: context/decisions.md
    condition: when the reasoning behind a tech choice is needed
  - target: context/conventions.md
    condition: when understanding how to use a technology in this codebase
  - target: context/setup.md
    condition: when installing toolchains or building CSS
last_updated: 2026-06-26
---

# Stack

## Core Technologies

- **Go 1.25** (`go.mod`) — backend language; standard `net/http` ServeMux, no web framework.
- Frontend: HTML + HTMX + vanilla JS. Server renders `html/template`; HTMX does partial swaps.
- **Tailwind CSS v4** (`@tailwindcss/cli`) — styling. `web/static/css/output.css` is built from
  `input.css`, never hand-written. Requires Node 22 to build.
- **PostgreSQL** (`github.com/lib/pq`, `database/sql`) — auth only (`cms_user_permission` table).
- **Node.js 22** — only for the Tailwind build and Playwright; not a runtime dependency of the server.

## Key Libraries

- **`github.com/patrickmn/go-cache`** — in-memory TTL cache backing every `Service[T]`. Not Redis.
- **`github.com/thoas/go-funk`** — `Find` / `Filter` over slices (e.g. predicate lookups in `Service[T]`,
  filtering archived items). Used instead of hand-rolled loops in service/handler code.
- **`github.com/chromedp/chromedp`** (+ `cdproto`) — headless-Chrome HTML→PDF. Not wkhtmltopdf.
- **`github.com/coreos/go-oidc/v3`** + **`golang.org/x/oauth2`** — Google OIDC login + token verify.
- **`github.com/golang-jwt/jwt/v5`** — signs/verifies the `cms_session` cookie (HS256, `SESSION_SECRET`).
- **`github.com/lib/pq`** — Postgres driver for the auth queries (raw parameterized SQL, no ORM).
- **`github.com/joho/godotenv`** — loads `.env` at startup (`config.LoadEnv` is fatal on failure).
- **`github.com/stretchr/testify`** — assertions/mocks in Go unit tests.
- **`@playwright/test`** — E2E browser tests under `tests/` (relies on the dev-login bypass).

## What We Deliberately Do NOT Use

- **No ORM (GORM/sqlx/ent).** Auth uses `database/sql` + `lib/pq` with parameterized queries;
  content uses the generic `Service[T]` over the db-service REST API — never SQL for content.
- **No SPA framework (React/Vue/Svelte).** HTMX + server-rendered templates only.
- **No second router / mux library (chi, gorilla, gin).** Standard `net/http` ServeMux only.
- **No CSS beyond Tailwind**, and `output.css` is generated — never edit or commit it (it's `.gitignore`d).
- **No Redis / external cache.** In-process `go-cache` is the only cache.

## Version Constraints

- **Go >= 1.25** — `Service[T]` and `utils.StringToIntType[T]` rely on generics.
- **Tailwind v4** (upgraded 2026-05) — its CLI build requires **Node 22**; older Node breaks the build.
- Handlers import **`text/template`** for building `FuncMap`s while `internal/views` uses
  **`html/template`**; they interoperate because `html/template.FuncMap` is an alias of
  `text/template.FuncMap`. Output HTML escaping is governed by `html/template` in `views`.
