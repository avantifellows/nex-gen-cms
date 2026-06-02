package db

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
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
