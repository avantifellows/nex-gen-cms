# Pattern Index

Lookup table for all pattern files in this directory. Check here before starting any task — if a pattern exists, follow it.

<!-- Reviewed 2026-10-01: rows below still match the files in this directory. -->

<!-- Each row maps a pattern file (or section) to its trigger — when should the agent load it?
     Row format uses a Markdown link in the first cell:
       simple   — `[name.md](name.md)` | when to use it
       anchored — `[name.md#task-first-task](name.md#task-first-task)` | when to use it
     Keep the table sorted alphabetically, one row per task (not per file).
     If you create a pattern, add it here; if you delete one, remove its row. -->

| Pattern | Use when |
|---------|----------|
| [add-content-resource.md#task-add-a-new-resource-type](add-content-resource.md#task-add-a-new-resource-type) | Adding a new content type (model + service + handler + routes + templates) |
| [add-content-resource.md#task-add-a-route-to-an-existing-handler](add-content-resource.md#task-add-a-route-to-an-existing-handler) | Adding a new route/endpoint to an existing handler |
| [debug-htmx-rendering.md](debug-htmx-rendering.md) | An HTMX route returns blank/500/redirect or doesn't swap |
| [generate-pdf.md](generate-pdf.md) | Editing or debugging question-paper/answer-sheet PDF generation (chromedp) |
| [protect-route.md](protect-route.md) | Adding/changing auth or role authorization on a route |

<!-- Reviewed 2026-09-29: still one row per pattern file in .mex/patterns/, alphabetically sorted. -->
