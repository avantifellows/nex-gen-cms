# Curriculum Config Manual QA

Use this checklist for CMS Curriculum Config Management QA before removing the LMS v1 workflow.

- Confirm only CMS Admin users can open `/admin/curriculum-config`, mutation endpoints, and `/admin/curriculum-config/export`; non-admin users should be rejected.
- Confirm schema-unavailable handling by pointing CMS at a database missing a required LMS Chapter Exam Config contract element; the page and export endpoint should show controlled unavailable responses.
- Confirm filters for exam track, grade, subject, chapter search, chapter id, and syllabus status apply only after pressing `Apply filters`.
- Confirm pagination, page size, and sort preserve the applied filters.
- Confirm `Export CSV` uses the applied filters and sort, ignores the current page, and downloads `curriculum-config-YYYY-MM-DD.csv`.
- Confirm CSV headers are `chapter_code`, `chapter_name`, `grade`, `subject`, `exam_track`, `is_in_syllabus`, `prescribed_minutes`, `prescribed_hours`, `coverage_sequence`, `updated_by_email`, and `updated_at`.
- Confirm CSV output omits config row id, chapter id, and lock token, and formula-escapes cells beginning with `=`, `+`, `-`, `@`, tab, or carriage return.
- Confirm zero matching export rows produce a header-only CSV.
- Confirm a realistic filtered export of at least 5,000 rows completes without exhausting the DB pool. CMS caps export reads at 10,000 rows and bounds export work with a 30-second timeout.
- Confirm add, edit, restore from out-of-syllabus, and confirmed remove-from-syllabus flows refresh the table with the applied filters preserved.
- Confirm impact counts, duplicate coverage order warnings, zero-minute warnings, and zero-topic chapter warnings appear before save/remove and after successful mutation.
- Confirm stale edit/remove submissions return visible conflict feedback and do not overwrite newer data.
- Confirm LMS Curriculum Logs, Chapter Completion rows, LMS navigation, LMS APIs, and CMS content caches are not changed by these CMS actions.
