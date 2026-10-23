package handlers

import (
	"bytes"
	"html/template"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/utils"
)

// renderAddProblem renders the add / edit / copy screen the way the handlers do, without the page layout.
func renderAddProblem(t *testing.T, data dto.ProblemData) string {
	t.Helper()
	funcs := template.FuncMap{
		"joinInt16":        utils.JoinInt16,
		"add":              utils.Add,
		"stringToInt":      utils.StringToInt,
		"toJson":           utils.ToJson,
		"getConceptName":   getConceptName,
		"dict":             utils.Dict,
		"emptyStringSlice": utils.EmptyStringSlice,
	}
	files := []string{"add_problem.html", "problem_type_options.html", "editor.html",
		"problem_answer_numerical.html", "input_tags.html"}
	for i := range files {
		files[i] = "../../web/html/" + files[i]
	}

	tmpl, err := template.New("add_problem.html").Funcs(funcs).ParseFiles(files...)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "content", data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out.String()
}

func TestChapterProblemsTabHasAddProblemButton(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/chapter_problems.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, "12"); err != nil {
		t.Fatalf("execute: %v", err)
	}

	want := `hx-get="/chapter/add-problem?chapter_id=12"`
	if !strings.Contains(out.String(), want) {
		t.Errorf("the Problems tab is missing the add button %q", want)
	}
}

func TestAddProblemRouteIsScopedToChapter(t *testing.T) {
	mainGo, err := os.ReadFile("../../cmd/main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainGo), `"/chapter/add-problem"`) {
		t.Errorf(`main.go should register "/chapter/add-problem"`)
	}
	if strings.Contains(string(mainGo), `"/topic/add-problem"`) {
		t.Errorf(`main.go should no longer register "/topic/add-problem"`)
	}
}

func TestAddProblemScreenForAChapter(t *testing.T) {
	html := renderAddProblem(t, dto.ProblemData{ChapterPtr: &models.Chapter{ID: 49}})

	// no topic yet (null), the chapter, and no problem id
	if !regexp.MustCompile(`createProblem\(event,\s*(<!--.*?-->\s*)?null\s*,\s*49\s*,\s*null\s*\)`).MatchString(html) {
		t.Errorf("the form should call createProblem(event, null, 49, null)")
	}
	if !strings.Contains(html, "Add New Problem") || strings.Contains(html, "Edit Problem") {
		t.Errorf("the title should read Add New Problem")
	}
	if !strings.Contains(html, "const currentChapterId = '49';") ||
		!strings.Contains(html, "const currentTopicId = '';") {
		t.Errorf("the script should know the chapter and no topic")
	}
}

// Edit keeps its own branch, and copy is now chapter-based like add: it also sends the chapter and no topic yet.
func TestAddProblemScreenStillRendersEditAndCopy(t *testing.T) {
	problem := &models.Problem{
		ID: 7, Subtype: "mcq_single_answer", TopicID: 5, ChapterID: 9,
		LangVersions: []models.LangVersion{{LangCode: "en"}},
	}

	edit := renderAddProblem(t, dto.ProblemData{ProblemPtr: problem})
	if !regexp.MustCompile(`createProblem\(event,\s*(<!--.*?-->\s*)?5\s*,\s*9\s*,\s*7\s*\)`).MatchString(edit) ||
		!strings.Contains(edit, "Edit Problem") {
		t.Errorf("edit mode should call createProblem(event, 5, 9, 7) under an Edit title")
	}

	cp := renderAddProblem(t, dto.ProblemData{
		HomeData:   dto.HomeData{CurriculumID: 1, GradeID: 10, SubjectID: 2},
		ProblemPtr: problem,
		ChapterPtr: &models.Chapter{ID: 9},
	})
	if !regexp.MustCompile(`createProblem\(event,\s*(<!--.*?-->\s*)?null\s*,\s*9\s*,\s*null\s*\)`).MatchString(cp) ||
		!strings.Contains(cp, "Copy Problem") {
		t.Errorf("copy mode should call createProblem(event, null, 9, null) under a Copy title")
	}
	// after saving a copy the user lands on the chapter, not on a topic screen
	if !strings.Contains(cp, "const chapterUrl = '/chapter?id=9&curriculum-dropdown=1&grade-dropdown=10&subject-dropdown=2';") {
		t.Errorf("a saved copy should open its chapter")
	}
}

