package remote_repo

import "fmt"

// APIError represents a non-success response from db-service.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("received non-success status code: %d", e.StatusCode)
}
