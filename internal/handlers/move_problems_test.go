package handlers

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"
)

// Moving problems asks for a chapter only: the dialog ends there and the Move button follows the chapter.
func TestMoveProblemsDialogAsksForAChapterOnly(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/move_problems_modal.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "7,8"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		// two dropdowns per row (curriculum, grade / subject, chapter), in a dialog narrower than before
		`grid grid-cols-2 gap-4`,
		`w-full max-w-2xl`,
		`class="form-select" hx-get="/api/chapters" onchange="toggleMoveButton()"`,
		`moveBtn.disabled = !chapter.value;`,
		"Then select Chapter.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the move dialog is missing %q", want)
		}
	}
	for _, unwanted := range []string{
		`id="topic-dropdown"`, `name="topic_id"`, "Topic", "beforeMoveProblemsTopicsRequest", "col-span-3", "max-w-3xl",
	} {
		if strings.Contains(html, unwanted) {
			t.Errorf("the move dialog should not mention %q any more", unwanted)
		}
	}
}

// A move leaves the problems' topics alone, so the handler must use the request type that has no topic.
func TestMoveProblemsHandlerSendsNoTopic(t *testing.T) {
	content, err := os.ReadFile("problem_handler.go")
	if err != nil {
		t.Fatalf("read problem_handler.go: %v", err)
	}
	source := strings.ReplaceAll(string(content), "\r\n", "\n")

	start := strings.Index(source, "func (h *ProblemsHandler) MoveProblems(")
	if start < 0 {
		t.Fatalf("could not find MoveProblems")
	}
	body := source[start:]
	if end := strings.Index(body[1:], "\nfunc "); end >= 0 {
		body = body[:end+1]
	}

	if !strings.Contains(body, "dto.MoveProblemsRequest{") {
		t.Errorf("MoveProblems should send a MoveProblemsRequest")
	}
	for _, unwanted := range []string{"MoveResourcesRequest", "topic_id", "TopicID", "Topic ID"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("MoveProblems should not mention %q: a move leaves topics alone", unwanted)
		}
	}
}
