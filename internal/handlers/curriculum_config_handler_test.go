package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/auth"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/avantifellows/nex-gen-cms/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = os.Chdir("../..")
}

type fakeCurriculumConfigRepository struct {
	readiness            curriculumconfig.Readiness
	listResult           curriculumconfig.ListResult
	options              curriculumconfig.FilterOptions
	chapterOptions       []curriculumconfig.ChapterOption
	getRow               *curriculumconfig.ListRow
	impactResult         curriculumconfig.ImpactResult
	mutationResult       curriculumconfig.MutationResult
	mutationErr          error
	listQueries          []curriculumconfig.ListQuery
	chapterOptionQueries []curriculumconfig.ChapterOptionsQuery
	impactQueries        []curriculumconfig.ImpactQuery
	createInputs         []curriculumconfig.CreateInput
	editIDs              []int64
	editInputs           []curriculumconfig.EditInput
}

func (f *fakeCurriculumConfigRepository) SchemaReadiness(context.Context) (curriculumconfig.Readiness, error) {
	return f.readiness, nil
}

func (f *fakeCurriculumConfigRepository) List(_ context.Context, query curriculumconfig.ListQuery) (curriculumconfig.ListResult, error) {
	f.listQueries = append(f.listQueries, query)
	return f.listResult, nil
}

func (f *fakeCurriculumConfigRepository) FilterOptions(context.Context) (curriculumconfig.FilterOptions, error) {
	return f.options, nil
}

func (f *fakeCurriculumConfigRepository) ChapterOptions(_ context.Context, query curriculumconfig.ChapterOptionsQuery) ([]curriculumconfig.ChapterOption, error) {
	f.chapterOptionQueries = append(f.chapterOptionQueries, query)
	return f.chapterOptions, nil
}

func (f *fakeCurriculumConfigRepository) Get(_ context.Context, id int64) (*curriculumconfig.ListRow, error) {
	f.editIDs = append(f.editIDs, id)
	return f.getRow, f.mutationErr
}

func (f *fakeCurriculumConfigRepository) Impact(_ context.Context, query curriculumconfig.ImpactQuery) (curriculumconfig.ImpactResult, error) {
	f.impactQueries = append(f.impactQueries, query)
	return f.impactResult, nil
}

func (f *fakeCurriculumConfigRepository) Create(_ context.Context, input curriculumconfig.CreateInput) (curriculumconfig.MutationResult, error) {
	f.createInputs = append(f.createInputs, input)
	return f.mutationResult, nil
}

func (f *fakeCurriculumConfigRepository) Edit(_ context.Context, input curriculumconfig.EditInput) (curriculumconfig.MutationResult, error) {
	f.editInputs = append(f.editInputs, input)
	return f.mutationResult, f.mutationErr
}

func (f *fakeCurriculumConfigRepository) RemoveFromSyllabus(context.Context, curriculumconfig.RemoveInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}

func (f *fakeCurriculumConfigRepository) ExportRows(context.Context, curriculumconfig.ListQuery) ([]curriculumconfig.ExportRow, error) {
	return nil, curriculumconfig.ErrNotImplemented
}

