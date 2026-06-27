---
name: decisions
description: Key architectural and technical decisions with reasoning. Load when making design choices or understanding why something is built a certain way.
triggers:
  - "why do we"
  - "why is it"
  - "decision"
  - "alternative"
  - "we chose"
edges:
  - target: context/architecture.md
    condition: when a decision relates to system structure
  - target: context/stack.md
    condition: when a decision relates to technology choice
  - target: context/auth.md
    condition: when a decision relates to login, roles, or the user store
  - target: context/deployment.md
    condition: when a decision affects build-at-deploy, infra, or release flow
last_updated: 2026-06-26
---

# Decisions

<!-- HOW TO USE THIS FILE:
     Each decision follows the format below.
     When a decision changes: DO NOT delete the old entry.
     Mark it as superseded, add the new entry above it.
     The history must be preserved — this is the event clock. -->

## Decision Log

### Server-side rendering with HTMX, not a SPA
**Date:** 2024-10-18
**Status:** Active
**Decision:** The UI is server-rendered `html/template` returning HTML fragments, enhanced by HTMX for
partial swaps. No JSON API for the browser, no client framework.
**Reasoning:** A small content-management UI maintained by a backend-leaning team; HTMX keeps state on
the server and avoids a separate frontend build/runtime.
**Alternatives considered:** React/Vue SPA (rejected — extra build, duplicated models, more moving parts).
**Consequences:** Handlers return HTML, not JSON. New screens mean new templates + a render call.
Client logic stays thin (`web/static/js`). Tailwind classes live in templates.

### Generic `Service[T]` for all content access
**Date:** 2024-10-18
**Status:** Active
**Decision:** One generic `services.Service[T]` provides cache-coordinated CRUD over the db-service API
for every model, rather than a hand-written service per entity.
**Reasoning:** Every content type has the same shape (list/get/add/update/archive with a TTL cache).
Generics collapse that into one tested implementation.
**Alternatives considered:** Per-model services (rejected — repetitive, drift-prone).
**Consequences:** Adding a content type needs no new service code — register `NewService[models.X]` in DI
and write a handler. The trade-off: callers pass endpoint strings + cache keys + predicate funcs, so
those must be kept consistent per handler.

### Two data sources — db-service API for content, Postgres only for auth
**Date:** 2026-05-21
**Status:** Active
**Decision:** Content lives behind the db-service REST API; the CMS connects to Postgres **only** to read
and manage the `cms_user_permission` table.
**Reasoning:** The db-service owns the content schema; the CMS owns nothing but its own user/role list.
Splitting keeps content ownership clear and avoids duplicating db-service logic.
**Alternatives considered:** Querying content tables directly in Postgres (rejected — bypasses db-service
ownership and validation).
**Consequences:** Never write content SQL here. Auth is the only `database/sql` usage (`db.CmsUserRepo`).

### Google OAuth + role-based access (viewer/editor/admin)
**Date:** 2026-05-21
**Status:** Active
**Decision:** Login is Google OIDC restricted to the `avantifellows.org` hosted domain; authorization is
role-based (`viewer` < `editor` < `admin`) with roles stored in `cms_user_permission` and carried in a
signed `cms_session` JWT cookie. Routes are guarded by `editor(...)`/`admin(...)` wrappers.
**Reasoning:** Real per-user accounts + revocable, role-scoped access; the org already uses Google Workspace.
**Alternatives considered:** Shared username/password basic auth (see superseded entry below).
**Consequences:** A user must exist and be active in `cms_user_permission` to sign in. See `context/auth.md`.

### Shared username/password basic auth
**Date:** 2025-10-08
**Status:** Superseded by "Google OAuth + role-based access (viewer/editor/admin)"
**Decision:** Gate the app behind a single `CMS_USERNAME`/`CMS_PASSWORD` checked against env vars.
**Reasoning:** Quickest way to keep staging private before real accounts existed.
**Consequences:** ~~Single shared credential, no per-user identity or roles.~~
**Superseded because:** No accountability, no granular access, and credentials lived in env. Replaced by
Google OAuth + the `cms_user_permission` role model (the `CMS_USERNAME`/`CMS_PASSWORD` vars were removed).

### PDF generation via headless Chrome (chromedp) with inlined CSS
**Date:** 2026-01-16
**Status:** Active
**Decision:** Question papers / answer sheets are rendered to PDF by driving headless Chrome with
chromedp: render the template to an HTML string, inline `output.css` into a `<style>` tag, load it via
CDP `Page.SetDocumentContent` (not a `data:` URL), wait for MathJax to finish, then `Page.PrintToPDF`.
**Reasoning:** Papers contain MathJax-typeset math and Tailwind styling; a real browser is the only
faithful renderer. CSS is inlined because headless Chrome can't resolve relative stylesheet links from
an in-memory document. `SetDocumentContent` avoids Chrome aborting navigation on large (>~2MB) `data:` URLs.
**Alternatives considered:** wkhtmltopdf / Go PDF libs (rejected — no MathJax/modern CSS fidelity).
**Consequences:** On EC2 the binary is the Playwright-installed Chromium (`/opt/playwright-browsers`);
locally chromedp finds the system Chrome. See `patterns/generate-pdf.md`.

### Generated Tailwind `output.css` is not committed
**Date:** 2026-06-01
**Status:** Active
**Decision:** `web/static/css/output.css` is `.gitignore`d and built from `input.css` at deploy time,
in CI, and locally (`npm run build:css`).
**Reasoning:** The generated file is large and churns on every class change, polluting diffs and causing
merge conflicts.
**Alternatives considered:** Committing the built CSS (rejected — noisy diffs/merges).
**Consequences:** A fresh clone has no styles until `npm run build:css` runs (`make run` does it for you).
The Tailwind v4 build needs Node 22.
