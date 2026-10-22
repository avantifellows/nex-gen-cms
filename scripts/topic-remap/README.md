# Topic remap (one-off, Oct 2026)

Moves ~11.4k new-CMS questions off their chapter's "Introduction" topic onto the topic an AI tagger assigned, and adds the `needs verification` tag to the ~1.8k doubtful ones so teachers check them first. Teachers clear the flag by removing the tag on the question's edit screen.

## Inputs

Two CSVs in `s3://af-data-dumps/nex-gen-cms/topic-remap/2026-10-22/`, pinned by `checksums.sha256`:

- `topic_tags_run1.csv`: the AI topic per question (`cms_row_id` = `resource.id`, `topic_id` = topic code).
- `run1_vs_blind_recheck.csv`: a second, blind tagging run. Where it disagrees with run 1 (`match` ≠ `same topic`), the question is flagged. Questions tagged by the `v2-closest` prompt are flagged too.

## What it changes

- `resource_topic.topic_id`, only for questions still on their chapter's default first topic slot (code ending `.1`, or `.0` on a few old topics; usually named "Introduction", but not always) **and** whose topic row hasn't changed since the CSV export (`EXPORT_CUTOFF` in `run.sh`). Anything else is a person's tag and is left alone. The target is resolved by topic code and must be in the same chapter and curriculum; anything else is skipped and counted.
- `tag`: creates `needs verification` if it doesn't exist.
- `resource.tag_ids`: appends that tag to flagged questions.

Everything runs in one transaction. Before committing, `apply` writes `backup.csv` (every changed row, before → after) to `~/.cms-topic-remap/<env>/`. `apply` also uploads it to `s3://af-data-dumps/nex-gen-cms/topic-remap/2026-10-22/backups/<env>/` (timestamped, plus `latest.csv`), so others can check what changed. `rollback` restores from the local copy, leaving alone any topic a person has changed since, and removes the tag only from questions that apply tagged. On another machine, point it at a specific backup with `REMAP_BACKUP=s3://…/backup-<time>-apply.csv`; it never fetches `latest.csv` by itself, because restoring an already rolled-back backup again would undo deliberate re-tags.

The `cms-data-import` role can read `af-data-dumps` and write only under `nex-gen-cms/topic-remap/*/backups/`.

## Running

Prerequisites: `psql`, and the AWS CLI with a `cms-data-import` profile. That profile assumes the read-only role `arn:aws:iam::111766607077:role/cms-data-import`; no access keys are involved.

```
# ~/.aws/config
[profile cms-data-import]
role_arn = arn:aws:iam::111766607077:role/cms-data-import
source_profile = default
region = ap-south-1
```

```
export REMAP_DATABASE_URL='postgres://…'   # the target DB, with write access for apply/rollback
./run.sh staging dry-run                   # plan and counts; writes nothing
./run.sh staging apply                     # dry run, confirm, then apply with the counts pinned
./run.sh staging rollback
```

`apply` aborts if the counts at write time differ from the ones you confirmed.
