package main

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/di"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
)

func main() {
	// New mux object is created here instead of using Default via http, so that we can create its mock in testing
	mux := http.NewServeMux()
	appComponentPtr, err := di.NewAppComponent()
	if err != nil {
		panic(err)
	}

	setup(new(Config), mux, appComponentPtr)
	http.ListenAndServe(":8080", mux)
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
	muxHandler.HandleFunc("/modules", handlers.GenericHandler)
	muxHandler.HandleFunc("/books", handlers.GenericHandler)
	muxHandler.HandleFunc("/major-tests", handlers.GenericHandler)
	muxHandler.HandleFunc("/add-chapter", handlers.GenericHandler)

	chaptersHandler := appComponentPtr.ChaptersHandler
	muxHandler.HandleFunc("/chapters", chaptersHandler.LoadChapters)
	muxHandler.HandleFunc("/api/curriculums", appComponentPtr.CurriculumsHandler.GetCurriculums)
	muxHandler.HandleFunc("/api/grades", appComponentPtr.GradesHandler.GetGrades)
	muxHandler.HandleFunc("/api/subjects", appComponentPtr.SubjectsHandler.GetSubjects)
	muxHandler.HandleFunc("/api/chapters", chaptersHandler.GetChapters)
	muxHandler.HandleFunc("/edit-chapter", chaptersHandler.EditChapter)
	muxHandler.HandleFunc("/update-chapter", chaptersHandler.UpdateChapter)
	muxHandler.HandleFunc("/create-chapter", chaptersHandler.AddChapter)
	muxHandler.HandleFunc("/delete-chapter", chaptersHandler.DeleteChapter)
}
