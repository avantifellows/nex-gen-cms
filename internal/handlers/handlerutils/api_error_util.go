package handlerutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
)

// WriteRemoteAPIError writes a client-safe error response for db-service failures.
func WriteRemoteAPIError(w http.ResponseWriter, fallbackMessage string, err error) {
	var apiErr *remote_repo.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
		msg := remoteErrorMessage(apiErr.Body, fallbackMessage)
		http.Error(w, msg, apiErr.StatusCode)
		return
	}

	http.Error(w, fmt.Sprintf("%s: %v", fallbackMessage, err), http.StatusInternalServerError)
}

func remoteErrorMessage(body string, fallback string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return fallback
	}

	var parsed struct {
		Errors struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}

	if msg := strings.TrimSpace(parsed.Errors.Message); msg != "" {
		return msg
	}

	return body
}
