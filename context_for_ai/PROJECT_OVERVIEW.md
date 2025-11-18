## Avanti Next Generation CMS — Project Overview

This document provides a comprehensive overview of the Next Generation CMS codebase. It is intended as a living reference for contributors (engineers and AI agents) to quickly understand the architecture, data flow, conventions, and extension points.


### High-level Summary

- **Language/Runtime**: Go 1.23 for backend; HTML + HTMX + TailwindCSS + vanilla JS for frontend; Playwright for E2E tests.
- **Architecture**: Thin Go HTTP server using the standard `net/http` mux, dependency-injected handlers, generic services backed by an in-memory cache and a remote API.
- **UI Composition**: Server-side rendered HTML templates (Go `html/template`) with HTMX for partial updates and navigation, Tailwind for styling.
- **Data Source**: A remote “DB service” accessed via REST with Bearer token (`DB_SERVICE_ENDPOINT`, `DB_SERVICE_TOKEN`).
- **PDF Generation**: Headless Chrome (chromedp) renders HTML (MathJax, Tailwind included) to PDF for question papers and answer sheets.


### Repository Layout

- `cmd/`
  - `main.go`: Entrypoint. Creates mux, builds DI `AppComponent`, registers routes.
  - `main_test.go`: Verifies route registration and environment loading.
- `config/`
  - `env.go`: .env loader and `GetEnv()` utility, fatal on missing/failed `.env`.
  - `env_test.go`: Tests for env loading behavior.
- `di/`
  - `app_component.go`: Dependency injection assembly of repositories, generic services, and handlers.
- `internal/constants/`
  - `constants.go`: Runtime constants (e.g., HTML folder), sort order enums, resource statuses.
- `internal/dto/`
  - View-models (`HomeData`, `PaperData`, `TopicsData`, `SortState`) passed into templates.
- `internal/handlers/`
  - Handlers for chapters, topics, tests, problems, etc. They translate HTTP to service calls and render templates.
  - `handlerutils/`: Shared handler helpers (subject/topic lookups).
- `internal/middleware/`
  - `htmx_middleware.go`: Enforces HTMX-originated requests for certain routes.
- `internal/models/`
  - Domain models with JSON mapping for the remote API.
- `internal/repositories/`
  - `local/`: `CacheRepository` (go-cache) and template execution helpers.
  - `remote/`: `APIRepository` for HTTP calls to the DB service.
- `internal/services/`
  - `service.go`: Generic `Service[T]` handling list/object CRUD with cache coordination.
- `utils/`
  - Helpers for conversion, math, paths, templating utilities (e.g., `Slice`, `Dict`, `ToJson`).
- `web/`
  - `html/`: Base and partial Go HTML templates.
  - `static/`: Tailwind output CSS and JS utilities (editor, math, images, nav-tracking, constants).
- `tests/`
  - Playwright tests and mocks for UI flows and HTMX interactions.
- `terraform/`
  - Complete AWS infrastructure as code: EC2, security groups, EIP, Cloudflare DNS, S3/DynamoDB backend.
  - `user-data.sh`: Instance initialization script with application deployment, NGINX setup, and SSL configuration.
- `.github/workflows/`
  - `deploy-staging.yml`: GitHub Actions CI/CD pipeline for automated deployment to AWS (staging).
  - `deploy-prod.yml`: GitHub Actions CI/CD pipeline for automated deployment to AWS (production).


### Runtime and Configuration

- **Port**: The server listens on `0.0.0.0:8080` (see `cmd/main.go`).
- **Environment**: `.env` is loaded via `github.com/joho/godotenv` from the application working directory.
  - `DB_SERVICE_ENDPOINT`: Base URL of the remote DB service (e.g., `https://api.example.com/`).
  - `DB_SERVICE_TOKEN`: Bearer token used for all calls to the DB service.
