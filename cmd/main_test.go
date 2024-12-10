package main

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/di"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
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

func TestSetup(t *testing.T) {
	mockConfig := new(MockConfig)
	mockConfig.On("LoadEnv", mock.Anything).Return(nil)

	// Create a new MockServeMux to capture registered routes
	mockServeMux := NewMockServeMux()
	appComponentPtr, _ := di.NewAppComponent()
	chaptersHandler := appComponentPtr.ChaptersHandler

	expectedRouteHandlers := []struct {
		pattern string
		handler http.Handler
	}{
		{"/web/", appComponentPtr.CssPathHandler},
		{"/", http.HandlerFunc(handlers.GenericHandler)},
		{"/modules", http.HandlerFunc(handlers.GenericHandler)},
		{"/books", http.HandlerFunc(handlers.GenericHandler)},
		{"/major-tests", http.HandlerFunc(handlers.GenericHandler)},
		{"/add-chapter", http.HandlerFunc(handlers.GenericHandler)},
		{"/chapters", http.HandlerFunc(chaptersHandler.LoadChapters)},
		{"/api/curriculums", http.HandlerFunc(appComponentPtr.CurriculumsHandler.GetCurriculums)},
		{"/api/grades", http.HandlerFunc(appComponentPtr.GradesHandler.GetGrades)},
		{"/api/subjects", http.HandlerFunc(appComponentPtr.SubjectsHandler.GetSubjects)},
		{"/api/chapters", http.HandlerFunc(chaptersHandler.GetChapters)},
		{"/edit-chapter", http.HandlerFunc(chaptersHandler.EditChapter)},
		{"/update-chapter", http.HandlerFunc(chaptersHandler.UpdateChapter)},
		{"/create-chapter", http.HandlerFunc(chaptersHandler.AddChapter)},
		{"/delete-chapter", http.HandlerFunc(chaptersHandler.DeleteChapter)},
	}

	setup(mockConfig, mockServeMux, appComponentPtr)

	// verify if runtime constants are  initialized
	assert.NotEmpty(t, constants.GetHtmlFolderPath(), "Runtime constants are not initilized")

	// verify if environment variables are loaded from .env file
	mockConfig.AssertCalled(t, "LoadEnv", mock.Anything)

	// Assert that expected route handlers are registered
	registeredRouteHandlers := mockServeMux.routeHandlers
	for _, expectedRH := range expectedRouteHandlers {
		pattern := expectedRH.pattern
		registeredHandler, ok := registeredRouteHandlers[pattern]
		assert.True(t, ok, "Route not registered: "+pattern)
		assert.True(t, areHandlersEqual(registeredHandler, expectedRH.handler), "Handler mismatch for pattern %s", pattern)
	}
	assert.Equal(t, len(expectedRouteHandlers), len(registeredRouteHandlers), "Unexpected number of routes registered")
}

// Function to compare function addresses using reflect (Using reflect because functions cannot be compared otherwise)
func areHandlersEqual(h1, h2 http.Handler) bool {
	return reflect.ValueOf(h1).Pointer() == reflect.ValueOf(h2).Pointer()
}
