package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
)

// The dialog templates are found relative to the project root, which the handler tests are not run from.
func renderAddTestDialog(t *testing.T, query url.Values) string {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// walk up to go.mod, so calling this twice in one test still lands on the root
	root := previous
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("no go.mod above %s", previous)
		}
		root = parent
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	constants.InitRuntimeConstant()

	request := httptest.NewRequest(http.MethodGet, "/tests/add-test-dialog?"+query.Encode(), nil)
	recorder := httptest.NewRecorder()
	(&TestsHandler{}).AddTestModal(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", recorder.Code, recorder.Body.String())
	}
	return strings.ReplaceAll(recorder.Body.String(), "\r\n", "\n")
}

func prefilledQuery(mode string) url.Values {
	query := url.Values{}
	query.Set("subtype", "chapter_test")
	query.Set("exam_id", "1")
	query.Set("curriculum_grades", `[{"curriculum_id":1,"grade_id":3},{"curriculum_id":1,"grade_id":4}]`)
	if mode != "" {
		query.Set("mode", mode)
	}
	return query
}

// One dialog serves new, copy and configuration; each mode shows its own title and primary button.
func TestAddTestDialogModes(t *testing.T) {
	cases := []struct {
		name         string
		query        url.Values
		title        string
		hiddenButton int // buttons of the other modes, which the dialog renders hidden
		confirmBtn   bool
	}{
		{"no query is a new test", url.Values{}, "Create New Test", 1, false},
		{"a prefilled query is a copy", prefilledQuery(""), "Copy Test", 1, false},
		{"mode=configuration changes the configuration", prefilledQuery("configuration"), "Test Configuration", 2, true},
		{"an unknown mode falls back to copy", prefilledQuery("bogus"), "Copy Test", 1, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := renderAddTestDialog(t, tc.query)

			if !strings.Contains(html, tc.title) {
				t.Errorf("the dialog should be titled %q", tc.title)
			}
			if got := strings.Count(html, `class="btn-primary hidden"`); got != tc.hiddenButton {
				t.Errorf("hidden primary buttons = %d, want %d", got, tc.hiddenButton)
			}
			if got := strings.Contains(html, `id="configuration-confirm-btn"`); got != tc.confirmBtn {
				t.Errorf("Confirm button present = %v, want %v", got, tc.confirmBtn)
			}
			if got := strings.Contains(html, `id="configuration-rule-status"`); got != tc.confirmBtn {
				t.Errorf("rule status present = %v, want %v", got, tc.confirmBtn)
			}
			if got := strings.Contains(html, "const savedExamId"); got != tc.confirmBtn {
				t.Errorf("configuration script present = %v, want %v", got, tc.confirmBtn)
			}
		})
	}
}

// The configuration dialog starts from what the edit screen holds, and Confirm stays off until something changes.
func TestConfigurationDialogIsPrefilledAndStartsDisabled(t *testing.T) {
	html := renderAddTestDialog(t, prefilledQuery("configuration"))

	for _, want := range []string{
		`const savedTestType = "chapter_test";`,
		`const savedExamId = String( 1 );`,
		`data-selected="3"`,
		`data-selected="4"`,
		`id="configuration-confirm-btn" class="btn-primary" disabled>Confirm</button>`,
		// Confirm only changes the edit screen, so closing the dialog must not leave through history
		`document.getElementById('addTestModal').remove();`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the configuration dialog is missing %q", want)
		}
	}
	if got := strings.Count(html, `data-selected="1"`); got != 2 {
		t.Errorf("both rows should start on curriculum 1, found %d", got)
	}
}

// Copy and new keep leaving through history; only configuration removes the dialog in place.
func TestOnlyConfigurationClosesTheDialogInPlace(t *testing.T) {
	for _, mode := range []string{"", "copy"} {
		html := renderAddTestDialog(t, prefilledQuery(mode))
		if strings.Contains(html, `document.getElementById('addTestModal').remove();`) {
			t.Errorf("mode %q should close through history", mode)
		}
		if !strings.Contains(html, "goBackAfterDelay(0);") {
			t.Errorf("mode %q should go back to close the dialog", mode)
		}
	}
}

// The select handlers (hx-on::after-request) run on htmx:after-request, which htmx fires after
// htmx:afterRequest. Confirm must read the rows after that, or it sees a select before its saved option is picked.
func TestConfigurationDialogRefreshesAfterSelectsPickTheirOption(t *testing.T) {
	html := renderAddTestDialog(t, prefilledQuery("configuration"))

	if !strings.Contains(html, `rowsContainer.addEventListener('htmx:after-request', refreshConfirm);`) {
		t.Errorf("the rows should refresh Confirm on htmx:after-request")
	}
	if strings.Contains(html, `'htmx:afterRequest', refreshConfirm`) {
		t.Errorf("htmx:afterRequest fires before the selects pick their saved option")
	}
}

// The edit screen wires the dialog in place and holds the choice until Save.
func TestEditScreenWiresTheConfigurationDialog(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_test.html")
	if err != nil {
		t.Fatalf("read add_test.html: %v", err)
	}
	html := strings.ReplaceAll(string(content), "\r\n", "\n")

	for _, want := range []string{
		`id="test-configuration-link"`,
		`hx-get="/tests/add-test-dialog" hx-target="#test-configuration-container" hx-swap="innerHTML"`,
		// htmx wraps hx-vals in braces after dropping the js: prefix, so the values have to be spread
		`hx-vals="js:...testConfigurationDialogVals()"`,
		`<div id="test-configuration-container"></div>`,
		`id="test-configuration-store"`,
		`id="test-configuration-pending"`,
		`function applyTestConfiguration(configuration, ruleChange)`,
		`function applyRuleToPage(examId, rule, jeeAdvancedExamId)`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("add_test.html is missing %q", want)
		}
	}

	// opening the dialog adds no history entry, so htmx never swaps the page out from under it
	start := strings.Index(html, `id="test-configuration-link"`)
	end := strings.Index(html[start:], "</button>")
	if start < 0 || end < 0 {
		t.Fatalf("could not find the Test Configuration link")
	}
	if strings.Contains(html[start:start+end], "hx-push-url") {
		t.Errorf("the Test Configuration link must not push a history entry")
	}
}

// Save sends the configuration held on the page, and curriculum_grades only when they changed.
func TestEditSaveSendsTheHeldConfiguration(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_test.html")
	if err != nil {
		t.Fatalf("read add_test.html: %v", err)
	}
	html := strings.ReplaceAll(string(content), "\r\n", "\n")

	start := strings.Index(html, "function buildTestJson(")
	if start < 0 {
		t.Fatalf("could not find buildTestJson")
	}
	body := html[start:]
	if end := strings.Index(body[1:], "\n    function "); end >= 0 {
		body = body[:end+1]
	}

	for _, want := range []string{
		"getTestConfigurationStore()",
		"curriculumGradeKeys(",
		"curriculum_grades",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("buildTestJson should use %q", want)
		}
	}
}
