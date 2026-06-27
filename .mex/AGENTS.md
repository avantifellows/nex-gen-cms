---
name: agents
description: Always-loaded project anchor. Read this first. Contains project identity, non-negotiables, commands, and pointer to ROUTER.md for full context.
last_updated: 2026-06-26
---

# nex-gen-cms

## What This Is
A Go (server-rendered HTML + HTMX) content-management UI for Avanti Fellows' educational content
(chapters, topics, tests, problems), backed by the remote db-service API, with Google-OAuth role-based access.

## Non-Negotiables
- Access content **only** through `services.Service[T]` / the db-service API — never write content SQL.
- Access auth data **only** through `db.CmsUserRepo` with parameterized SQL. Never commit secrets / `.env`.
- Authorize on the server: guard mutating routes in `cmd/main.go` with `editor(...)`/`admin(...)`; the
  `cms_role` cookie and JS gating are cosmetic.
- Render only via `views.ExecuteTemplate(s)`; every helper a template uses must be in its `FuncMap`.
- Never edit or commit `web/static/css/output.css` — it is generated from `input.css`.

## Commands
- Run: `make run` (builds CSS, then `go run ./cmd` on `:8080`)
- CSS watch: `make css-watch` (or `npm run dev:css`)
- Test (Go): `go test ./...`
- Test (E2E): `npx playwright test` (server up on `:8080`, `DEV_LOGIN_EMAIL` set)
- Build: `make build`

## Scaffold Growth
After meaningful work, run GROW:
- Ground: what changed in reality?
- Record: update `ROUTER.md` and relevant `context/` files
- Orient: create or update a `patterns/` runbook if this can recur
- Write: bump `last_updated` on changed scaffold files and run `mex log` when rationale matters

The scaffold grows from real work, not just setup. See the GROW step in `ROUTER.md` for details.

## Navigation
At the start of every session, read `ROUTER.md` before doing anything else.
For full project context, patterns, and task guidance — everything is there.
