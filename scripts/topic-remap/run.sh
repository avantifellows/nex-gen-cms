#!/usr/bin/env bash
# Remap AI-tagged question topics (and add the "needs verification" tag), or undo it. See README.md.
#
#   REMAP_DATABASE_URL=postgres://… ./run.sh <staging|prod> dry-run
#   REMAP_DATABASE_URL=postgres://… ./run.sh <staging|prod> apply
#   REMAP_DATABASE_URL=postgres://… ./run.sh <staging|prod> rollback
set -euo pipefail

ENV_NAME=${1:?usage: run.sh <staging|prod> <dry-run|apply|rollback>}
MODE=${2:?usage: run.sh <staging|prod> <dry-run|apply|rollback>}
case "$ENV_NAME" in staging|prod) ;; *) echo "env must be staging or prod" >&2; exit 2 ;; esac
case "$MODE" in dry-run|apply|rollback) ;; *) echo "mode must be dry-run, apply or rollback" >&2; exit 2 ;; esac
: "${REMAP_DATABASE_URL:?set REMAP_DATABASE_URL to the $ENV_NAME database}"

HERE=$(cd "$(dirname "$0")" && pwd)
S3_PREFIX=s3://af-data-dumps/nex-gen-cms/topic-remap/2026-10-22
AWS_PROFILE=${AWS_PROFILE:-cms-data-import}
# Kept between runs: rollback needs the backup that apply wrote here.
WORK=${REMAP_WORK_DIR:-$HOME/.cms-topic-remap/$ENV_NAME}
mkdir -p "$WORK"

for f in topic_tags_run1.csv run1_vs_blind_recheck.csv; do
  [ -f "$WORK/$f" ] || aws s3 cp "$S3_PREFIX/$f" "$WORK/$f" --profile "$AWS_PROFILE" --only-show-errors
done
(cd "$WORK" && shasum -a 256 -c "$HERE/checksums.sha256")

BACKUPS=$S3_PREFIX/backups/$ENV_NAME
upload() { aws s3 cp "$1" "$BACKUPS/$2" --profile "$AWS_PROFILE" --only-show-errors && echo "uploaded $BACKUPS/$2"; }

psql_run() { (cd "$WORK" && psql "$REMAP_DATABASE_URL" -X -q -v ON_ERROR_STOP=1 -v env="$ENV_NAME" "$@"); }

case "$MODE" in
  dry-run)
    psql_run -f "$HERE/remap_apply.sql"
    ;;
  apply)
    out=$(psql_run -f "$HERE/remap_apply.sql")
    echo "$out"
    n_remap=$(sed -n 's/^to remap: \([0-9]*\) .*/\1/p' <<<"$out")
    n_flag=$(sed -n 's/.*to flag "needs verification": \([0-9]*\)$/\1/p' <<<"$out")
    [ -f "$WORK/backup.csv" ] && { echo "backup.csv already exists in $WORK: roll back or move it first" >&2; exit 1; }
    read -r -p "Apply $n_remap remaps and $n_flag flags to $ENV_NAME? Type '$ENV_NAME' to confirm: " ok
    [ "$ok" = "$ENV_NAME" ] || { echo aborted; exit 1; }
    if ! psql_run -v apply=yes -v expect_remap="$n_remap" -v expect_flag="$n_flag" -f "$HERE/remap_apply.sql" | tail -2; then
      rm -f "$WORK/backup.csv"   # the transaction rolled back; a backup of it would mislead rollback
      echo "apply FAILED: nothing was changed" >&2; exit 1
    fi
    # Share the backup so others can verify the run (and roll back from another machine via REMAP_BACKUP).
    upload "$WORK/backup.csv" "backup-$(date +%Y%m%d-%H%M%S)-apply.csv"
    upload "$WORK/backup.csv" "latest.csv"
    ;;
  rollback)
    # Only the local backup, or one named explicitly. Never fetch latest.csv on our own: after a
    # rollback it is stale, and restoring from it again would undo deliberate re-tags by teachers.
    if [ ! -f "$WORK/backup.csv" ]; then
      [ -n "${REMAP_BACKUP:-}" ] || { echo "no backup.csv in $WORK; set REMAP_BACKUP=s3://… to a specific backup" >&2; exit 1; }
      aws s3 cp "$REMAP_BACKUP" "$WORK/backup.csv" --profile "$AWS_PROFILE" --only-show-errors
    fi
    [ "$(wc -l < "$WORK/backup.csv")" -gt 1 ] || { echo "backup.csv in $WORK is empty: nothing to roll back" >&2; exit 1; }
    psql_run -f "$HERE/remap_rollback.sql"
    read -r -p "Roll back on $ENV_NAME? Type '$ENV_NAME' to confirm: " ok
    [ "$ok" = "$ENV_NAME" ] || { echo aborted; exit 1; }
    psql_run -v apply=yes -f "$HERE/remap_rollback.sql" | tail -1
    stamp=$(date +%Y%m%d-%H%M%S)
    # Record which backup was rolled back. The local copy is renamed, so rollback can't run twice by accident.
    upload "$WORK/backup.csv" "backup-$stamp-rolled-back.csv"
    mv "$WORK/backup.csv" "$WORK/backup.rolled-back-$stamp.csv"
    ;;
esac
