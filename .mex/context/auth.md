---
name: auth
description: Login, sessions, roles, and the Postgres user store. Load when touching authentication, authorization, route guards, or cms_user_permission.
triggers:
  - "auth"
  - "login"
  - "logout"
  - "session"
  - "role"
  - "permission"
  - "oauth"
  - "google"
  - "admin"
  - "cms_user_permission"
edges:
  - target: context/architecture.md
    condition: when you need the overall request flow the auth middleware sits in front of
  - target: context/decisions.md
    condition: when you need why OAuth + roles replaced basic auth
  - target: patterns/protect-route.md
    condition: when adding or changing the auth/role guard on a route
  - target: context/setup.md
    condition: when configuring OAuth env vars or the DEV_LOGIN_EMAIL bypass
last_updated: 2026-06-26
---

# Auth

Login is **Google OIDC**; authorization is **role-based** (`viewer` < `editor` < `admin`). User
identity and roles live in the Postgres `cms_user_permission` table — the only direct DB usage in the app.

## Components

- **`internal/auth/oauth.go`** — `GoogleAuth` (OIDC provider + verifier + oauth2 config). Builds the
  consent URL with a `state` cookie and `hd=avantifellows.org`; `Exchange` verifies state, exchanges the
  code, verifies the ID token, and **re-checks** `email_verified` and the `hd` (hosted-domain) claim
  server-side. `NewGoogleAuth` returns `(nil, nil)` when OAuth env vars are unset (login disabled, not an error).
- **`internal/auth/session.go`** — issues/reads the session. Two cookies:
  - `cms_session` — **HttpOnly** signed JWT (`SessionClaims{UserID, Email, Role}`, HS256 via `SESSION_SECRET`,
    12h expiry). This is the source of truth for authorization.
  - `cms_role` — **non-HttpOnly** mirror of the role, for JS to gate UI (e.g. the Admin nav link).
    **Cosmetic only** — never trust it server-side.
- **`internal/auth/roles.go`** — role constants + `AtLeast(have, need)` rank comparison + `ValidRole`.
- **`internal/auth/context.go`** — `WithSession` / `FromContext` carry `*SessionClaims` on the request context.
- **`internal/middleware/auth.go`** — `RequireLogin` (wraps the whole mux), `RequireRole` / `RequireRoleFunc`.
- **`internal/repositories/db/cms_user_repo.go`** — `CmsUserRepo`: `GetByEmail` (case-insensitive),
  `List`, `Create`, `SetActive` (soft delete/restore), `UpdateRole`, `UpdateLastLogin`. Parameterized SQL only.
- **`internal/handlers/login_handler.go`** — `Login`, `StartGoogleAuth`, `GoogleCallback`, `DevLogin`, `Logout`.
- **`internal/handlers/admin_users_handler.go`** — admin-only user management (`/admin/users*`).

## Flow

1. `cmd/main.go` wraps the mux in `middleware.RequireLogin(mux, exceptions...)`. Exceptions (no session
   required): `/login`, `/favicon.ico`, `/web/static/css/output.css`, `/auth/google/start`,
   `/auth/google/callback`, `/dev-login`.
2. `RequireLogin` reads `cms_session`. Missing/invalid → redirect to `/login` (or `HX-Redirect: /login`
   with 401 for HTMX). Valid → attach claims to context, continue.
3. **Login:** `/auth/google/start` → Google consent → `/auth/google/callback`. The callback verifies the
   token, looks up the email in `cms_user_permission`, rejects unknown (`not authorized`) or inactive
   (`access revoked`) users, then `IssueSession` and redirect to `/home`.
4. **Authorization:** mutating routes are wrapped in `cmd/main.go` with `editor(...)` or `admin(...)`
   (aliases for `RequireRoleFunc(RoleEditor|RoleAdmin, ...)`). `RequireRole` checks `AtLeast(claims.Role, need)`;
   too-low → 403 (`HX-Reswap: none` + 403 for HTMX).
5. **Dev bypass:** `POST /dev-login` signs in as `DEV_LOGIN_EMAIL` (must exist & be active). Only useful
   when that env var is set; intended for local dev and Playwright. Never set it in production.

## Gotchas

- **Real authorization is server-side only.** The `cms_role` cookie and any JS UI-gating are convenience;
  every protected action must be guarded by `editor(...)`/`admin(...)` in `cmd/main.go`.
- **`Secure` cookies are gated by `APP_ENV=production`.** Locally over HTTP a `Secure` cookie would be
  silently dropped, so it's off unless `APP_ENV=production`.
- **No `SESSION_SECRET` → no auth.** `ReadSession` returns nil and `IssueSession` errors, so everything
  redirects to `/login`. Set it.
- **Sign-in requires a db row.** OAuth success is not enough — the email must exist and be `is_active` in
  `cms_user_permission`. The first admin (`pritam@avantifellows.org`) is seeded by the db-service migration.
- **Hosted-domain restriction is enforced twice:** `hd` is passed to Google's chooser AND re-verified on
  the returned ID token. Don't rely on the request param alone.
- **OAuth being unconfigured is not fatal.** `NewGoogleAuth` returns `(nil, nil)`; the login page just
  hides Google sign-in (`GoogleConfigured=false`). Auth still works via `DEV_LOGIN_EMAIL` locally.
- **New users are added via `/admin/users`** (admin role), not by editing the DB by hand in normal flow.
