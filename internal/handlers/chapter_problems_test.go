package handlers

import (
	"bytes"
	"html/template"
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
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html", "../../web/html/problem_type_options.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		`hx-get="/api/chapter/problems?chapter-dropdown=12"`, // tbody refresh
		`var chapterId = '12';`, // initial load script
		`id="problemTableBody"`, // the move modal refreshes this id
		// topic filter: loads this chapter's topics; its changes only hide rows, so no request carries its value
		`hx-get="/api/topics?view=dropdown-all&id=12"`,
		`hx-trigger="refreshProblems"`,
		`hx-include="#curriculum-dropdown, #subject-dropdown"`,
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

// The chapter's problems load in one request and the topic filter works on the loaded rows, so the table
// must not be paged: no limit, no offset, no infinite scroll, and no topic value sent to the server.
func TestChapterProblemsTemplateIsNotPaged(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html", "../../web/html/problem_type_options.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, unwanted := range []string{
		"infinite-scroll", "limit=", "offset=", "data-sentinel", "chapter-problems-sentinel", "topic-dropdown",
	} {
		if strings.Contains(html, unwanted) {
			t.Errorf("the chapter tab must not be paged or filter on the server, found %q", unwanted)
		}
	}

	for _, want := range []string{
		`hx-sync="this:replace"`,
		// htmx marks the loader while the tbody's own request runs (input.css shows it then), and the
		// row buttons must not inherit it
		`hx-indicator="#chapter-problems-loader" hx-disinherit="hx-indicator"`,
		`<div id="chapter-problems-loader" class="justify-center `,
		// a filter change and every (re)load narrow the rows locally
		`Object.values(filterEls).forEach(el => el.addEventListener('change', applyFilters));`,
		`hx-on::after-swap="if (event.detail.elt === this) applyFilters()"`,
		`hasId(row.dataset.topicIds, filters.topic)`,
		// a hidden row is unchecked, so Move Problems cannot carry a problem the user cannot see
		`row.querySelector('input[name="select-problem"]').checked = false`,
		`tr:not(.hidden) input[name="select-problem"]`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered tab is missing %q", want)
		}
	}
	// the spinner centres itself
	if !strings.Contains(html, `w-6 h-6 mx-auto border-4`) {
		t.Errorf("the spinner must centre itself with mx-auto:\n%s", html)
	}
}

// Each row carries its topic ids (empty when it has none), which is what the local topic filter reads.
func TestChapterProblemRowCarriesTopicIDs(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problem_row.html"))
	problems := []*models.Problem{
		{ID: 1, Code: "P1", TopicIDs: []int16{7, 9}, LangVersions: []models.LangVersion{{LangCode: "en"}}},
		{ID: 2, Code: "P2", LangVersions: []models.LangVersion{{LangCode: "en"}}},
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, problems); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{`data-topic-ids="7,9"`, `data-topic-ids=""`} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered rows are missing %q:\n%s", want, html)
		}
	}
}

// Besides the topic, each row carries what the other local filters read: type, difficulty, skills and the
// day it was created (empty when db-service sent none).
func TestChapterProblemRowCarriesFilterValues(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problem_row.html"))
	problems := []*models.Problem{
		{ID: 1, Code: "P1", Subtype: "integer_type", DifficultyLevel: "hard", SkillIDs: []int16{3, 4},
			InsertedAt: "2026-10-28T09:30:00", LangVersions: []models.LangVersion{{LangCode: "en"}}},
		{ID: 2, Code: "P2", LangVersions: []models.LangVersion{{LangCode: "en"}}},
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, problems); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		`data-subtype="integer_type"`, `data-difficulty="hard"`, `data-skill-ids="3,4"`, `data-created="2026-10-28"`,
		`data-subtype=""`, `data-difficulty=""`, `data-skill-ids=""`, `data-created=""`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered rows are missing %q:\n%s", want, html)
		}
	}
}

func TestProblemCreatedDate(t *testing.T) {
	tests := map[string]string{
		"2026-10-28T09:30:00":    "2026-10-28",
		"2026-10-28T09:30:00.12Z": "2026-10-28",
		"2026-10-28":             "2026-10-28",
		"":                       "",
		"2026-10":                "",
	}
	for insertedAt, want := range tests {
		if got := (models.Problem{InsertedAt: insertedAt}).CreatedDate(); got != want {
			t.Errorf("CreatedDate(%q) = %q, want %q", insertedAt, got, want)
		}
	}
}

