package main

import (
	"net/http"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

func main() {
	// New mux object is created here instead of using Default via http, so that we can create its mock in testing
	mux := http.NewServeMux()
	setup(new(Config), mux)
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

func setup(configLoader ConfigLoader, muxHandler MuxHandler) {
	constants.InitRuntimeConstant()

	// Load environment variables
	configLoader.LoadEnv(new(config.Env))

	// this is for output.css file used in home.html
	muxHandler.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("./web"))))

	muxHandler.HandleFunc("/", handlers.GenericHandler)
	muxHandler.HandleFunc("/modules", handlers.GenericHandler)
	muxHandler.HandleFunc("/books", handlers.GenericHandler)
	muxHandler.HandleFunc("/major-tests", handlers.GenericHandler)
	muxHandler.HandleFunc("/add-chapter", handlers.GenericHandler)

	// Initialize repositories
	cacheRepo := local_repo.NewCacheRepository(5*time.Minute, 10*time.Minute)
	apiRepo := remote_repo.NewAPIRepository()

	// Initialize service
	topicsService := services.NewService[models.Topic](cacheRepo, apiRepo)
	chaptersService := services.NewChapterService(cacheRepo, apiRepo)
	curriculumsService := services.NewService[models.Curriculum](cacheRepo, apiRepo)
	gradesService := services.NewService[models.Grade](cacheRepo, apiRepo)
	subjectsService := services.NewService[models.Subject](cacheRepo, apiRepo)

	// Initialize handlers
	// topicsHandler := handlers.NewTopicsHandler(topicsService)
	chaptersHandler := handlers.NewChaptersHandler(chaptersService, topicsService)
	curriculumsHandler := handlers.NewCurriculumsHandler(curriculumsService)
	gradesHandler := handlers.NewGradesHandler(gradesService)
	subjectsHandler := handlers.NewSubjectsHandler(subjectsService)

	muxHandler.HandleFunc("/chapters", chaptersHandler.LoadChapters)
	muxHandler.HandleFunc("/api/curriculums", curriculumsHandler.GetCurriculums)
	muxHandler.HandleFunc("/api/grades", gradesHandler.GetGrades)
	muxHandler.HandleFunc("/api/subjects", subjectsHandler.GetSubjects)
	muxHandler.HandleFunc("/api/chapters", chaptersHandler.GetChapters)
	muxHandler.HandleFunc("/edit-chapter", chaptersHandler.EditChapter)
	muxHandler.HandleFunc("/update-chapter", chaptersHandler.UpdateChapter)
	muxHandler.HandleFunc("/create-chapter", chaptersHandler.AddChapter)
	muxHandler.HandleFunc("/delete-chapter", chaptersHandler.DeleteChapter)
}
