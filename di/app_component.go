package di

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/auth"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/models"
	pgrepo "github.com/avantifellows/nex-gen-cms/internal/repositories/db"
	local_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/local"
	remote_repo "github.com/avantifellows/nex-gen-cms/internal/repositories/remote"
	"github.com/avantifellows/nex-gen-cms/internal/services"
)

type AppComponent struct {
	DB                 *sql.DB
	CssPathHandler     http.Handler
	LoginHandler       *handlers.LoginHandler
	AdminUsersHandler  *handlers.AdminUsersHandler
	ChaptersHandler    *handlers.ChaptersHandler
	ResourcesHandler   *handlers.ResourcesHandler
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
	// Auth-related dependencies (Postgres + Google OIDC) are constructed before app handlers so that
	// failures here surface as startup errors rather than runtime 500s.
	database, err := pgrepo.Open()
	if err != nil {
		return nil, err
	}
	usersRepo := pgrepo.NewCmsUserRepo(database)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	googleAuth, err := auth.NewGoogleAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Existing content services (cache + DB-service API)
	cacheRepo := local_repo.NewCacheRepository(5*time.Minute, 10*time.Minute)
	apiRepo := remote_repo.NewAPIRepository()

	chaptersService := services.NewService[models.Chapter](cacheRepo, apiRepo)
	resourcesService := services.NewService[models.Resource](cacheRepo, apiRepo)
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

	cssPathHandler := http.StripPrefix("/web/", http.FileServer(http.Dir("./web")))
	loginHandler := handlers.NewLoginHandler(googleAuth, usersRepo)
	adminUsersHandler := handlers.NewAdminUsersHandler(usersRepo)
	chaptersHandler := handlers.NewChaptersHandler(chaptersService, topicsService)
	resourcesHandler := handlers.NewResourcesHandler(resourcesService)
	topicsHandler := handlers.NewTopicsHandler(topicsService)
	conceptsHandler := handlers.NewConceptsHandler(conceptsService)
	curriculumsHandler := handlers.NewCurriculumsHandler(curriculumsService)
	gradesHandler := handlers.NewGradesHandler(gradesService)
	subjectsHandler := handlers.NewSubjectsHandler(subjectsService)
	skillsHandler := handlers.NewSkillsHandler(skillsService)
	testsHandler := handlers.NewTestsHandler(testsService, subjectsService, problemsService, testRulesService,
		curriculumsService, gradesService, examsService)
	problemsHandler := handlers.NewProblemsHandler(problemsService, skillsService, subjectsService, topicsService,
		tagsService)
	tagsHandler := handlers.NewTagsHandler(tagsService)
	examsHandler := handlers.NewExamsHandler(examsService)

	return &AppComponent{
		DB:                 database,
		CssPathHandler:     cssPathHandler,
		LoginHandler:       loginHandler,
		AdminUsersHandler:  adminUsersHandler,
		ChaptersHandler:    chaptersHandler,
		ResourcesHandler:   resourcesHandler,
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