- **Template path**: Determined at runtime by `internal/constants`. Defaults to `web/html`. A special case exists for Windows test runs.
- **Deployment**: Terraform configuration for AWS EC2 deployment with NGINX reverse proxy, SSL via Let's Encrypt, and GitHub Actions CI/CD (see `terraform/` directory).


### Request Flow Overview

1. Browser loads `GET /` which is treated as `/home`. `handlers.GenericHandler` renders a top-level base template `home.html` with nav and dropdowns.
2. HTMX enhances navigation and partial updates. Selecting nav tabs or dropdowns issues HTMX requests to endpoints (e.g., `/chapters`, `/api/chapters`).
3. Handlers invoke a generic `Service[T]` to fetch from cache or the remote API via repositories.
4. Handlers render server-side templates using `local_repo.ExecuteTemplate(s)` with optional `template.FuncMap` helpers.
5. For certain routes, `HTMXMiddleware` enforces that requests originate from HTMX; otherwise users are redirected to `/chapters`.


### Route Map (HTTP -> Handler -> Template/Behavior)

- `GET /` → `GenericHandler` → resolves to `/home` → renders `home.html`.
- Static assets: `GET /web/*` → serves from `./web` via `http.StripPrefix`.

- Chapters
  - `GET /chapters` → `ChaptersHandler.LoadChapters` → `home.html` + `chapters.html` (base + content block).
  - `GET /api/curriculums` → `CurriculumsHandler.GetCurriculums` → `curriculums.html` options.
  - `GET /api/grades` → `GradesHandler.GetGrades` → `grades.html` options.
  - `GET /api/subjects` → `SubjectsHandler.GetSubjects` → `subjects.html` options.
  - `GET /api/chapters` → `ChaptersHandler.GetChapters` → `chapter_row.html` or `chapter_dropdown.html`.
  - `GET /chapter` → `ChaptersHandler.GetChapter` → base + `chapter.html`.
  - `GET /topics` → `ChaptersHandler.LoadTopics` → `topics.html` shell.
  - `GET /api/topics` → `ChaptersHandler.GetTopics` → `topic_row.html` or `topic_dropdown.html`.
  - `GET /edit-chapter` (HTMX-only) → `ChaptersHandler.EditChapter` → base + `edit_chapter.html`.
  - `PATCH /update-chapter` → `ChaptersHandler.UpdateChapter` → `update_success.html`.
  - `POST /create-chapter` → `ChaptersHandler.AddChapter` → returns `chapter_row.html` for insertion.
  - `DELETE /delete-chapter` → `ChaptersHandler.DeleteChapter`.

- Topics
  - `GET /add-topic` → `TopicsHandler.OpenAddTopic` → `add_topic.html`.
  - `POST /create-topic` → `TopicsHandler.AddTopic` → `topic_row.html`.
  - `DELETE /delete-topic` → `TopicsHandler.DeleteTopic`.
  - `GET /edit-topic` (HTMX-only) → `TopicsHandler.EditTopic` → `edit_topic.html`.
  - `PATCH /update-topic` → `TopicsHandler.UpdateTopic` → `update_success.html`.
  - `GET /topic` → `TopicsHandler.GetTopic` → base + `topic.html`.

- Concepts
  - `GET /api/concepts` (optional `topic_id`) → `ConceptsHandler.GetConcepts` → `concept_row.html`.

