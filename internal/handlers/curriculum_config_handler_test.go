package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
	readiness curriculumconfig.Readiness
}

func (f *fakeCurriculumConfigRepository) SchemaReadiness(context.Context) (curriculumconfig.Readiness, error) {
	return f.readiness, nil
}

func (f *fakeCurriculumConfigRepository) List(context.Context, curriculumconfig.ListQuery) (curriculumconfig.ListResult, error) {
	return curriculumconfig.ListResult{}, curriculumconfig.ErrNotImplemented
}

func (f *fakeCurriculumConfigRepository) FilterOptions(context.Context) (curriculumconfig.FilterOptions, error) {
	return curriculumconfig.FilterOptions{}, curriculumconfig.ErrNotImplemented
}

func (f *fakeCurriculumConfigRepository) ChapterOptions(context.Context, curriculumconfig.ChapterOptionsQuery) ([]curriculumconfig.ChapterOption, error) {
	return nil, curriculumconfig.ErrNotImplemented
}

func (f *fakeCurriculumConfigRepository) Impact(context.Context, curriculumconfig.ImpactQuery) (curriculumconfig.ImpactResult, error) {
	return curriculumconfig.ImpactResult{}, curriculumconfig.ErrNotImplemented
}

func (f *fakeCurriculumConfigRepository) Create(context.Context, curriculumconfig.CreateInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}

func (f *fakeCurriculumConfigRepository) Edit(context.Context, curriculumconfig.EditInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
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
}

func TestCurriculumConfigHTMXPlaceholderReturnsControlledUnavailableResponse(t *testing.T) {
	handler := NewCurriculumConfigHandler(&fakeCurriculumConfigRepository{
		readiness: curriculumconfig.Readiness{Ready: true, MutationReady: true},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/table", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()

	handler.Table(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "Curriculum Config table is not available in this slice")
	assert.False(t, strings.Contains(rec.Body.String(), "<html"))
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
