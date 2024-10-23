package main

import (
	"net/http"
	"testing"

	"github.com/avantifellows/nex-gen-cms/config"
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
	mux    *http.ServeMux
	routes map[string]http.HandlerFunc
}

func NewMockServeMux() *MockServeMux {
	return &MockServeMux{
		mux:    http.NewServeMux(),
		routes: make(map[string]http.HandlerFunc),
	}
}

func (m *MockServeMux) Handle(pattern string, handler http.Handler) {
	m.mux.Handle(pattern, handler)
}

func (m *MockServeMux) HandleFunc(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
	m.routes[pattern] = handlerFunc
	m.mux.HandleFunc(pattern, handlerFunc)
}

func TestSetup(t *testing.T) {
	mockConfig := new(MockConfig)
	mockConfig.On("LoadEnv", mock.Anything).Return(nil)

	// Create a new MockServeMux to capture registered routes
	mockServeMux := NewMockServeMux()
	expectedRoutes := []string{
		"/", "/modules", "/books", "/major-tests", "/add-chapter",
		"/chapters", "/api/curriculums", "/api/grades", "/api/subjects",
		"/api/chapters", "/edit-chapter", "/update-chapter", "/create-chapter", "/delete-chapter",
	}

	setup(mockConfig, mockServeMux)

	mockConfig.AssertCalled(t, "LoadEnv", mock.Anything)

	registeredRoutes := mockServeMux.routes
	// Assert that all expected routes are registered
	for _, route := range expectedRoutes {
		_, ok := registeredRoutes[route]
		assert.True(t, ok, "Route not registered: "+route)
	}

	// Assert the number of registered routes
	assert.Equal(t, len(expectedRoutes), len(registeredRoutes), "Unexpected number of routes registered")
}
