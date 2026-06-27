---
name: protect-route
description: Add or change authentication / role authorization on a route. Use whenever a new route mutates data or must be restricted by role.
triggers:
  - "protect route"
  - "require login"
  - "require role"
  - "editor only"
  - "admin only"
  - "authorize"
  - "permission on route"
edges:
  - target: context/auth.md
    condition: for how sessions, roles, cookies, and the middleware chain work
  - target: patterns/add-content-resource.md
    condition: when the route being protected is a new resource/handler route
last_updated: 2026-06-26
---

# Protect a Route

## Context

Every route is already behind `middleware.RequireLogin` (wrapped around the whole mux in `cmd/main.go`,
minus the exceptions list). So "protecting" a route means adding a **role** requirement on top, or adding
it to the login exceptions. Read `context/auth.md` first. Role hierarchy: `viewer` < `editor` < `admin`.

## Steps
1. Decide the minimum role. Read-only public-to-logged-in routes need no extra guard. Anything that
   mutates content should be `editor(...)`; user management is `admin(...)`.
2. In `cmd/main.go` `setup()`, wrap the handler:
   - `muxHandler.HandleFunc("/create-thing", editor(thingHandler.Create))`
   - `muxHandler.HandleFunc("/admin/things", admin(thingHandler.List))`
   - HTMX-only **and** role-gated → compose: `middleware.RequireHTMX(middleware.RequireRole(auth.RoleEditor, http.HandlerFunc(h.Edit)))`.
3. If a route must skip login entirely (rare), add its exact path to the `exceptions` slice in `main.go`.
4. To read the current user inside a handler: `claims := auth.FromContext(r.Context())` (set by `RequireLogin`).

## Gotchas
- `editor` and `admin` are local aliases in `cmd/main.go` for `middleware.RequireRoleFunc(auth.Role…, h)`.
  Use them — don't re-implement role checks in the handler body.
- **Never gate on the `cms_role` cookie or JS server-side** — it's a non-HttpOnly mirror for UI only.
  Authorization is the signed `cms_session` JWT, enforced by the middleware.
- HTMX requests get `HX-Redirect: /login` (401) when unauthenticated and `HX-Reswap: none` (403) when the
  role is too low — so a forbidden action silently no-ops on the client instead of swapping garbage in.
- Adding a path to `exceptions` makes it fully public — only do it for genuinely unauthenticated endpoints
  (login, OAuth callback, public CSS, favicon, dev-login).
- The `RequireRole` wrappers re-read the session if it isn't already on the context, so they're safe to use
  even outside the global `RequireLogin` chain.

## Verify
- [ ] Unauthenticated request to the route redirects to `/login` (or `HX-Redirect` for HTMX).
- [ ] A `viewer` hitting an `editor`/`admin` route gets 403; the right role succeeds.
- [ ] `go build ./...` passes; the route still appears in `cmd/main_test.go`'s expectations if it asserts routes.

## Debug
- 302/redirect to `/login` when you expected access → session missing/invalid (`SESSION_SECRET` unset, or
  cookie not sent). Check `auth.ReadSession`.
- 403 for a user who should pass → check their `role` in `cms_user_permission` and `AtLeast(have, need)`.
- Route unexpectedly public → confirm it isn't in the `exceptions` list and is actually wrapped.

## Update Scaffold
- [ ] If a new role-gating convention emerged, update `context/auth.md`.
- [ ] Update `.mex/ROUTER.md` "Current Project State" if access rules materially changed.
