package main

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/di"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/middleware"
)

func main() {
	// New mux object is created here instead of using Default via http, so that we can create its mock in testing
	mux := http.NewServeMux()
	appComponentPtr, err := di.NewAppComponent()
	if err != nil {
		panic(err)
	}

	setup(new(Config), mux, appComponentPtr)
	http.ListenAndServe("0.0.0.0:8080", mux)
}

type ConfigLoader interface {
	LoadEnv(loader config.EnvLoader)
}

type Config struct{}

// Config implements ConfigLoader.
func (c *Config) LoadEnv(loader config.EnvLoader) {
	config.LoadEnv(loader)
}

// Created to make setup() function testable (by implementing this interface for its Mock MockServeMux in main_test.go)
type MuxHandler interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
}

func setup(configLoader ConfigLoader, muxHandler MuxHandler, appComponentPtr *di.AppComponent) {
	constants.InitRuntimeConstant()

	// Load environment variables
	configLoader.LoadEnv(new(config.Env))

	// this is for output.css file used in home.html
	muxHandler.Handle("/web/", appComponentPtr.CssPathHandler)

	muxHandler.HandleFunc("/", handlers.GenericHandler)
	muxHandler.HandleFunc("/add-chapter", handlers.GenericHandler)

	chaptersHandler := appComponentPtr.ChaptersHandler
	muxHandler.HandleFunc("/chapters", chaptersHandler.LoadChapters)
	muxHandler.HandleFunc("/api/curriculums", appComponentPtr.CurriculumsHandler.GetCurriculums)
	muxHandler.HandleFunc("/api/grades", appComponentPtr.GradesHandler.GetGrades)
	muxHandler.HandleFunc("/api/subjects", appComponentPtr.SubjectsHandler.GetSubjects)
	muxHandler.HandleFunc("/api/skills", appComponentPtr.SkillsHandler.GetSkills)

	muxHandler.HandleFunc("/api/chapters", chaptersHandler.GetChapters)
	muxHandler.Handle("/edit-chapter", middleware.RequireHTMX(http.HandlerFunc(chaptersHandler.EditChapter)))
	muxHandler.HandleFunc("/update-chapter", chaptersHandler.UpdateChapter)
	muxHandler.HandleFunc("/create-chapter", chaptersHandler.AddChapter)
	muxHandler.HandleFunc("/delete-chapter", chaptersHandler.DeleteChapter)
	muxHandler.HandleFunc("/chapter", chaptersHandler.GetChapter)
	muxHandler.HandleFunc("/topics", chaptersHandler.LoadTopics)
	muxHandler.HandleFunc("/api/topics", chaptersHandler.GetTopics)

	topicsHandler := appComponentPtr.TopicsHandler
	muxHandler.HandleFunc("/add-topic", topicsHandler.OpenAddTopic)
	muxHandler.HandleFunc("/create-topic", topicsHandler.AddTopic)
	muxHandler.HandleFunc("/delete-topic", topicsHandler.DeleteTopic)
	muxHandler.Handle("/edit-topic", middleware.RequireHTMX(http.HandlerFunc(topicsHandler.EditTopic)))
	muxHandler.HandleFunc("/update-topic", topicsHandler.UpdateTopic)
	muxHandler.HandleFunc("/topic", topicsHandler.GetTopic)

	conceptsHandler := appComponentPtr.ConceptsHandler
	muxHandler.HandleFunc("/api/concepts", conceptsHandler.GetConcepts)

	testsHandler := appComponentPtr.TestsHandler
	muxHandler.HandleFunc("/tests", testsHandler.LoadTests)
	muxHandler.HandleFunc("/api/tests", testsHandler.GetTests)
	muxHandler.HandleFunc("/test", testsHandler.GetTest)
	muxHandler.HandleFunc("/api/test/problems", testsHandler.GetTestProblems)
	muxHandler.HandleFunc("/tests/add-test", testsHandler.AddTest)
	muxHandler.HandleFunc("/add-question-to-test", testsHandler.AddQuestionToTest)
	muxHandler.HandleFunc("/create-test", testsHandler.CreateTest)
	muxHandler.HandleFunc("/tests/edit-test", testsHandler.EditTest)
	muxHandler.Handle("/tests/add-test-dialog", middleware.RequireHTMX(http.HandlerFunc(testsHandler.AddTestModal)))
	muxHandler.HandleFunc("/add-curriculum-grade-selects", testsHandler.AddCurriculumGradeDropdowns)
	muxHandler.HandleFunc("/update-test", testsHandler.UpdateTest)

	problemsHandler := appComponentPtr.ProblemsHandler
	muxHandler.HandleFunc("/problem", problemsHandler.GetProblem)
	muxHandler.HandleFunc("/api/topic/problems", problemsHandler.GetTopicProblems)
	muxHandler.HandleFunc("/problems", problemsHandler.LoadProblems)
	muxHandler.HandleFunc("/topic/add-problem", problemsHandler.AddProblem)
	muxHandler.HandleFunc("/create-problem", problemsHandler.CreateProblem)
	muxHandler.HandleFunc("/problems/edit-problem", problemsHandler.EditProblem)
	muxHandler.HandleFunc("/update-problem", problemsHandler.UpdateProblem)
	muxHandler.HandleFunc("/archive-problem", problemsHandler.ArchiveProblem)

	tagsHandler := appComponentPtr.TagsHandler
	muxHandler.HandleFunc("/api/tags", tagsHandler.GetTags)

	modulesHandler := appComponentPtr.ModulesHandler
	muxHandler.HandleFunc("/modules", modulesHandler.LoadModules)

	booksHandler := appComponentPtr.BooksHandler
	muxHandler.HandleFunc("/books", booksHandler.LoadBooks)
}
