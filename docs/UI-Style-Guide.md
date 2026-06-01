# Avanti Fellows CMS — UI Style Guide

> **Purpose:** Reference for the nex-gen-cms design system — colors, typography, spacing, the shared utility classes, and HTML snippets that match the conventions used across our Go html/template files.

> **Design Language:** Warm Professional — Avanti Brand Maroon (matching `hr-feedback.avantifellows.org`)
>
> **Font:** Inter (loaded via Google Fonts in `web/html/home.html` and `web/html/login.html`)
>
> **Icons:** Font Awesome 6 (loaded from `cdnjs.cloudflare.com` in the base template)
>
> **Shared classes:** Defined in [`input.css`](../input.css) under `@layer components`. Tokens live in [`tailwind.config.js`](../tailwind.config.js) under `theme.extend`. Use Tailwind class names (e.g. `bg-accent`, `text-ink-muted`), never raw hex values.
>
> **Sibling style guide:** This guide was adapted from [`af_lms/docs/UI-Style-Guide.md`](https://github.com/avantifellows/af_lms/blob/main/docs/UI-Style-Guide.md). Both apps share the same brand palette and design principles so the CMS, the LMS, and `hr-feedback.avantifellows.org` look like one product.

---

## Table of Contents

1. [Design Philosophy](#1-design-philosophy)
2. [Color System](#2-color-system)
3. [Typography](#3-typography)
4. [Spacing & Sizing](#4-spacing--sizing)
5. [Shared Utility Classes](#5-shared-utility-classes)
6. [Page Layouts](#6-page-layouts)
7. [Headers & Navigation](#7-headers--navigation)
8. [Cards & Panels](#8-cards--panels)
9. [Buttons](#9-buttons)
10. [Form Inputs](#10-form-inputs)
11. [Tables](#11-tables)
12. [Status Badges & Pills](#12-status-badges--pills)
13. [Modals & Dialogs](#13-modals--dialogs)
14. [Problem Editor & Specialised Components](#14-problem-editor--specialised-components)
15. [HTMX Loading States](#15-htmx-loading-states)
16. [Responsive & Mobile](#16-responsive--mobile)
17. [Rebuilding the CSS](#17-rebuilding-the-css)

---

## 1. Design Philosophy

The CMS UI follows the **Warm Professional** principles shared with the rest of the AF ecosystem:

- **Rounded corners** (`rounded-lg`) on cards, buttons, inputs, form sections — approachable, modern.
- **Soft shadows** (`shadow-card`, `shadow-xl`) for depth and card separation.
- **Subtle borders** define hierarchy. `border-b border-border` for headers, `border-b-2 border-border-accent` for colored section dividers.
- **Uppercase headings + tracking** (`uppercase tracking-wide` or `tracking-tight`) for a structured feel.
- **Monospace numbers.** Numeric data (codes, counts, durations, dates) uses `font-mono` — this is what makes the data-rich screens feel like an admin tool, not a marketing page.
- **48px minimum touch targets** on tab items / radio labels, **44px** on buttons and inputs.
- **Hover + active states on every interactive element** — buttons darken, rows tint, links underline.
- **Maroon accent** (`#ad2f2f`) is the only interactive color — no hardcoded `blue-500`/`text-blue-600` anywhere.
- **Warm beige page** (`#f5efe8`) with **cream cards** (`#fffaf5`) — never pure white surfaces.
- **AF logo** in the header is served from the CDN (`cdn.avantifellows.org/af_logos/avanti_logo_black_text.webp`).
- **Brand color variety** — coral, gold, amber, blue used sparingly for badges, section dividers, status accents so the screen doesn't feel monotone.

---

## 2. Color System

All colors are defined as Tailwind tokens in `tailwind.config.js`. Use the class names (`bg-accent`, `text-ink`, `border-border-accent`), never raw hex.

### Brand palette

| Color | Hex | Role |
|-------|-----|------|
| AF Maroon | `#ad2f2f` | **Primary accent** — buttons, links, active states |
| AF Dark Brown | `#261410` | Primary text |
| AF Beige | `#f5efe8` | Page background |
| AF Cream | `#fffaf5` | Card backgrounds |
| AF Muted Brown | `#685851` | Secondary/muted text |
| Brand Coral | `#E96D57` | Stat accents, variety color |
| Brand Gold | `#FFD063` | Warnings, difficulty stars |
| Brand Amber | `#FFB763` | Section dividers, subject headers |
| Brand Blue | `#9AC4FA` | Info, concept chips |
| Brand Salmon | `#FF9683` | Decorative (available, used sparingly) |
| Brand Orange | `#D77C11` | Available in palette, currently unused |

### Design tokens

#### Accent

| Class | Value | Usage |
|-------|-------|-------|
| `bg-accent` / `text-accent` / `border-accent` | `#ad2f2f` | Primary buttons, active nav tab, links, focus rings |
| `bg-accent-hover` / `text-accent-hover` | `#8a2525` | Hover states for accent elements |
| `text-text-on-accent` | `#ffffff` | White text on accent backgrounds |

#### Backgrounds

| Class | Value | Usage |
|-------|-------|-------|
| `bg-bg` | `#f5efe8` | Page background (warm beige) — applied to `<body>` |
| `bg-bg-card` | `#fffaf5` | Card / panel background (cream) |
| `bg-bg-card-alt` | `#f3ece5` | Table headers, disabled inputs, alt-row striping |
| `bg-bg-input` | `#ffffff` | Input field background |
| `bg-bg-hover` | `rgba(173, 47, 47, 0.06)` | Row / item hover background |

#### Text

| Class | Value | Usage |
|-------|-------|-------|
| `text-ink` | `#261410` | Headings, main content (dark brown) |
| `text-ink-secondary` | `#685851` | Supporting text (taupe) |
| `text-ink-muted` | `#685851` | Labels, captions, metadata, table headers |

#### Borders

| Class | Value | Usage |
|-------|-------|-------|
| `border-border` (also bare `border`) | `rgba(38, 20, 16, 0.15)` | Default borders (translucent brown) |
| `border-border-accent` | `#ad2f2f` | Section dividers under titles, table-head underline |

#### Status

| Class | Value | Usage |
|-------|-------|-------|
| `bg-danger` / `text-danger` | `#ad2f2f` | Errors, destructive actions (same hex as accent) |
| `bg-danger-bg` | `rgba(173, 47, 47, 0.08)` | Light danger background — alerts, deactivate hover |
| `text-success` / `bg-success` | `#1e6b4b` | Success states (forest green) |
| `bg-success-bg` | `rgba(30, 107, 75, 0.12)` | Light success background — alerts, active pills |
| `text-warning` | `#8c5a1d` | Warning text (brown-gold) |
| `bg-warning-bg` | `rgba(140, 90, 29, 0.08)` | Light warning background |
| `border-warning-border` | `#8c5a1d` | Warning border |
| `text-info` | `#9AC4FA` | Info states |
| `bg-info-bg` | `rgba(154, 196, 250, 0.15)` | Light info background |

#### Brand color range (variety)

Use these to break the monotone — section dividers, subject headers, badge tints, difficulty stars. Pick by hierarchy / semantic feel, not at random.

| Class | Value | Used for |
|-------|-------|----------|
| `text-brand-coral` / `border-brand-coral` | `#E96D57` | Decorative accents |
| `text-brand-gold` / `border-brand-gold` | `#FFD063` | Difficulty stars, warning highlights |
| `text-brand-amber` / `border-brand-amber` | `#FFB763` | Subject headers on test detail (`border-l-4 border-brand-amber`) |
| `text-brand-blue` / `border-brand-blue` | `#9AC4FA` | Concept chips, info accents |
| `bg-brand-coral-bg` | `rgba(233, 109, 87, 0.10)` | Light coral tint |
| `bg-brand-gold-bg` | `rgba(255, 208, 99, 0.15)` | Light gold tint |
| `bg-brand-amber-bg` | `rgba(255, 183, 99, 0.12)` | Light amber tint |
| `bg-brand-blue-bg` | `rgba(154, 196, 250, 0.15)` | Light blue tint — concept chip background |

### Semantic category colors

We don't currently render category-coded badges (problem subtype, test type, role) with distinct hues — they all use the muted/accent badge styles below. If you add a new screen where a single field carries 3-5 mutually-exclusive categories that the user needs to scan visually (e.g. role = viewer/editor/admin), reach for the brand color range above rather than introducing new hex values.

---

## 3. Typography

### Font

**Inter**, loaded via Google Fonts in the base template:

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap"
      rel="stylesheet">
```

Inter is set as the default `font-sans` in the Tailwind config, so any element inherits it unless `font-mono` is applied.

### Scale

| Element | Classes | Example |
|---------|---------|---------|
| Page title | `page-title` (`text-xl sm:text-2xl font-bold uppercase tracking-tight`) | CMS USERS, ADD NEW PROBLEM |
| Section heading | `section-title` (`text-lg font-bold uppercase tracking-wide border-b-2 border-border-accent pb-2 mb-4`) | STATEMENT / TEXT, OPTIONS |
| Card title | `text-lg font-bold uppercase tracking-wide text-ink` | (used inline) |
| Form label | `form-label` (`text-xs font-bold uppercase tracking-wide text-ink-muted mb-1`) | CODE, NAME, DIFFICULTY |
| Body text | `text-sm text-ink` | Default content |
| Supporting text | `text-sm text-ink-muted` | Descriptions, table captions |
| Metadata | `text-xs text-ink-muted` | Hints, "loading…" placeholders |
| Codes / numbers | `font-mono` | `MTH-11-JEE-17`, `P002290`, `180 min` |

### Rules

- **Uppercase + tracking** for headings, labels, badges, button text — structured, professional feel.
- **`font-mono`** for every code, count, duration, date and email — this is what makes the screens feel data-rich.
- **Never ALL CAPS on body text** — only headings, labels, buttons, badges.

---

## 4. Spacing & Sizing

### Touch targets

| Element | Minimum height |
|---------|----------------|
| Buttons (default `.btn`) | `min-h-[44px]` |
| Buttons (`.btn-lg`) | `min-h-[48px]` |
| Buttons (`.btn-sm`) | `min-h-[36px]` |
| Inputs / Selects (`.form-input`, `.form-select`) | `min-h-[44px]` |
| Compact selects in nav (`.form-select-compact`) | `h-10` |
| Sub-tabs | `min-h-[40px]` |

### Border radius scale

| Element | Radius |
|---------|--------|
| Buttons | `rounded-lg` |
| Cards | `rounded-lg` (default), `rounded-xl` (modal) |
| Inputs / Selects | `rounded-lg` |
| Badges | `rounded-full` |
| Modal shells | `rounded-xl` |

### Shadow scale

| Elevation | Class | Usage |
|-----------|-------|-------|
| Subtle (default for cards) | `shadow-card` | Cards, sticky bars, primary panels |
| Prominent | `shadow-xl` | Modals |

`shadow-card` is a custom token defined in the Tailwind config; it's a softer pair of brown-tinted shadows than the default Tailwind `shadow-md`.

---

## 5. Shared Utility Classes

Defined in [`input.css`](../input.css). Apply by class name in templates — no imports, no JS.

### Buttons

```html
<button class="btn-primary">Primary</button>
<button class="btn-secondary">Cancel</button>
<button class="btn-ghost">Link-style</button>
<button class="btn-danger">Delete</button>
<button class="btn-danger-ghost">Remove</button>

<button class="btn-primary btn-sm">Small</button>
<button class="btn-primary">Medium (default, 44px)</button>
<button class="btn-primary btn-lg">Large (48px)</button>
```

All variants include `rounded-lg`, `uppercase tracking-wide`, `transition-colors`, `focus-visible:ring-2`, and `disabled:opacity-50`.

### Card

```html
<div class="card card-pad">Standard (cream, shadow-card, p-6)</div>
<div class="card card-pad-sm">Tight (p-4)</div>
<div class="card shadow-xl rounded-xl p-6">Modal-level</div>
```

### Inputs

```html
<input class="form-input" placeholder="Search…">
<select class="form-select"><option>All</option></select>
<select class="form-select-compact">…</select>  <!-- For nav bar -->
<label class="form-label" for="code">School Code</label>
```

### Badges

Use these for status-style pills (Active, Completed, Pending):

```html
<span class="badge-success">active</span>
<span class="badge-warning">in progress</span>
<span class="badge-danger">archived</span>
<span class="badge-muted">inactive</span>
<span class="badge-info">info</span>
<span class="badge-accent">primary</span>
```

For a "+ Add new X" inline link (used at the bottom of list tables):

```html
<h4 class="add-new-link" onclick="…">
  <a href="#">+ Add New Chapter</a>
</h4>
```

### Tables

```html
<div class="card overflow-x-auto">
  <table class="app-table">
    <thead>
      <tr>
        <th>Code</th>
        <th>Name</th>
        <th class="text-center">Actions</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td class="font-mono">MTH-11-JEE-17</td>
        <td>Fundamentals of Mathematics</td>
        <td class="text-center"><!-- action buttons --></td>
      </tr>
    </tbody>
  </table>
</div>
```

`.app-table` styles `thead`, `th`, `td`, and `tbody tr:hover` automatically — you only need to add column alignment helpers where needed.

---

## 6. Page Layouts

### Base template

[`web/html/home.html`](../web/html/home.html) provides the standard chrome:

```html
<body class="bg-bg text-ink min-h-screen">
  <header class="bg-bg-card border-b border-border shadow-card">
    <div class="max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-8">
      <!-- logo + nav + dropdowns + sign out -->
    </div>
  </header>
  <main class="max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-8 py-6">
    {{ block "content" . }}…{{ end }}
  </main>
</body>
```

### Detail pages

Wrap the content in a `.card.card-pad` so the back button + title sit on a cream surface against the beige page:

```html
{{ define "content" }}
<div class="card card-pad mb-4">
  <div class="flex items-center mb-4">
    <button class="btn-ghost btn-sm" onclick="window.history.back()">
      <i class="fa-solid fa-chevron-left"></i> Back
    </button>
    <h3 class="ml-4 text-lg font-bold uppercase tracking-wide text-ink">
      {{ getName .ChapterPtr "en" }}
    </h3>
  </div>
  <!-- … -->
</div>
{{ end }}
```

---

## 7. Headers & Navigation

### Top nav (in the base template)

```html
<header class="bg-bg-card border-b border-border shadow-card">
  <div class="max-w-[1600px] mx-auto px-4 sm:px-6 lg:px-8">
    <nav>
      <div class="nav nav-tabs flex flex-wrap items-center gap-y-2 border-b-0">
        <a href="/chapters" class="shrink-0 mr-4 sm:mr-6 flex items-center">
          <img src="https://cdn.avantifellows.org/af_logos/avanti_logo_black_text.webp"
               alt="Avanti Fellows" class="h-8 sm:h-9">
        </a>

        <button class="nav-link" hx-get="/chapters" …>Chapters</button>
        <button class="nav-link" hx-get="/tests" …>Tests</button>
        <button class="nav-link" hx-get="/problems" …>Problems</button>

        <select class="form-select-compact ms-4 sm:ms-6 w-auto" id="curriculum-dropdown" …>
          …
        </select>

        <button hx-post="/logout" class="btn-secondary btn-sm ms-auto my-2">
          Sign out
        </button>
      </div>
    </nav>
  </div>
</header>
```

`.nav-link` is the state-machine class — JS toggles `.active` on the matching tab, which applies the maroon underline (`border-b-2 border-accent`).

**Mobile notes:** Use `flex-wrap` + `gap-y-2` so items wrap instead of colliding. Tighten gaps to `gap-3` on mobile, `sm:gap-6` on desktop. Add `shrink-0` on the logo to prevent compression.

### Sub-tabs (chapter / topic detail)

```html
<div class="flex flex-wrap gap-3 py-2 border-b border-border">
  <div class="sub-tab active" id="topics-sub-tab">Topics</div>
  <div class="sub-tab" id="resources-sub-tab">Resources</div>
</div>
```

JS toggles `.active` to switch panels.

### Section dividers

Use `.section-title` for the standard subsection header inside a card:

```html
<h2 class="section-title mt-6">Options</h2>
```

For a thicker divider over a major section:

```html
<div class="border-b-4 border-border-accent pb-4 mb-6">
  <h2 class="text-lg font-bold text-ink uppercase tracking-wide">Section Title</h2>
</div>
```

---

## 8. Cards & Panels

Use `.card` as the base, then pick padding:

```html
<div class="card card-pad">Standard (p-6)</div>
<div class="card card-pad-sm">Tight (p-4)</div>
<div class="card overflow-hidden">…</div>           <!-- For wrapping tables -->
<div class="card shadow-xl rounded-xl p-6">…</div>  <!-- Modal shell -->
```

---

## 9. Buttons

Use `.btn-*` everywhere a button or button-styled link is needed.

```html
<!-- Action bar at the bottom of a form -->
<div class="flex gap-3 justify-end pt-2">
  <button type="button" class="btn-secondary" onclick="window.history.back()">Cancel</button>
  <button type="submit" class="btn-primary btn-lg">Save Changes</button>
</div>

<!-- Inline "+ Add new" link -->
<a href="#" class="add-new-link" hx-get="…">+ Add concept from other topic</a>

<!-- Icon-only row action — uses the legacy .action-button class -->
<button class="action-button" hx-get="/edit-chapter?id={{.ID}}" title="Edit">
  <i class="fa-solid fa-pen"></i>
</button>
```

---

## 10. Form Inputs

```html
<div>
  <label class="form-label" for="code">Code</label>
  <input id="code" type="text" class="form-input font-mono"
         placeholder="e.g. MTH-11-JEE-17">
</div>

<div>
  <label class="form-label" for="type">Type</label>
  <select id="type" class="form-select">
    <option value="">Select type</option>
    <option value="major_test">Major Test</option>
  </select>
</div>
```

For dropdowns that sit in the global nav bar, prefer `.form-select-compact` (smaller height, lighter border) so they don't dominate.

For the difficulty radio chips used in [`add_problem.html`](../web/html/add_problem.html), use the `has-[:checked]` pattern:

```html
<label class="inline-flex items-center gap-2 px-4 py-2 rounded-lg border border-border bg-bg-card cursor-pointer
              hover:bg-bg-hover
              has-[:checked]:border-accent has-[:checked]:bg-accent has-[:checked]:text-text-on-accent
              transition-colors">
  <input type="radio" name="difficulty" value="easy" class="accent-accent">
  <span class="font-bold">1 — Easy</span>
</label>
```

---

## 11. Tables

```html
<div class="card overflow-x-auto">
  <table class="app-table">
    <thead>
      <tr>
        <th>
          <a href="#" class="hover:text-accent" hx-get="/api/tests?col=1">
            Code <i class="fas fa-sort"></i>
          </a>
        </th>
        <th>Name</th>
        <th class="text-center">Marks</th>
        <th class="text-center">Actions</th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td class="font-mono">SB-MT-07-16</td>
        <td>
          <a class="text-accent hover:text-accent-hover hover:underline font-medium"
             href="/test?id=…">Major Test 7</a>
        </td>
        <td class="text-center font-mono">360</td>
        <td class="text-center">…</td>
      </tr>
    </tbody>
  </table>
</div>
```

Don't add manual `bg-bg-card-alt` to `<thead>` or `border-b` to `<tr>` — `.app-table` handles those.

---

## 12. Status Badges & Pills

For binary-status pills (active/inactive, completed/in-progress), use `.badge-*`:

```html
<span class="badge-success">active</span>
<span class="badge-muted">inactive</span>
```

For semantic category badges where each color carries meaning (e.g. role types if added later), apply brand color classes directly rather than overloading the badge variants:

```html
<span class="badge bg-brand-blue-bg text-ink">teacher</span>
<span class="badge bg-brand-coral-bg text-accent-hover">admin</span>
```

---

## 13. Modals & Dialogs

The standard modal shell:

```html
<div class="fixed inset-0 bg-ink/40 flex justify-center items-center z-50 p-4">
  <div class="card shadow-xl rounded-xl w-full max-w-md p-6 max-h-[90vh] overflow-y-auto">
    <h2 class="page-title mb-6">Move Problems</h2>

    <!-- body -->
    <form class="space-y-4">
      <!-- … -->
    </form>

    <div class="flex gap-3 justify-end pt-4 border-t border-border">
      <button type="button" class="btn-secondary" onclick="…">Cancel</button>
      <button type="submit" class="btn-primary">Move</button>
    </div>
  </div>
</div>
```

**Scrollbar rule:** Only the inner modal body should scroll. Either give the inner `.card` `max-h-[90vh] overflow-y-auto` (as above, for short forms), or split it into a header / `flex-1 overflow-y-auto` body / footer when the body needs to scroll independently. Never nest two `overflow-y-auto` containers.

Backdrop is `bg-ink/40` (a translucent dark-brown wash) — not `bg-black/30`, because pure black against beige looks harsh.

---

## 14. Problem Editor & Specialised Components

A handful of components are unique to the CMS and live as their own classes in `input.css`:

| Class | Purpose | Used in |
|-------|---------|---------|
| `.editor` | Rich-text contenteditable surface (toolbar + preview) | `editor.html` (shared by problem, solution, options, instructions) |
| `.mcq-tab`, `.mcq-tab-add` | Per-option tabs in the MCQ editor | `add_problem.html` |
| `.chip`, `.chip-box`, `.chip-editor` | Editable comma-separated tag input (positive / negative marks per section) | `chip_box_cells.html`, `test_chip_editor.html` |
| `.concept-chip` | Pill for selected concepts (blue tint) | rich-text editor + tags input |
| `.sub-tab` | State-machine sub-navigation (Topics / Resources, Concepts / Problems / Resources) | `chapter.html`, `topic.html` |
| `.action-button` | Icon-only row action (pencil, trash, copy, download) | every list row |
| `.answer-pdf-table-row` | Border + padding for the answer-sheet PDF tables | `answer_sheet.html` |

MathJax wrapping rules at the bottom of `input.css` force formulae to wrap, unless the formula contains an `<mtable>` (matrix), in which case it stays on one line.

---

## 15. HTMX Loading States

Spinners use the brand border + accent top-border pattern:

```html
<div class="hidden flex justify-center py-4" id="some-loader">
  <div class="w-6 h-6 border-4 border-border border-t-accent rounded-full animate-spin"></div>
</div>
```

`input.css` already wires `.htmx-request` visibility for the well-known loader ids (`#search-btn-loader`, `#searched-test-loader`, `#question-bank-loader`, `#topic-problems-loader`) and form-state classes (`#move-problems-form.htmx-request .btn-text { display: none; }` etc.). New loaders should follow the same naming so the existing rules apply, or add a fresh selector to the components layer.

For an inline button spinner (e.g. on a submit button that fires HTMX):

```html
<button class="btn-primary group min-w-[120px]" hx-post="…">
  <span class="group-[.htmx-request]:hidden">Move</span>
  <span class="hidden group-[.htmx-request]:inline-flex">
    <div class="w-4 h-4 border-2 border-text-on-accent border-t-transparent rounded-full animate-spin"></div>
  </span>
</button>
```

---

## 16. Responsive & Mobile

- **Progressive padding** on detail pages: `px-4 sm:px-6 lg:px-8`.
- **Grid breakpoints**: `grid-cols-1 md:grid-cols-3 gap-6` for the type / skills / concepts row in the problem editor.
- **Wrap nav items** on narrow screens with `flex-wrap` + `gap-y-2` (already applied in the base template).
- **Touch targets** ≥44 px for buttons / inputs, ≥48 px for tab items / radio labels.
- **Tab overflow**: add `overflow-x-auto` on tab bars that may exceed the viewport.
- **Tables**: keep `overflow-x-auto` on the wrapping `.card` so horizontal scroll is graceful on phones.

---

## 17. Rebuilding the CSS

You only need to run Tailwind locally if you change `input.css` or `tailwind.config.js`. The compiled `web/static/css/output.css` is checked in.

```bash
npm install        # First time only
npm run build:css  # Rebuild after editing tokens or components
```

If you add a class that's only referenced from server-fetched HTML (e.g. PDF instructions loaded over fetch), add it to the `safelist` in `tailwind.config.js` so Tailwind doesn't tree-shake it away.

For day-to-day editing, run the build once after each change to `input.css` — there's no watch mode wired up by default. Builds finish in ~200 ms.

---

## Cross-references

- Original guide (LMS): <https://github.com/avantifellows/af_lms/blob/main/docs/UI-Style-Guide.md>
- HR app this palette comes from: <https://hr-feedback.avantifellows.org>
- Tokens / Tailwind config: [`tailwind.config.js`](../tailwind.config.js)
- Shared classes: [`input.css`](../input.css)
- Base template (nav, font loader, dropdowns): [`web/html/home.html`](../web/html/home.html)