// The filters beside the topic one: type, difficulty, skill, and the start and end day. They only hide rows.
func TestChapterProblemsTemplateHasTheOtherFilters(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html", "../../web/html/problem_type_options.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, want := range []string{
		`id="problems-type-filter"`, `<option value="" selected>All Types</option>`, `value="mcq_single_answer"`,
		`id="problems-difficulty-filter"`, `<option value="easy">Easy</option>`, `<option value="hard">Hard</option>`,
		// the skill options load once, and only when the page does not have them yet (Back restores them)
		`id="problems-skill-filter"`,
		`hx-get="/api/skills" hx-trigger="load[document.getElementById('problems-skill-filter').options.length < 2]"`,
		`hx-swap="beforeend"`,
		`<input type="date" id="problems-start-date"`, `<input type="date" id="problems-end-date"`,
		// the date range is inclusive, on the day a problem was created
		`created >= filters.from`, `created <= filters.to`,
		// the chosen filters are kept on the table, so Back can put them back
		`tbody.setAttribute('data-filters', JSON.stringify(filters));`,
		`JSON.parse(tbody.getAttribute('data-filters') || '{}')`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered tab is missing %q", want)
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

// Back navigation restores the saved page, so nothing on it may re-fetch what it already has.
func TestChapterProblemsTemplateRestoresFromHistory(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html", "../../web/html/problem_type_options.html"))
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
		`Object.entries(filterEls).forEach(([name, el]) => {`,
		`if (el.type === 'checkbox') el.checked = !!saved[name];`,
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
	}
	wants := map[string]string{
		"nav-tracker.js defines the key":                  `REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY = "refreshChapterProblemsOnBack"`,
		"add/edit problem save sets it":                   `sessionStorage.setItem(REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY, '1')`,
		"deleting a problem from its screen sets it":      `sessionStorage.setItem(REFRESH_CHAPTER_PROBLEMS_ON_BACK_KEY, '1')`,
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

// Problems now live on the chapter screen, so nothing may still open a Problems tab on the topic screen.
func TestTopicScreenHasNoProblemsTab(t *testing.T) {
	removed := map[string][]string{
		"../../web/html/topic.html":       {"topic-problems-tab", "/topic/problems", "topicTabOnce"},
		"../../web/html/add_problem.html": {"topicTabOnce", "topic-problems-tab"},
		"../../cmd/main.go":               {`"/topic/problems"`, "LoadTopicProblems", "/api/topic/problems", "GetTopicProblems"},
	}

	for path, unwanted := range removed {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, text := range unwanted {
			if strings.Contains(string(content), text) {
				t.Errorf("%s still mentions %q", path, text)
			}
		}
	}

	for _, gone := range []string{"../../web/html/topic_problems.html", "../../web/html/topic_problem_row.html"} {
		if _, err := os.Stat(gone); err == nil {
			t.Errorf("%s should have been removed", gone)
		}
	}
}

// Another screen can ask the chapter screen to open a given tab once, by name, before the tabs load themselves.
func TestChapterScreenOpensTheRequestedTabOnce(t *testing.T) {
	content, err := os.ReadFile("../../web/html/chapter.html")
	if err != nil {
		t.Fatalf("read chapter.html: %v", err)
	}
	script := string(content)

	for _, want := range []string{
		`sessionStorage.getItem('chapterTabOnce')`,
		// the request is forgotten at once, so it applies to one arrival only
		`sessionStorage.removeItem('chapterTabOnce')`,
		// an unknown tab name must not leave every tab inactive
		`if (tabId && document.getElementById(tabId)) {`,
		`activateSubTab(tabId);`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("chapter.html is missing %q", want)
		}
	}

	// the request is read after the tabs exist and before htmx processes them: it sits inside the script block
	scriptAt := strings.Index(script, "<script>")
	openerAt := strings.Index(script, "sessionStorage.getItem('chapterTabOnce')")
	if scriptAt < 0 || openerAt < scriptAt {
		t.Errorf("the opener should be part of the screen's script, after its tabs")
	}
}

// The toolbar reads left to right: the topic filter, then adding a problem, then moving the selected ones.
func TestChapterProblemsToolbarOrder(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html", "../../web/html/problem_type_options.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	filterAt := strings.Index(html, `id="problems-topic-filter"`)
	addAt := strings.Index(html, `/chapter/add-problem?chapter_id=12`)
	moveAt := strings.Index(html, `id="move-problems-btn"`)
	if filterAt < 0 || addAt < 0 || moveAt < 0 {
		t.Fatalf("the toolbar is missing a control: filter=%d add=%d move=%d", filterAt, addAt, moveAt)
	}
	if !(filterAt < addAt && addAt < moveAt) {
		t.Errorf("expected the topic filter, then the add button, then the move button")
	}
}
