package handlers

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"
)

// Copying a problem asks for a chapter only: the dialog ends there and the Copy button follows the chapter.
func TestCopyProblemDialogAsksForAChapterOnly(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/copy_problem_modal.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "7"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		`id="copy-problem-chapter-dropdown"`,
		// two dropdowns per row (curriculum, grade / subject, chapter), in a dialog narrower than before
		`grid grid-cols-2 gap-4`,
		`w-full max-w-2xl`,
		`hx-get="/chapter/copy-problem"`,
		`onchange="toggleCopyProblemButton()"`,
		`copyBtn.disabled = !chapterDropdown.value;`,
		"Then select Chapter.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the copy dialog is missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"copy-problem-topic-dropdown", `name="topic_id"`, "Topic", "/topic/copy-problem", "col-span-3", "max-w-3xl",
	} {
		if strings.Contains(html, unwanted) {
			t.Errorf("the copy dialog should not mention %q any more", unwanted)
		}
	}
}

func TestCopyProblemRoutesAreScopedToTheChapter(t *testing.T) {
	files := map[string]string{
		"../../cmd/main.go":                       `"/chapter/copy-problem"`,
		"../../web/html/chapter_problem_row.html": `hx-get="/chapter/copy-problem-dialog?id={{.ID}}"`,
		"../../web/html/search_problem_row.html":  `hx-get="/chapter/copy-problem-dialog?id={{.ID}}"`,
	}
	for path, want := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(content), want) {
			t.Errorf("%s should contain %q", path, want)
		}
		if strings.Contains(string(content), "/topic/copy-problem") {
			t.Errorf("%s should not use the topic copy routes any more", path)
		}
	}

	mainGo, err := os.ReadFile("../../cmd/main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainGo), `"/chapter/copy-problem-dialog"`) {
		t.Errorf("main.go should register the chapter copy dialog route")
	}
}