func TestCurriculumConfigPageRendersThroughBaseTemplateWhenReady(t *testing.T) {
	constants.InitRuntimeConstant()
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		listResult: curriculumconfig.ListResult{
			Rows: []curriculumconfig.ListRow{{
				ID:                12,
				ChapterID:         44,
				ChapterCode:       "MATH-001",
				ChapterName:       "Quadratic Equations",
				Grade:             "11",
				Subject:           "Mathematics",
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 90,
				PrescribedHours:   "1.5 hours",
				CoverageSequence:  7,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         time.Date(2026, 6, 3, 9, 30, 0, 0, time.UTC),
				LockToken:         "14983",
			}},
			TotalRows:  1,
			Page:       1,
			Limit:      50,
			TotalPages: 1,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config", nil)
	rec := httptest.NewRecorder()

	handler.Page(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<title>Avanti Next Generation CMS</title>")
	assert.Contains(t, body, "Curriculum Config")
	assert.Contains(t, body, `id="curriculum-config-page"`)
	assert.Contains(t, body, `data-hide-global-filters="true"`)
	assert.Contains(t, body, "MATH-001")
	assert.Contains(t, body, "Quadratic Equations")
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/new?exam_track=jee_main`)
	assert.Contains(t, body, `hx-target="#curriculum-config-side-panel"`)
	assert.NotContains(t, body, "14983")
}

func TestCurriculumConfigHTMXTableRendersDefaultRowsAsPartial(t *testing.T) {
	constants.InitRuntimeConstant()
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		listResult: curriculumconfig.ListResult{
			Rows: []curriculumconfig.ListRow{{
				ID:                12,
				ChapterID:         44,
				ChapterCode:       "MATH-001",
				ChapterName:       "Quadratic Equations",
				Grade:             "11",
				Subject:           "Mathematics",
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 90,
				PrescribedHours:   "1.5 hours",
				CoverageSequence:  7,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         time.Date(2026, 6, 3, 9, 30, 0, 0, time.UTC),
				LockToken:         "14983",
			}},
			TotalRows:  1,
			Page:       1,
			Limit:      50,
			TotalPages: 1,
		},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Table(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.listQueries, 1)
	assert.Equal(t, curriculumconfig.ListQuery{
		ExamTrack:      "jee_main",
		SyllabusStatus: "in_syllabus",
		Page:           1,
		Limit:          50,
		Sort:           "curriculum",
		Direction:      "asc",
	}, repo.listQueries[0])
	body := rec.Body.String()
	assert.Contains(t, body, "MATH-001")
	assert.Contains(t, body, "Quadratic Equations")
	assert.Contains(t, body, "44")
	assert.Contains(t, body, "11")
	assert.Contains(t, body, "Mathematics")
	assert.Contains(t, body, "JEE Main")
	assert.Contains(t, body, "In syllabus")
	assert.Contains(t, body, "90 min")
	assert.Contains(t, body, "1.5 hours")
	assert.Contains(t, body, "admin@avantifellows.org")
	assert.NotContains(t, body, "14983")
	assert.False(t, strings.Contains(body, "<html"))
}

func TestCurriculumConfigTableNormalizesInvalidAppliedQueryBeforeListing(t *testing.T) {
	constants.InitRuntimeConstant()
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		listResult: curriculumconfig.ListResult{
			Rows:       nil,
			TotalRows:  0,
			Page:       1,
			Limit:      50,
			TotalPages: 0,
		},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table?exam_track=bad&syllabus_status=bad&page=-3&limit=999&sort=drop_table&dir=sideways&grade=-1&chapter_id=abc", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Table(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.listQueries, 1)
	assert.Equal(t, curriculumconfig.ListQuery{
		ExamTrack:      "jee_main",
		SyllabusStatus: "in_syllabus",
		Page:           1,
		Limit:          50,
		Sort:           "curriculum",
		Direction:      "asc",
	}, repo.listQueries[0])
}

func TestCurriculumConfigPageRendersExplicitApplyFiltersFromOptions(t *testing.T) {
	constants.InitRuntimeConstant()
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		options: curriculumconfig.FilterOptions{
			ExamTracks: []curriculumconfig.Option{{Value: "jee_main", Label: "JEE Main"}, {Value: "neet", Label: "NEET"}},
			Grades:     []curriculumconfig.Option{{Value: "11", Label: "Grade 11"}, {Value: "12", Label: "Grade 12"}},
			Subjects:   []curriculumconfig.Option{{Value: "2", Label: "Chemistry"}},
		},
		listResult: curriculumconfig.ListResult{Rows: nil, TotalRows: 0, Page: 1, Limit: 20, TotalPages: 0},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config?exam_track=neet&grade=12&subject=2&search=organic&chapter_id=77&syllabus_status=all&limit=20&sort=updated_at&dir=desc", nil)
	rec := httptest.NewRecorder()

	handler.Page(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.listQueries, 1)
	assert.Equal(t, curriculumconfig.ListQuery{
		ExamTrack:      "neet",
		Grade:          "12",
		Subject:        "2",
		Search:         "organic",
		ChapterID:      "77",
		SyllabusStatus: "all",
		Page:           1,
		Limit:          20,
		Sort:           "updated_at",
		Direction:      "desc",
	}, repo.listQueries[0])
	body := rec.Body.String()
	assert.Contains(t, body, `id="curriculum-config-filter-form"`)
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/table"`)
	assert.Contains(t, body, `hx-target="#curriculum-config-table"`)
	assert.Contains(t, body, `<button type="submit"`)
	assert.Contains(t, body, `<option value="neet" selected>NEET</option>`)
	assert.Contains(t, body, `<option value="12" selected>Grade 12</option>`)
	assert.Contains(t, body, `<option value="2" selected>Chemistry</option>`)
	assert.Contains(t, body, `name="search" value="organic"`)
	assert.Contains(t, body, `name="chapter_id" value="77"`)
	assert.NotContains(t, body, `select id="curriculum-config-exam-track" name="exam_track" hx-get`)
	assert.NotContains(t, body, `input id="curriculum-config-search" name="search" hx-get`)
}

func TestCurriculumConfigTablePreservesAppliedFiltersForPaginationPageSizeAndSorting(t *testing.T) {
	constants.InitRuntimeConstant()
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		listResult: curriculumconfig.ListResult{
			Rows: []curriculumconfig.ListRow{{
				ID:                12,
				ChapterID:         77,
				ChapterCode:       "CHEM-001",
				ChapterName:       "Organic Chemistry",
				Grade:             "12",
				Subject:           "Chemistry",
				ExamTrack:         "neet",
				IsInSyllabus:      true,
				PrescribedMinutes: 60,
				PrescribedHours:   "1 hour",
				CoverageSequence:  1,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         time.Date(2026, 6, 3, 9, 30, 0, 0, time.UTC),
			}},
			TotalRows:  42,
			Page:       2,
			Limit:      20,
			TotalPages: 3,
		},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table?exam_track=neet&grade=12&subject=Chemistry&search=organic&chapter_id=77&syllabus_status=all&page=2&limit=20&sort=updated_at&dir=desc", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Table(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `id="curriculum-config-applied-filters"`)
	assert.Contains(t, body, `name="exam_track" value="neet"`)
	assert.Contains(t, body, `name="grade" value="12"`)
	assert.Contains(t, body, `name="subject" value="Chemistry"`)
	assert.Contains(t, body, `name="search" value="organic"`)
	assert.Contains(t, body, `name="chapter_id" value="77"`)
	assert.Contains(t, body, `name="syllabus_status" value="all"`)
	assert.Contains(t, body, `name="page" value="2"`)
	assert.Contains(t, body, `name="limit" value="20"`)
	assert.Contains(t, body, `name="sort" value="updated_at"`)
	assert.Contains(t, body, `name="dir" value="desc"`)
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/table?exam_track=neet&amp;grade=12&amp;subject=Chemistry&amp;search=organic&amp;chapter_id=77&amp;syllabus_status=all&amp;page=1&amp;limit=20&amp;sort=updated_at&amp;dir=desc"`)
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/table?exam_track=neet&amp;grade=12&amp;subject=Chemistry&amp;search=organic&amp;chapter_id=77&amp;syllabus_status=all&amp;page=3&amp;limit=20&amp;sort=updated_at&amp;dir=desc"`)
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/table?exam_track=neet&amp;grade=12&amp;subject=Chemistry&amp;search=organic&amp;chapter_id=77&amp;syllabus_status=all&amp;page=1&amp;limit=20&amp;sort=chapter_code&amp;dir=asc"`)
	assert.Contains(t, body, `id="curriculum-config-refresh"`)
	assert.Contains(t, body, `id="curriculum-config-page-size-form"`)
	assert.Contains(t, body, `<option value="20" selected>20 rows</option>`)
}

func TestCurriculumConfigHTMXTableRendersUsefulEmptyState(t *testing.T) {
	constants.InitRuntimeConstant()
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		listResult: curriculumconfig.ListResult{
			Rows:       nil,
			TotalRows:  0,
			Page:       1,
			Limit:      50,
			TotalPages: 0,
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Table(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "No LMS Chapter Exam Config rows found")
	assert.Contains(t, body, "No in-syllabus JEE Main rows match the current filters.")
	assert.NotContains(t, body, "<html")
}

func TestCurriculumConfigNewRendersAddPanelWithAppliedFiltersPreserved(t *testing.T) {
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/new?exam_track=neet&grade=12&subject=Chemistry&search=organic&chapter_id=77&syllabus_status=all&page=2&limit=20&sort=updated_at&dir=desc", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.New(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `id="curriculum-config-add-panel"`)
	assert.Contains(t, body, `hx-post="/admin/curriculum-config/create"`)
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/chapter-options"`)
	assert.Contains(t, body, `hx-get="/admin/curriculum-config/impact"`)
	assert.Contains(t, body, `name="exam_track" value="neet"`)
	assert.Contains(t, body, `name="grade" value="12"`)
	assert.Contains(t, body, `name="subject" value="Chemistry"`)
	assert.Contains(t, body, `name="search" value="organic"`)
	assert.Contains(t, body, `name="page" value="2"`)
	assert.NotContains(t, body, "<html")
}

func TestCurriculumConfigEditRendersSidePanelWithImmutableIdentityAndAppliedFilters(t *testing.T) {
	updatedAt := time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC)
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		getRow: &curriculumconfig.ListRow{
			ID:                99,
			ChapterID:         44,
			ChapterCode:       "MATH-001",
			ChapterName:       "Quadratic Equations",
			Grade:             "11",
			Subject:           "Mathematics",
			ExamTrack:         "jee_main",
			IsInSyllabus:      false,
			PrescribedMinutes: 0,
			PrescribedHours:   "0 hours",
			CoverageSequence:  7,
			UpdatedByEmail:    "admin@avantifellows.org",
			UpdatedAt:         updatedAt,
			LockToken:         "15001",
		},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/edit?id=99&exam_track=neet&grade=12&subject=Chemistry&search=organic&chapter_id=77&syllabus_status=all&page=2&limit=20&sort=updated_at&dir=desc", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Edit(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []int64{99}, repo.editIDs)
	body := rec.Body.String()
	assert.Contains(t, body, `id="curriculum-config-edit-panel"`)
	assert.Contains(t, body, `hx-post="/admin/curriculum-config/update"`)
	assert.Contains(t, body, "MATH-001")
	assert.Contains(t, body, "Quadratic Equations")
	assert.Contains(t, body, "Chapter ID 44")
	assert.Contains(t, body, "JEE Main")
	assert.NotContains(t, body, `name="chapter_id"`)
	assert.NotContains(t, body, `name="exam_track"`)
	assert.Contains(t, body, `name="id" value="99"`)
	assert.Contains(t, body, `name="lock_token" value="15001"`)
	assert.Contains(t, body, `<option value="out_of_syllabus" selected>Out of syllabus</option>`)
	assert.Contains(t, body, `name="filter_exam_track" value="neet"`)
	assert.Contains(t, body, `name="filter_page" value="2"`)
	assert.NotContains(t, body, "<html")
}

func TestCurriculumConfigChapterOptionsRendersMetadataWarnings(t *testing.T) {
	existingID := int64(12)
	existingInSyllabus := true
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		chapterOptions: []curriculumconfig.ChapterOption{{
			ChapterID:           44,
			ChapterCode:         "MATH-001",
			ChapterName:         "Quadratic Equations",
			Grade:               "11",
			Subject:             "Mathematics",
			TopicCount:          0,
			ExistingConfigID:    &existingID,
			ExistingInSyllabus:  &existingInSyllabus,
			ExistingExamTrack:   "jee_main",
			HasZeroTopicWarning: true,
			HasDuplicateConfig:  true,
		}},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/chapter-options?exam_track=jee_main&grade=11&subject=Mathematics&search=quad", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.ChapterOptions(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.chapterOptionQueries, 1)
	assert.Equal(t, curriculumconfig.ChapterOptionsQuery{ExamTrack: "jee_main", Grade: "11", Subject: "Mathematics", Search: "quad"}, repo.chapterOptionQueries[0])
	body := rec.Body.String()
	assert.Contains(t, body, "MATH-001")
	assert.Contains(t, body, "Quadratic Equations")
	assert.Contains(t, body, "0 topics")
	assert.Contains(t, body, "already has a JEE Main config")
	assert.Contains(t, body, "no topics")
}

func TestCurriculumConfigImpactRendersWarningsCountsAndUnavailableState(t *testing.T) {
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		impactResult: curriculumconfig.ImpactResult{
			SummaryRows:        6,
			ActiveLogs:         2,
			ChapterCompletions: 4,
			Warnings: []curriculumconfig.Warning{{
				Code:    "zero_minutes_in_syllabus",
				Message: "This in-syllabus row has zero prescribed minutes.",
			}},
		},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/impact?chapter_id=44&exam_track=neet&is_in_syllabus=true&prescribed_minutes=0&coverage_sequence=7", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Impact(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.impactQueries, 1)
	assert.Equal(t, curriculumconfig.ImpactQuery{ChapterID: 44, ExamTrack: "neet", IsInSyllabus: true, PrescribedMinutes: 0, CoverageSequence: 7}, repo.impactQueries[0])
	body := rec.Body.String()
	assert.Contains(t, body, "zero prescribed minutes")
	assert.Contains(t, body, "Summary rows: 6")
	assert.Contains(t, body, "Active logs: 2")
	assert.Contains(t, body, "Chapter completions: 4")
}

func TestCurriculumConfigImpactAcceptsConfigIDForEditPreview(t *testing.T) {
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		impactResult: curriculumconfig.ImpactResult{
			SummaryRows: 6,
		},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/impact?config_id=99&chapter_id=44&exam_track=neet&is_in_syllabus=true&prescribed_minutes=0&coverage_sequence=7", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Impact(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.impactQueries, 1)
	assert.Equal(t, curriculumconfig.ImpactQuery{ConfigID: 99, ChapterID: 44, ExamTrack: "neet", IsInSyllabus: true, PrescribedMinutes: 0, CoverageSequence: 7}, repo.impactQueries[0])
}

func TestCurriculumConfigImpactParsesEditSyllabusStatus(t *testing.T) {
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
	}
	handler := NewCurriculumConfigHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/impact?config_id=99&chapter_id=44&exam_track=neet&syllabus_status=out_of_syllabus&prescribed_minutes=0&coverage_sequence=7", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Impact(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.impactQueries, 1)
	assert.False(t, repo.impactQueries[0].IsInSyllabus)
}

