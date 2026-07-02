---
name: add-content-resource
description: Add a new content resource type (model + service + handler + routes + templates), or add a route to an existing handler. The most common extension tasks.
triggers:
  - "add resource"
  - "add model"
  - "new entity"
  - "add endpoint"
  - "add route"
  - "new handler"
  - "add CRUD"
edges:
  - target: context/conventions.md
    condition: for naming, structure, and the cache/archive/multi-lang code patterns
  - target: context/architecture.md
    condition: to understand the Service[T] + DI + render flow being extended
  - target: patterns/protect-route.md
    condition: when the new route mutates data and needs an editor/admin guard
  - target: patterns/debug-htmx-rendering.md
    condition: when the new route renders blank or the fragment doesn't swap
last_updated: 2026-06-26
---

# Add a Content Resource / Route

## Context

Content types (chapters, topics, tests, problems, resources, …) all share one generic
`services.Service[T]` over the db-service API + `go-cache`. Adding a type needs **no new service code**.
Read `context/conventions.md` (naming, archive, multi-lang, cache-pointer rules) and
`context/architecture.md` (the request flow) first. Look at `internal/handlers/chapter_handler.go` and
`internal/models/chapter.go` as the reference implementation.

## Task: Add a new resource type

### Steps
1. **Model** — create `internal/models/<name>.go`: an exported struct with `json` tags matching the
   db-service response. Use `[]<Name>Lang` for multi-language names. Add `New<Name>(...)` and (for PATCH)
   a `BuildMap(...)` returning `map[string]any`. Include `StatusID int8 \`json:"cms_status_id,omitempty"\``
   if the type is archivable.
2. **DI** — in `di/app_component.go`: add `<name>Service := services.NewService[models.<Name>](cacheRepo, apiRepo)`,
   construct the handler, add a field to `AppComponent`, and assign it in the returned struct.
3. **Handler** — create `internal/handlers/<name>_handler.go`: declare the per-handler consts at the top
   (`<name>EndPoint`, `<name>Key`, template names), a `<Name>sHandler` struct + `New<Name>sHandler(...)`,
   and the HTTP methods (`Load…`, `Get…`, `Add…`, `Update…`, `Archive…`). Call the service, then render
   via `views.ExecuteTemplate(s)` with the `FuncMap` of any helpers the template uses.
4. **Routes** — register paths in `cmd/main.go` `setup()`. Wrap mutating routes with `editor(...)`;
   wrap HTMX-only screens with `middleware.RequireHTMX`.
5. **Templates** — add files under `web/html/` (`snake_case.html`; `_row` / `_dropdown` / `_modal`
   fragments as needed). Full pages render `home.html` (base) + the content block.
6. **Frontend (if needed)** — add JS under `web/static/js/`; new Tailwind classes require `npm run build:css`.
7. **Tests** — add a Playwright spec under `tests/` for new user flows.

### Gotchas
- Filter `StatusID != constants.StatusArchived` in list handlers; archive (PATCH `cms_status_id`) instead
  of hard delete.
- The cache holds `*[]*T` shared across requests — copy a pointer before mutating it per request.
- db-service endpoint strings are inconsistent about leading slashes (`"chapter"` vs `"/skill"`); they are
  concatenated onto `DB_SERVICE_ENDPOINT` — match the db-service route exactly, avoid double slashes.
- Every helper a template calls (`getName`, `add`, `dict`, …) must be in the `FuncMap` you pass, or the
  parse fails and `views` returns a 500 / blank.

### Verify
- [ ] `go build ./...` and `go test ./...` pass.
- [ ] Route hit returns the expected fragment (curl with `-H "HX-Request: true"` if HTMX-only).
- [ ] Archived items don't appear in lists; mutating routes are role-guarded.
- [ ] New templates render (no "Template Parsing/Execution Error" in server logs).

## Task: Add a route to an existing handler

### Steps
1. Add the method to the existing `internal/handlers/<name>_handler.go` (follow the verb-first naming).
2. Reuse the handler's existing `Service[T]`, endpoint const, and cache key — don't introduce a new client.
3. Register the path in `cmd/main.go` `setup()` with the right guard (`editor`/`admin`/`RequireHTMX`).
4. Add/extend the template + its `FuncMap`.

### Gotchas
- If it mutates content via the service, keep the cache consistent by using `UpdateObject` /
  `AddObject` / `ArchiveObject` (which update the cached list) rather than `Post` + manual cache edits.
- HTMX-only routes return a redirect to `/chapters` for non-HTMX requests (`RequireHTMX`) — test with the header.

## Update Scaffold
- [ ] Update `.mex/ROUTER.md` "Current Project State" if a new capability now works.
- [ ] Update `.mex/context/` files if a convention or external endpoint changed.
- [ ] If this task surfaced a new gotcha, update this pattern.
