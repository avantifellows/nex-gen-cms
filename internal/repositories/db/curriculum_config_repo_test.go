package db

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurriculumConfigReadinessSucceedsWhenSchemaContractIsPresent(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()
	expectSuccessfulReadinessQueries(mock, nil)

	readiness, err := NewCurriculumConfigRepo(database).SchemaReadiness(context.Background())

	require.NoError(t, err)
	assert.True(t, readiness.Ready)
	assert.True(t, readiness.MutationReady)
	assert.Empty(t, readiness.Reasons)
	assert.Empty(t, readiness.MutationReasons)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigReadinessReportsMissingStructuralColumns(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()
	expectSuccessfulReadinessQueries(mock, map[string]struct{}{"lms_chapter_exam_configs.updated_at": {}})

	readiness, err := NewCurriculumConfigRepo(database).SchemaReadiness(context.Background())

	require.NoError(t, err)
	assert.False(t, readiness.Ready)
	assert.False(t, readiness.MutationReady)
	assert.Contains(t, readiness.Reasons, "missing column lms_chapter_exam_configs.updated_at")
	assert.Contains(t, readiness.MutationReasons, "missing column lms_chapter_exam_configs.updated_at")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigReadinessReportsMissingConstraintsAndIndexes(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()
	expectColumnQuery(mock, nil)
	expectConstraintQueryExcept(mock, map[string]struct{}{
		"lms_chapter_exam_configs_exam_track_check": {},
	})
	expectIndexQueryExcept(mock, map[string]struct{}{
		"lms_curriculum_logs_active_scope_index": {},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT chapter_id, exam_track, COUNT(*)")).
		WillReturnRows(sqlmock.NewRows([]string{"chapter_id", "exam_track", "count"}))

	readiness, err := NewCurriculumConfigRepo(database).SchemaReadiness(context.Background())

	require.NoError(t, err)
	assert.False(t, readiness.Ready)
	assert.False(t, readiness.MutationReady)
	assert.Contains(t, readiness.Reasons, "missing constraint lms_chapter_exam_configs_exam_track_check")
	assert.Contains(t, readiness.Reasons, "missing index lms_curriculum_logs_active_scope_index")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigReadinessSeparatesDuplicateDataFromStructuralReadiness(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()
	expectColumnQuery(mock, nil)
	expectConstraintQuery(mock)
	expectIndexQuery(mock)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT chapter_id, exam_track, COUNT(*)")).
		WillReturnRows(sqlmock.NewRows([]string{"chapter_id", "exam_track", "count"}).AddRow(44, "jee_main", 2))

	readiness, err := NewCurriculumConfigRepo(database).SchemaReadiness(context.Background())

	require.NoError(t, err)
	assert.True(t, readiness.Ready)
	assert.False(t, readiness.MutationReady)
	assert.Contains(t, readiness.MutationReasons, "duplicate LMS Chapter Exam Config rows for chapter_id=44 exam_track=jee_main")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigReadinessCachesSuccessAndRetriesFailure(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()
	expectSuccessfulReadinessQueries(mock, map[string]struct{}{"lms_chapter_exam_configs.updated_at": {}})
	expectSuccessfulReadinessQueries(mock, nil)

	repo := NewCurriculumConfigRepo(database)
	first, err := repo.SchemaReadiness(context.Background())
	require.NoError(t, err)
	assert.False(t, first.Ready)

	second, err := repo.SchemaReadiness(context.Background())
	require.NoError(t, err)
	assert.True(t, second.Ready)

	third, err := repo.SchemaReadiness(context.Background())
	require.NoError(t, err)
	assert.True(t, third.Ready)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigListUsesDefaultFiltersAndMapsJoinedRows(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	updatedAt := time.Date(2026, 6, 3, 9, 30, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*)")).
		WithArgs("jee_main", true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("jsonb_array_elements(ch.name)")).
		WithArgs("jee_main", true, 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}).AddRow(int64(12), int64(44), "MATH-001", "Quadratic Equations", "11", "Mathematics", "jee_main", true, 90, 7, "admin@avantifellows.org", updatedAt, "14983"))

	result, err := NewCurriculumConfigRepo(database).List(context.Background(), curriculumconfig.ListQuery{})

	require.NoError(t, err)
	assert.Equal(t, 1, result.TotalRows)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 1, result.TotalPages)
	require.Len(t, result.Rows, 1)
	row := result.Rows[0]
	assert.Equal(t, int64(12), row.ID)
	assert.Equal(t, int64(44), row.ChapterID)
	assert.Equal(t, "MATH-001", row.ChapterCode)
	assert.Equal(t, "Quadratic Equations", row.ChapterName)
	assert.Equal(t, "11", row.Grade)
	assert.Equal(t, "Mathematics", row.Subject)
	assert.Equal(t, "jee_main", row.ExamTrack)
	assert.True(t, row.IsInSyllabus)
	assert.Equal(t, 90, row.PrescribedMinutes)
	assert.Equal(t, "1.5 hours", row.PrescribedHours)
	assert.Equal(t, 7, row.CoverageSequence)
	assert.Equal(t, "admin@avantifellows.org", row.UpdatedByEmail)
	assert.Equal(t, updatedAt, row.UpdatedAt)
	assert.Equal(t, "14983", row.LockToken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigListReturnsEmptyFirstPageWhenNoRowsMatch(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*)")).
		WithArgs("jee_main", true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY c.exam_track ASC, g.number ASC, COALESCE(sn.name, '') ASC, c.coverage_sequence ASC, ch.code ASC, COALESCE(chn.name, '') ASC, c.id ASC")).
		WithArgs("jee_main", true, 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}))

	result, err := NewCurriculumConfigRepo(database).List(context.Background(), curriculumconfig.ListQuery{})

	require.NoError(t, err)
	assert.Empty(t, result.Rows)
	assert.Equal(t, 0, result.TotalRows)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.TotalPages)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigListAppliesFiltersPaginationAndTotals(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*)")).
		WithArgs("neet", "12", "Chemistry", "77", "%organic%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(42))
	mock.ExpectQuery(regexp.QuoteMeta("(ch.code ILIKE $5 OR COALESCE(chn.name, '') ILIKE $5)")).
		WithArgs("neet", "12", "Chemistry", "77", "%organic%", 10, 20).
		WillReturnRows(emptyCurriculumConfigListRows())

	result, err := NewCurriculumConfigRepo(database).List(context.Background(), curriculumconfig.ListQuery{
		ExamTrack:      "neet",
		Grade:          "12",
		Subject:        "Chemistry",
		Search:         "organic",
		ChapterID:      "77",
		SyllabusStatus: "all",
		Page:           3,
		Limit:          10,
		Sort:           "chapter_name",
		Direction:      "desc",
	})

	require.NoError(t, err)
	assert.Equal(t, 42, result.TotalRows)
	assert.Equal(t, 3, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 5, result.TotalPages)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigListNormalizesInvalidQueryBeforeSQL(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*)")).
		WithArgs("jee_main", true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY c.exam_track ASC, g.number ASC, COALESCE(sn.name, '') ASC, c.coverage_sequence ASC, ch.code ASC, COALESCE(chn.name, '') ASC, c.id ASC")).
		WithArgs("jee_main", true, 50, 0).
		WillReturnRows(emptyCurriculumConfigListRows())

	result, err := NewCurriculumConfigRepo(database).List(context.Background(), curriculumconfig.ListQuery{
		ExamTrack:      "bad",
		Grade:          "-1",
		Subject:        "all",
		ChapterID:      "abc",
		SyllabusStatus: "bad",
		Page:           -4,
		Limit:          999,
		Sort:           "unsafe",
		Direction:      "sideways",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 50, result.Limit)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigListSupportsDescendingCurriculumSortWithTieBreakers(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*)")).
		WithArgs("neet", true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY c.exam_track DESC, g.number DESC, COALESCE(sn.name, '') DESC, c.coverage_sequence DESC, ch.code DESC, COALESCE(chn.name, '') DESC, c.id ASC")).
		WithArgs("neet", true, 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}))

	_, err := NewCurriculumConfigRepo(database).List(context.Background(), curriculumconfig.ListQuery{
		ExamTrack: "neet",
		Sort:      "curriculum",
		Direction: "desc",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigListSortsWhitelistedColumnWithDeterministicTieBreakers(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*)")).
		WithArgs("jee_advanced", true).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("ORDER BY c.updated_at DESC, c.exam_track ASC, g.number ASC, COALESCE(sn.name, '') ASC, c.coverage_sequence ASC, ch.code ASC, COALESCE(chn.name, '') ASC, c.id ASC")).
		WithArgs("jee_advanced", true, 100, 0).
		WillReturnRows(emptyCurriculumConfigListRows())

	_, err := NewCurriculumConfigRepo(database).List(context.Background(), curriculumconfig.ListQuery{
		ExamTrack: "jee_advanced",
		Limit:     100,
		Sort:      "updated_at",
		Direction: "desc",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigFilterOptionsComeFromDirectPostgresReads(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT value, label")).
		WillReturnRows(sqlmock.NewRows([]string{"value", "label"}).
			AddRow("11", "Grade 11").
			AddRow("12", "Grade 12"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT s.id::text AS value")).
		WillReturnRows(sqlmock.NewRows([]string{"value", "label"}).
			AddRow("2", "Chemistry").
			AddRow("3", "Physics"))

	options, err := NewCurriculumConfigRepo(database).FilterOptions(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []curriculumconfig.Option{
		{Value: "jee_main", Label: "JEE Main"},
		{Value: "jee_advanced", Label: "JEE Advanced"},
		{Value: "neet", Label: "NEET"},
	}, options.ExamTracks)
	assert.Equal(t, []curriculumconfig.Option{
		{Value: "11", Label: "Grade 11"},
		{Value: "12", Label: "Grade 12"},
	}, options.Grades)
	assert.Equal(t, []curriculumconfig.Option{
		{Value: "2", Label: "Chemistry"},
		{Value: "3", Label: "Physics"},
	}, options.Subjects)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigChapterOptionsIncludeDisplayTopicAndDuplicateMetadata(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("existing.id AS existing_config_id")).
		WithArgs("jee_main", "11", "Mathematics", "%quadratic%").
		WillReturnRows(sqlmock.NewRows([]string{
			"chapter_id", "chapter_code", "chapter_name", "grade", "subject", "topic_count",
			"existing_config_id", "existing_in_syllabus", "existing_exam_track",
		}).
			AddRow(int64(44), "MATH-001", "Quadratic Equations", "11", "Mathematics", 0, int64(12), true, "jee_main").
			AddRow(int64(45), "MATH-002", "Sequences", "11", "Mathematics", 3, nil, nil, nil))

	options, err := NewCurriculumConfigRepo(database).ChapterOptions(context.Background(), curriculumconfig.ChapterOptionsQuery{
		ExamTrack: "jee_main",
		Grade:     "11",
		Subject:   "Mathematics",
		Search:    "quadratic",
	})

	require.NoError(t, err)
	require.Len(t, options, 2)
	assert.Equal(t, curriculumconfig.ChapterOption{
		ChapterID:           44,
		ChapterCode:         "MATH-001",
		ChapterName:         "Quadratic Equations",
		Grade:               "11",
		Subject:             "Mathematics",
		TopicCount:          0,
		ExistingConfigID:    int64Ptr(12),
		ExistingInSyllabus:  boolPtr(true),
		ExistingExamTrack:   "jee_main",
		HasZeroTopicWarning: true,
		HasDuplicateConfig:  true,
	}, options[0])
	assert.Equal(t, curriculumconfig.ChapterOption{
		ChapterID:   45,
		ChapterCode: "MATH-002",
		ChapterName: "Sequences",
		Grade:       "11",
		Subject:     "Mathematics",
		TopicCount:  3,
	}, options[1])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigCreateValidatesPositiveAndOutOfSyllabusInputs(t *testing.T) {
	database, _, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	_, err := NewCurriculumConfigRepo(database).Create(context.Background(), curriculumconfig.CreateInput{
		ChapterID:         0,
		ExamTrack:         "bad",
		IsInSyllabus:      false,
		PrescribedMinutes: 30,
		CoverageSequence:  0,
		AdminEmail:        "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chapter id must be positive")
	assert.Contains(t, err.Error(), "exam track is invalid")
	assert.Contains(t, err.Error(), "coverage order must be positive")
	assert.Contains(t, err.Error(), "out-of-syllabus rows must have zero prescribed minutes")
	assert.Contains(t, err.Error(), "admin email is required")
}

func TestCurriculumConfigCreateInsertsAuditFieldsAndReturnsWarningsAndImpact(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	updatedAt := time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM lms_chapter_exam_configs")).
		WithArgs(int64(44), "jee_main").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO lms_chapter_exam_configs")).
		WithArgs(int64(44), "jee_main", true, 0, 7, "admin@avantifellows.org").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}).AddRow(int64(99), int64(44), "MATH-001", "Quadratic Equations", "11", "Mathematics", "jee_main", true, 0, 7, "admin@avantifellows.org", updatedAt, "15001"))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM lms_chapter_exam_configs other")).
		WithArgs(int64(99), "jee_main", "11", "Mathematics", 7).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(t.id)::int AS topic_count")).
		WithArgs(int64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"topic_count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM school sc")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(6))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(DISTINCT log.id) FROM lms_curriculum_logs log")).
		WithArgs(int64(44), "jee_main").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM lms_curriculum_chapter_completions completion")).
		WithArgs(int64(44), "jee_main").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	result, err := NewCurriculumConfigRepo(database).Create(context.Background(), curriculumconfig.CreateInput{
		ChapterID:         44,
		ExamTrack:         "jee_main",
		IsInSyllabus:      true,
		PrescribedMinutes: 0,
		CoverageSequence:  7,
		AdminEmail:        "admin@avantifellows.org",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Row)
	assert.Equal(t, int64(99), result.Row.ID)
	assert.Equal(t, "admin@avantifellows.org", result.Row.UpdatedByEmail)
	assert.Equal(t, "0 hours", result.Row.PrescribedHours)
	assert.Equal(t, curriculumconfig.ImpactResult{SummaryRows: 6, ActiveLogs: 2, ChapterCompletions: 4}, result.Impact)
	assert.Equal(t, []curriculumconfig.Warning{
		{Code: "duplicate_coverage_order", Message: "Another in-syllabus LMS Chapter Exam Config uses coverage order 7 for JEE Main Grade 11 Mathematics."},
		{Code: "zero_minutes_in_syllabus", Message: "This in-syllabus row has zero prescribed minutes."},
		{Code: "zero_topic_chapter", Message: "This chapter has no topics."},
	}, result.Warnings)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigCreateMapsPreDetectedAndConcurrentDuplicatesToSameError(t *testing.T) {
	for _, tc := range []struct {
		name   string
		expect func(sqlmock.Sqlmock)
	}{
		{
			name: "pre-detected",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM lms_chapter_exam_configs")).
					WithArgs(int64(44), "jee_main").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(12)))
			},
		},
		{
			name: "concurrent unique violation",
			expect: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM lms_chapter_exam_configs")).
					WithArgs(int64(44), "jee_main").
					WillReturnRows(sqlmock.NewRows([]string{"id"}))
				mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO lms_chapter_exam_configs")).
					WithArgs(int64(44), "jee_main", true, 60, 7, "admin@avantifellows.org").
					WillReturnError(&pq.Error{Code: "23505", Constraint: "lms_chapter_exam_configs_chapter_track_unique"})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, mock, cleanup := newCurriculumConfigReadinessMock(t)
			defer cleanup()
			tc.expect(mock)

			_, err := NewCurriculumConfigRepo(database).Create(context.Background(), curriculumconfig.CreateInput{
				ChapterID:         44,
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 60,
				CoverageSequence:  7,
				AdminEmail:        "admin@avantifellows.org",
			})

			require.Error(t, err)
			assert.EqualError(t, err, "duplicate LMS Chapter Exam Config already exists for this chapter and exam track")
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCurriculumConfigGetReturnsEditableRowByID(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	updatedAt := time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC)
	expectGetConfigRow(mock, int64(99), updatedAt, true)

	row, err := NewCurriculumConfigRepo(database).Get(context.Background(), 99)

	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, int64(99), row.ID)
	assert.Equal(t, int64(44), row.ChapterID)
	assert.Equal(t, "MATH-001", row.ChapterCode)
	assert.Equal(t, "Quadratic Equations", row.ChapterName)
	assert.Equal(t, "jee_main", row.ExamTrack)
	assert.True(t, row.IsInSyllabus)
	assert.Equal(t, "1 hour", row.PrescribedHours)
	assert.Equal(t, "15001", row.LockToken)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigEditValidatesEditableFieldsAndLockToken(t *testing.T) {
	database, _, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	_, err := NewCurriculumConfigRepo(database).Edit(context.Background(), curriculumconfig.EditInput{
		ID:                0,
		IsInSyllabus:      false,
		PrescribedMinutes: 30,
		CoverageSequence:  0,
		LockToken:         "",
		AdminEmail:        "",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "config id must be positive")
	assert.Contains(t, err.Error(), "coverage order must be positive")
	assert.Contains(t, err.Error(), "out-of-syllabus rows must have zero prescribed minutes")
	assert.ErrorIs(t, err, curriculumconfig.ErrStaleLock)
}

func TestCurriculumConfigEditRejectsInSyllabusToOutOfSyllabusChange(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	expectGetConfigRow(mock, int64(99), time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC), true)

	_, err := NewCurriculumConfigRepo(database).Edit(context.Background(), curriculumconfig.EditInput{
		ID:                99,
		IsInSyllabus:      false,
		PrescribedMinutes: 0,
		CoverageSequence:  7,
		LockToken:         "15001",
		AdminEmail:        "admin@avantifellows.org",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Use the dedicated remove action")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigEditRestoresOutOfSyllabusAndPreservesInsertedAuditFields(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	updatedAt := time.Date(2026, 6, 3, 10, 30, 0, 0, time.UTC)
	expectGetConfigRow(mock, int64(99), time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC), false)
	mock.ExpectQuery(regexp.QuoteMeta("SET is_in_syllabus = $2, prescribed_minutes = $3, coverage_sequence = $4, updated_by_email = $5, updated_at = NOW()")).
		WithArgs(int64(99), true, 0, 8, "admin@avantifellows.org", "15001").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}).AddRow(int64(99), int64(44), "MATH-001", "Quadratic Equations", "11", "Mathematics", "jee_main", true, 0, 8, "admin@avantifellows.org", updatedAt, "15002"))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM lms_chapter_exam_configs other")).
		WithArgs(int64(99), "jee_main", "11", "Mathematics", 8).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(t.id)::int AS topic_count")).
		WithArgs(int64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"topic_count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM school sc")).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(6))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(DISTINCT log.id) FROM lms_curriculum_logs log")).
		WithArgs(int64(44), "jee_main").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM lms_curriculum_chapter_completions completion")).
		WithArgs(int64(44), "jee_main").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))

	result, err := NewCurriculumConfigRepo(database).Edit(context.Background(), curriculumconfig.EditInput{
		ID:                99,
		IsInSyllabus:      true,
		PrescribedMinutes: 0,
		CoverageSequence:  8,
		LockToken:         "15001",
		AdminEmail:        "admin@avantifellows.org",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Row)
	assert.True(t, result.Row.IsInSyllabus)
	assert.Equal(t, 8, result.Row.CoverageSequence)
	assert.Equal(t, "admin@avantifellows.org", result.Row.UpdatedByEmail)
	assert.Equal(t, curriculumconfig.ImpactResult{SummaryRows: 6, ActiveLogs: 2, ChapterCompletions: 4}, result.Impact)
	assert.Equal(t, []curriculumconfig.Warning{
		{Code: "duplicate_coverage_order", Message: "Another in-syllabus LMS Chapter Exam Config uses coverage order 8 for JEE Main Grade 11 Mathematics."},
		{Code: "zero_minutes_in_syllabus", Message: "This in-syllabus row has zero prescribed minutes."},
		{Code: "zero_topic_chapter", Message: "This chapter has no topics."},
	}, result.Warnings)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigEditReturnsUnavailableImpactWithoutBlockingValidUpdate(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	updatedAt := time.Date(2026, 6, 3, 10, 30, 0, 0, time.UTC)
	expectGetConfigRow(mock, int64(99), time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC), false)
	mock.ExpectQuery(regexp.QuoteMeta("SET is_in_syllabus = $2, prescribed_minutes = $3, coverage_sequence = $4, updated_by_email = $5, updated_at = NOW()")).
		WithArgs(int64(99), true, 60, 8, "admin@avantifellows.org", "15001").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}).AddRow(int64(99), int64(44), "MATH-001", "Quadratic Equations", "11", "Mathematics", "jee_main", true, 60, 8, "admin@avantifellows.org", updatedAt, "15002"))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM lms_chapter_exam_configs other")).
		WithArgs(int64(99), "jee_main", "11", "Mathematics", 8).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(t.id)::int AS topic_count")).
		WithArgs(int64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"topic_count"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM school sc")).
		WillReturnError(context.DeadlineExceeded)

	result, err := NewCurriculumConfigRepo(database).Edit(context.Background(), curriculumconfig.EditInput{
		ID:                99,
		IsInSyllabus:      true,
		PrescribedMinutes: 60,
		CoverageSequence:  8,
		LockToken:         "15001",
		AdminEmail:        "admin@avantifellows.org",
	})

	require.NoError(t, err)
	assert.True(t, result.Impact.Unavailable)
	assert.Empty(t, result.Warnings)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigEditRejectsMismatchedLockTokenAsConflict(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	expectGetConfigRow(mock, int64(99), time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC), false)
	mock.ExpectQuery(regexp.QuoteMeta("SET is_in_syllabus = $2, prescribed_minutes = $3, coverage_sequence = $4, updated_by_email = $5, updated_at = NOW()")).
		WithArgs(int64(99), true, 60, 8, "admin@avantifellows.org", "stale").
		WillReturnError(sql.ErrNoRows)

	_, err := NewCurriculumConfigRepo(database).Edit(context.Background(), curriculumconfig.EditInput{
		ID:                99,
		IsInSyllabus:      true,
		PrescribedMinutes: 60,
		CoverageSequence:  8,
		LockToken:         "stale",
		AdminEmail:        "admin@avantifellows.org",
	})

	require.ErrorIs(t, err, curriculumconfig.ErrStaleLock)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCurriculumConfigImpactReturnsWarningsAndUnavailableStateDoesNotBlock(t *testing.T) {
	database, mock, cleanup := newCurriculumConfigReadinessMock(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("selected.grade, selected.subject")).
		WithArgs(int64(44), "jee_main", 7, int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count", "grade", "subject"}).AddRow(1, "11", "Mathematics"))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(t.id)::int AS topic_count")).
		WithArgs(int64(44)).
		WillReturnRows(sqlmock.NewRows([]string{"topic_count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta("COUNT(*) FROM school sc")).
		WillReturnError(context.DeadlineExceeded)

	result, err := NewCurriculumConfigRepo(database).Impact(context.Background(), curriculumconfig.ImpactQuery{
		ChapterID:         44,
		ExamTrack:         "jee_main",
		IsInSyllabus:      true,
		PrescribedMinutes: 0,
		CoverageSequence:  7,
	})

	require.NoError(t, err)
	assert.True(t, result.Unavailable)
	assert.Equal(t, []curriculumconfig.Warning{
		{Code: "duplicate_coverage_order", Message: "Another in-syllabus LMS Chapter Exam Config uses coverage order 7 for JEE Main Grade 11 Mathematics."},
		{Code: "zero_minutes_in_syllabus", Message: "This in-syllabus row has zero prescribed minutes."},
		{Code: "zero_topic_chapter", Message: "This chapter has no topics."},
	}, result.Warnings)
	require.NoError(t, mock.ExpectationsWereMet())
}

func emptyCurriculumConfigListRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
		"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
	})
}

func expectGetConfigRow(mock sqlmock.Sqlmock, id int64, updatedAt time.Time, inSyllabus bool) {
	mock.ExpectQuery(regexp.QuoteMeta("WHERE c.id = $1")).
		WithArgs(id).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "chapter_id", "chapter_code", "chapter_name", "grade", "subject", "exam_track",
			"is_in_syllabus", "prescribed_minutes", "coverage_sequence", "updated_by_email", "updated_at", "lock_token",
		}).AddRow(id, int64(44), "MATH-001", "Quadratic Equations", "11", "Mathematics", "jee_main", inSyllabus, 60, 7, "admin@avantifellows.org", updatedAt, "15001"))
}

