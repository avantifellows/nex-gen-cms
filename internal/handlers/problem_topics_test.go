package handlers

import (
	"bytes"
	"html/template"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/dto"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	"github.com/avantifellows/nex-gen-cms/utils"
)

func TestProblemTopicsKeepsTheOrderAndSkipsMissingOnes(t *testing.T) {
	known := map[int16]*models.Topic{5: topicNamed(5, "Kinematics"), 90: topicNamed(90, "Optics")}
	lookup := func(id int16) *models.Topic { return known[id] }

	got := problemTopics([]int16{90, 404, 5}, lookup)
	want := []*models.Topic{known[90], known[5]}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("problemTopics = %v, want %v", got, want)
	}
	if got := problemTopics(nil, lookup); len(got) != 0 {
		t.Errorf("a problem without topics should list none, got %v", got)
	}
}

func renderProblemScreen(t *testing.T, data dto.ProblemData) string {
	t.Helper()
	tmpl := template.Must(template.New("problem.html").Funcs(template.FuncMap{
		"add": utils.Add, "stringToInt": utils.StringToInt, "seq": utils.Seq,
		"getName": getConceptName, "langName": utils.LangName,
	}).ParseFiles("../../web/html/problem.html"))
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "content", data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out.String()
}

func problemForScreen() *models.Problem {
	return &models.Problem{
		Code: "P0000007", Subtype: "mcq_single_answer", ChapterID: 9,
		LangVersions: []models.LangVersion{{LangCode: "en"}},
	}
}

// The problem screen lists every topic of the problem, which may belong to other chapters, and none when it has none.
func TestProblemScreenListsAllTopics(t *testing.T) {
	html := renderProblemScreen(t, dto.ProblemData{
		ProblemPtr: problemForScreen(),
		ChapterPtr: &models.Chapter{Code: "C1", Name: []models.ChapterLang{{LangCode: "en", ChapterName: "Motion"}}},
		TopicPtrs: []*models.Topic{
			{Code: "T5", Name: []models.TopicLang{{LangCode: "en", TopicName: "Kinematics"}}},
			{Code: "T90", Name: []models.TopicLang{{LangCode: "en", TopicName: "Optics"}}},
		},
	})
	if !strings.Contains(html, "Topics:") || !strings.Contains(html, "T5 — Kinematics, T90 — Optics") {
		t.Errorf("expected both topics in one Topics line:\n%s", html)
	}

	html = renderProblemScreen(t, dto.ProblemData{ProblemPtr: problemForScreen()})
	if strings.Contains(html, "Topic") {
		t.Errorf("a problem without topics should show no topic label")
	}
}

// The single legacy topic is no longer read when the problem screen is built.
func TestProblemScreenDoesNotUseTheLegacyTopic(t *testing.T) {
	for path, unwanted := range map[string][]string{
		"../../web/html/problem.html": {".TopicPtr ", ".TopicPtr\n", ".TopicPtr}"},
		"problem_handler.go":          {"strconv.Itoa(int(selectedProblemPtr.TopicID))", "TopicPtr:"},
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := strings.ReplaceAll(string(raw), "\r\n", "\n")
		for _, text := range unwanted {
			if strings.Contains(content, text) {
				t.Errorf("%s still uses %q", path, text)
			}
		}
	}
}
