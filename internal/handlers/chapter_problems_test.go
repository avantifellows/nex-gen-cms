package handlers

import (
	"bytes"
	"html/template"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
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
		`hx-get="/api/chapter/problems?chapter-dropdown=12&limit=10&offset=0"`, // tbody refresh
		`var chapterId = '12';`, // initial load script
		`id="problemTableBody"`, // the move modal refreshes this id
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
	if !strings.Contains(html, "`/api/chapter/problems?chapter-dropdown=${chapterId}&limit=10&offset=0`") {
		t.Errorf("initial-load script URL should carry only the chapter:\n%s", html)
	}
	if strings.Contains(html, "subject-dropdown=${") || strings.Contains(html, "curriculum-dropdown=${") {
		t.Errorf("initial-load script must not repeat values that hx-include already adds:\n%s", html)
	}
}

// The infinite-scroll extension drives the loader (the hidden class) and loads the next page when the
// sentinel scrolls into view, so the template must declare both and start with the loader hidden.
func TestChapterProblemsTemplateIsLazyLoaded(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		`hx-ext="infinite-scroll"`,
		`data-limit="10" data-offset="0"`,
		`hx-sync="this:replace"`, // a stale response must not land after the topic filter changed
		`data-loader="#chapter-problems-loader"`,
		`data-sentinel="#chapter-problems-sentinel"`,
		`hx-get="/api/chapter/problems?chapter-dropdown=12&limit=10&offset=0"`,
		`<div id="chapter-problems-sentinel"></div>`,
		`<div id="chapter-problems-loader" class="hidden `,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered tab is missing %q", want)
		}
	}
	// the loader is a block once the extension removes "hidden", so the spinner centres itself
	if !strings.Contains(html, `w-6 h-6 mx-auto border-4`) {
		t.Errorf("the spinner must centre itself with mx-auto:\n%s", html)
	}
	if strings.Contains(html, "hx-indicator") {
		t.Errorf("the extension drives the loader, hx-indicator would fight it:\n%s", html)
	}
}

// The extension only reacts to the endpoints it knows, so the chapter list must be one of them.
func TestInfiniteScrollKnowsChapterProblemsEndpoint(t *testing.T) {
	js, err := os.ReadFile("../../web/static/js/infinite-scroll.js")
	if err != nil {
		t.Fatalf("read infinite-scroll.js: %v", err)
	}
	if !strings.Contains(string(js), `path?.startsWith("/api/chapter/problems")`) {
		t.Errorf("infinite-scroll.js does not handle /api/chapter/problems")
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

func TestProblemsPagination(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantQuery string
		wantLimit int
	}{
		{"not paged without limit", "chapter-dropdown=12", "", 0},
		{"offset alone does not page", "chapter-dropdown=12&offset=20", "", 0},
		{"first page", "limit=10", "&limit=10&offset=0", 10},
		{"later page", "limit=10&offset=20", "&limit=10&offset=20", 10},
		{"invalid limit falls back to 10", "limit=abc&offset=10", "&limit=10&offset=10", 10},
		{"limit below 1 falls back to 10", "limit=0", "&limit=10&offset=0", 10},
		{"negative offset falls back to 0", "limit=10&offset=-5", "&limit=10&offset=0", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("parse query: %v", err)
			}
			gotQuery, gotLimit := problemsPagination(values)
			if gotQuery != tt.wantQuery || gotLimit != tt.wantLimit {
				t.Errorf("problemsPagination() = (%q, %d), want (%q, %d)", gotQuery, gotLimit, tt.wantQuery, tt.wantLimit)
			}
		})
	}
}

// Back navigation restores the saved page, so nothing on it may re-fetch what it already has.
func TestChapterProblemsTemplateRestoresFromHistory(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		// the topic options are loaded only when the page does not have them yet
		`hx-trigger="load[document.getElementById('problems-topic-filter').options.length < 2]"`,
		// the script tells a fresh load from a restored one and restores the topic from the saved params
		`if (tbody.children.length === 0) {`,
		`function restoreChapterProblems() {`,
		`tbody.getAttribute('data-active-params')`,
		`htmx.trigger(tbody, 'infinite-scroll:restore');`,
		// selected rows survive the history snapshot: every change is mirrored into the saved attribute
		`var syncCheckedAttributes = () =>`,
		`cb.toggleAttribute('checked', cb.checked)`,
		// a flag set by the add / edit / delete screens makes the restored list reload
		`sessionStorage.getItem(REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY)`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered tab is missing %q", want)
		}
	}
	// nothing may be registered on the document or body, where it would outlive this screen
	if strings.Contains(html, "document.body.addEventListener") || strings.Contains(html, "beforeHistorySave") {
		t.Errorf("the tab must not register page-level listeners")
	}
}

func TestChapterScreenDoesNotRefetchRestoredProblemsTab(t *testing.T) {
	tmpl := template.Must(template.New("chapter.html").Funcs(template.FuncMap{
		"getName": func(any, string) string { return "Motion" },
	}).ParseFiles("../../web/html/chapter.html"))
	data := dto.ChapterData{
		HomeData:   dto.HomeData{CurriculumID: 3, GradeID: 10, SubjectID: 2},
		ChapterPtr: &models.Chapter{ID: 49},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "content", data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// the tab loads on page load only while it is active and its content area is still empty
	want := `hx-trigger="load[document.getElementById('problems-sub-tab').classList.contains('active') && ` +
		`!document.getElementById('subContent').children.length], click"`
	if !strings.Contains(out.String(), want) {
		t.Errorf("the Problems tab should not reload when restored with its content: %s", out.String())
	}
}

// These screens change what the chapter's list shows, so each must flag it for reload on Back.
func TestScreensFlagChapterProblemsForReload(t *testing.T) {
	files := map[string]string{
		"nav-tracker.js defines the key":                  "../../web/static/js/nav-tracker.js",
		"add/edit problem save sets it":                   "../../web/html/add_problem.html",
		"deleting a problem from its screen sets it":      "../../web/html/problem.html",
		"infinite-scroll.js restores a page from history": "../../web/static/js/infinite-scroll.js",
	}
	wants := map[string]string{
		"nav-tracker.js defines the key":                  `REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY = "refreshChapterProblemsOnBack"`,
		"add/edit problem save sets it":                   `sessionStorage.setItem(REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY, '1')`,
		"deleting a problem from its screen sets it":      `sessionStorage.setItem(REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY, '1')`,
		"infinite-scroll.js restores a page from history": `name === 'infinite-scroll:restore'`,
	}

	for name, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(content), wants[name]) {
			t.Errorf("%s: %s does not contain %q", name, path, wants[name])
		}
	}
}