- Tests
  - `GET /tests` → `TestsHandler.LoadTests` → base + `tests.html` (+ `test_type_options.html`).
  - `GET /api/tests` → `TestsHandler.GetTests` → `test_row.html` (sorting via sessionStorage on client).
  - `GET /test` → `TestsHandler.GetTest` → base + `test.html`.
  - `GET /api/test/problems` → `TestsHandler.GetTestProblems` → `test_problem_row.html`.
  - `POST /tests/add-test` → `TestsHandler.AddTest` → base + `add_test.html` (+ multiple partials) for test composition.
  - `POST /add-question-to-test` → `TestsHandler.AddQuestionToTest` → returns fragments depending on subject/subtype context (various `dest_*` templates).
  - `POST /create-test` → `TestsHandler.CreateTest`.
  - `GET /tests/edit-test` → `TestsHandler.EditTest` → base + `add_test.html` (edit mode) + partials.
  - `GET /tests/add-test-dialog` (HTMX-only) → `TestsHandler.AddTestModal` → modal HTML.
  - `GET /add-curriculum-grade-selects` → `TestsHandler.AddCurriculumGradeDropdowns`.
  - `PATCH /update-test` → `TestsHandler.UpdateTest`.
  - `PATCH /archive-test` → `TestsHandler.ArchiveTest`.
  - `GET /download-pdf?type=questions|answers` → `TestsHandler.DownloadPdf` → PDF stream.

- Problems
  - `GET /problem` → `ProblemsHandler.GetProblem` → base + `problem.html`.
  - `GET /api/topic/problems` → `ProblemsHandler.GetTopicProblems` → either `src_problem_row.html` or `topic_problem_row.html` (filtering by difficulty, subtype, selection).
  - `GET /problems` → `ProblemsHandler.LoadProblems` → `problems.html` (shell for topic problems).
  - `GET /topic/add-problem` → `ProblemsHandler.AddProblem` → base + `add_problem.html` (+ editor partials).
  - `POST /create-problem` → `ProblemsHandler.CreateProblem` (raw JSON passthrough to API).
  - `GET /problems/edit-problem` → `ProblemsHandler.EditProblem` → base + `add_problem.html` (+ editor partials).
  - `PATCH /update-problem` → `ProblemsHandler.UpdateProblem`.
  - `PATCH /archive-problem` → `ProblemsHandler.ArchiveProblem`.

- Lookups & Aux
  - `GET /api/skills` → `SkillsHandler.GetSkills` → `skills.html` (multi-select matrix with selected IDs).
  - `GET /api/tags` → `TagsHandler.GetTags` → `tag_row.html` (filtered by `q`, excluding already selected tags).


### Service and Repository Layer

Generic service encapsulates cache + remote API for any model type `T`:

- `GetList(urlEndPoint, cacheKey, onlyCache, onlyRemote) (*[]*T, error)`
  - Reads from cache unless `onlyRemote == true`.
  - On remote fetch, JSON unmarshals into `[]*T` and caches under `cacheKey`.
- `GetObject(idStr, predicate, cacheKey, urlEndPoint) (*T, error)`
  - Attempts to find in cached list using `predicate`, falls back to GET (with optional `/{id}` suffix).
- `AddObject(body, cacheKey, urlEndPoint) (*T, error)`
  - POSTs to remote, appends the returned object to cached list if present.
- `UpdateObject(idStr, urlEndPoint, body, cacheKey, predicate) (*T, error)`
  - PATCHes remote; if cached list is present, replaces the matched item.
- `DeleteObject(idStr, keepPredicate, cacheKey, urlEndPoint) error`
  - DELETEs remote; prunes cached list via `keepPredicate`.
- `ArchiveObject(idStr, urlEndPoint, body, cacheKey, keepPredicate) error`
  - PATCH with an archive payload; prunes cached list.

Repositories:

- `internal/repositories/local/CacheRepository`: wraps `github.com/patrickmn/go-cache` for in-memory TTL caching.
- `internal/repositories/local.ExecuteTemplate(s)`: uniform helpers to render one or many templates with optional `template.FuncMap`.
- `internal/repositories/remote/APIRepository`:
  - Builds `apiUrl = DB_SERVICE_ENDPOINT + urlEndPoint`, sets `Authorization: Bearer DB_SERVICE_TOKEN` and `Content-Type: application/json`.
  - 10s timeout; returns body bytes or an error if status not in 2xx.

DI (`di/app_component.go`) wires:

