package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
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

func (r *CurriculumConfigRepo) List(ctx context.Context, query curriculumconfig.ListQuery) (curriculumconfig.ListResult, error) {
	query = curriculumconfig.NormalizeListQuery(query)
	whereSQL, args := buildListWhere(query)

	var totalRows int
	countSQL := `
		SELECT COUNT(*)
		FROM lms_chapter_exam_configs c
		` + listDisplayJoins() + `
		` + whereSQL
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&totalRows); err != nil {
		return curriculumconfig.ListResult{}, err
	}

	offset := (query.Page - 1) * query.Limit
	rowArgs := append(append([]any{}, args...), query.Limit, offset)
	rowsSQL := `
		SELECT
			c.id,
			c.chapter_id,
			ch.code AS chapter_code,
			COALESCE(chn.name, '') AS chapter_name,
			g.number::text AS grade,
			COALESCE(sn.name, '') AS subject,
			c.exam_track,
			c.is_in_syllabus,
			c.prescribed_minutes,
			c.coverage_sequence,
			c.updated_by_email,
			c.updated_at,
			c.xmin::text AS lock_token
		FROM lms_chapter_exam_configs c
		` + listDisplayJoins() + `
		` + whereSQL + `
		ORDER BY ` + listOrderBy(query) + `
		LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	rows, err := r.db.QueryContext(ctx, rowsSQL, rowArgs...)
	if err != nil {
		return curriculumconfig.ListResult{}, err
	}
	defer rows.Close()

	var resultRows []curriculumconfig.ListRow
	for rows.Next() {
		var row curriculumconfig.ListRow
		if err := rows.Scan(
			&row.ID,
			&row.ChapterID,
			&row.ChapterCode,
			&row.ChapterName,
			&row.Grade,
			&row.Subject,
			&row.ExamTrack,
			&row.IsInSyllabus,
			&row.PrescribedMinutes,
			&row.CoverageSequence,
			&row.UpdatedByEmail,
			&row.UpdatedAt,
			&row.LockToken,
		); err != nil {
			return curriculumconfig.ListResult{}, err
		}
		row.PrescribedHours = PrescribedHoursLabel(row.PrescribedMinutes)
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return curriculumconfig.ListResult{}, err
	}

	totalPages := 0
	if totalRows > 0 {
		totalPages = int(math.Ceil(float64(totalRows) / float64(query.Limit)))
	}
	return curriculumconfig.ListResult{
		Rows:       resultRows,
		TotalRows:  totalRows,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *CurriculumConfigRepo) FilterOptions(ctx context.Context) (curriculumconfig.FilterOptions, error) {
	grades, err := r.optionRows(ctx, `
		SELECT value, label
		FROM (
			SELECT DISTINCT g.number::text AS value, 'Grade ' || g.number::text AS label, g.number AS sort_number
			FROM lms_chapter_exam_configs c
			`+listDisplayJoins()+`
		) grade_options
		ORDER BY sort_number
	`)
	if err != nil {
		return curriculumconfig.FilterOptions{}, err
	}
	subjects, err := r.optionRows(ctx, `
		SELECT DISTINCT s.id::text AS value, COALESCE(sn.name, '') AS label
		FROM lms_chapter_exam_configs c
		`+listDisplayJoins()+`
		ORDER BY COALESCE(sn.name, '')
	`)
	if err != nil {
		return curriculumconfig.FilterOptions{}, err
	}
	return curriculumconfig.FilterOptions{
		ExamTracks: []curriculumconfig.Option{
			{Value: "jee_main", Label: "JEE Main"},
			{Value: "jee_advanced", Label: "JEE Advanced"},
			{Value: "neet", Label: "NEET"},
		},
		Grades:   grades,
		Subjects: subjects,
	}, nil
}

func (r *CurriculumConfigRepo) optionRows(ctx context.Context, sql string) ([]curriculumconfig.Option, error) {
	rows, err := r.db.QueryContext(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []curriculumconfig.Option
	for rows.Next() {
		var option curriculumconfig.Option
		if err := rows.Scan(&option.Value, &option.Label); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return options, nil
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

func buildListWhere(query curriculumconfig.ListQuery) (string, []any) {
	clauses := []string{"c.exam_track = $1"}
	args := []any{query.ExamTrack}
	nextArg := 2

	if query.SyllabusStatus != "all" {
		clauses = append(clauses, fmt.Sprintf("c.is_in_syllabus = $%d", nextArg))
		args = append(args, query.SyllabusStatus == "in_syllabus")
		nextArg++
	}
	if query.Grade != "" {
		clauses = append(clauses, fmt.Sprintf("g.number::text = $%d", nextArg))
		args = append(args, query.Grade)
		nextArg++
	}
	if query.Subject != "" {
		clauses = append(clauses, fmt.Sprintf("(s.id::text = $%d OR COALESCE(sn.name, '') = $%d)", nextArg, nextArg))
		args = append(args, query.Subject)
		nextArg++
	}
	if query.ChapterID != "" {
		clauses = append(clauses, fmt.Sprintf("c.chapter_id::text = $%d", nextArg))
		args = append(args, query.ChapterID)
		nextArg++
	}
	if query.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(ch.code ILIKE $%d OR COALESCE(chn.name, '') ILIKE $%d)", nextArg, nextArg))
		args = append(args, "%"+query.Search+"%")
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func listDisplayJoins() string {
	return `
		JOIN chapter ch ON ch.id = c.chapter_id
		JOIN grade g ON g.id = ch.grade_id
		JOIN subject s ON s.id = ch.subject_id
		LEFT JOIN LATERAL (
			SELECT elem->>'chapter' AS name
			FROM jsonb_array_elements(ch.name) elem
			WHERE elem->>'lang_code' = 'en'
			LIMIT 1
		) chn ON true
		LEFT JOIN LATERAL (
			SELECT elem->>'subject' AS name
			FROM jsonb_array_elements(s.name) elem
			WHERE elem->>'lang_code' = 'en'
			LIMIT 1
		) sn ON true`
}

func listOrderBy(query curriculumconfig.ListQuery) string {
	if query.Sort == "curriculum" {
		direction := strings.ToUpper(query.Direction)
		return "c.exam_track " + direction + ", g.number " + direction + ", COALESCE(sn.name, '') " + direction + ", c.coverage_sequence " + direction + ", ch.code " + direction + ", COALESCE(chn.name, '') " + direction + ", c.id ASC"
	}

	sortColumns := map[string]string{
		"exam_track":        "c.exam_track",
		"grade":             "g.number",
		"subject":           "COALESCE(sn.name, '')",
		"coverage_sequence": "c.coverage_sequence",
		"chapter_code":      "ch.code",
		"chapter_name":      "COALESCE(chn.name, '')",
		"updated_at":        "c.updated_at",
	}
	column := sortColumns[query.Sort]
	return column + " " + strings.ToUpper(query.Direction) + ", c.exam_track ASC, g.number ASC, COALESCE(sn.name, '') ASC, c.coverage_sequence ASC, ch.code ASC, COALESCE(chn.name, '') ASC, c.id ASC"
}

func PrescribedHoursLabel(minutes int) string {
	hours := float64(minutes) / 60
	if minutes%60 == 0 {
		if minutes == 60 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", minutes/60)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", hours), "0"), ".") + " hours"
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
