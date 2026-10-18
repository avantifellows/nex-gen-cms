package handlers

import (
	"bytes"
	"html/template"
	"os"
	"strings"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

func TestChapterProblemsQuery(t *testing.T) {
	tests := []struct {
		name      string
		chapterID string
		topicID   string
		want      string
		wantErr   string
	}{
		{"chapter only (all topics)", "12", "", "?chapter_id=12", ""},
		{"chapter with topic filter", "12", "45", "?chapter_id=12&topic_id=45", ""},
		{"missing chapter", "", "45", "", "Invalid Chapter ID"},
		{"non-numeric chapter", "abc", "", "", "Invalid Chapter ID"},
		{"non-numeric topic", "12", "abc", "", "Invalid Topic ID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chapterProblemsQuery(tt.chapterID, tt.topicID)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("query = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChapterProblemsTemplateUsesChapterEndpoint(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		`hx-get="/api/chapter/problems?chapter-dropdown=12"`, // tbody refresh
		`var chapterId = '12';`,                              // initial load script
		`id="problemTableBody"`,                              // the move modal refreshes this id
		// topic filter: loads this chapter's topics, and its changes reload the table with its value included
		`hx-get="/api/topics?view=dropdown-all&id=12"`,
		`hx-trigger="refreshProblems, change from:#problems-topic-filter"`,
		`#curriculum-dropdown, #subject-dropdown, #problems-topic-filter`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered tab is missing %q", want)
		}
	}
	if strings.Contains(html, "/topic/") || strings.Contains(html, "/api/topic/") {
		t.Errorf("chapter tab must not call topic endpoints:\n%s", html)
	}

	// The initial-load script must only send the chapter: the tbody's hx-include adds curriculum, subject and
	// the topic filter, so repeating them in the URL sent them twice.
	if !strings.Contains(html, "`/api/chapter/problems?chapter-dropdown=${chapterId}`") {
		t.Errorf("initial-load script URL should carry only the chapter:\n%s", html)
	}
	if strings.Contains(html, "subject-dropdown=${") || strings.Contains(html, "curriculum-dropdown=${") {
		t.Errorf("initial-load script must not repeat values that hx-include already adds:\n%s", html)
	}
}

// input.css hides the loaders by id (display none, flex while htmx-request), so a loader id without a rule
// stays visible forever.
func TestChapterProblemsLoaderHasCssRule(t *testing.T) {
	css, err := os.ReadFile("../../input.css")
	if err != nil {
		t.Fatalf("read input.css: %v", err)
	}
	rules := string(css)

	for _, want := range []string{"#chapter-problems-loader {", "#chapter-problems-loader.htmx-request {"} {
		if !strings.Contains(rules, want) {
			t.Errorf("input.css has no rule %q for the chapter problems loader", want)
		}
	}
}

func TestTopicDropdownAllTemplate(t *testing.T) {
	tmpl := template.Must(template.New("topic_dropdown_all.html").Funcs(template.FuncMap{"getName": getTopicName}).
		ParseFiles("../../web/html/topic_dropdown_all.html"))
	topics := []*models.Topic{
		{ID: 5, Name: []models.TopicLang{{LangCode: "en", TopicName: "Kinematics"}}},
		{ID: 6, Name: []models.TopicLang{{LangCode: "en", TopicName: "Dynamics"}}},
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, topics); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	// a visible, selectable blank option so "all topics" can be picked again after choosing a topic
	if !strings.Contains(html, `<option value="" selected>All Topics</option>`) {
		t.Errorf("missing the All Topics option:\n%s", html)
	}
	for _, want := range []string{`<option value="5">Kinematics</option>`, `<option value="6">Dynamics</option>`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q:\n%s", want, html)
		}
	}
}
