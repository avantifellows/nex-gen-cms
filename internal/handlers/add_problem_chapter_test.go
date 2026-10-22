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
