package handlers

import (
	"bytes"
	"html/template"
	"strings"
	"testing"
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
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered tab is missing %q", want)
		}
	}
	if strings.Contains(html, "/topic/") || strings.Contains(html, "/api/topic/") {
		t.Errorf("chapter tab must not call topic endpoints:\n%s", html)
	}
}