- One cache repo instance (5m default, 10m cleanup).
- One remote repo instance.
- One `Service[T]` per model type (Chapters, Topics, Concepts, Curriculums, Grades, Subjects, Skills, Tests, Problems, Tags, Test Rules).
- One handler per vertical.


### Domain Models (selected fields)

- `Chapter`: `ID int16`, `Code string`, `Name []ChapterLang`, `CurriculumID int16`, `GradeID int8`, `SubjectID int8`, `Topics []*Topic`.
- `Topic`: `ID int16`, `Name []TopicLang`, `Code string`, `ChapterID int16`, `CurriculumID int16`.
- `Concept`: `ID int16`, `Name []ConceptLang`, `TopicID int16`.
- `Curriculum`: `ID int16`, `Name string`, `Code string`.
- `Grade`: `ID int8`, `Number int8`.
- `Subject`: `ID int8`, `Name []SubjectLang`, `Code string`, `ParentID int8`, `ParentName []SubjectLang`.
- `Skill`: `ID int16`, `Name string`.
- `Problem`: Rich object including `MetaData` with HTML (question, options, solutions), `SkillIDs`, `SubjectID`, `TopicID`, `TagIDs`, `Status`.
- `Test`: `Name []ResName`, `Code`, `Type`, `Subtype`, `ExamIDs []int8`, `CurriculumGrades []CurriculumGrade`, `TypeParams` (duration, marks, per-subject sections), `Status`.
- `TestRule`: Per exam+test-type configuration including subjects, marking scheme, instructions (HTML).

DTOs:

- `HomeData`: Pivotal view-model holding selected IDs, pointers to selected `Chapter|Topic|Problem|Test`, map of problems, and an optional `TestRule`.
- `TopicsData`: For topics tab (ChapterId + sorting state).
- `PaperData`: For PDF rendering (test + problems + rule).


### Templates and Frontend Behavior

- Base template: `web/html/home.html` renders the nav (Chapters/Tests) and the three dropdowns (Curriculum, Grade, Subject). It includes:
  - HTMX (`unpkg`), Tailwind CSS output (`/web/static/css/output.css`), Font Awesome, MathLive, MathJax.
  - Custom JS: `/web/static/js/constants.js`, `/web/static/js/nav-tracker.js`.
  - Custom HTMX extensions:
    - `mathJaxLoader`: After swaps, conditionally runs `MathJax.typesetPromise()` for nodes with `data-mathjax`.
    - Debounced tab highlight sync with URL via `htmx:afterSettle`.
  - Session storage keys: `chaptersLoaded` and `testsLoaded` gate redundant fetches.

- Chapters templates: `chapters.html` defines a content block with a table. A custom `chaptersLoader` HTMX extension suppresses premature/duplicate `/api/chapters` requests while dropdowns initialize/restore.

- Tests templates: `tests.html` manages client-side sorting state via `sessionStorage` and augments `/api/tests` requests with `sortColumn` and `sortOrder` in `htmx:configRequest`.

- Problem editor: Rich-text with math & images
  - `editor.html` includes toolbar and containers.
  - `static/js/editor.js`: WYSIWYG controls (bold/italic/underline, font, size, list, paragraph types, line-height), table inserter, link inserter, HR, fullscreen toggle, preview/code-view toggles.
  - `static/js/editor-math.js`: MathLive inline input to LaTeX, rendered by MathJax in the preview.
  - `static/js/editor-image.js`: Inline image insertion via `FileReader`.


### PDF Generation

`TestsHandler.DownloadPdf` renders question papers and answer sheets using chromedp:

- Renders template (`question_paper.html` or `answer_sheet.html`) to an HTML string.
- Inlines Tailwind CSS by reading `web/static/css/output.css` and injecting a `<style>` tag.
- Navigates a headless Chrome page to `data:text/html,<html>` and ensures MathJax completes typesetting.
- Prints to A4 PDF with custom header/footer and margins; streams the PDF as the HTTP response.


