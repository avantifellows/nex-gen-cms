package config

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEnv is a mock of Env which will implement EnvLoader for testing
type MockEnv struct {
	mock.Mock
}

// MockEnv is implementing EnvLoader here
func (m *MockEnv) Load() error {
	args := m.Called()
	return args.Error(0)
}

// Test for LoadEnv when Load() succeeds
func TestLoadEnv_Success(t *testing.T) {
	// Setup the mock & its expected behavior
	mockEnv := new(MockEnv)
	mockEnv.On("Load").Return(nil)

	// Call LoadEnv
	LoadEnv(mockEnv)

	// Assert that Load() was called
	mockEnv.AssertCalled(t, "Load")
}

// Test for LoadEnv when Load() fails
func TestLoadEnv_Failure(t *testing.T) {
	// Setup the mock to return an error
	mockEnv := new(MockEnv)
	mockEnv.On("Load").Return(fmt.Errorf("failed to load env"))

	// Capture the log output to verify fatal error
	var logOutput bytes.Buffer
	log.SetOutput(&logOutput)
	// Restore original log output
	defer log.SetOutput(os.Stderr)

	// Override fatalf to panic instead of calling log.Fatalf
	originalFatalf := fatalf
	fatalf = func(format string, v ...any) {
		log.Printf(format, v...)
		// Panic instead of os.Exit
		panic(fmt.Sprintf(format, v...))
	}
	// Restore original fatalf after test
	defer func() { fatalf = originalFatalf }()

	// Expect the test to fail due to log.Fatalf()
	assert.Panics(t, func() {
		LoadEnv(mockEnv)
	})

	// Verify that the log output contains the expected error message
	logMsg := logOutput.String()
	assert.Contains(t, logMsg, "Error loading .env file")
}

func TestGetEnv_WhenSet(t *testing.T) {
	key := "TEST_ENV"
	expectedValue := "some_value"
	os.Setenv(key, expectedValue)
	defer os.Unsetenv(key)

	result := GetEnv(key, "default_value")
	if result != expectedValue {
		t.Errorf("GetEnv(%s, default_value) = %s; want %s", key, result, expectedValue)
	}
}

func TestGetEnv_WhenNotSet(t *testing.T) {
	key := "TEST_ENV"
	defaultValue := "default_value"

	result := GetEnv(key, defaultValue)
	if result != defaultValue {
		t.Errorf("GetEnv(%s, default_value) = %s; want %s", key, result, defaultValue)
	}
}
