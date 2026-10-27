package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// The Add Test question bank loads a chapter's problems on its own: the topic is only an optional filter.
func TestQuestionBankLoadsByChapterWithOptionalTopic(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_test.html")
	if err != nil {
		t.Fatalf("read add_test.html: %v", err)
	}
	html := strings.ReplaceAll(string(content), "\r\n", "\n")

	for _, want := range []string{
		`hx-get="/api/question-bank/problems?include_paragraph_siblings=true"`,
		`hx-trigger="change[!sessionStorage.getItem('selectedTopic')] from:#chapter-dropdown, change from:(#topic-dropdown, .level-checkbox, #ptype-dropdown)"`,
		`hx-include="#chapter-dropdown, #topic-dropdown,`,
		`hx-sync="this:replace"`,
		`hx-get="/api/topics?view=dropdown-all"`,
		`<option value="">All Topics</option>`,
		`if (this.value || event.isTrusted) sessionStorage.setItem('selectedTopic', this.value);`,
		`if (event.isTrusted) sessionStorage.removeItem('selectedTopic');`,
		`function reloadQuestionBankProblems()`,
		// a new chapter resets the topic at once: its topics load afterwards, and the problems request must not
		// send the old chapter's topic
		`onchange="resetTopicDropdown(); if (this.value) {`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("add_test.html is missing %q", want)
		}
	}
	for _, unwanted := range []string{"/api/topic/problems", "Select Topic", "reloadSelectedTopicProblems"} {
		if strings.Contains(html, unwanted) {
			t.Errorf("add_test.html should not mention %q any more", unwanted)
		}
	}

	// a topic dropdown reload must not wipe the problems that the chapter change has just requested
	start := strings.Index(html, `if (name === "htmx:afterSwap")`)
	end := strings.Index(html, `} else if (name === "htmx:configRequest")`)
	if start < 0 || end < start {
		t.Fatalf("could not find the cascade-clear afterSwap handler")
	}
	handler := html[start:end]
	if strings.Index(handler, "problemList.innerHTML = ''") < strings.Index(handler, `if (targetId === "chapter-dropdown")`) {
		t.Errorf("the problem list must be cleared only inside the chapter-dropdown branch")
	}
	if strings.Count(handler, `if (targetId ===`) != 1 {
		t.Errorf("only the chapter dropdown swap should reset anything")
	}
}

func TestQuestionBankRequestWithoutAChapterIsEmpty(t *testing.T) {
	handler := &ProblemsHandler{}
	for _, chapter := range []string{"", "Select Chapter"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/api/question-bank/problems?chapter-dropdown="+strings.ReplaceAll(chapter, " ", "%20"), nil)
		handler.GetQuestionBankProblems(recorder, request)
		if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
			t.Errorf("chapter %q: got %d with %d bytes, want an empty 200", chapter, recorder.Code, recorder.Body.Len())
		}
	}
}

func TestQuestionBankRequestWithABadChapterOrTopicIsRejected(t *testing.T) {
	handler := &ProblemsHandler{}
	for _, query := range []string{"chapter-dropdown=abc", "chapter-dropdown=5&topic-dropdown=xyz"} {
		recorder := httptest.NewRecorder()
		handler.GetQuestionBankProblems(recorder, httptest.NewRequest(http.MethodGet, "/api/question-bank/problems?"+query, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", query, recorder.Code)
		}
	}
}