### Testing

- **Go unit tests**
  - `cmd/main_test.go`: Route registration and env loading flow using a mock Mux and testify.
  - `config/env_test.go`: Success/failure paths for `.env` loading and `GetEnv()` semantics.

- **Playwright E2E tests** (`tests/`)
  - `test-home.spec.ts`: Verifies nav visibility, dropdown population, `onLoaded` events, and state restoration via `sessionStorage` with delayed HTMX triggers.
  - `test-chapters.spec.ts`: Verifies chapters table headers, add/hide form, delete flow (dialog confirm/deny), edit flow (API calls + payload verification).
  - `test-add_chapter.spec.ts`: Verifies creation flow and DOM insertion/reset.
  - `mock.ts`: HTTP route interceptors for dropdowns, chapters list, and edit/update flows (merging base `home.html` with content fragments).
  - `utils.ts`: Shared constants (base URL and dropdown fixtures).

Run tests:

```bash
# Backend unit tests
go test ./...

# Frontend E2E tests
npm install
npm run build:css
# Ensure the Go server is running locally on port 8080 before this step
npx playwright test
```


### Build & Run (Development)

```bash
# 1) Dependencies for frontend tooling (Tailwind, Playwright)
npm install

# 2) Build Tailwind CSS
npm run build:css

# 3) Configure environment (example)
cat > .env << 'EOF'
DB_SERVICE_ENDPOINT=https://your-db-service.example.com/
DB_SERVICE_TOKEN=yourBearerToken
EOF

# 4) Run the Go server
go run ./cmd

# Visit http://localhost:8080
```


### Deployment & Infrastructure

The project includes Terraform configuration for AWS deployment with GitHub Actions CI/CD:

#### Infrastructure Components
- **EC2 Instance**: ARM-based Amazon Linux 2023 (`t4g.small` for staging, `t4g.medium` for prod)
- **Load Balancer**: NGINX reverse proxy with SSL termination
- **SSL Certificate**: Let's Encrypt certificate via Certbot (auto-renewal)
- **DNS**: Cloudflare A record pointing to Elastic IP
- **Storage**: S3 + DynamoDB backend for Terraform state management

#### Deployment Files
- `terraform/`: Complete Terraform configuration
  - `main.tf`: Core infrastructure (EC2, security groups, EIP, Cloudflare DNS)
  - `user-data.sh`: Instance initialization script (idempotent, runs on every boot)
  - `variables.tf`: Input variables for customization
  - `backend.tf`: S3/DynamoDB state backend configuration
  - `bootstrap.sh`: One-time setup script for Terraform backend resources
- `.github/workflows/deploy-staging.yml`: GitHub Actions workflow for automated deployment
- `.github/workflows/deploy-prod.yml`: GitHub Actions workflow for production deployment

#### Environment Setup
The deployment script (`user-data.sh`) creates a `.env` file in `/opt/nex-gen-cms/` with:
- `DB_SERVICE_ENDPOINT`: Database service URL
- `DB_SERVICE_TOKEN`: Authentication token

The application runs as a systemd service (`nexgencms`) under a dedicated `app` user, with NGINX proxying requests from port 80/443 to the Go server on port 8080.

#### Deployment Flow
1. **Bootstrap** (one-time): Run `terraform/bootstrap.sh` to create S3 bucket and DynamoDB table
2. **Configure**: Set Terraform variables in `terraform.tfvars`
3. **Deploy**:
   - Staging: GitHub Actions automatically deploys on push to `main`.
   - Production: GitHub Actions deploys on push to `release` (separate backend key `nex-gen-cms/prod.tfstate`).
4. **Updates**: Instance automatically pulls latest code on reboot; GitHub Actions can trigger reboots for code-only changes. For local ops, `terraform apply` will re-render user data and rebuild the app as needed.

