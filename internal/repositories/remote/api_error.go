package remote_repo

import "fmt"

// APIError represents a non-success response from db-service.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
	}
	return fmt.Sprintf("received non-success status code: %d", e.StatusCode)
}