func int64Ptr(value int64) *int64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func newCurriculumConfigReadinessMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	database, mock, err := sqlmock.New()
	require.NoError(t, err)
	return database, mock, func() {
		_ = database.Close()
	}
}

func expectSuccessfulReadinessQueries(mock sqlmock.Sqlmock, missingColumns map[string]struct{}) {
	expectColumnQuery(mock, missingColumns)
	expectConstraintQuery(mock)
	expectIndexQuery(mock)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT chapter_id, exam_track, COUNT(*)")).
		WillReturnRows(sqlmock.NewRows([]string{"chapter_id", "exam_track", "count"}))
}

func expectColumnQuery(mock sqlmock.Sqlmock, missing map[string]struct{}) {
	rows := sqlmock.NewRows([]string{"table_name", "column_name"})
	for table, columns := range requiredCurriculumConfigColumns() {
		for _, column := range columns {
			if _, skip := missing[table+"."+column]; skip {
				continue
			}
			rows.AddRow(table, column)
		}
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT table_name, column_name")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)
}

func expectConstraintQuery(mock sqlmock.Sqlmock) {
	expectConstraintQueryExcept(mock, nil)
}

func expectConstraintQueryExcept(mock sqlmock.Sqlmock, missing map[string]struct{}) {
	rows := sqlmock.NewRows([]string{"conname"})
	for _, name := range []string{
		"lms_chapter_exam_configs_exam_track_check",
		"lms_chapter_exam_configs_prescribed_minutes_check",
		"lms_chapter_exam_configs_coverage_sequence_check",
		"lms_chapter_exam_configs_out_of_syllabus_minutes_check",
		"lms_curriculum_logs_exam_track_check",
		"lms_curriculum_logs_duration_minutes_check",
		"lms_curriculum_log_topics_log_topic_unique",
		"lms_curriculum_chapter_completions_exam_track_check",
	} {
		if _, skip := missing[name]; !skip {
			rows.AddRow(name)
		}
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT c.conname")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)
}

func expectIndexQuery(mock sqlmock.Sqlmock) {
	expectIndexQueryExcept(mock, nil)
}

func expectIndexQueryExcept(mock sqlmock.Sqlmock, missing map[string]struct{}) {
	rows := sqlmock.NewRows([]string{"indexname"})
	for _, name := range []string{
		"lms_chapter_exam_configs_chapter_track_unique",
		"lms_chapter_exam_configs_exam_track_chapter_id_index",
		"lms_curriculum_logs_active_scope_index",
		"lms_curriculum_logs_active_scope_date_index",
		"lms_curriculum_logs_log_date_index",
		"lms_curriculum_log_topics_log_id_index",
		"lms_curriculum_log_topics_topic_id_index",
		"lms_curriculum_chapter_completions_active_unique",
		"lms_curriculum_chapter_completions_scope_index",
	} {
		if _, skip := missing[name]; !skip {
			rows.AddRow(name)
		}
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT indexname")).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(rows)
}
