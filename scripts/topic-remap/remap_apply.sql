-- Remap new-CMS questions from their chapter's "Introduction" topic to the AI-assigned topic, and tag
-- doubtful ones "needs verification". One transaction: all or nothing. Run via run.sh, not directly.
--
-- Inputs: topic_tags_run1.csv and run1_vs_blind_recheck.csv. run.sh downloads them from S3 (read-only
-- cms-data-import role), checks them against checksums.sha256, and runs this file from that download
-- directory; psql's \copy can only read local files, so this file never talks to S3 itself.
-- Variables: env (label), export_cutoff (required; UTC timestamp when the CSV was exported), apply=yes to
-- commit, expect_remap / expect_flag (apply aborts unless both match).
-- Output: backup.csv (every row this run changes, before → after; flag=true only where this run adds the tag), written before COMMIT; remap_rollback.sql reads it.
--
-- Rules: only questions with a single topic row that (a) is on the chapter's default first slot (code
-- ending .1, or .0 on a few old topics) and (b) hasn't changed since export_cutoff. Names don't work as
-- the test: the slot is "Introduction" in most chapters but "Introudctions" (JA-11M2.1) or "Introduction
-- to Definite Integration" (JM-12M20.1) in others, while real topics like "Introduction to Determinants"
-- sit elsewhere. A question on any other topic, or changed after the export, is a person's tag and is
-- left alone. Then: the candidate for the
-- question's curriculum (or, for old unprefixed topics, the one in the same chapter); target resolved by
-- topic code (the CSV's topic_row_id is stale for some rows); target must be in the same chapter.
-- Flag: the blind recheck disagreed with run 1, or the tag came from the "v2-closest" prompt.

\if :{?apply}
\else
  \set apply no
\endif
\if :{?export_cutoff}
\else
  \echo 'export_cutoff is required (UTC timestamp of the CSV export)'
  \quit
\endif
\if :{?expect_remap}
\else
  \set expect_remap -1
\endif
\if :{?expect_flag}
\else
  \set expect_flag -1
\endif

BEGIN;

CREATE TEMP TABLE run1 (
  problem_id text, topic_id text, confidence text, needs_second_topic text, model text, prompt_version text,
  cms_row_id bigint, topic_row_id text, additional_topics text, team_review text
) ON COMMIT DROP;
\copy run1 FROM 'topic_tags_run1.csv' CSV HEADER

CREATE TEMP TABLE recheck (
  problem_id text, cms_row_id bigint, curriculum text, subject text, has_diagram text, run1_topic_id text,
  run1_topic_name text, run1_second_topic text, team_review text, recheck_topic_id text,
  recheck_topic_name text, recheck_second_topic text, match text, recheck_reason text
) ON COMMIT DROP;
\copy recheck FROM 'run1_vs_blind_recheck.csv' CSV HEADER

CREATE TEMP TABLE remap_input ON COMMIT DROP AS
SELECT r.cms_row_id AS resource_id, r.topic_id AS target_code,
       concat_ws('+',
         CASE WHEN c.match IS NOT NULL AND c.match <> 'same topic' THEN 'recheck:' || c.match END,
         CASE WHEN r.prompt_version LIKE '%closest%' THEN 'v2-closest' END) AS flag_reason
FROM run1 r
LEFT JOIN recheck c ON c.problem_id = r.problem_id AND c.run1_topic_id = r.topic_id;

CREATE TEMP TABLE remap_plan ON COMMIT DROP AS
WITH cur AS (
  SELECT rt.id AS rt_id, rt.resource_id, rt.topic_id, t.code, t.chapter_id,
         t.code ~ '\.[01]$' AS on_default_slot,
         rt.updated_at <= :'export_cutoff'::timestamp AS unchanged_since_export,
         count(*) OVER (PARTITION BY rt.resource_id) AS n_rows
  FROM resource_topic rt JOIN topic t ON t.id = rt.topic_id
  WHERE rt.resource_id IN (SELECT resource_id FROM remap_input)
),
topic_by_code AS (
  SELECT DISTINCT ON (code) id, code, chapter_id FROM topic ORDER BY code, id
),
cand AS (
  SELECT i.resource_id, i.target_code, i.flag_reason,
         c.rt_id, c.topic_id AS from_topic_id, c.code AS from_code, c.chapter_id AS from_chapter,
         c.on_default_slot, c.unchanged_since_export, c.n_rows, tt.id AS to_topic_id, tt.chapter_id AS to_chapter
  FROM remap_input i
  JOIN cur c USING (resource_id)
  LEFT JOIN topic_by_code tt ON tt.code = i.target_code
  WHERE (position('-' IN c.code) > 0 AND split_part(c.code, '-', 1) = split_part(i.target_code, '-', 1))
     OR (position('-' IN c.code) = 0 AND tt.chapter_id = c.chapter_id)
),
one AS (
  SELECT *, count(*) OVER (PARTITION BY resource_id) AS n_cand FROM cand
)
SELECT *,
  CASE
    WHEN n_rows <> 1 THEN 'skip:multiple_topic_rows'
    WHEN n_cand <> 1 THEN 'skip:ambiguous'
    WHEN from_topic_id = to_topic_id THEN 'skip:already_on_target'
    WHEN NOT on_default_slot THEN 'skip:already_tagged_by_person'
    WHEN NOT unchanged_since_export THEN 'skip:changed_since_export'
    WHEN to_topic_id IS NULL THEN 'skip:target_topic_not_found'
    WHEN to_chapter <> from_chapter THEN 'skip:target_in_other_chapter'
    ELSE 'remap'
  END AS action
FROM one;

ALTER TABLE remap_plan ADD COLUMN flag boolean;
UPDATE remap_plan SET flag = (flag_reason <> '' AND action IN ('remap', 'skip:already_on_target'));

\echo
\echo '== Plan (' :'env' ')'
SELECT action, count(*) FROM remap_plan GROUP BY 1
UNION ALL SELECT 'not_found_in_db', count(DISTINCT resource_id) FROM remap_input
  WHERE resource_id NOT IN (SELECT resource_id FROM resource_topic)
UNION ALL SELECT 'no_candidate_for_curriculum', count(DISTINCT resource_id) FROM remap_input
  WHERE resource_id IN (SELECT resource_id FROM resource_topic)
    AND resource_id NOT IN (SELECT resource_id FROM remap_plan)
ORDER BY 1;
\echo 'Skipped because the AI topic is in another chapter (chapter-fix candidates):'
SELECT resource_id, from_code, target_code FROM remap_plan WHERE action = 'skip:target_in_other_chapter' ORDER BY 1;

SELECT count(*) FILTER (WHERE action = 'remap') AS n_remap,
       count(*) FILTER (WHERE flag) AS n_flag
FROM remap_plan \gset
\echo 'to remap:' :n_remap ', to flag "needs verification":' :n_flag

SELECT (:'apply' = 'yes') AS do_apply,
       (:expect_remap = :n_remap AND :expect_flag = :n_flag) AS counts_ok \gset

\if :do_apply
  \if :counts_ok
  \else
    \echo 'ABORT: counts differ from expected (remap' :n_remap 'vs' :expect_remap ', flag' :n_flag 'vs' :expect_flag ')'
    ROLLBACK;
    \quit
  \endif

  INSERT INTO tag (name, description, inserted_at, updated_at)
  SELECT 'needs verification', 'AI-assigned topic; a teacher should check it and remove this tag',
         now() AT TIME ZONE 'utc', now() AT TIME ZONE 'utc'
  WHERE NOT EXISTS (SELECT 1 FROM tag WHERE name = 'needs verification');
  SELECT min(id) AS tag_id FROM tag WHERE name = 'needs verification' \gset

  -- Rows this run changes; flag only where this run adds the tag. Built as a table because \copy
  -- can't interpolate psql variables.
  CREATE TEMP TABLE remap_backup_rows ON COMMIT DROP AS
  SELECT p.rt_id, p.resource_id, p.from_topic_id,
         CASE WHEN p.action = 'remap' THEN p.to_topic_id ELSE p.from_topic_id END AS to_topic_id,
         (p.flag AND NOT (:tag_id::int = ANY (coalesce(r.tag_ids, '{}')))) AS flag,
         :tag_id::int AS tag_id
    FROM remap_plan p JOIN resource r ON r.id = p.resource_id
   WHERE p.action = 'remap' OR (p.flag AND NOT (:tag_id::int = ANY (coalesce(r.tag_ids, '{}'))));
  \copy (SELECT * FROM remap_backup_rows) TO 'backup.csv' CSV HEADER

  UPDATE resource_topic rt
     SET topic_id = p.to_topic_id, updated_at = now() AT TIME ZONE 'utc'
    FROM remap_plan p
   WHERE p.action = 'remap' AND rt.id = p.rt_id AND rt.topic_id = p.from_topic_id;

  UPDATE resource r
     SET tag_ids = array_append(coalesce(r.tag_ids, '{}'), :tag_id::int), updated_at = now() AT TIME ZONE 'utc'
    FROM remap_plan p
   WHERE p.flag AND r.id = p.resource_id AND NOT (:tag_id::int = ANY (coalesce(r.tag_ids, '{}')));

  COMMIT;
  \echo 'APPLIED: remapped' :n_remap ', flagged' :n_flag ', tag id' :tag_id '; backup in backup.csv'
\else
  ROLLBACK;
  \echo 'DRY RUN: nothing written.'
\endif
