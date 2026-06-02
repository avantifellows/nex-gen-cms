package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/di"
	"github.com/avantifellows/nex-gen-cms/internal/auth"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/curriculumconfig"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockConfig struct {
	mock.Mock
}

func (m *MockConfig) LoadEnv(loader config.EnvLoader) {
	m.Called(loader)
}

type MockServeMux struct {
	mux           *http.ServeMux
	routeHandlers map[string]http.Handler
	/**
	  following extra attribute would be required if we don't want to convert handlerFunc parameter
	  to HandlerFunc type in HandleFunc() implemented below for MockServeMux. In that case all expected values
	  & verifications will also be separate in the same way as done for routeHandlers
	*/
	// routeHandlerFuncs map[string]http.HandlerFunc
}

func NewMockServeMux() *MockServeMux {
	return &MockServeMux{
		mux:           http.NewServeMux(),
		routeHandlers: make(map[string]http.Handler),
	}
}

func (m *MockServeMux) Handle(pattern string, handler http.Handler) {
	m.routeHandlers[pattern] = handler
	m.mux.Handle(pattern, handler)
}

func (m *MockServeMux) HandleFunc(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
	m.routeHandlers[pattern] = http.HandlerFunc(handlerFunc)
	m.mux.HandleFunc(pattern, handlerFunc)
}

type routeTestCurriculumConfigRepo struct{}

func (routeTestCurriculumConfigRepo) SchemaReadiness(context.Context) (curriculumconfig.Readiness, error) {
	return curriculumconfig.Readiness{Ready: true, MutationReady: true}, nil
}
func (routeTestCurriculumConfigRepo) List(context.Context, curriculumconfig.ListQuery) (curriculumconfig.ListResult, error) {
	return curriculumconfig.ListResult{}, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) Get(context.Context, int64) (*curriculumconfig.ListRow, error) {
	return nil, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) FilterOptions(context.Context) (curriculumconfig.FilterOptions, error) {
	return curriculumconfig.FilterOptions{}, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) ChapterOptions(context.Context, curriculumconfig.ChapterOptionsQuery) ([]curriculumconfig.ChapterOption, error) {
	return nil, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) Impact(context.Context, curriculumconfig.ImpactQuery) (curriculumconfig.ImpactResult, error) {
	return curriculumconfig.ImpactResult{}, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) Create(context.Context, curriculumconfig.CreateInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) Edit(context.Context, curriculumconfig.EditInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) RemoveFromSyllabus(context.Context, curriculumconfig.RemoveInput) (curriculumconfig.MutationResult, error) {
	return curriculumconfig.MutationResult{}, curriculumconfig.ErrNotImplemented
}
func (routeTestCurriculumConfigRepo) ExportRows(context.Context, curriculumconfig.ListQuery) ([]curriculumconfig.ExportRow, error) {
	return []curriculumconfig.ExportRow{{
		ChapterCode:       "MATH-001",
		ChapterName:       "Quadratic Equations",
		Grade:             "11",
		Subject:           "Mathematics",
		ExamTrack:         "jee_main",
		IsInSyllabus:      true,
		PrescribedMinutes: 60,
		PrescribedHours:   "1 hour",
		CoverageSequence:  1,
		UpdatedByEmail:    "admin@avantifellows.org",
		UpdatedAt:         time.Date(2026, 6, 3, 9, 30, 0, 0, time.UTC),
	}}, nil
}

func TestSetup(t *testing.T) {
	mockConfig := new(MockConfig)
	mockConfig.On("LoadEnv", mock.Anything).Return(nil)

	mockServeMux := NewMockServeMux()
	appComponentPtr := &di.AppComponent{
		CssPathHandler:          http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		CurriculumConfigHandler: handlers.NewCurriculumConfigHandler(routeTestCurriculumConfigRepo{}),
	}

	setup(mockConfig, mockServeMux, appComponentPtr)

	// verify if runtime constants are  initialized
	assert.NotEmpty(t, constants.GetHtmlFolderPath(), "Runtime constants are not initilized")

	// verify if environment variables are loaded from .env file
	mockConfig.AssertCalled(t, "LoadEnv", mock.Anything)

	// Assert that expected route handlers are registered
	registeredRouteHandlers := mockServeMux.routeHandlers
	for _, pattern := range []string{
		"/web/",
		"/",
		"/login",
		"/logout",
		"/admin/users",
		"/chapters",
		"/api/curriculums",
		"/api/grades",
		"/api/subjects",
		"/api/chapters",
		"/tests",
		"/problems",
		"/api/tags",
		"/api/exams",
	} {
		_, ok := registeredRouteHandlers[pattern]
		assert.True(t, ok, "Route not registered: "+pattern)
	}
}

func TestSetupRegistersCurriculumConfigRoutes(t *testing.T) {
	mockConfig := new(MockConfig)
	mockConfig.On("LoadEnv", mock.Anything).Return(nil)

	mockServeMux := NewMockServeMux()
	appComponentPtr := &di.AppComponent{
		CssPathHandler:          http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		CurriculumConfigHandler: handlers.NewCurriculumConfigHandler(routeTestCurriculumConfigRepo{}),
	}

	setup(mockConfig, mockServeMux, appComponentPtr)

	expectedRoutes := []string{
		"/admin/curriculum-config",
		"/admin/curriculum-config/table",
		"/admin/curriculum-config/new",
		"/admin/curriculum-config/edit",
		"/admin/curriculum-config/remove",
		"/admin/curriculum-config/chapter-options",
		"/admin/curriculum-config/impact",
		"/admin/curriculum-config/create",
		"/admin/curriculum-config/update",
		"/admin/curriculum-config/remove-from-syllabus",
		"/admin/curriculum-config/export",
	}
	for _, route := range expectedRoutes {
		_, ok := mockServeMux.routeHandlers[route]
		assert.True(t, ok, "Route not registered: "+route)
	}
}

func TestSetupRegistersWorkingCurriculumConfigExportRoute(t *testing.T) {
	t.Setenv("SESSION_SECRET", "curriculum-config-route-secret")
	mockConfig := new(MockConfig)
	mockConfig.On("LoadEnv", mock.Anything).Return(nil)

	mockServeMux := NewMockServeMux()
	appComponentPtr := &di.AppComponent{
		CssPathHandler:          http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		CurriculumConfigHandler: handlers.NewCurriculumConfigHandler(routeTestCurriculumConfigRepo{}),
	}
	setup(mockConfig, mockServeMux, appComponentPtr)

	login := httptest.NewRecorder()
	assert.NoError(t, auth.IssueSession(login, 1, "admin@avantifellows.org", auth.RoleAdmin))
	req := httptest.NewRequest(http.MethodGet, "/admin/curriculum-config/export", nil)
	for _, cookie := range login.Result().Cookies() {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()

	mockServeMux.routeHandlers["/admin/curriculum-config/export"].ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/csv; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "MATH-001")
}
