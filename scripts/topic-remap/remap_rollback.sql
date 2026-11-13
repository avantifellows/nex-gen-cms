-- Undo remap_apply.sql using backup.csv. One transaction. Run via run.sh (rollback), not directly.
-- Only restores rows still on the topic the remap set; a topic a person changed since is left alone
-- and counted. Removes the "needs verification" tag only from questions this apply tagged; the tag
-- record itself is kept (harmless when unused, and it may predate the apply).

\if :{?apply}
\else
  \set apply no
\endif

BEGIN;

CREATE TEMP TABLE remap_backup (rt_id bigint, resource_id bigint, from_topic_id bigint, to_topic_id bigint,
                                flag boolean, tag_id int) ON COMMIT DROP;
\copy remap_backup FROM 'backup.csv' CSV HEADER

\echo '== Rollback plan (' :'env' ')'
SELECT count(*) FILTER (WHERE b.from_topic_id <> b.to_topic_id AND rt.topic_id = b.to_topic_id) AS topics_to_restore,
       count(*) FILTER (WHERE b.from_topic_id <> b.to_topic_id AND rt.topic_id = b.from_topic_id) AS already_restored,
       count(*) FILTER (WHERE b.from_topic_id <> b.to_topic_id AND rt.topic_id IS DISTINCT FROM b.to_topic_id
                          AND rt.topic_id IS DISTINCT FROM b.from_topic_id) AS changed_by_person_left_alone,
       count(*) FILTER (WHERE b.flag) AS tags_to_remove
FROM remap_backup b LEFT JOIN resource_topic rt ON rt.id = b.rt_id;

UPDATE resource_topic rt
   SET topic_id = b.from_topic_id, updated_at = now() AT TIME ZONE 'utc'
  FROM remap_backup b
 WHERE rt.id = b.rt_id AND b.from_topic_id <> b.to_topic_id AND rt.topic_id = b.to_topic_id;

UPDATE resource r
   SET tag_ids = array_remove(r.tag_ids, b.tag_id), updated_at = now() AT TIME ZONE 'utc'
  FROM remap_backup b
 WHERE b.flag AND r.id = b.resource_id AND b.tag_id = ANY (r.tag_ids);

SELECT (:'apply' = 'yes') AS do_apply \gset
\if :do_apply
  COMMIT;
  \echo 'ROLLED BACK.'
\else
  ROLLBACK;
  \echo 'DRY RUN only (nothing written). Add -v apply=yes to roll back.'
\endif
