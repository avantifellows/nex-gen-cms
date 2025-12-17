package di

import (
	"net/http"
	"strings"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

type AppComponent struct {
	CssPathHandler     http.Handler
	LoginHandler       *handlers.LoginHandler
	ChaptersHandler    *handlers.ChaptersHandler
	TopicsHandler      *handlers.TopicsHandler
	ConceptsHandler    *handlers.ConceptsHandler
	CurriculumsHandler *handlers.CurriculumsHandler
	GradesHandler      *handlers.GradesHandler
	SubjectsHandler    *handlers.SubjectsHandler
	SkillsHandler      *handlers.SkillsHandler
	TestsHandler       *handlers.TestsHandler
	ProblemsHandler    *handlers.ProblemsHandler
	TagsHandler        *handlers.TagsHandler
	ExamsHandler       *handlers.ExamsHandler
}

func NewAppComponent() (*AppComponent, error) {
	// Initialize repositories
	cacheRepo := local_repo.NewCacheRepository(5*time.Minute, 10*time.Minute)
	apiRepo := remote_repo.NewAPIRepository()

	// Initialize service
	chaptersService := services.NewService[models.Chapter](cacheRepo, apiRepo)
	topicsService := services.NewService[models.Topic](cacheRepo, apiRepo)
	conceptsService := services.NewService[models.Concept](cacheRepo, apiRepo)
	curriculumsService := services.NewService[models.Curriculum](cacheRepo, apiRepo)
	gradesService := services.NewService[models.Grade](cacheRepo, apiRepo)
	subjectsService := services.NewService[models.Subject](cacheRepo, apiRepo)
	skillsService := services.NewService[models.Skill](cacheRepo, apiRepo)
	testsService := services.NewService[models.Test](cacheRepo, apiRepo)
	problemsService := services.NewService[models.Problem](cacheRepo, apiRepo)
	tagsService := services.NewService[models.Tag](cacheRepo, apiRepo)
	testRulesService := services.NewService[models.TestRule](cacheRepo, apiRepo)
	examsService := services.NewService[models.Exam](cacheRepo, apiRepo)

	// Initialize handlers
	fileServer := http.StripPrefix("/web/", http.FileServer(http.Dir("./web")))
	cssPathHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS for fonts so they can be fetched from a data: URL origin
		if strings.HasSuffix(r.URL.Path, ".ttf") {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Range, Content-Type, Accept")
		}
		fileServer.ServeHTTP(w, r)
	})
	loginHandler := handlers.NewLoginHandler()
	chaptersHandler := handlers.NewChaptersHandler(chaptersService, topicsService)
	topicsHandler := handlers.NewTopicsHandler(topicsService)
	conceptsHandler := handlers.NewConceptsHandler(conceptsService)
	curriculumsHandler := handlers.NewCurriculumsHandler(curriculumsService)
	gradesHandler := handlers.NewGradesHandler(gradesService)
	subjectsHandler := handlers.NewSubjectsHandler(subjectsService)
	skillsHandler := handlers.NewSkillsHandler(skillsService)
	testsHandler := handlers.NewTestsHandler(testsService, subjectsService, problemsService, testRulesService,
		curriculumsService, gradesService)
	problemsHandler := handlers.NewProblemsHandler(problemsService, skillsService, subjectsService, topicsService,
		tagsService)
	tagsHandler := handlers.NewTagsHandler(tagsService)
	examsHandler := handlers.NewExamsHandler(examsService)

	return &AppComponent{
		CssPathHandler:     cssPathHandler,
		LoginHandler:       loginHandler,
		ChaptersHandler:    chaptersHandler,
		TopicsHandler:      topicsHandler,
		ConceptsHandler:    conceptsHandler,
		CurriculumsHandler: curriculumsHandler,
		GradesHandler:      gradesHandler,
		SubjectsHandler:    subjectsHandler,
		SkillsHandler:      skillsHandler,
		TestsHandler:       testsHandler,
		ProblemsHandler:    problemsHandler,
		TagsHandler:        tagsHandler,
		ExamsHandler:       examsHandler,
	}, nil
}
