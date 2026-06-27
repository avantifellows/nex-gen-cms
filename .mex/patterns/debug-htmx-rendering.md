---
name: debug-htmx-rendering
description: Diagnose the most common failure boundary — an HTMX request that returns blank, 500s, redirects, or doesn't swap into the page. Use when a fragment route "does nothing".
triggers:
  - "blank response"
  - "htmx not swapping"
  - "fragment empty"
  - "template error"
  - "nothing happens"
  - "500 on render"
  - "redirect to chapters"
edges:
  - target: context/conventions.md
    condition: for the template/render and FuncMap rules being violated
  - target: context/auth.md
    condition: when the symptom is a redirect to /login or a 403
  - target: patterns/add-content-resource.md
    condition: when debugging a route you just added
last_updated: 2026-06-26
---

# Debug HTMX / Template Rendering

## Context

The riskiest boundary in this app is handler → `views.ExecuteTemplate(s)` → HTMX swap. Failures are
quiet: a 500 from a bad template parse, a silent empty body from a guard clause, or a redirect HTMX
ignores. Reproduce by hitting the route with curl and the right headers before touching code.

```bash
# Mimic an HTMX request (some routes require this header):
curl -i -H "HX-Request: true" -b "cms_session=<paste-cookie>" \
  "http://localhost:8080/api/chapters?curriculum-dropdown=1&grade-dropdown=1&subject-dropdown=1"
```

## What to check, in order
1. **Status / headers.** `302 → /login` = no/invalid session. `HX-Redirect: /login` (401) = HTMX
   unauthenticated. `403` (with `HX-Reswap: none`) = role too low. `303 → /chapters` = a `RequireHTMX`
   route hit without the `HX-Request` header. See `context/auth.md`.
2. **Empty 200 body.** Many handlers `return` early when required query params are missing/zero —
   e.g. `getCurriculumGradeSubjectIds` yields `0` and the handler bails. Confirm you sent
   `curriculum-dropdown` / `grade-dropdown` / `subject-dropdown` (note: form-field names, not `*_id`).
3. **500 "Internal Server Error" + server log `Template Parsing/Execution Error`.** A template references
   a helper not in the passed `FuncMap`, a missing field, or a wrong template filename. Cross-check the
   `FuncMap` keys against `{{ ... }}` calls in the template, and the template const against the file in `web/html/`.
4. **Swaps but looks unstyled.** `output.css` not built — run `npm run build:css`.
5. **Stale data after an edit.** The `go-cache` list wasn't updated/invalidated. Confirm the mutation went
   through `Service[T]` `UpdateObject`/`AddObject`/`ArchiveObject` (which fix the cached list) rather than a
   raw `Post`. Cache TTL is 5m.
6. **Wrong fragment for the context.** Several handlers branch on `?view=list|dropdown|...`; verify the
   `view` param matches the template you expect.

## Gotchas
- `views.ExecuteTemplate` uses `template.Must` and will **panic** on a parse error for the single-file
  path; `ExecuteTemplates` logs and returns a 500. Check server stdout, not just the browser.
- Form field names are hyphenated consts (`curriculum-dropdown`), but db-service query params are
  underscored (`curriculum_id`) — mixing them yields an empty/`0` parse and an early return.
- A handler that calls `http.Error` after already writing part of the response will produce a garbled body.

## Verify (after fixing)
- [ ] curl with the correct headers/params returns the expected fragment and 200.
- [ ] No `Template Parsing/Execution Error` in server logs.
- [ ] The element actually swaps in the browser (HTMX target + swap attributes are correct).

## Update Scaffold
- [ ] If the failure was a recurring, non-obvious trap, add it to `context/conventions.md` or this pattern.
