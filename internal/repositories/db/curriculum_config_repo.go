package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"

	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/lib/pq"
)

type CurriculumConfigRepo struct {
	db *sql.DB

	mu     sync.RWMutex
	cached *curriculumconfig.Readiness
}

func NewCurriculumConfigRepo(database *sql.DB) *CurriculumConfigRepo {
	return &CurriculumConfigRepo{db: database}
}

func (r *CurriculumConfigRepo) SchemaReadiness(ctx context.Context) (curriculumconfig.Readiness, error) {
	r.mu.RLock()
	if r.cached != nil {
		readiness := *r.cached
		r.mu.RUnlock()
		return readiness, nil
	}
	r.mu.RUnlock()

	readiness, err := r.checkSchemaReadiness(ctx)
	if err != nil {
		return curriculumconfig.Readiness{}, err
	}
	if readiness.Ready && readiness.MutationReady {
		r.mu.Lock()
		cached := readiness
		r.cached = &cached
		r.mu.Unlock()
	}
	return readiness, nil
}

func (r *CurriculumConfigRepo) List(context.Context, curriculumconfig.ListQuery) (curriculumconfig.ListResult, error) {
	return curriculumconfig.ListResult{}, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) FilterOptions(context.Context) (curriculumconfig.FilterOptions, error) {
	return curriculumconfig.FilterOptions{}, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) ChapterOptions(context.Context, curriculumconfig.ChapterOptionsQuery) ([]curriculumconfig.ChapterOption, error) {
	return nil, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) Impact(context.Context, curriculumconfig.ImpactQuery) (curriculumconfig.ImpactResult, error) {
	return curriculumconfig.ImpactResult{}, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) Create(context.Context, curriculumconfig.CreateInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) Edit(context.Context, curriculumconfig.EditInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) RemoveFromSyllabus(context.Context, curriculumconfig.RemoveInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) ExportRows(context.Context, curriculumconfig.ListQuery) ([]curriculumconfig.ExportRow, error) {
	return nil, curriculumconfig.ErrNotImplemented
}

func (r *CurriculumConfigRepo) checkSchemaReadiness(ctx context.Context) (curriculumconfig.Readiness, error) {
	readiness := curriculumconfig.Readiness{Ready: true, MutationReady: true}

	missingColumns, err := r.missingColumns(ctx)
	if err != nil {
		return readiness, err
	}
	for _, missing := range missingColumns {
		readiness.Ready = false
		readiness.MutationReady = false
		readiness.Reasons = append(readiness.Reasons, "missing column "+missing)
	}

	missingConstraints, err := r.missingConstraints(ctx)
	if err != nil {
		return readiness, err
	}
	for _, missing := range missingConstraints {
		readiness.Ready = false
		readiness.MutationReady = false
		readiness.Reasons = append(readiness.Reasons, "missing constraint "+missing)
	}

	missingIndexes, err := r.missingIndexes(ctx)
	if err != nil {
		return readiness, err
	}
	for _, missing := range missingIndexes {
		readiness.Ready = false
		readiness.MutationReady = false
		readiness.Reasons = append(readiness.Reasons, "missing index "+missing)
	}

	duplicate, err := r.duplicateConfig(ctx)
	if err != nil {
		return readiness, err
	}
	if duplicate != "" {
		readiness.MutationReady = false
		readiness.MutationReasons = append(readiness.MutationReasons, duplicate)
	}
	if !readiness.Ready {
		readiness.MutationReasons = append(readiness.MutationReasons, readiness.Reasons...)
	}
	return readiness, nil
}

func (r *CurriculumConfigRepo) missingColumns(ctx context.Context) ([]string, error) {
	required := requiredCurriculumConfigColumns()
	tables := make([]string, 0, len(required))
	for table := range required {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	rows, err := r.db.QueryContext(ctx, `
		SELECT table_name, column_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = ANY($1)
	`, pq.Array(tables))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]map[string]struct{}, len(required))
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			return nil, err
		}
		if seen[table] == nil {
			seen[table] = make(map[string]struct{})
		}
		seen[table][column] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var missing []string
	for _, table := range tables {
		for _, column := range required[table] {
			if _, ok := seen[table][column]; !ok {
				missing = append(missing, table+"."+column)
			}
		}
	}
	return missing, nil
}

func (r *CurriculumConfigRepo) missingConstraints(ctx context.Context) ([]string, error) {
	required := []string{
		"lms_chapter_exam_configs_exam_track_check",
		"lms_chapter_exam_configs_prescribed_minutes_check",
		"lms_chapter_exam_configs_coverage_sequence_check",
		"lms_chapter_exam_configs_out_of_syllabus_minutes_check",
		"lms_curriculum_logs_exam_track_check",
		"lms_curriculum_logs_duration_minutes_check",
		"lms_curriculum_log_topics_log_topic_unique",
		"lms_curriculum_chapter_completions_exam_track_check",
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.conname
		FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = 'public'
		  AND c.conname = ANY($1)
	`, pq.Array(required))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return missingNames(rows, required)
}

func (r *CurriculumConfigRepo) missingIndexes(ctx context.Context) ([]string, error) {
	required := []string{
		"lms_chapter_exam_configs_chapter_track_unique",
		"lms_chapter_exam_configs_exam_track_chapter_id_index",
		"lms_curriculum_logs_active_scope_index",
		"lms_curriculum_logs_active_scope_date_index",
		"lms_curriculum_logs_log_date_index",
		"lms_curriculum_log_topics_log_id_index",
		"lms_curriculum_log_topics_topic_id_index",
		"lms_curriculum_chapter_completions_active_unique",
		"lms_curriculum_chapter_completions_scope_index",
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = 'public'
		  AND indexname = ANY($1)
	`, pq.Array(required))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return missingNames(rows, required)
}

func (r *CurriculumConfigRepo) duplicateConfig(ctx context.Context) (string, error) {
	var chapterID int64
	var examTrack string
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT chapter_id, exam_track, COUNT(*)
		FROM lms_chapter_exam_configs
		GROUP BY chapter_id, exam_track
		HAVING COUNT(*) > 1
		LIMIT 1
	`).Scan(&chapterID, &examTrack, &count)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("duplicate LMS Chapter Exam Config rows for chapter_id=%d exam_track=%s", chapterID, examTrack), nil
}

func missingNames(rows *sql.Rows, required []string) ([]string, error) {
	seen := make(map[string]struct{}, len(required))
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		seen[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var missing []string
	for _, name := range required {
		if _, ok := seen[name]; !ok {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

func requiredCurriculumConfigColumns() map[string][]string {
	return map[string][]string{
		"lms_chapter_exam_configs": {
			"id", "chapter_id", "exam_track", "is_in_syllabus", "prescribed_minutes",
			"coverage_sequence", "inserted_by_email", "updated_by_email", "inserted_at", "updated_at",
		},
		"chapter": {"id", "code", "name", "grade_id", "subject_id"},
		"grade":   {"id", "number"},
		"subject": {"id", "name", "code"},
		"topic":   {"id", "chapter_id"},
		"school":  {"id", "code", "program_ids"},
		"program": {"id", "name"},
		"lms_curriculum_logs": {
			"id", "school_code", "program_id", "grade_id", "subject_id", "exam_track", "deleted_at",
		},
		"lms_curriculum_log_topics": {
			"id", "curriculum_log_id", "topic_id",
		},
		"lms_curriculum_chapter_completions": {
			"id", "school_code", "program_id", "chapter_id", "exam_track", "deleted_at",
		},
	}
}
