package di

import (
	"net/http"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

type AppComponent struct {
	CssPathHandler     http.Handler
	ChaptersHandler    *handlers.ChaptersHandler
	CurriculumsHandler *handlers.CurriculumsHandler
	GradesHandler      *handlers.GradesHandler
	SubjectsHandler    *handlers.SubjectsHandler
}

func NewAppComponent() (*AppComponent, error) {
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
	cssPathHandler := http.StripPrefix("/web/", http.FileServer(http.Dir("./web")))
	chaptersHandler := handlers.NewChaptersHandler(chaptersService, topicsService)
	// topicsHandler := handlers.NewTopicsHandler(topicsService)
	curriculumsHandler := handlers.NewCurriculumsHandler(curriculumsService)
	gradesHandler := handlers.NewGradesHandler(gradesService)
	subjectsHandler := handlers.NewSubjectsHandler(subjectsService)

	return &AppComponent{
		CssPathHandler:     cssPathHandler,
		ChaptersHandler:    chaptersHandler,
		CurriculumsHandler: curriculumsHandler,
		GradesHandler:      gradesHandler,
		SubjectsHandler:    subjectsHandler,
	}, nil
}
