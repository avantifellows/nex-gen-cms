---
name: generate-pdf
description: Work on or debug PDF generation (question papers / answer sheets) via headless Chrome (chromedp). Use when touching DownloadPdf, PDF templates, or diagnosing broken/blank/timed-out PDFs.
triggers:
  - "pdf"
  - "download pdf"
  - "chromedp"
  - "question paper"
  - "answer sheet"
  - "mathjax"
  - "print to pdf"
edges:
  - target: context/architecture.md
    condition: for where DownloadPdf sits in the overall flow
  - target: context/decisions.md
    condition: for why chromedp + inlined CSS + SetDocumentContent were chosen
  - target: context/setup.md
    condition: when PDF fails locally due to a missing Chrome
  - target: context/deployment.md
    condition: when PDF fails on EC2 (Playwright Chromium path or missing fonts)
last_updated: 2026-09-29
---

# PDF Generation (chromedp)

## Context

`TestsHandler.DownloadPdf` (`internal/handlers/test_handler.go`) renders three PDF types — `questions`,
`questions_with_answers`, `answers` — by driving headless Chrome. The math is MathJax-typeset and the
layout is Tailwind, so a real browser is the only faithful renderer. Read the "PDF via headless Chrome"
entry in `context/decisions.md`. Shared markup lives in `web/html/test_pdf_shared.html`; per-type templates
are `question_paper.html`, `question_paper_with_answers.html`, `answer_sheet.html`.

Since 2026-09, a rendered PDF is also cached in S3 by content hash (`context/decisions.md` — "Cache
generated test PDFs in S3") — `DownloadPdf` only reaches Steps 3-6 below on a cache miss.

## Steps (the rendering pipeline, in order)
1. Render the chosen template (+ `test_pdf_shared.html`) to an HTML string with the PDF `FuncMap`
   (`getName`, `add`, `labels`, `dict`, `capitalize`, `getSectionName`, `stringToInt`, `trim`, `getChapterName`).
2. Inline CSS: read `web/static/css/output.css` and inject it (plus a white-background override) before
   `</head>`. Headless Chrome can't resolve the relative stylesheet link from an in-memory document.
   `pdfContentHash` hashes this HTML plus the header HTML and `pdfCacheVersion`; if it matches
   `Test.TypeParams.PdfUrls[pdfCacheKey]` and the S3 object still exists, `DownloadPdf` redirects to a
   presigned URL here and skips the rest of this pipeline entirely.
3. Pick the Chrome binary: if `/opt/playwright-browsers` exists (EC2), use the Playwright Chromium at
   `chromium-*/chrome-linux/chrome` with `--no-sandbox --disable-gpu --headless`; otherwise use the system
   Chrome via the default chromedp allocator.
4. Load HTML via CDP `Page.SetDocumentContent` after `Navigate("about:blank")` — **not** a `data:` URL
   (Chrome aborts navigation `net::ERR_ABORTED` for `data:` URLs over ~2MB).
5. Wait for `window.load`, then poll `#mathjax-done` (set to `"true"` by the page after MathJax finishes)
   for up to 50s, then flush `document.fonts.ready` + two `requestAnimationFrame`s.
6. `Page.PrintToPDF` (A4, print background on, custom header/footer) → stream as
   `Content-Disposition: attachment`. On the response path, a detached background goroutine then uploads
   the PDF to S3 and PATCHes the cache entry onto the test (re-fetched with `onlyRemote=true` first, so it
   can't revert a concurrent edit made during the render).

## Gotchas
- **CSS must be inlined**, not linked — and the white-background `<style>` override is required because
  `output.css` paints the app's warm-beige background, which is wrong for a printed page.
- **Don't reintroduce `data:` URLs** for the document — large papers exceed Chrome's limit and abort.
- **The MathJax gate is `#mathjax-done`**, not a fixed sleep. If a template doesn't set it, the poll times
  out after ~50s with `MathJax typeset did not complete`. Keep the shared "mathjax-done" wiring intact.
- **EC2 vs local Chrome differ.** On the server the binary is Playwright-installed under
  `/opt/playwright-browsers`; locally chromedp finds system Chrome. Missing Chrome on EC2 →
  "Playwright Chromium not found". Fonts must be installed on the box (`fontconfig`) or text renders wrong.
- Whole render is bounded by a 60s context timeout.
- **Changing any `PrintToPDF` option isn't reflected in `pdfContentHash`** (it only hashes the HTML/header
  strings) — bump `pdfCacheVersion` when you do, or stale cached PDFs keep serving the old output.
- **The cache is best-effort and silent.** `AWS_S3_BUCKET` unset, an S3 error, or a stale/missing cached
  object all just fall through to a normal render — never surfaced to the caller. If you suspect the cache
  is serving wrong output, check `Test.TypeParams.PdfUrls` and the server logs (`log.Printf("pdf cache: ...")`
  lines), not the HTTP response.

## Verify
- [ ] `GET /download-pdf?...&type=questions` (and `answers`, `questions_with_answers`) returns a valid PDF.
- [ ] Math renders (not raw `$...$`), backgrounds are white, header/footer + page numbers present.
- [ ] No `net::ERR_ABORTED`, no MathJax-timeout in server logs for a large test.
- [ ] Repeating the same download redirects (302) to a presigned S3 URL instead of re-rendering; changing
      the test's content (or `pdfCacheVersion`) produces a fresh render + a new cache entry.

## Debug
- **Blank/partial math** → `#mathjax-done` never reached `"true"`; check the shared template's MathJax hook.
- **`net::ERR_ABORTED` / nothing renders** → something switched back to a `data:` URL; must be `SetDocumentContent`.
- **Unstyled or beige PDF** → `output.css` wasn't built/inlined, or the white-bg override was removed.
- **"Playwright Chromium not found" (EC2)** → the `/opt/playwright-browsers/chromium-*` dir is missing.
- **Wrong/boxed glyphs on EC2** → missing system fonts (`fc-cache`/`fontconfig`).
- **Download always re-renders, cache never seems to hit** → confirm `AWS_S3_BUCKET`/`AWS_REGION` are set
  and the EC2 instance role has `s3:GetObject`/`s3:PutObject` on the bucket (`terraform/iam.tf`); check for
  `pdf cache: ...` log lines (HeadObject/presign/upload/refetch/persist failures are all logged, never
  returned as an error to the client).
- **Serving stale content after an edit** → `pdfContentHash` didn't change, meaning nothing you edited is
  part of the hashed HTML/header — if you changed a `PrintToPDF` option, bump `pdfCacheVersion`.

## Update Scaffold
- [ ] If the render pipeline, the Chrome-path logic, or the S3 cache behavior changes, update this pattern
      and `context/decisions.md`.
