---
name: architecture
description: How the major pieces of this project connect and flow. Load when working on system design, integrations, or understanding how components interact.
triggers:
  - "architecture"
  - "system design"
  - "how does X connect to Y"
  - "integration"
  - "flow"
edges:
  - target: context/stack.md
    condition: when specific technology details or library versions are needed
  - target: context/decisions.md
    condition: when understanding why the architecture is structured this way
  - target: context/auth.md
    condition: when the request involves login, sessions, roles, or the Postgres user store
  - target: context/conventions.md
    condition: when extending a component and you need the code patterns
  - target: context/deployment.md
    condition: when you need how the running server sits behind NGINX/EC2
last_updated: 2026-06-26
---

# Architecture

## System Overview

A thin Go HTTP server (`net/http` ServeMux) that server-side renders HTML and uses HTMX
for partial page swaps. There is **no SPA** — the browser receives HTML fragments, not JSON.

Request flow:
1. `cmd/main.go` wraps the whole mux in `middleware.RequireLogin` (except a small exceptions
   list: `/login`, `/auth/google/*`, `/dev-login`, static CSS, favicon).
2. `RequireLogin` reads & verifies the `cms_session` JWT cookie → attaches `*SessionClaims`
   to the request context, or redirects to `/login` (HX-Redirect for HTMX requests).
3. The mux routes to a handler. Mutating routes are wrapped with `editor(...)`/`admin(...)`
   role guards; some are wrapped with `middleware.RequireHTMX` (HTMX-origin only).
4. The handler calls a generic `services.Service[T]`, which reads from an in-memory cache
   (`go-cache`) or fetches from the remote **db-service** REST API (`APIRepository`, Bearer token).
5. The handler renders templates via `views.ExecuteTemplate(s)` with a `template.FuncMap` of
   helpers → an HTML fragment (e.g. `chapter_row.html`) or a full page (`home.html` base + content block).
6. HTMX swaps the returned fragment into the DOM client-side.

Auth data (users/roles) is the **only** thing read from Postgres directly (`db.CmsUserRepo`);
all content data (chapters/topics/tests/problems/…) comes from the db-service API.

## Key Components

- **`di.AppComponent`** (`di/app_component.go`) — dependency-injection assembly. Constructs the
  Postgres pool + `GoogleAuth` (fail-fast at startup), one shared cache repo + API repo, one
  `Service[T]` per model, and one handler per vertical. The single place wiring is added.
- **`services.Service[T]`** (`internal/services/service.go`) — generic CRUD over cache + remote API:
  `GetList / GetObject / AddObject / UpdateObject / DeleteObject / ArchiveObject / Post`. Adding a
  new content type needs **no new service code** — just `NewService[models.X]`.
- **`remote_repo.APIRepository`** (`internal/repositories/remote`) — single HTTP client to the
  db-service. Prepends `DB_SERVICE_ENDPOINT`, sets `Authorization: Bearer DB_SERVICE_TOKEN`, 30s
  timeout, non-2xx → error. All content reads/writes funnel through here.
- **`local_repo.CacheRepository`** (`internal/repositories/local`) — `go-cache` TTL store
  (5m default expiry / 10m cleanup). Holds `*[]*T` lists keyed by per-handler cache keys.
- **`db.CmsUserRepo`** (`internal/repositories/db`) — parameterized SQL against the
  `cms_user_permission` Postgres table. The only direct DB access in the app. See `context/auth.md`.
- **`handlers.*`** — one struct per vertical (`ChaptersHandler`, `TestsHandler`, `ProblemsHandler`,
  `ResourcesHandler`, `LoginHandler`, `AdminUsersHandler`, …). Translate HTTP ↔ service calls and
  render templates. Cross-handler helpers live in `handlers/handlerutils/`.
- **`views.ExecuteTemplate(s)`** (`internal/views/render.go`) — the only template-render entry
  point. Resolves paths via `constants.GetHtmlFolderPath()` (`web/html`).
- **`TestsHandler.DownloadPdf`** — headless-Chrome (chromedp) HTML→PDF for question papers /
  answer sheets. See `patterns/generate-pdf.md`.

## External Dependencies

- Content store — the db-service REST API (`DB_SERVICE_ENDPOINT`, `DB_SERVICE_TOKEN`) is the source
  of truth for ALL content (chapters, topics, concepts, tests, problems, resources, skills, tags, exams,
  curriculums, grades, subjects, test rules). Accessed only via `Service[T]` / `APIRepository`.
- Auth store — PostgreSQL (`DATABASE_URL`), used only for the `cms_user_permission` table
  (auth/roles), the same DB that hosts db-service. Accessed only via `db.CmsUserRepo`.
- Login — Google OAuth / OIDC (`accounts.google.com`). Restricted to the `avantifellows.org`
  hosted domain; ID token verified server-side. See `context/auth.md`.
- PDF rendering — headless Chrome via chromedp renders question-paper/answer-sheet HTML (incl. MathJax) to PDF.
- Frontend CDNs — HTMX, MathJax, MathLive, Font Awesome, loaded in `web/html/home.html`.

## What Does NOT Exist Here

- **No ORM and no app-owned content schema.** This service never writes content SQL; content is the
  db-service's responsibility. Postgres here is for auth users only.
- **No SPA / client-side framework** — server-rendered `html/template` + HTMX only. No React/Vue, no JSON API for the UI.
- **No second HTTP router** — standard library `net/http` ServeMux only.
- **No background jobs / queues / schedulers.**
- **No file/object storage layer** — problem images are inlined into HTML (base64) by the editor.
