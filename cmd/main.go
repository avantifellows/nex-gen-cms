package main

import (
	"net/http"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// this is for output.css file used in home.html
	http.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("./web"))))

	http.HandleFunc("/", handlers.GenericHandler)
	http.HandleFunc("/modules", handlers.GenericHandler)
	http.HandleFunc("/books", handlers.GenericHandler)
	http.HandleFunc("/major-tests", handlers.GenericHandler)
	http.HandleFunc("/add-chapter", handlers.GenericHandler)

	// Initialize repositories
	cacheRepo := local_repo.NewCacheRepository(5*time.Minute, 10*time.Minute)
	apiRepo := remote_repo.NewAPIRepository()

	// Initialize service
	topicsService := services.NewService[models.Topic](cacheRepo, apiRepo)
	chaptersService := services.NewService[models.Chapter](cacheRepo, apiRepo)
	curriculumsService := services.NewService[models.Curriculum](cacheRepo, apiRepo)
	gradesService := services.NewService[models.Grade](cacheRepo, apiRepo)
	subjectsService := services.NewService[models.Subject](cacheRepo, apiRepo)

	// Initialize handlers
	topicsHandler := handlers.NewTopicsHandler(topicsService)
	chaptersHandler := handlers.NewChaptersHandler(chaptersService, topicsService)
	curriculumsHandler := handlers.NewCurriculumsHandler(curriculumsService)
	gradesHandler := handlers.NewGradesHandler(gradesService)
	subjectsHandler := handlers.NewSubjectsHandler(subjectsService)

	http.HandleFunc("/chapters", chaptersHandler.LoadChapters)
	http.HandleFunc("/api/curriculums", curriculumsHandler.GetCurriculums)
	http.HandleFunc("/api/grades", gradesHandler.GetGrades)
	http.HandleFunc("/api/subjects", subjectsHandler.GetSubjects)
	http.HandleFunc("/api/chapters", chaptersHandler.GetChapters)
	http.Handle("/edit-chapter", handlers.RequireHTMX(http.HandlerFunc(chaptersHandler.EditChapter)))
	http.HandleFunc("/update-chapter", chaptersHandler.UpdateChapter)
	http.HandleFunc("/create-chapter", chaptersHandler.AddChapter)
	http.HandleFunc("/delete-chapter", chaptersHandler.DeleteChapter)
	http.HandleFunc("/chapter", chaptersHandler.GetChapter)
	http.HandleFunc("/topics", chaptersHandler.LoadTopics)
	http.HandleFunc("/api/topics", chaptersHandler.GetTopics)
	http.HandleFunc("/add-topic", topicsHandler.OpenAddTopic)
	http.HandleFunc("/create-topic", topicsHandler.AddTopic)
	http.HandleFunc("/delete-topic", topicsHandler.DeleteTopic)
	http.Handle("/edit-topic", handlers.RequireHTMX(http.HandlerFunc(topicsHandler.EditTopic)))
	http.HandleFunc("/update-topic", topicsHandler.UpdateTopic)

	http.ListenAndServe(":8080", nil)
}
