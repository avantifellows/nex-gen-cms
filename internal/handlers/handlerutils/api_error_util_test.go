package handlerutils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
)

func TestWriteRemoteAPIErrorPassesThroughClientError(t *testing.T) {
	rec := httptest.NewRecorder()
	err := &remote_repo.APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Body:       "This test code has already been used.",
	}

	WriteRemoteAPIError(rec, "Error adding test", err)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnprocessableEntity)
	}
	if body := rec.Body.String(); body != "This test code has already been used.\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestWriteRemoteAPIErrorExtractsErrorsMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	err := &remote_repo.APIError{
		StatusCode: http.StatusUnprocessableEntity,
		Body: `{
			"errors": {
				"message": "This test code has already been used.",
				"detail": "Unprocessable Entity"
			}
		}`,
	}

	WriteRemoteAPIError(rec, "Error adding test", err)

	if body := rec.Body.String(); body != "This test code has already been used.\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestWriteRemoteAPIErrorUsesInternalServerErrorForUpstreamFailures(t *testing.T) {
	rec := httptest.NewRecorder()
	err := &remote_repo.APIError{
		StatusCode: http.StatusBadGateway,
		Body:       "upstream unavailable",
	}

	WriteRemoteAPIError(rec, "Error adding test", err)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestWriteRemoteAPIErrorUsesInternalServerErrorForNonAPIErrors(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteRemoteAPIError(rec, "Error adding test", errors.New("network timeout"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