// The screen is scoped to a chapter now, so the topic branches it used to have must be gone.
func TestAddProblemTemplateHasNoTopicPtr(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_problem.html")
	if err != nil {
		t.Fatalf("read add_problem.html: %v", err)
	}
	if strings.Contains(string(content), ".TopicPtr") {
		t.Errorf("add_problem.html should not use .TopicPtr any more")
	}
}

// The Topics checklist is part of the shared screen, so add, edit and copy all show it.
func TestAddProblemScreenShowsTopicsField(t *testing.T) {
	problem := &models.Problem{
		ID: 7, Subtype: "mcq_single_answer", ChapterID: 9,
		LangVersions: []models.LangVersion{{LangCode: "en"}},
	}
	screens := map[string]dto.ProblemData{
		"add":  {ChapterPtr: &models.Chapter{ID: 9}},
		"edit": {ProblemPtr: problem},
		"copy": {ProblemPtr: problem, ChapterPtr: &models.Chapter{ID: 9}},
	}

	for mode, data := range screens {
		html := renderAddProblem(t, data)
		for _, want := range []string{
			`<p class="form-label" id="topics-label">Topics</p>`,
			// a scrollable list of about 7.5 rows (max-h-60 = 15rem), labelled by the heading above it
			`id="topics-checklist" role="group" aria-labelledby="topics-label"`,
			`max-h-60 overflow-y-auto`,
			`Loading topics...`,
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s screen is missing %q", mode, want)
			}
		}
		if strings.Contains(html, `id="topics-dropdown"`) {
			t.Errorf("%s screen should show checkboxes, not a multi-select", mode)
		}
	}
}

func renderTopicChecklist(t *testing.T, topics []*models.Topic) string {
	t.Helper()
	tmpl := template.Must(template.New("topic_checklist.html").Funcs(template.FuncMap{"getName": getTopicName}).
		ParseFiles("../../web/html/topic_checklist.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, topics); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out.String()
}

// Each topic is one labelled checkbox row, and a chapter without topics says so instead of staying blank.
func TestTopicChecklistTemplate(t *testing.T) {
	html := renderTopicChecklist(t, []*models.Topic{
		{ID: 5, Name: []models.TopicLang{{LangCode: "en", TopicName: "Kinematics"}}},
		{ID: 6, Name: []models.TopicLang{{LangCode: "en", TopicName: "Dynamics"}}},
	})

	if strings.Count(html, `type="checkbox"`) != 2 {
		t.Errorf("expected one checkbox per topic, got: %s", html)
	}
	for _, want := range []string{
		`name="topic-ids" value="5"`, `<span>Kinematics</span>`,
		`name="topic-ids" value="6"`, `<span>Dynamics</span>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q: %s", want, html)
		}
	}
	if strings.Contains(html, "No topics in this chapter") {
		t.Errorf("the empty message must not show when there are topics")
	}

	empty := renderTopicChecklist(t, nil)
	if !strings.Contains(empty, "No topics in this chapter") || strings.Contains(empty, "checkbox") {
		t.Errorf("an empty chapter should show only the empty message, got: %s", empty)
	}
}

// The list loads the problem's chapter topics itself on load, for add, edit and copy alike.
func TestAddProblemScreenLoadsChapterTopics(t *testing.T) {
	problem := &models.Problem{
		ID: 7, Subtype: "mcq_single_answer", ChapterID: 9,
		LangVersions: []models.LangVersion{{LangCode: "en"}},
	}
	screens := map[string]dto.ProblemData{
		"add":  {ChapterPtr: &models.Chapter{ID: 9}},
		"edit": {ProblemPtr: problem},
		"copy": {ProblemPtr: problem, ChapterPtr: &models.Chapter{ID: 9}},
	}

	for mode, data := range screens {
		html := renderAddProblem(t, data)
		for _, want := range []string{
			// not before the curriculum dropdown has a value: the topics are filtered by curriculum
			`hx-trigger="load[document.getElementById('curriculum-dropdown').value], onLoaded from:#curriculum-dropdown"`,
			`hx-include="#curriculum-dropdown"`,
			`hx-get="/api/topics?view=checklist&id=9"`,
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s screen is missing %q", mode, want)
			}
		}
	}
}
