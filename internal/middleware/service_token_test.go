package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireServiceToken(t *testing.T) {
	const token = "s3cr3t-token"

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name       string
		envToken   string
		authHeader string
		wantStatus int
	}{
		{"valid token", token, "Bearer " + token, http.StatusOK},
		{"wrong token", token, "Bearer nope", http.StatusUnauthorized},
		{"missing header", token, "", http.StatusUnauthorized},
		{"missing bearer prefix", token, token, http.StatusUnauthorized},
		{"env unset fails closed", "", "Bearer " + token, http.StatusUnauthorized},
		{"env unset and header empty", "", "", http.StatusUnauthorized},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CMS_SERVICE_TOKEN", tc.envToken)

			req := httptest.NewRequest(http.MethodGet, "/api/service/tests", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()

			RequireServiceToken(okHandler).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}
