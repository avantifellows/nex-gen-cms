package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/lib/pq"
)

const curriculumConfigExportRowLimit = 10000

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

func (r *CurriculumConfigRepo) Get(ctx context.Context, id int64) (*curriculumconfig.ListRow, error) {
	if id < 1 {
		return nil, errors.New("config id must be positive")
	}
	row := curriculumconfig.ListRow{}
	err := r.db.QueryRowContext(ctx, `
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
		`+listDisplayJoins()+`
		WHERE c.id = $1
	`, id).Scan(
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
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("LMS Chapter Exam Config does not exist")
	}
	if err != nil {
		return nil, err
	}
	row.PrescribedHours = PrescribedHoursLabel(row.PrescribedMinutes)
	return &row, nil
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

func (r *CurriculumConfigRepo) ChapterOptions(ctx context.Context, query curriculumconfig.ChapterOptionsQuery) ([]curriculumconfig.ChapterOption, error) {
	query.ExamTrack = normalizeExamTrack(query.ExamTrack)
	query.Grade = strings.TrimSpace(query.Grade)
	query.Subject = strings.TrimSpace(query.Subject)
	query.Search = strings.TrimSpace(query.Search)

	clauses := []string{"TRUE"}
	args := []any{query.ExamTrack}
	nextArg := 2
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
	if query.Search != "" {
		clauses = append(clauses, fmt.Sprintf("(ch.code ILIKE $%d OR COALESCE(chn.name, '') ILIKE $%d)", nextArg, nextArg))
		args = append(args, "%"+query.Search+"%")
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			ch.id AS chapter_id,
			ch.code AS chapter_code,
			COALESCE(chn.name, '') AS chapter_name,
			g.number::text AS grade,
			COALESCE(sn.name, '') AS subject,
			COUNT(t.id)::int AS topic_count,
			existing.id AS existing_config_id,
			existing.is_in_syllabus AS existing_in_syllabus,
			existing.exam_track AS existing_exam_track
		FROM chapter ch
		JOIN grade g ON g.id = ch.grade_id
		JOIN subject s ON s.id = ch.subject_id
		LEFT JOIN topic t ON t.chapter_id = ch.id
		LEFT JOIN lms_chapter_exam_configs existing
			ON existing.chapter_id = ch.id
		   AND existing.exam_track = $1
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
		) sn ON true
		WHERE `+strings.Join(clauses, " AND ")+`
		GROUP BY ch.id, ch.code, chn.name, g.number, sn.name, existing.id, existing.is_in_syllabus, existing.exam_track
		ORDER BY g.number ASC, COALESCE(sn.name, '') ASC, ch.code ASC, COALESCE(chn.name, '') ASC, ch.id ASC
		LIMIT 50
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []curriculumconfig.ChapterOption
	for rows.Next() {
		var option curriculumconfig.ChapterOption
		var existingID sql.NullInt64
		var existingInSyllabus sql.NullBool
		var existingExamTrack sql.NullString
		if err := rows.Scan(
			&option.ChapterID,
			&option.ChapterCode,
			&option.ChapterName,
			&option.Grade,
			&option.Subject,
			&option.TopicCount,
			&existingID,
			&existingInSyllabus,
			&existingExamTrack,
		); err != nil {
			return nil, err
		}
		if existingID.Valid {
			option.ExistingConfigID = &existingID.Int64
			option.HasDuplicateConfig = true
		}
		if existingInSyllabus.Valid {
			option.ExistingInSyllabus = &existingInSyllabus.Bool
		}
		if existingExamTrack.Valid {
			option.ExistingExamTrack = existingExamTrack.String
		}
		option.HasZeroTopicWarning = option.TopicCount == 0
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return options, nil
}

func (r *CurriculumConfigRepo) Impact(ctx context.Context, query curriculumconfig.ImpactQuery) (curriculumconfig.ImpactResult, error) {
	warnings, err := r.previewWarnings(ctx, query)
	if err != nil {
		return curriculumconfig.ImpactResult{}, err
	}
	result, err := r.impactCounts(ctx, query)
	result.Warnings = warnings
	return result, err
}

func (r *CurriculumConfigRepo) impactCounts(ctx context.Context, query curriculumconfig.ImpactQuery) (curriculumconfig.ImpactResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var result curriculumconfig.ImpactResult
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM school sc
		JOIN program p ON p.id = ANY(sc.program_ids)
		WHERE COALESCE(sc.code, '') <> ''
		  AND UPPER(COALESCE(p.name, '')) IN ('COE', 'NODAL')
	`).Scan(&result.SummaryRows); err != nil {
		result.Unavailable = true
		return result, nil
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT log.id) FROM lms_curriculum_logs log
		JOIN lms_curriculum_log_topics log_topic ON log_topic.curriculum_log_id = log.id
		JOIN topic t ON t.id = log_topic.topic_id
		WHERE t.chapter_id = $1
		  AND log.exam_track = $2
		  AND log.deleted_at IS NULL
	`, query.ChapterID, query.ExamTrack).Scan(&result.ActiveLogs); err != nil {
		result.Unavailable = true
		return result, nil
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM lms_curriculum_chapter_completions completion
		WHERE completion.chapter_id = $1
		  AND completion.exam_track = $2
		  AND completion.deleted_at IS NULL
	`, query.ChapterID, query.ExamTrack).Scan(&result.ChapterCompletions); err != nil {
		result.Unavailable = true
		return result, nil
	}
	return result, nil
}

func (r *CurriculumConfigRepo) Create(ctx context.Context, input curriculumconfig.CreateInput) (curriculumconfig.MutationResult, error) {
	if err := validateCreateInput(input); err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	if err := r.ensureNoExistingConfig(ctx, input.ChapterID, input.ExamTrack); err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	row, err := r.insertConfig(ctx, input)
	if err != nil {
		return curriculumconfig.MutationResult{}, mapCreateError(err)
	}
	warnings, err := r.warningsForConfig(ctx, *row)
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	impact, err := r.impactCounts(ctx, curriculumconfig.ImpactQuery{
		ConfigID:          row.ID,
		ChapterID:         row.ChapterID,
		ExamTrack:         row.ExamTrack,
		IsInSyllabus:      row.IsInSyllabus,
		PrescribedMinutes: row.PrescribedMinutes,
		CoverageSequence:  row.CoverageSequence,
	})
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	return curriculumconfig.MutationResult{Row: row, Warnings: warnings, Impact: impact}, nil
}

func (r *CurriculumConfigRepo) Edit(ctx context.Context, input curriculumconfig.EditInput) (curriculumconfig.MutationResult, error) {
	if err := validateEditInput(input); err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	current, err := r.Get(ctx, input.ID)
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	if current.IsInSyllabus && !input.IsInSyllabus {
		return curriculumconfig.MutationResult{}, errors.New("Use the dedicated remove action to remove an in-syllabus row from syllabus")
	}
	row, err := r.updateConfig(ctx, input)
	if err != nil {
		return curriculumconfig.MutationResult{}, mapEditError(err)
	}
	warnings, err := r.warningsForConfig(ctx, *row)
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	impact, err := r.impactCounts(ctx, curriculumconfig.ImpactQuery{
		ConfigID:          row.ID,
		ChapterID:         row.ChapterID,
		ExamTrack:         row.ExamTrack,
		IsInSyllabus:      row.IsInSyllabus,
		PrescribedMinutes: row.PrescribedMinutes,
		CoverageSequence:  row.CoverageSequence,
	})
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	return curriculumconfig.MutationResult{Row: row, Warnings: warnings, Impact: impact}, nil
}

func (r *CurriculumConfigRepo) RemoveFromSyllabus(ctx context.Context, input curriculumconfig.RemoveInput) (curriculumconfig.MutationResult, error) {
	if err := validateRemoveInput(input); err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	current, err := r.Get(ctx, input.ID)
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	if !current.IsInSyllabus {
		if strings.TrimSpace(current.LockToken) != strings.TrimSpace(input.LockToken) {
			return curriculumconfig.MutationResult{}, curriculumconfig.ErrStaleLock
		}
		return curriculumconfig.MutationResult{}, errors.New("LMS Chapter Exam Config is already out of syllabus")
	}
	row, err := r.removeConfigFromSyllabus(ctx, input)
	if err != nil {
		return curriculumconfig.MutationResult{}, mapEditError(err)
	}
	warnings, err := r.warningsForConfig(ctx, *row)
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	impact, err := r.impactCounts(ctx, curriculumconfig.ImpactQuery{
		ConfigID:          row.ID,
		ChapterID:         row.ChapterID,
		ExamTrack:         row.ExamTrack,
		IsInSyllabus:      row.IsInSyllabus,
		PrescribedMinutes: row.PrescribedMinutes,
		CoverageSequence:  row.CoverageSequence,
	})
	if err != nil {
		return curriculumconfig.MutationResult{}, err
	}
	return curriculumconfig.MutationResult{Row: row, Warnings: warnings, Impact: impact}, nil
}

func (r *CurriculumConfigRepo) ExportRows(ctx context.Context, query curriculumconfig.ListQuery) ([]curriculumconfig.ExportRow, error) {
	query = curriculumconfig.NormalizeListQuery(query)
	whereSQL, args := buildListWhere(query)
	rowArgs := append(append([]any{}, args...), curriculumConfigExportRowLimit)

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rowsSQL := `
		SELECT
			ch.code AS chapter_code,
			COALESCE(chn.name, '') AS chapter_name,
			g.number::text AS grade,
			COALESCE(sn.name, '') AS subject,
			c.exam_track,
			c.is_in_syllabus,
			c.prescribed_minutes,
			c.coverage_sequence,
			c.updated_by_email,
			c.updated_at
		FROM lms_chapter_exam_configs c
		` + listDisplayJoins() + `
		` + whereSQL + `
		ORDER BY ` + listOrderBy(query) + `
		LIMIT $` + strconv.Itoa(len(args)+1)
	rows, err := r.db.QueryContext(ctx, rowsSQL, rowArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var exportRows []curriculumconfig.ExportRow
	for rows.Next() {
		var row curriculumconfig.ExportRow
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
		row.PrescribedHours = PrescribedHoursLabel(row.PrescribedMinutes)
		exportRows = append(exportRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return exportRows, nil
}

func (r *CurriculumConfigRepo) ensureNoExistingConfig(ctx context.Context, chapterID int64, examTrack string) error {
	var id int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM lms_chapter_exam_configs
		WHERE chapter_id = $1 AND exam_track = $2
		LIMIT 1
	`, chapterID, examTrack).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("duplicate LMS Chapter Exam Config already exists for this chapter and exam track")
}

func (r *CurriculumConfigRepo) insertConfig(ctx context.Context, input curriculumconfig.CreateInput) (*curriculumconfig.ListRow, error) {
	row := curriculumconfig.ListRow{}
	err := r.db.QueryRowContext(ctx, `
		WITH inserted AS (
			INSERT INTO lms_chapter_exam_configs (
				chapter_id,
				exam_track,
				is_in_syllabus,
				prescribed_minutes,
				coverage_sequence,
				inserted_by_email,
				updated_by_email
			)
			VALUES ($1, $2, $3, $4, $5, $6, $6)
			RETURNING id, chapter_id, exam_track, is_in_syllabus, prescribed_minutes, coverage_sequence, updated_by_email, updated_at, xmin::text
		)
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
		FROM inserted c
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
		) sn ON true
	`, input.ChapterID, input.ExamTrack, input.IsInSyllabus, input.PrescribedMinutes, input.CoverageSequence, strings.TrimSpace(input.AdminEmail)).Scan(
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
	)
	if err != nil {
		return nil, err
	}
	row.PrescribedHours = PrescribedHoursLabel(row.PrescribedMinutes)
	return &row, nil
}

func (r *CurriculumConfigRepo) updateConfig(ctx context.Context, input curriculumconfig.EditInput) (*curriculumconfig.ListRow, error) {
	row := curriculumconfig.ListRow{}
	err := r.db.QueryRowContext(ctx, `
		WITH updated AS (
			UPDATE lms_chapter_exam_configs c
			SET is_in_syllabus = $2, prescribed_minutes = $3, coverage_sequence = $4, updated_by_email = $5, updated_at = NOW()
			WHERE c.id = $1
			  AND c.xmin::text = $6
			RETURNING c.id, c.chapter_id, c.exam_track, c.is_in_syllabus, c.prescribed_minutes, c.coverage_sequence, c.updated_by_email, c.updated_at, c.xmin::text
		)
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
		FROM updated c
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
		) sn ON true
	`, input.ID, input.IsInSyllabus, input.PrescribedMinutes, input.CoverageSequence, strings.TrimSpace(input.AdminEmail), strings.TrimSpace(input.LockToken)).Scan(
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
	)
	if err != nil {
		return nil, err
	}
	row.PrescribedHours = PrescribedHoursLabel(row.PrescribedMinutes)
	return &row, nil
}

func (r *CurriculumConfigRepo) removeConfigFromSyllabus(ctx context.Context, input curriculumconfig.RemoveInput) (*curriculumconfig.ListRow, error) {
	row := curriculumconfig.ListRow{}
	err := r.db.QueryRowContext(ctx, `
		WITH updated AS (
			UPDATE lms_chapter_exam_configs c
			SET is_in_syllabus = false, prescribed_minutes = 0, updated_by_email = $2, updated_at = NOW()
			WHERE c.id = $1
			  AND c.xmin::text = $3
			RETURNING c.id, c.chapter_id, c.exam_track, c.is_in_syllabus, c.prescribed_minutes, c.coverage_sequence, c.updated_by_email, c.updated_at, c.xmin::text
		)
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
		FROM updated c
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
		) sn ON true
	`, input.ID, strings.TrimSpace(input.AdminEmail), strings.TrimSpace(input.LockToken)).Scan(
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
	)
	if err != nil {
		return nil, err
	}
	row.PrescribedHours = PrescribedHoursLabel(row.PrescribedMinutes)
	return &row, nil
}

func (r *CurriculumConfigRepo) warningsForConfig(ctx context.Context, row curriculumconfig.ListRow) ([]curriculumconfig.Warning, error) {
	var warnings []curriculumconfig.Warning
	if row.IsInSyllabus {
		var duplicateCoverageCount int
		if err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM lms_chapter_exam_configs other
			JOIN chapter other_chapter ON other_chapter.id = other.chapter_id
			JOIN grade other_grade ON other_grade.id = other_chapter.grade_id
			JOIN subject other_subject ON other_subject.id = other_chapter.subject_id
			LEFT JOIN LATERAL (
				SELECT elem->>'subject' AS name
				FROM jsonb_array_elements(other_subject.name) elem
				WHERE elem->>'lang_code' = 'en'
				LIMIT 1
			) other_subject_name ON true
			WHERE other.id <> $1
			  AND other.exam_track = $2
			  AND other.is_in_syllabus = true
			  AND other_grade.number::text = $3
			  AND COALESCE(other_subject_name.name, '') = $4
			  AND other.coverage_sequence = $5
		`, row.ID, row.ExamTrack, row.Grade, row.Subject, row.CoverageSequence).Scan(&duplicateCoverageCount); err != nil {
			return nil, err
		}
		if duplicateCoverageCount > 0 {
			warnings = append(warnings, curriculumconfig.Warning{
				Code:    "duplicate_coverage_order",
				Message: fmt.Sprintf("Another in-syllabus LMS Chapter Exam Config uses coverage order %d for %s Grade %s %s.", row.CoverageSequence, examTrackLabel(row.ExamTrack), row.Grade, row.Subject),
			})
		}
		if row.PrescribedMinutes == 0 {
			warnings = append(warnings, curriculumconfig.Warning{
				Code:    "zero_minutes_in_syllabus",
				Message: "This in-syllabus row has zero prescribed minutes.",
			})
		}
	}
	var topicCount int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(t.id)::int AS topic_count
		FROM chapter ch
		LEFT JOIN topic t ON t.chapter_id = ch.id
		WHERE ch.id = $1
	`, row.ChapterID).Scan(&topicCount); err != nil {
		return nil, err
	}
	if topicCount == 0 {
		warnings = append(warnings, curriculumconfig.Warning{
			Code:    "zero_topic_chapter",
			Message: "This chapter has no topics.",
		})
	}
	return warnings, nil
}

func (r *CurriculumConfigRepo) previewWarnings(ctx context.Context, query curriculumconfig.ImpactQuery) ([]curriculumconfig.Warning, error) {
	var warnings []curriculumconfig.Warning
	if query.IsInSyllabus {
		var duplicateCoverageCount int
		var grade, subject string
		if err := r.db.QueryRowContext(ctx, `
			SELECT COUNT(*), selected.grade, selected.subject
			FROM (
				SELECT selected_grade.number::text AS grade, COALESCE(selected_subject_name.name, '') AS subject
				FROM chapter selected_chapter
				JOIN grade selected_grade ON selected_grade.id = selected_chapter.grade_id
				JOIN subject selected_subject ON selected_subject.id = selected_chapter.subject_id
				LEFT JOIN LATERAL (
					SELECT elem->>'subject' AS name
					FROM jsonb_array_elements(selected_subject.name) elem
					WHERE elem->>'lang_code' = 'en'
					LIMIT 1
				) selected_subject_name ON true
				WHERE selected_chapter.id = $1
			) selected,
			lms_chapter_exam_configs other
			JOIN chapter other_chapter ON other_chapter.id = other.chapter_id
			JOIN grade other_grade ON other_grade.id = other_chapter.grade_id
			JOIN subject other_subject ON other_subject.id = other_chapter.subject_id
			LEFT JOIN LATERAL (
				SELECT elem->>'subject' AS name
				FROM jsonb_array_elements(other_subject.name) elem
				WHERE elem->>'lang_code' = 'en'
				LIMIT 1
			) other_subject_name ON true
			WHERE other.exam_track = $2
			  AND other.is_in_syllabus = true
			  AND other_grade.number::text = selected.grade
			  AND COALESCE(other_subject_name.name, '') = selected.subject
			  AND other.coverage_sequence = $3
			  AND ($4::bigint = 0 OR other.id <> $4)
			GROUP BY selected.grade, selected.subject
		`, query.ChapterID, query.ExamTrack, query.CoverageSequence, query.ConfigID).Scan(&duplicateCoverageCount, &grade, &subject); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if duplicateCoverageCount > 0 {
			warnings = append(warnings, curriculumconfig.Warning{
				Code:    "duplicate_coverage_order",
				Message: fmt.Sprintf("Another in-syllabus LMS Chapter Exam Config uses coverage order %d for %s Grade %s %s.", query.CoverageSequence, examTrackLabel(query.ExamTrack), grade, subject),
			})
		}
		if query.PrescribedMinutes == 0 {
			warnings = append(warnings, curriculumconfig.Warning{
				Code:    "zero_minutes_in_syllabus",
				Message: "This in-syllabus row has zero prescribed minutes.",
			})
		}
	}
	var topicCount int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(t.id)::int AS topic_count
		FROM chapter ch
		LEFT JOIN topic t ON t.chapter_id = ch.id
		WHERE ch.id = $1
	`, query.ChapterID).Scan(&topicCount); err != nil {
		return nil, err
	}
	if topicCount == 0 {
		warnings = append(warnings, curriculumconfig.Warning{
			Code:    "zero_topic_chapter",
			Message: "This chapter has no topics.",
		})
	}
	return warnings, nil
}

func mapCreateError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return errors.New("duplicate LMS Chapter Exam Config already exists for this chapter and exam track")
		case "23503":
			return errors.New("chapter does not exist")
		case "23514":
			switch pqErr.Constraint {
			case "lms_chapter_exam_configs_exam_track_check":
				return errors.New("exam track is invalid")
			case "lms_chapter_exam_configs_prescribed_minutes_check":
				return errors.New("prescribed minutes must be non-negative")
			case "lms_chapter_exam_configs_coverage_sequence_check":
				return errors.New("coverage order must be positive")
			case "lms_chapter_exam_configs_out_of_syllabus_minutes_check":
				return errors.New("out-of-syllabus rows must have zero prescribed minutes")
			}
		}
	}
	return err
}

func mapEditError(err error) error {
	if err == sql.ErrNoRows {
		return curriculumconfig.ErrStaleLock
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23514":
			switch pqErr.Constraint {
			case "lms_chapter_exam_configs_prescribed_minutes_check":
				return errors.New("prescribed minutes must be non-negative")
			case "lms_chapter_exam_configs_coverage_sequence_check":
				return errors.New("coverage order must be positive")
			case "lms_chapter_exam_configs_out_of_syllabus_minutes_check":
				return errors.New("out-of-syllabus rows must have zero prescribed minutes")
			}
		}
	}
	return err
}

func examTrackLabel(value string) string {
	switch value {
	case "jee_main":
		return "JEE Main"
	case "jee_advanced":
		return "JEE Advanced"
	case "neet":
		return "NEET"
	default:
		return value
	}
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

func normalizeExamTrack(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "jee_advanced":
		return "jee_advanced"
	case "neet":
		return "neet"
	default:
		return "jee_main"
	}
}

func validateCreateInput(input curriculumconfig.CreateInput) error {
	var problems []string
	if input.ChapterID < 1 {
		problems = append(problems, "chapter id must be positive")
	}
	if !isAllowedExamTrack(input.ExamTrack) {
		problems = append(problems, "exam track is invalid")
	}
	if input.PrescribedMinutes < 0 {
		problems = append(problems, "prescribed minutes must be non-negative")
	}
	if input.CoverageSequence < 1 {
		problems = append(problems, "coverage order must be positive")
	}
	if !input.IsInSyllabus && input.PrescribedMinutes != 0 {
		problems = append(problems, "out-of-syllabus rows must have zero prescribed minutes")
	}
	if strings.TrimSpace(input.AdminEmail) == "" {
		problems = append(problems, "admin email is required")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validateEditInput(input curriculumconfig.EditInput) error {
	var problems []string
	missingLock := strings.TrimSpace(input.LockToken) == ""
	if input.ID < 1 {
		problems = append(problems, "config id must be positive")
	}
	if input.PrescribedMinutes < 0 {
		problems = append(problems, "prescribed minutes must be non-negative")
	}
	if input.CoverageSequence < 1 {
		problems = append(problems, "coverage order must be positive")
	}
	if !input.IsInSyllabus && input.PrescribedMinutes != 0 {
		problems = append(problems, "out-of-syllabus rows must have zero prescribed minutes")
	}
	if missingLock {
		problems = append(problems, "lock token is required")
	}
	if strings.TrimSpace(input.AdminEmail) == "" {
		problems = append(problems, "admin email is required")
	}
	if len(problems) == 0 {
		return nil
	}
	message := strings.Join(problems, "; ")
	if missingLock {
		return fmt.Errorf("%w: %s", curriculumconfig.ErrStaleLock, message)
	}
	return errors.New(message)
}

func validateRemoveInput(input curriculumconfig.RemoveInput) error {
	var problems []string
	missingLock := strings.TrimSpace(input.LockToken) == ""
	if input.ID < 1 {
		problems = append(problems, "config id must be positive")
	}
	if missingLock {
		problems = append(problems, "lock token is required")
	}
	if strings.TrimSpace(input.AdminEmail) == "" {
		problems = append(problems, "admin email is required")
	}
	if len(problems) == 0 {
		return nil
	}
	message := strings.Join(problems, "; ")
	if missingLock {
		return fmt.Errorf("%w: %s", curriculumconfig.ErrStaleLock, message)
	}
	return errors.New(message)
}

func isAllowedExamTrack(value string) bool {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "jee_main", "jee_advanced", "neet":
		return true
	default:
		return false
	}
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
