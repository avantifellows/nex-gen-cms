# nex-gen-cms — Agent Guide

Go server-rendered CMS (`net/http` + HTMX + Tailwind) for Avanti Fellows content, backed by the
db-service API with Google-OAuth role-based access.

## Start here

This repo's context lives in the `.mex/` scaffold. Before doing anything else:

1. Read **`.mex/ROUTER.md`** — current project state, the routing table, and the per-task behavioural
   contract (CONTEXT → BUILD → VERIFY → DEBUG → GROW).
2. Read **`.mex/AGENTS.md`** — project identity, non-negotiables, and commands.

From there, `.mex/ROUTER.md` routes you to the right file: architecture, stack, conventions, decisions,
setup, auth, and deployment context under `.mex/context/`, and task runbooks under `.mex/patterns/`
(start with `.mex/patterns/INDEX.md`).

Keep that scaffold as the single source of truth — don't duplicate it here. When reality changes, update
`.mex/` (see the **GROW** step in `.mex/ROUTER.md`), not this file.

## Project references not in the scaffold

- **UI:** read `docs/UI-Style-Guide.md` before adding or changing screens (Warm Professional design
  language; use Tailwind token classes from `input.css` like `bg-accent`/`text-ink`, never hardcoded hex).