func TestCurriculumConfigCreateParsesFormStampsAdminAndRefreshesAppliedFilters(t *testing.T) {
	constants.InitRuntimeConstant()
	createdAt := time.Date(2026, 6, 3, 10, 15, 0, 0, time.UTC)
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		mutationResult: curriculumconfig.MutationResult{
			Row: &curriculumconfig.ListRow{
				ID:                99,
				ChapterID:         44,
				ChapterCode:       "MATH-001",
				ChapterName:       "Quadratic Equations",
				Grade:             "11",
				Subject:           "Mathematics",
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 0,
				PrescribedHours:   "0 hours",
				CoverageSequence:  7,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         createdAt,
			},
			Warnings: []curriculumconfig.Warning{{Code: "zero_minutes_in_syllabus", Message: "This in-syllabus row has zero prescribed minutes."}},
			Impact:   curriculumconfig.ImpactResult{SummaryRows: 6, ActiveLogs: 2, ChapterCompletions: 4},
		},
		listResult: curriculumconfig.ListResult{
			Rows: []curriculumconfig.ListRow{{
				ID:                99,
				ChapterID:         44,
				ChapterCode:       "MATH-001",
				ChapterName:       "Quadratic Equations",
				Grade:             "11",
				Subject:           "Mathematics",
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 0,
				PrescribedHours:   "0 hours",
				CoverageSequence:  7,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         createdAt,
			}},
			TotalRows:  1,
			Page:       2,
			Limit:      20,
			TotalPages: 3,
		},
	}
	handler := NewCurriculumConfigHandler(repo)
	form := url.Values{
		"chapter_id":             {"44"},
		"exam_track":             {"jee_main"},
		"is_in_syllabus":         {"true"},
		"prescribed_minutes":     {"0"},
		"coverage_sequence":      {"7"},
		"filter_exam_track":      {"neet"},
		"filter_grade":           {"12"},
		"filter_subject":         {"Chemistry"},
		"filter_search":          {"organic"},
		"filter_chapter_id":      {"77"},
		"filter_syllabus_status": {"all"},
		"filter_page":            {"2"},
		"filter_limit":           {"20"},
		"filter_sort":            {"updated_at"},
		"filter_dir":             {"desc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/curriculum-config/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(auth.WithSession(req.Context(), &auth.SessionClaims{Email: "admin@avantifellows.org", Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()

	handler.Create(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.createInputs, 1)
	assert.Equal(t, curriculumconfig.CreateInput{
		ChapterID:         44,
		ExamTrack:         "jee_main",
		IsInSyllabus:      true,
		PrescribedMinutes: 0,
		CoverageSequence:  7,
		AdminEmail:        "admin@avantifellows.org",
	}, repo.createInputs[0])
	require.Len(t, repo.listQueries, 1)
	assert.Equal(t, curriculumconfig.ListQuery{
		ExamTrack:      "neet",
		Grade:          "12",
		Subject:        "Chemistry",
		Search:         "organic",
		ChapterID:      "77",
		SyllabusStatus: "all",
		Page:           2,
		Limit:          20,
		Sort:           "updated_at",
		Direction:      "desc",
	}, repo.listQueries[0])
	body := rec.Body.String()
	assert.Contains(t, body, "Created LMS Chapter Exam Config")
	assert.Contains(t, body, "zero prescribed minutes")
	assert.Contains(t, body, "Summary rows: 6")
	assert.Contains(t, body, `id="curriculum-config-row-99"`)
	assert.Contains(t, body, `name="exam_track" value="neet"`)
	assert.Contains(t, body, `name="page" value="2"`)
}

func TestCurriculumConfigUpdateParsesEditableFieldsLockTokenAndRefreshesAppliedFilters(t *testing.T) {
	constants.InitRuntimeConstant()
	updatedAt := time.Date(2026, 6, 3, 10, 30, 0, 0, time.UTC)
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
		mutationResult: curriculumconfig.MutationResult{
			Row: &curriculumconfig.ListRow{
				ID:                99,
				ChapterID:         44,
				ChapterCode:       "MATH-001",
				ChapterName:       "Quadratic Equations",
				Grade:             "11",
				Subject:           "Mathematics",
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 0,
				PrescribedHours:   "0 hours",
				CoverageSequence:  8,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         updatedAt,
			},
			Warnings: []curriculumconfig.Warning{{Code: "zero_minutes_in_syllabus", Message: "This in-syllabus row has zero prescribed minutes."}},
			Impact:   curriculumconfig.ImpactResult{SummaryRows: 6, ActiveLogs: 2, ChapterCompletions: 4},
		},
		listResult: curriculumconfig.ListResult{
			Rows: []curriculumconfig.ListRow{{
				ID:                99,
				ChapterID:         44,
				ChapterCode:       "MATH-001",
				ChapterName:       "Quadratic Equations",
				Grade:             "11",
				Subject:           "Mathematics",
				ExamTrack:         "jee_main",
				IsInSyllabus:      true,
				PrescribedMinutes: 0,
				PrescribedHours:   "0 hours",
				CoverageSequence:  8,
				UpdatedByEmail:    "admin@avantifellows.org",
				UpdatedAt:         updatedAt,
			}},
			TotalRows:  1,
			Page:       2,
			Limit:      20,
			TotalPages: 3,
		},
	}
	handler := NewCurriculumConfigHandler(repo)
	form := url.Values{
		"id":                     {"99"},
		"syllabus_status":        {"in_syllabus"},
		"prescribed_minutes":     {"0"},
		"coverage_sequence":      {"8"},
		"lock_token":             {"15001"},
		"filter_exam_track":      {"neet"},
		"filter_grade":           {"12"},
		"filter_subject":         {"Chemistry"},
		"filter_search":          {"organic"},
		"filter_chapter_id":      {"77"},
		"filter_syllabus_status": {"all"},
		"filter_page":            {"2"},
		"filter_limit":           {"20"},
		"filter_sort":            {"updated_at"},
		"filter_dir":             {"desc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/curriculum-config/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(auth.WithSession(req.Context(), &auth.SessionClaims{Email: "admin@avantifellows.org", Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, repo.editInputs, 1)
	assert.Equal(t, curriculumconfig.EditInput{
		ID:                99,
		IsInSyllabus:      true,
		PrescribedMinutes: 0,
		CoverageSequence:  8,
		LockToken:         "15001",
		AdminEmail:        "admin@avantifellows.org",
	}, repo.editInputs[0])
	require.Len(t, repo.listQueries, 1)
	assert.Equal(t, curriculumconfig.ListQuery{
		ExamTrack:      "neet",
		Grade:          "12",
		Subject:        "Chemistry",
		Search:         "organic",
		ChapterID:      "77",
		SyllabusStatus: "all",
		Page:           2,
		Limit:          20,
		Sort:           "updated_at",
		Direction:      "desc",
	}, repo.listQueries[0])
	body := rec.Body.String()
	assert.Contains(t, body, "Updated LMS Chapter Exam Config")
	assert.Contains(t, body, "zero prescribed minutes")
	assert.Contains(t, body, "Summary rows: 6")
	assert.Contains(t, body, `id="curriculum-config-row-99"`)
	assert.Contains(t, body, `name="exam_track" value="neet"`)
	assert.Contains(t, body, `name="page" value="2"`)
}

func TestCurriculumConfigUpdateRejectsSubmittedImmutableIdentity(t *testing.T) {
	repo := &fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
	}
	handler := NewCurriculumConfigHandler(repo)
	form := url.Values{
		"id":                 {"99"},
		"chapter_id":         {"44"},
		"exam_track":         {"neet"},
		"syllabus_status":    {"in_syllabus"},
		"prescribed_minutes": {"60"},
		"coverage_sequence":  {"8"},
		"lock_token":         {"15001"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/curriculum-config/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(auth.WithSession(req.Context(), &auth.SessionClaims{Email: "admin@avantifellows.org", Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Empty(t, repo.editInputs)
	assert.Contains(t, rec.Body.String(), "Chapter and exam-track identity cannot be changed from the edit form")
}

func TestCurriculumConfigUpdateMapsStaleLockToConflictFeedback(t *testing.T) {
	repo := &fakeCurriculumConfigRepository{
		readiness:   curriculumconfig.Readiness{Ready: true, MutationReady: true},
		mutationErr: curriculumconfig.ErrStaleLock,
	}
	handler := NewCurriculumConfigHandler(repo)
	form := url.Values{
		"id":                 {"99"},
		"syllabus_status":    {"in_syllabus"},
		"prescribed_minutes": {"60"},
		"coverage_sequence":  {"8"},
		"lock_token":         {"stale"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/curriculum-config/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req = req.WithContext(auth.WithSession(req.Context(), &auth.SessionClaims{Email: "admin@avantifellows.org", Role: auth.RoleAdmin}))
	rec := httptest.NewRecorder()

	handler.Update(rec, req)

	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "This LMS Chapter Exam Config changed while you were editing")
}

func TestCurriculumConfigPageShowsControlledUnavailableStateWhenSchemaIsNotReady(t *testing.T) {
	constants.InitRuntimeConstant()
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{
			Ready:   false,
			Reasons: []string{"missing column lms_chapter_exam_configs.updated_at"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config", nil)
	rec := httptest.NewRecorder()

	handler.Page(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Curriculum Config unavailable")
	assert.Contains(t, body, "missing column lms_chapter_exam_configs.updated_at")
}

func TestCurriculumConfigHTMXRequestShowsControlledUnavailableStateWhenSchemaIsNotReady(t *testing.T) {
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{
			Ready:   false,
			Reasons: []string{"missing index lms_chapter_exam_configs_chapter_track_unique"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Table(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "Curriculum Config unavailable")
	assert.Contains(t, body, "missing index lms_chapter_exam_configs_chapter_track_unique")
	assert.NotContains(t, body, "<html")
}

func TestCurriculumConfigEndpointsRequireCMSAdminAccess(t *testing.T) {
	t.Setenv("SESSION_SECRET", "curriculum-config-test-secret")
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
	})
	adminOnly := middleware.RequireRole(auth.RoleAdmin, http.HandlerFunc(handler.Page))

	viewerLogin := httptest.NewRecorder()
	require.NoError(t, auth.IssueSession(viewerLogin, 11, "viewer@avantifellows.org", auth.RoleViewer))
	viewerReq := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config", nil)
	for _, cookie := range viewerLogin.Result().Cookies() {
		viewerReq.AddCookie(cookie)
	}
	viewerRec := httptest.NewRecorder()

	adminOnly.ServeHTTP(viewerRec, viewerReq)

	require.Equal(t, http.StatusForbidden, viewerRec.Code)

	htmxReq := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table", nil)
	htmxReq.Header.Set("HX-Request", "true")
	htmxRec := httptest.NewRecorder()

	middleware.RequireRole(auth.RoleAdmin, http.HandlerFunc(handler.Table)).ServeHTTP(htmxRec, htmxReq)

	require.Equal(t, http.StatusUnauthorized, htmxRec.Code)
	assert.Equal(t, "/login", htmxRec.Header().Get("HX-Redirect"))
}
