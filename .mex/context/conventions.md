---
name: conventions
description: How code is written in this project — naming, structure, patterns, and style. Load when writing new code or reviewing existing code.
triggers:
  - "convention"
  - "pattern"
  - "naming"
  - "style"
  - "how should I"
  - "what's the right way"
edges:
  - target: context/architecture.md
    condition: when a convention depends on understanding the system structure
  - target: patterns/add-content-resource.md
    condition: when adding a new resource type or a route to an existing handler
  - target: context/auth.md
    condition: when the convention involves role guards or session/role data
last_updated: 2026-06-26
---

# Conventions

## Naming

- **Go files:** `snake_case` (`chapter_handler.go`, `cms_user_repo.go`, `api_repository.go`).
- **Templates:** `snake_case.html`. `GenericHandler` maps a route to a file by replacing `-` with `_`
  (`/add-chapter` → `add_chapter.html`). Fragments are suffixed by shape: `_row`, `_dropdown`, `_modal`.
- **Handlers:** one struct per vertical `XxxHandler` with a `NewXxxHandler(...)` constructor.
  Exported methods are the HTTP handlers, verb-first: `LoadXxx`, `GetXxx`, `AddXxx`/`Create…`,
  `UpdateXxx`, `ArchiveXxx`, `DeleteXxx`.
- **Per-handler consts** declared at the top of each handler file: endpoint, cache key, and template
  names (e.g. `chaptersEndPoint = "chapter"`, `chaptersKey = "chapters"`, `chapterRowTemplate = "chapter_row.html"`).
- **Models:** exported structs with `json` tags matching the db-service. Multi-language names are
  `[]XxxLang` slices of `{Name, LangCode}` (e.g. `ChapterLang{ChapterName, LangCode}`). Each model
  has a `NewXxx(...)` constructor and often a `BuildMap(...)` returning the PATCH payload.
- **Shared form/query keys** are consts: `CURRICULUM_DROPDOWN_NAME = "curriculum-dropdown"`,
  `GRADE_DROPDOWN_NAME`, `SUBJECT_DROPDOWN_NAME` (form field names), `QUERY_PARAM_CURRICULUM_ID = "curriculum_id"`.
- **Postgres columns** (`cms_user_permission`): `snake_case` (`is_active`, `last_login_at`, `inserted_at`).
- **Roles:** lowercase strings — `viewer`, `editor`, `admin` (constants in `internal/auth/roles.go`).

## Structure

- Layout: `cmd/` entrypoint + the route table; `di/` wiring; `config/` env; `internal/{handlers,
  services,repositories,models,dto,middleware,auth,constants,views}`; `utils/`; `web/{html,static}`.
- **All content data access goes through `services.Service[T]`.** Never call `APIRepository` or hit a
  db-service endpoint directly from a handler.
- **All auth data access goes through `db.CmsUserRepo`** with parameterized SQL. Never inline SQL in handlers.
- Handlers translate HTTP ↔ service calls and render templates. No business logic in templates.
  Shared helpers across handlers live in `handlers/handlerutils/`.
- **Routes are registered only in `cmd/main.go` `setup()`.** Wrap mutating routes with `editor(...)`
  or `admin(...)`; wrap HTMX-only routes with `middleware.RequireHTMX`.
- **DTOs** (`internal/dto`) are view-models passed to templates. Screen DTOs embed `dto.HomeData`
  (curriculum/grade/subject ids shared across screens).
- **Template rendering only via `views.ExecuteTemplate` / `ExecuteTemplates`** — never call
  `template.ParseFiles` directly except inside `views` / `GenericHandler`.

## Patterns

**Generic service CRUD — pass endpoint + cacheKey + a predicate, let the service keep the cache consistent:**
```go
// Update: predicate matches the cached item to replace.
h.chaptersService.UpdateObject(idStr, chaptersEndPoint, chapterMap, chaptersKey,
    func(c *models.Chapter) bool { return c.ID == id })
```

**Archive, don't hard-delete, for content — PATCH a status, then filter it out of lists:**
```go
chapterMap := map[string]any{"cms_status_id": constants.StatusArchived}
h.chaptersService.ArchiveObject(idStr, chaptersEndPoint, chapterMap, chaptersKey,
    func(c *models.Chapter) bool { return c.ID != id }) // keep predicate
// In list handlers:
*chapters = funk.Filter(*chapters, func(c *models.Chapter) bool {
    return c.StatusID != constants.StatusArchived }).([]*models.Chapter)
```

**Multi-language names — never assign a bare string:**
```go
// Correct
Name: []ChapterLang{{ChapterName: name, LangCode: "en"}}
ch.GetNameByLang("en")
// Wrong: ch.Name = name
```

**Never mutate a cached pointer — the list cache holds `[]*T` shared across requests. Copy first:**
```go
localChapter := *selectedChapterPtr // shallow copy before per-request mutation
localChapter.Topics = nil
```

**Authorization is server-side at the route. The `cms_role` cookie / JS gating is cosmetic only:**
```go
muxHandler.HandleFunc("/create-chapter", editor(chaptersHandler.AddChapter))
```

## Verify Checklist

Before presenting any code:
- [ ] `go build ./...` compiles and `go test ./...` passes.
- [ ] Content data access goes through `Service[T]` (not `APIRepository`/raw HTTP); auth data through
      `CmsUserRepo` with parameterized SQL.
- [ ] New route is registered in `cmd/main.go`, wrapped with `editor(...)`/`admin(...)` if it mutates,
      and `RequireHTMX` if it must be HTMX-only.
- [ ] New template is under `web/html/`, rendered via `views.ExecuteTemplate(s)`, name matches the
      handler reference, and the `FuncMap` includes **every** helper the template calls.
- [ ] List handlers filter out `StatusArchived`; content deletes are archives (`cms_status_id`).
- [ ] Multi-lang names use `[]XxxLang` (not raw strings); no mutation of cached `[]*T` pointers.
- [ ] If Tailwind classes changed, `npm run build:css` was run; `output.css` was not hand-edited or committed.
