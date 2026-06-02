# Project Guidance

## Quality Checks

- Run `gofmt -l` on changed Go files and require no output before review or handoff.
- Run `go test ./...` for backend changes.
- Run `npm run build:css` after template or Tailwind class changes.
- Playwright coverage lives under `tests/` and is optional unless the change touches an existing browser-level workflow or adds browser coverage explicitly.

## Curriculum Config

- Curriculum Config handlers use server-rendered full pages plus HTMX partials; do not introduce a separate client app for this workflow.
- Table, pagination, sort, export, and post-mutation refreshes use the last applied filter contract: `exam_track`, `grade`, `subject`, `search`, `chapter_id`, `syllabus_status`, `page`, `limit`, `sort`, and `dir`.
- Mutation forms preserve applied filters with hidden `filter_*` fields so create, update, and remove responses can refresh the same table context.
- Impact preview requests must include the candidate row context, not just the changed field: chapter id, exam track, syllabus status, prescribed minutes, and coverage order. Edit previews also include the config id so duplicate coverage checks can exclude the current row.
- Curriculum Config repository code uses direct Postgres queries against LMS tables and does not call LMS APIs or existing CMS content caches.
