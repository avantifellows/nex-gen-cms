package handlers

import (
	"bytes"
	"html/template"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
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

	// the chapter and no problem id; topics come from the checklist, not from an argument
	if !regexp.MustCompile(`createProblem\(event,\s*(<!--.*?-->\s*)?49\s*,\s*null\s*\)`).MatchString(html) {
		t.Errorf("the form should call createProblem(event, 49, null)")
	}
	if !strings.Contains(html, "Add New Problem") || strings.Contains(html, "Edit Problem") {
		t.Errorf("the title should read Add New Problem")
	}
	// the chapter reaches the script through createProblem; it keeps no chapter or topic constant of its own
	if strings.Contains(html, "currentChapterId") || strings.Contains(html, "currentTopicId") {
		t.Errorf("the script should not keep a chapter or topic constant")
	}
}

// Edit keeps its own branch, and copy is now chapter-based like add: it also sends the chapter and no topic yet.
func TestAddProblemScreenStillRendersEditAndCopy(t *testing.T) {
	problem := &models.Problem{
		ID: 7, Subtype: "mcq_single_answer", ChapterID: 9,
		LangVersions: []models.LangVersion{{LangCode: "en"}},
	}

	edit := renderAddProblem(t, dto.ProblemData{ProblemPtr: problem})
	if !regexp.MustCompile(`createProblem\(event,\s*(<!--.*?-->\s*)?9\s*,\s*7\s*\)`).MatchString(edit) ||
		!strings.Contains(edit, "Edit Problem") {
		t.Errorf("edit mode should call createProblem(event, 9, 7) under an Edit title")
	}

	cp := renderAddProblem(t, dto.ProblemData{
		HomeData:   dto.HomeData{CurriculumID: 1, GradeID: 10, SubjectID: 2},
		ProblemPtr: problem,
		ChapterPtr: &models.Chapter{ID: 9},
	})
	if !regexp.MustCompile(`createProblem\(event,\s*(<!--.*?-->\s*)?9\s*,\s*null\s*\)`).MatchString(cp) ||
		!strings.Contains(cp, "Copy Problem") {
		t.Errorf("copy mode should call createProblem(event, 9, null) under a Copy title")
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
			// a scrollable list of a fixed height of about 7.5 rows (h-60 = 15rem), labelled by the heading above it
			`id="topics-checklist" role="group" aria-labelledby="topics-label"`,
			`class="h-60 overflow-y-auto`,
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

func renderTopicChecklist(t *testing.T, data dto.TopicChecklistData) string {
	t.Helper()
	tmpl := template.Must(template.New("topic_checklist.html").Funcs(template.FuncMap{"getName": getTopicName}).
		ParseFiles("../../web/html/topic_checklist.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	return out.String()
}

func topicNamed(id int16, name string) *models.Topic {
	return &models.Topic{ID: id, Name: []models.TopicLang{{LangCode: "en", TopicName: name}}}
}

// Each topic is one labelled checkbox row, and a chapter without topics says so instead of staying blank.
func TestTopicChecklistTemplate(t *testing.T) {
	html := renderTopicChecklist(t, dto.TopicChecklistData{
		Topics: []*models.Topic{topicNamed(5, "Kinematics"), topicNamed(6, "Dynamics")},
	})

	if strings.Count(html, `type="checkbox"`) != 2 || strings.Contains(html, "checked") {
		t.Errorf("expected two unticked checkboxes, got: %s", html)
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

	empty := renderTopicChecklist(t, dto.TopicChecklistData{})
	if !strings.Contains(empty, "No topics in this chapter") || strings.Contains(empty, "checkbox") {
		t.Errorf("an empty chapter should show only the empty message, got: %s", empty)
	}
}

// A problem's own topics come back ticked, including those that are not in the chapter's list.
func TestTopicChecklistTemplateTicksSelectedTopics(t *testing.T) {
	html := renderTopicChecklist(t, dto.TopicChecklistData{
		Topics:   []*models.Topic{topicNamed(5, "Kinematics"), topicNamed(6, "Dynamics")},
		Selected: map[int16]bool{6: true, 90: true},
		Other:    []dto.TopicChecklistExtra{{Topic: topicNamed(90, "Optics"), Note: "other chapter"}},
	})

	if !strings.Contains(html, `value="6" class="accent-accent" checked>`) {
		t.Errorf("the selected chapter topic should be ticked: %s", html)
	}
	if strings.Contains(html, `value="5" class="accent-accent" checked`) {
		t.Errorf("a topic that is not selected must stay unticked: %s", html)
	}
	if !strings.Contains(html, `value="90" class="accent-accent" checked>`) ||
		!strings.Contains(html, `Optics <span class="text-ink-muted">(other chapter)</span>`) {
		t.Errorf("a selected topic of another chapter should be listed ticked with its note: %s", html)
	}
}

func TestParseTopicIDs(t *testing.T) {
	tests := map[string][]int16{
		"":          nil,
		"5":         {5},
		"5,9":       {5, 9},
		" 5 , 9 ,":  {5, 9},
		"5,abc,9,x": {5, 9},
	}
	for input, want := range tests {
		if got := parseTopicIDs(input); !reflect.DeepEqual(got, want) {
			t.Errorf("parseTopicIDs(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestBuildTopicChecklist(t *testing.T) {
	chapterTopics := []*models.Topic{topicNamed(5, "Kinematics"), topicNamed(6, "Dynamics")}
	archived := topicNamed(7, "Old topic")
	archived.StatusID = constants.StatusArchived
	known := map[int16]*models.Topic{7: archived, 90: topicNamed(90, "Optics")}
	lookup := func(id int16) *models.Topic { return known[id] }

	data := buildTopicChecklist(chapterTopics, []int16{6, 90, 7, 90, 404}, lookup)

	if !reflect.DeepEqual(data.Selected, map[int16]bool{6: true, 90: true, 7: true, 404: true}) {
		t.Errorf("Selected = %v", data.Selected)
	}
	if len(data.Topics) != 2 {
		t.Errorf("the chapter's own topics must be kept, got %d", len(data.Topics))
	}
	// 6 is listed already, 404 is unknown, and 90 is asked for twice: only 90 and 7 are extra, once each
	if len(data.Other) != 2 || data.Other[0].Topic.ID != 90 || data.Other[0].Note != "other chapter" ||
		data.Other[1].Topic.ID != 7 || data.Other[1].Note != "archived" {
		t.Errorf("Other = %+v", data.Other)
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
			`hx-get="/api/topics?view=checklist&id=9`, // edit and copy continue with &selected=…
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s screen is missing %q", mode, want)
			}
		}
	}
}

// Edit and copy ask for their topics to come back ticked; a new problem has none to tick.
func TestAddProblemScreenAsksForTheProblemsTopicsTicked(t *testing.T) {
	problem := &models.Problem{
		ID: 7, Subtype: "mcq_single_answer", ChapterID: 9, TopicIDs: []int16{5, 90},
		LangVersions: []models.LangVersion{{LangCode: "en"}},
	}
	want := `hx-get="/api/topics?view=checklist&id=9&selected=5,90"`

	for mode, data := range map[string]dto.ProblemData{
		"edit": {ProblemPtr: problem},
		"copy": {ProblemPtr: problem, ChapterPtr: &models.Chapter{ID: 9}},
	} {
		if html := renderAddProblem(t, data); !strings.Contains(html, want) {
			t.Errorf("%s screen should request %q", mode, want)
		}
	}

	if html := renderAddProblem(t, dto.ProblemData{ChapterPtr: &models.Chapter{ID: 9}}); strings.Contains(html, "view=checklist&id=9&selected=") {
		t.Errorf("a new problem has no topics to tick")
	}
}

// Topics replace concepts on this screen, so the concept prefill must not render any more.
func TestAddProblemScreenDoesNotPrefillConcepts(t *testing.T) {
	problem := &models.Problem{
		ID: 7, Subtype: "mcq_single_answer", ChapterID: 9,
		LangVersions: []models.LangVersion{{LangCode: "en"}},
		Concepts:     []models.Concept{{ID: 3, Name: []models.ConceptLang{{LangCode: "en", ConceptName: "Velocity"}}}},
	}

	html := renderAddProblem(t, dto.ProblemData{ProblemPtr: problem})
	for _, unwanted := range []string{"already mapped concepts", "Velocity", "exclude="} {
		if strings.Contains(html, unwanted) {
			t.Errorf("the edit screen should not prefill concepts, found %q", unwanted)
		}
	}
}

// The main problem is saved with the ticked topics. The script lives in the template, so these checks read it.
func TestAddProblemSavesTheTickedTopics(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_problem.html")
	if err != nil {
		t.Fatalf("read add_problem.html: %v", err)
	}
	script := strings.ReplaceAll(string(content), "\r\n", "\n")

	for _, want := range []string{
		// the topic is no longer an argument, and the payload carries the ticked ids
		"window.createProblem = function (event, chapterId, problemId) {",
		"function buildProblemJson(chapterId) {",
		`topicIds: getCheckedTopicIds(topicsChecklist),`,
		`querySelectorAll('input[name="topic-ids"]:checked')`,
		"topic_ids: topicIds,",
		// topic_ids is the full set, so saving before the list loaded must be refused
		"topicsChecklist.dataset.loaded = 'true';",
		"Topics are still loading. Please wait a moment and try again.",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("add_problem.html is missing %q", want)
		}
	}

	// the main problem sends neither the legacy topic nor concepts
	mainStart := strings.Index(script, "function buildProblemJson(chapterId) {")
	mainEnd := strings.Index(script, "scheduleEditorBoot(")
	if mainStart < 0 || mainEnd < mainStart {
		t.Fatalf("could not find buildProblemJson")
	}
	main := script[mainStart:mainEnd]
	for _, unwanted := range []string{`\btopicId\b`, `\bconceptIds\b`, `concepts-dropdown`} {
		if regexp.MustCompile(unwanted).MatchString(main) {
			t.Errorf("the main problem payload must not use %q any more", unwanted)
		}
	}
}

// The picker asks only for a grade and a chapter, then lists that chapter's topics as checkbox rows.
func TestAddTopicModalTemplate(t *testing.T) {
	tmpl := template.Must(template.ParseFiles("../../web/html/add_topic_modal.html"))
	var out bytes.Buffer
	if err := tmpl.Execute(&out, nil); err != nil {
		t.Fatalf("execute: %v", err)
	}
	html := out.String()

	for _, unwanted := range []string{"add-topic-curriculum-dropdown", "add-topic-subject-dropdown"} {
		if strings.Contains(html, unwanted) {
			t.Errorf("the picker must not ask for the curriculum or subject again, found %q", unwanted)
		}
	}
	for _, want := range []string{
		`id="add-topic-grade-dropdown"`, `id="add-topic-chapter-dropdown"`,
		// curriculum and subject are the ones of the screen behind the dialog, not chosen again here
		`hx-include="#add-topic-grade-dropdown, #curriculum-dropdown, #subject-dropdown"`,
		// the chosen chapter's topics come from the checklist view, filtered by that curriculum
		`hx-get="/api/topics?view=checklist" hx-trigger="change from:#add-topic-chapter-dropdown"`,
		`hx-include="#add-topic-chapter-dropdown, #curriculum-dropdown"`,
		// the topics list has the same fixed height (7.5 rows) as the problem's own list
		`id="add-topic-checklist" class="h-60 overflow-y-auto`,
		`id="add-topic-btn" type="button" disabled`,
		`window.addPickedTopics = function`,
		`window.closeAddTopicModal = function`,
		// the ticked topics are added to the list the opening link pointed at
		`window._topicModalTargetChecklist`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the topic picker is missing %q", want)
		}
	}
}

// Topics replace concepts for the main problem: the screen offers the topic picker and no Concepts field.
func TestAddProblemScreenOffersTopicPickerInsteadOfConcepts(t *testing.T) {
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
			`hx-get="/chapter/add-problem/add-topic-dialog"`,
			`window._topicModalTargetChecklist = document.getElementById('topics-checklist')`,
			`+ Add topic from other chapter`,
		} {
			if !strings.Contains(html, want) {
				t.Errorf("%s screen is missing %q", mode, want)
			}
		}
		if strings.Contains(html, `id="concepts-dropdown"`) {
			t.Errorf("%s screen should not have the main Concepts field any more", mode)
		}
	}

	content, err := os.ReadFile("../../web/html/add_problem.html")
	if err != nil {
		t.Fatalf("read add_problem.html: %v", err)
	}
	if strings.Contains(string(content), "mainConceptsDropdown") {
		t.Errorf("the script should not keep a handle on the removed Concepts field")
	}

	mainGo, err := os.ReadFile("../../cmd/main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	if !strings.Contains(string(mainGo), `"/chapter/add-problem/add-topic-dialog"`) {
		t.Errorf("main.go should register the topic picker route")
	}
}

// Type, Skills and Difficulty stack in the left 40% and Topics takes the right 60%.
func TestAddProblemScreenLayoutPutsSkillsBelowTypeAndTopicsOnTheRight(t *testing.T) {
	html := renderAddProblem(t, dto.ProblemData{ChapterPtr: &models.Chapter{ID: 9}})

	gridAt := strings.Index(html, `grid grid-cols-1 md:grid-cols-5`)
	leftColumnAt := strings.Index(html, `<div class="space-y-6 md:col-span-2">`)
	typeAt := strings.Index(html, `for="problem-type-dropdown"`)
	skillsAt := strings.Index(html, `for="skills-dropdown"`)
	difficultyAt := strings.Index(html, `Difficulty Rating`) // the first one is the main problem's
	rightColumnAt := strings.Index(html, `<div class="md:col-span-3">`)
	topicsAt := strings.Index(html, `id="topics-label"`)

	for name, at := range map[string]int{
		"grid": gridAt, "left column": leftColumnAt, "type": typeAt, "skills": skillsAt,
		"difficulty": difficultyAt, "right column": rightColumnAt, "topics": topicsAt,
	} {
		if at < 0 {
			t.Fatalf("the layout is missing its %s", name)
		}
	}

	// a five column grid: the left column (Type, Skills, Difficulty) takes two, the right column (Topics) three
	if !(gridAt < leftColumnAt && leftColumnAt < typeAt && typeAt < skillsAt && skillsAt < difficultyAt &&
		difficultyAt < rightColumnAt && rightColumnAt < topicsAt) {
		t.Errorf("expected Type, Skills and Difficulty in the left column, then Topics in the right one")
	}

	// the difficulty choice no longer sits as its own row under the paragraph editor
	paragraphAt := strings.Index(html, `id="paragraphDiv"`)
	languageTabsAt := strings.Index(html, `id="langTabs"`)
	if paragraphAt < 0 || languageTabsAt < paragraphAt {
		t.Fatalf("could not find the paragraph editor and the language tabs")
	}
	if strings.Contains(html[paragraphAt:languageTabsAt], "Difficulty Rating") {
		t.Errorf("the difficulty choice should have moved out of the row below the paragraph editor")
	}
}

// The additional comprehension problems pick their topics the way the main problem does, and have no concepts.
func TestAdditionalProblemBlockHasATopicsChecklist(t *testing.T) {
	html := renderAddProblem(t, dto.ProblemData{ChapterPtr: &models.Chapter{ID: 9}})

	start := strings.Index(html, `<template id="tmplAdditionalProblem">`)
	end := strings.Index(html, `<template id="tmplAnsNumeric">`)
	if start < 0 || end < start {
		t.Fatalf("could not find the additional problem template")
	}
	block := html[start:end]

	for _, want := range []string{
		`additional-topics-checklist h-60 overflow-y-auto`,
		`hx-get="/chapter/add-problem/add-topic-dialog"`,
		// the picker adds to this block's own list
		`window._topicModalTargetChecklist = this.closest('.additional-problem').querySelector('.additional-topics-checklist')`,
		`+ Add topic from other chapter`,
	} {
		if !strings.Contains(block, want) {
			t.Errorf("the additional problem block is missing %q", want)
		}
	}
	for _, unwanted := range []string{
		"additional-topic-dropdown", "additional-concepts-dropdown", "add-concept-dialog", "Concepts",
	} {
		if strings.Contains(block, unwanted) {
			t.Errorf("the additional problem block should not have %q any more", unwanted)
		}
	}
}

// Every additional problem is saved with its own ticked topics, and nothing about concepts is left on the screen.
func TestAdditionalProblemsSaveTheirTickedTopics(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_problem.html")
	if err != nil {
		t.Fatalf("read add_problem.html: %v", err)
	}
	script := strings.ReplaceAll(string(content), "\r\n", "\n")

	for _, want := range []string{
		"function fillAdditionalTopics(problemBlock) {",
		// a new block copies the main problem's list: the same rows, with the same ones ticked
		"Array.from(topicsChecklist.children, row => row.cloneNode(true))",
		"topicsChecklist.addEventListener('htmx:afterSwap', copy, { once: true });",
		"topicIds: getCheckedTopicIds(problemBlock.querySelector('.additional-topics-checklist')),",
		"topic_ids: topicIds,",
		// saving waits until every block's topics are in
		"block.querySelector('.additional-topics-checklist').dataset.loaded !== 'true'",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("add_problem.html is missing %q", want)
		}
	}

	for _, unwanted := range []string{
		"loadConceptOptions", "loadAdditionalConcepts", "loadTopicOptions", "conceptsOptionsByTopic",
		"topicsOptionsHtml", "currentTopicId", "topic_id:", "additional-topic-dropdown",
		// no request of its own any more: the chapter id and the loader that used it are gone
		"currentChapterId", "loadAdditionalTopics", "fetchText(`/api/topics",
	} {
		if strings.Contains(script, unwanted) {
			t.Errorf("add_problem.html should not contain %q any more", unwanted)
		}
	}
	if strings.Contains(strings.ToLower(script), "concept") {
		t.Errorf("add_problem.html should not mention concepts any more")
	}
}

// The concept modal had no other user than this screen, so its template, route and handler are gone.
func TestConceptModalIsGone(t *testing.T) {
	if _, err := os.Stat("../../web/html/add_concept_modal.html"); err == nil {
		t.Errorf("add_concept_modal.html should have been removed")
	}

	for path, unwanted := range map[string][]string{
		"../../cmd/main.go":                          {"add-concept-dialog", "AddConceptModal"},
		"../../internal/handlers/problem_handler.go": {"AddConceptModal", "addConceptModalTemplate"},
	} {
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
}

// An additional problem is laid out like the main one: skills and difficulty in the left 40%, topics in the right 60%.
func TestAdditionalProblemBlockLayoutMatchesTheMainBlock(t *testing.T) {
	html := renderAddProblem(t, dto.ProblemData{ChapterPtr: &models.Chapter{ID: 9}})

	start := strings.Index(html, `<template id="tmplAdditionalProblem">`)
	end := strings.Index(html, `<template id="tmplAnsNumeric">`)
	if start < 0 || end < start {
		t.Fatalf("could not find the additional problem template")
	}
	block := html[start:end]

	gridAt := strings.Index(block, `grid grid-cols-1 md:grid-cols-5`)
	leftColumnAt := strings.Index(block, `<div class="space-y-6 md:col-span-2">`)
	skillsAt := strings.Index(block, `Skills</p>`)
	difficultyAt := strings.Index(block, `Difficulty Rating`)
	rightColumnAt := strings.Index(block, `<div class="md:col-span-3">`)
	topicsAt := strings.Index(block, `Topics</p>`)

	for name, at := range map[string]int{
		"grid": gridAt, "left column": leftColumnAt, "skills": skillsAt,
		"difficulty": difficultyAt, "right column": rightColumnAt, "topics": topicsAt,
	} {
		if at < 0 {
			t.Fatalf("the additional problem block is missing its %s", name)
		}
	}

	if !(gridAt < leftColumnAt && leftColumnAt < skillsAt && skillsAt < difficultyAt &&
		difficultyAt < rightColumnAt && rightColumnAt < topicsAt) {
		t.Errorf("expected Skills and Difficulty in the left column, then Topics in the right one")
	}
}

// A saved copy opens its destination chapter on the Problems tab; add and edit go back to the list they came from.
func TestSavedCopyOpensTheChaptersProblemsTab(t *testing.T) {
	content, err := os.ReadFile("../../web/html/add_problem.html")
	if err != nil {
		t.Fatalf("read add_problem.html: %v", err)
	}
	script := strings.ReplaceAll(string(content), "\r\n", "\n")

	copyAt := strings.Index(script, "if (isCopyPage) {")
	otherwiseAt := strings.Index(script[copyAt+1:], "} else {")
	if copyAt < 0 || otherwiseAt < 0 {
		t.Fatalf("could not find the copy and the add or edit branches of the save handling")
	}
	copyBranch := script[copyAt : copyAt+1+otherwiseAt]
	addOrEditBranch := script[copyAt+1+otherwiseAt:]

	request := "sessionStorage.setItem('chapterTabOnce', 'problems-sub-tab');"
	if !strings.Contains(copyBranch, request) {
		t.Errorf("the copy branch should ask the chapter screen to open its Problems tab")
	}
	// the request has to be stored before the chapter is loaded, or the chapter would read nothing
	if strings.Index(copyBranch, request) > strings.Index(copyBranch, "htmx.ajax('GET', chapterUrl") {
		t.Errorf("the request must be stored before the chapter is loaded")
	}
	if !strings.Contains(copyBranch, "const chapterUrl = '/chapter?id=") {
		t.Errorf("a saved copy should open its chapter")
	}

	// add and edit keep going back to their list, and must not open a tab of their own
	if strings.Contains(addOrEditBranch, "chapterTabOnce") {
		t.Errorf("add and edit should not ask for a tab: they go back to the list they came from")
	}
	if !strings.Contains(addOrEditBranch, "goBackAfterDelay(0);") {
		t.Errorf("add and edit should still go back")
	}
}
