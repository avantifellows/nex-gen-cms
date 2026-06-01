# nex-gen-cms

## How to Run locally?

### Prerequisites:
1. **Install Dbservice:** Install and run it locally following the steps mentioned over [here](https://github.com/avantifellows/db-service/blob/main/docs/INSTALLATION.md).
2. **Install Go (>= 1.25):** Check with `go version`. Install from [golang.org](https://go.dev/dl/) if missing.
3. **Postgres access:** the CMS connects directly to the Postgres database that hosts `cms_user_permission` (same DB as db-service). For staging you can use the staging RDS credentials; for local you can point at your local db-service Postgres.
4. **Google OAuth client:** Create (or reuse) a client in [Google Cloud Console](https://console.cloud.google.com/) and add `http://localhost:8080/auth/google/callback` as an authorized redirect URI.

### Getting started:
To run the CMS locally, follow these steps:
1. Clone the repository to your local machine.
   
   ```
   git clone https://github.com/avantifellows/nex-gen-cms.git
   ```
2. Create a `.env` file at the project root.
3. Add the following keys:

   ```
   # db-service API (for content: chapters/topics/tests/etc.)
   DB_SERVICE_ENDPOINT=http://localhost:4000/api/
   DB_SERVICE_TOKEN=<BEARER_TOKEN used in your local db-service .env>

   # Postgres direct connection (for cms_user_permission auth lookups)
   DATABASE_URL=postgres://postgres:postgres@localhost:5432/dbservice_dev?sslmode=disable

   # Google OAuth (login)
   GOOGLE_CLIENT_ID=<from GCP Console>
   GOOGLE_CLIENT_SECRET=<from GCP Console>
   OAUTH_REDIRECT_URL=http://localhost:8080/auth/google/callback

   # Session cookie signing key (any long random string; openssl rand -base64 48)
   SESSION_SECRET=<random>

   # Optional: when set, the login page exposes a "Sign in as <email>" button that
   # bypasses Google OAuth. Local dev convenience only — do NOT set in production.
   DEV_LOGIN_EMAIL=

   # Set to "production" to require HTTPS cookies. Leave unset locally.
   APP_ENV=
   ```

   The user named in `DATABASE_URL` must be able to SELECT/INSERT/UPDATE `cms_user_permission`. First admin (`pritam@avantifellows.org`) is already seeded by the db-service migration.
4. Navigate to the project directory.
 
   ```
   cd <path to local project root folder>
   ```
5. Run this command to download all necessary dependencies for the project.

   ```
   go mod tidy
   ```
6. Build the CSS bundle. `web/static/css/output.css` is generated (not committed), so a
   fresh clone has no styles until you build it once:

   ```
   npm install        # first time only
   npm run build:css
   ```
7. Run the application by running:

   ```
   go run ./cmd
   ```

   Or do steps 6–7 in one go with `make run` (builds CSS, then starts the server). While
   editing templates/styles, run `make css-watch` (or `npm run dev:css`) in a second terminal
   to rebuild CSS automatically.
8. Open your browser and go to http://localhost:8080 to view the application.

### Temporary Branches to use until it gets merged to main:
1. **nex-gen-cms:** [feat/tests](https://github.com/avantifellows/nex-gen-cms/tree/feat/tests)
2. **db-service:** [adding-language-table](https://github.com/avantifellows/db-service/tree/adding-language-table)

### UI Style Guide

The CMS follows the **Warm Professional** design language shared with the rest of the AF product family (af_lms, hr-feedback.avantifellows.org). Brand tokens, typography rules, component conventions and HTML snippets are documented in [`docs/UI-Style-Guide.md`](docs/UI-Style-Guide.md). Read that before adding new screens or editing existing ones so the look stays consistent.

### Tailwind Setup:

`web/static/css/output.css` is a **generated** artifact and is **not committed** (it's
`.gitignore`d). It is built from `input.css` by Tailwind — at deploy time (`terraform/user-data.sh`),
in CI (`playwright.yml`), and locally. This keeps the giant generated file out of diffs and merges.

#### ✅ Running the App

Build the CSS once after cloning (and whenever you pull template/style changes):

```bash
npm install        # first time only
npm run build:css
```

`make run` does this for you (builds CSS, then runs the server).

#### 🛠️ Editing Styles

While changing `input.css` or template classes, run a watcher so CSS rebuilds on save:

```bash
npm run dev:css    # or: make css-watch
```
