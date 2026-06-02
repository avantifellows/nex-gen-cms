# Next Generation CMS

Next Generation CMS is the content and admin surface for Avanti curriculum material and related global configuration used by learner-facing and school-facing products.

## Language

### Curriculum

**Curriculum Config Management**:
An admin-only workflow for changing global LMS Chapter Exam Config values that affect all schools using the configured chapter and exam track.
_Avoid_: Content config, syllabus config

**LMS Chapter Exam Config**:
An exam-track-specific configuration for a chapter that records whether it is in syllabus, the prescribed lecture time, and the coverage order.
_Avoid_: Chapter setup, syllabus row

**Exam Track**:
The exam-specific curriculum lens selected by a user, such as JEE Main, JEE Advanced, or NEET.
_Avoid_: Stream, orientation

### Roles & Access

**CMS Admin**:
A CMS user with the `admin` role who can manage users and global LMS Chapter Exam Config rows.
_Avoid_: LMS Admin, superuser

## Relationships

- A **Chapter** can have one **LMS Chapter Exam Config** per **Exam Track**
- **Curriculum Config Management** changes live **LMS Chapter Exam Config** rows directly
- **Curriculum Config Management** is global and is not scoped to a school or program
- **Curriculum Config Management** is exposed from the CMS Admin area
- Only a **CMS Admin** can use **Curriculum Config Management**
- CMS manages **LMS Chapter Exam Config** rows through direct Postgres access, not through LMS APIs

## Scope Boundaries

- **Curriculum Config Management** in CMS has parity with the LMS v1 workflow: filters, pagination, add, edit, remove-from-syllabus, impact counts, and CSV export
- CMS adapts access control to its own role model; LMS passcode users, PMs, and Program Admins do not exist in CMS
- The CMS page and navigation use `Curriculum Config`; detailed row copy may use **LMS Chapter Exam Config**
- The first CMS implementation does not change LMS navigation or remove the LMS `/curriculum-summary/config` page
- LMS entry-point changes and LMS feature removal happen only after the CMS implementation is verified
- CMS checks schema readiness before rendering or mutating **LMS Chapter Exam Config** rows
- CMS uses a Postgres `xmin::text` lock token for optimistic concurrency; `updated_at` is audit display data
- Removing an in-syllabus row from syllabus is a dedicated confirmed action, not a normal edit toggle
- Restoring an out-of-syllabus row is allowed through edit
- Duplicate coverage order values produce warnings but do not block saving
- Adding a config can target any chapter; chapters with no topics are allowed but should be warned about
- A chapter can have at most one **LMS Chapter Exam Config** per **Exam Track**
- CSV export uses the active filters and ignores pagination
- CSV export omits internal ids and lock tokens
- Bulk CSV import is out of scope for CMS v1
- **Curriculum Config Management** is a dedicated CMS Admin page, not part of the Admin user-management page
- **Curriculum Config Management** edits happen in a modal or side panel, not inline in the table
- Filter changes require an explicit apply action
- Table and modal updates use server-rendered HTMX partials
- Impact preview HTMX requests carry the candidate chapter, exam track, syllabus status, prescribed minutes, and coverage order so warnings and counts match the pending save
- Mutation responses preserve the last applied table filters through hidden `filter_*` form fields
- CMS v1 does not embed a separate client app for **Curriculum Config Management**
- CMS preserves LMS audit fields: create stamps inserted and updated email fields, while edit and remove stamp only the updated email field
- **Curriculum Config Management** uses direct DB queries and does not use or invalidate the existing CMS content caches
- **Curriculum Config Management** changes do not mutate LMS Curriculum Logs or Chapter Completion records
- Impact counts are shown before save and returned again after save using the saved row
- Removing the LMS `/curriculum-summary/config` page is a follow-up after the CMS implementation is verified
- Repository tests cover **Curriculum Config Management** business rules, while handler tests cover routing, access, and template responses
- CMS preserves LMS error semantics for stale writes, invalid payloads, and schema unavailable states while adapting responses to HTMX UI
- Browser-level coverage is valuable but not mandatory for CMS v1 if repository and handler coverage plus manual QA cover the workflow
- CMS acceptance criteria should mirror the LMS v1 Curriculum Config QA checklist, excluding LMS navigation and removal
- Follow-up LMS removal is out of scope for the CMS add-only issue

## Example dialogue

> **Dev:** "Should the CMS page call this Syllabus Config?"
> **Domain expert:** "No. Use **Curriculum Config** for the page and **LMS Chapter Exam Config** for each row, because these rows drive LMS Curriculum behavior globally."

> **Dev:** "Should editors who can edit chapters also edit **Curriculum Config**?"
> **Domain expert:** "No. **Curriculum Config Management** changes global LMS behavior, so it belongs to **CMS Admins** only."

> **Dev:** "Should CMS call the LMS config APIs?"
> **Domain expert:** "No. CMS owns this admin surface and should write the live **LMS Chapter Exam Config** rows directly."

> **Dev:** "Can duplicate coverage order be saved?"
> **Domain expert:** "Yes. Warn the **CMS Admin**, but do not block the save."

> **Dev:** "Should a chapter with no topics be blocked from config?"
> **Domain expert:** "No. Allow it, but warn the **CMS Admin** that there are no topics under that chapter."

> **Dev:** "Should the table allow inline editing?"
> **Domain expert:** "No. Use a modal or side panel so each global config change is explicit."

> **Dev:** "Should saving config update existing curriculum logs?"
> **Domain expert:** "No. **Curriculum Config Management** changes global config only; logs and chapter completions remain unchanged."

> **Dev:** "Should LMS feature removal be part of the CMS issue?"
> **Domain expert:** "No. First verify the CMS page, then handle LMS removal separately."

## Flagged ambiguities

- "config" can refer to many CMS settings; resolved: this feature is specifically **Curriculum Config Management** for **LMS Chapter Exam Config** rows.
- "admin" can refer to LMS roles or CMS roles; resolved: CMS uses **CMS Admin** for this workflow.
