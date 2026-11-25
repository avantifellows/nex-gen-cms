package handlers

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// A test has exactly one curriculum-grade pair, so buildTestData must reject anything else
// before it touches any service (the zero-value handler would panic if it got that far).
func TestBuildTestDataRequiresExactlyOneCurriculumGradePair(t *testing.T) {
	cases := map[string]url.Values{
		"no pair":           {"modal-testType": {"major_test"}, "modal-examType": {"1"}},
		"two pairs":         {"curriculum[]": {"1", "2"}, "grade[]": {"11", "12"}, "modal-testType": {"major_test"}, "modal-examType": {"1"}},
		"curriculum only":   {"curriculum[]": {"1"}, "modal-testType": {"major_test"}, "modal-examType": {"1"}},
		"grade only":        {"grade[]": {"11"}, "modal-testType": {"major_test"}, "modal-examType": {"1"}},
		"two curricula one": {"curriculum[]": {"1", "2"}, "grade[]": {"11"}, "modal-testType": {"major_test"}, "modal-examType": {"1"}},
	}

	for name, form := range cases {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/tests/add-test", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			_, err := (&TestsHandler{}).buildTestData(request)
			if err == nil || !strings.Contains(err.Error(), "exactly one curriculum and grade") {
				t.Fatalf("expected the exactly-one-pair error, got %v", err)
			}
		})
	}
}