See `terraform/README.md` for detailed setup instructions.


### Conventions & Utilities

- Sorting:
  - Chapters/topics sorting state is currently server-managed via `dto.SortState` and toggled by query params.
  - Tests list sorting is client-managed (sessionStorage + HTMX request augmentation) and visual icons update on re-render.
- Template helpers (commonly passed via `template.FuncMap`):
  - `slice`, `add`, `seq`, `joinInt16`, `dict`, `toJson`, `capitalize`, `getName`/`getParentName`/`getParentId`, `problemDisplaySubtype`.
- Status/archival: Archived resources are filtered out in UI lists.
- Middleware: Routes like `/edit-chapter`, `/edit-topic`, `/tests/add-test-dialog` require an HTMX header; otherwise a redirect to `/chapters` occurs.


### Extending the System

To add a new resource type (e.g., “Passage”):

1. Model: Create `internal/models/passage.go` with JSON tags matching the remote API.
2. Service: No new service code required—use `services.NewService[models.Passage]`.
3. DI: Register a `Service[Passage]` in `di/app_component.go` and wire a new handler.
4. Handler: Create `internal/handlers/passage_handler.go` exposing desired routes (list/get/create/update/delete), calling the generic service.
5. Routes: Register paths in `cmd/main.go`’s `setup()`.
6. Templates: Add HTML templates to `web/html/` and wire rendering via `local_repo.ExecuteTemplate(s)`.
7. Frontend: If needed, add JS under `web/static/js/`. Update Tailwind build if new classes are introduced.
8. Tests: Add Playwright specs under `tests/` and Go unit tests as appropriate.


### Known Notes & Pitfalls

- The HTML folder path logic checks for a Windows-style `\\cmd` suffix during tests; macOS/Linux tests run with default `web/html`.
- Some endpoint strings include a leading slash (e.g., `"/skill"`), others do not (e.g., `"topic"`). The remote repo concatenates them to `DB_SERVICE_ENDPOINT`; ensure no accidental double slashes or missing slashes at deployment.
- Chapters sorting is server-stateful, whereas Tests sorting is client-stateful; future refactor could unify this behavior.
- `DownloadPdf` inlines CSS because headless Chrome cannot resolve relative stylesheet links from an in-memory document.


### Glossary

- **HTMX**: Library enabling partial HTML over the wire with declarative attributes.
- **chromedp**: Go bindings for Chrome DevTools Protocol, used for HTML→PDF rendering.
- **go-cache**: In-memory key/value store with TTL; used for caching lists across requests.
- **Service[T]**: Generic data-access layer wrapping cache + remote API for a model type.


### Quick Links (Files to Start From)

- Entry: `cmd/main.go` → routes & bootstrap
- DI: `di/app_component.go` → how services/handlers are wired
- Services: `internal/services/service.go` → generic data layer
- Remote API: `internal/repositories/remote/api_repository.go`
- Templates: `web/html/home.html` (base), plus content templates under `web/html/`
- Tests: `tests/` (Playwright), `cmd/main_test.go`, `config/env_test.go`
- Deployment: `terraform/README.md` → infrastructure setup, `terraform/main.tf` → AWS resources, `.github/workflows/deploy-staging.yml` → CI/CD pipeline


### Maintainers' Checklist for Changes

- Add/modify routes in `cmd/main.go`.
- Wire new services/handlers in `di/app_component.go`.
- Keep template names synchronized with route expectations (e.g., `chapter_row.html`).
- Use `Service[T]` for remote access and cache consistency across CRUD.
- Add Playwright coverage for new user flows; keep Tailwind build updated if classes change.
- For deployment changes: Update Terraform variables in `terraform.tfvars`, test infrastructure changes in staging before production.
- Environment variables: Add new variables to both `.env` (local development) and `terraform/user-data.sh` (deployment).


This document should evolve alongside the codebase; please update sections when introducing new resources, routes, or architectural changes.


