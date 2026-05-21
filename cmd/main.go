package main

import (
	"log"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/di"
	"github.com/avantifellows/nex-gen-cms/internal/auth"
	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/handlers"
	"github.com/avantifellows/nex-gen-cms/internal/middleware"
)

func main() {
	// Load .env before DI so the OAuth + DB pool can read their env vars at construction time.
	// setup() also calls LoadEnv (via the mockable ConfigLoader interface) so tests stay unchanged;
	// godotenv.Load is a no-op the second time around.
	config.LoadEnv(new(config.Env))

	mux := http.NewServeMux()
	appComponentPtr, err := di.NewAppComponent()
	if err != nil {
		log.Fatalf("startup: %v", err)
	}

	setup(new(Config), mux, appComponentPtr)

	// Paths that don't require a session cookie.
	exceptions := []string{
		"/login",
		"/favicon.ico",
		"/web/static/css/output.css",
		"/auth/google/start",
		"/auth/google/callback",
		"/dev-login",
	}

	addr := "0.0.0.0:8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, middleware.RequireLogin(mux, exceptions...)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

type ConfigLoader interface {
	LoadEnv(loader config.EnvLoader)
}

type Config struct{}

func (c *Config) LoadEnv(loader config.EnvLoader) {
	config.LoadEnv(loader)
}

type MuxHandler interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(string, func(http.ResponseWriter, *http.Request))
}

// editor and admin are short aliases to keep the route table readable.
func editor(h http.HandlerFunc) http.HandlerFunc {
	return middleware.RequireRoleFunc(auth.RoleEditor, h)
}
func admin(h http.HandlerFunc) http.HandlerFunc {
	return middleware.RequireRoleFunc(auth.RoleAdmin, h)
}

func setup(configLoader ConfigLoader, muxHandler MuxHandler, appComponentPtr *di.AppComponent) {
	constants.InitRuntimeConstant()
	configLoader.LoadEnv(new(config.Env))

	muxHandler.Handle("/web/", appComponentPtr.CssPathHandler)

	muxHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		http.NotFound(w, r)
	})
	muxHandler.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	loginHandler := appComponentPtr.LoginHandler
	muxHandler.HandleFunc("/login", loginHandler.Login)
	muxHandler.HandleFunc("/logout", loginHandler.Logout)
	muxHandler.HandleFunc("/auth/google/start", loginHandler.StartGoogleAuth)
	muxHandler.HandleFunc("/auth/google/callback", loginHandler.GoogleCallback)
	// Dev-only bypass — guarded inside the handler by DEV_LOGIN_EMAIL being unset.
	muxHandler.HandleFunc("/dev-login", loginHandler.DevLogin)

	muxHandler.HandleFunc("/home", handlers.GenericHandler)
	muxHandler.HandleFunc("/add-chapter", handlers.GenericHandler)

	// Admin user management
	adminUsers := appComponentPtr.AdminUsersHandler
	muxHandler.HandleFunc("/admin/users", admin(adminUsers.List))
	muxHandler.HandleFunc("/admin/users/active", admin(adminUsers.SetActive))
	muxHandler.HandleFunc("/admin/users/role", admin(adminUsers.UpdateRole))

	chaptersHandler := appComponentPtr.ChaptersHandler
	muxHandler.HandleFunc("/chapters", chaptersHandler.LoadChapters)
	muxHandler.HandleFunc("/api/curriculums", appComponentPtr.CurriculumsHandler.GetCurriculums)
	muxHandler.HandleFunc("/api/grades", appComponentPtr.GradesHandler.GetGrades)
	muxHandler.HandleFunc("/api/subjects", appComponentPtr.SubjectsHandler.GetSubjects)
	muxHandler.HandleFunc("/api/skills", appComponentPtr.SkillsHandler.GetSkills)

	muxHandler.HandleFunc("/api/chapters", chaptersHandler.GetChapters)
	muxHandler.Handle("/edit-chapter", middleware.RequireHTMX(middleware.RequireRole(auth.RoleEditor, http.HandlerFunc(chaptersHandler.EditChapter))))
	muxHandler.HandleFunc("/update-chapter", editor(chaptersHandler.UpdateChapter))
	muxHandler.HandleFunc("/create-chapter", editor(chaptersHandler.AddChapter))
	muxHandler.HandleFunc("/archive-chapter", editor(chaptersHandler.ArchiveChapter))
	muxHandler.HandleFunc("/chapter", chaptersHandler.GetChapter)
	muxHandler.HandleFunc("/topics", chaptersHandler.LoadTopics)
	muxHandler.HandleFunc("/api/topics", chaptersHandler.GetTopics)
	muxHandler.HandleFunc("/chapter/resources", chaptersHandler.LoadResources)

	topicsHandler := appComponentPtr.TopicsHandler
	muxHandler.HandleFunc("/add-topic", editor(topicsHandler.OpenAddTopic))
	muxHandler.HandleFunc("/create-topic", editor(topicsHandler.AddTopic))
	muxHandler.HandleFunc("/archive-topic", editor(topicsHandler.ArchiveTopic))
	muxHandler.Handle("/edit-topic", middleware.RequireHTMX(middleware.RequireRole(auth.RoleEditor, http.HandlerFunc(topicsHandler.EditTopic))))
	muxHandler.HandleFunc("/update-topic", editor(topicsHandler.UpdateTopic))
	muxHandler.HandleFunc("/topic", topicsHandler.GetTopic)
	muxHandler.HandleFunc("/topic/resources", topicsHandler.LoadResources)

	resourcesHandler := appComponentPtr.ResourcesHandler
	muxHandler.HandleFunc("/add-resource", editor(resourcesHandler.OpenAddResource))
	muxHandler.HandleFunc("/create-resource", editor(resourcesHandler.AddResource))
	muxHandler.HandleFunc("/api/resources", resourcesHandler.GetResources)
	muxHandler.Handle("/edit-resource", middleware.RequireHTMX(middleware.RequireRole(auth.RoleEditor, http.HandlerFunc(resourcesHandler.EditResource))))
	muxHandler.HandleFunc("/update-resource", editor(resourcesHandler.UpdateResource))
	muxHandler.HandleFunc("/delete-resource", editor(resourcesHandler.DeleteResource))
	muxHandler.HandleFunc("/resources/move-resource", editor(resourcesHandler.LoadMoveResources))
	muxHandler.HandleFunc("/move-resource", editor(resourcesHandler.MoveResource))

	conceptsHandler := appComponentPtr.ConceptsHandler
	muxHandler.HandleFunc("/api/concepts", conceptsHandler.GetConcepts)

	testsHandler := appComponentPtr.TestsHandler
	muxHandler.HandleFunc("/tests", testsHandler.LoadTests)
	muxHandler.HandleFunc("/api/tests", testsHandler.GetTests)
	muxHandler.HandleFunc("/api/search-tests", testsHandler.GetSearchTests)
	muxHandler.HandleFunc("/test", testsHandler.GetTest)
	muxHandler.HandleFunc("/api/test/problems", testsHandler.GetTestProblems)
	muxHandler.HandleFunc("/api/test/subjectwise-problems", testsHandler.GetSubjectwiseTestProblems)
	muxHandler.HandleFunc("/tests/add-test", editor(testsHandler.AddTest))
	muxHandler.HandleFunc("/add-question-to-test", editor(testsHandler.AddQuestionToTest))
	muxHandler.HandleFunc("/create-test", editor(testsHandler.CreateTest))
	muxHandler.HandleFunc("/tests/edit-test", editor(testsHandler.EditTest))
	muxHandler.Handle("/tests/add-test-dialog", middleware.RequireHTMX(middleware.RequireRole(auth.RoleEditor, http.HandlerFunc(testsHandler.AddTestModal))))
	muxHandler.HandleFunc("/add-curriculum-grade-selects", editor(testsHandler.AddCurriculumGradeDropdowns))
	muxHandler.HandleFunc("/update-test", editor(testsHandler.UpdateTest))
	muxHandler.HandleFunc("/update-test-subject", editor(testsHandler.UpdateTestSubject))
	muxHandler.HandleFunc("/archive-test", editor(testsHandler.ArchiveTest))
	muxHandler.HandleFunc("/download-pdf", testsHandler.DownloadPdf)
	muxHandler.HandleFunc("/tests/copy-test", editor(testsHandler.CopyTest))
	muxHandler.HandleFunc("/tests/validate-test", testsHandler.ValidateTest)

	problemsHandler := appComponentPtr.ProblemsHandler
	muxHandler.HandleFunc("/problems", problemsHandler.LoadProblems)
	muxHandler.HandleFunc("/problem", problemsHandler.GetProblem)
	muxHandler.HandleFunc("/api/topic/problems", problemsHandler.GetTopicProblems)
	muxHandler.HandleFunc("/topic/problems", problemsHandler.LoadTopicProblems)
	muxHandler.HandleFunc("/topic/add-problem", editor(problemsHandler.AddProblem))
	muxHandler.HandleFunc("/topic/add-problem/add-concept-dialog", editor(problemsHandler.AddConceptModal))
	muxHandler.HandleFunc("/create-problem", editor(problemsHandler.CreateProblem))
	muxHandler.HandleFunc("/problems/edit-problem", editor(problemsHandler.EditProblem))
	muxHandler.HandleFunc("/update-problem", editor(problemsHandler.UpdateProblem))
	muxHandler.HandleFunc("/archive-problem", editor(problemsHandler.ArchiveProblem))
	muxHandler.HandleFunc("/api/search-problems", problemsHandler.GetSearchProblems)
	muxHandler.HandleFunc("/problems/test-associations", problemsHandler.LoadTestAssociations)
	muxHandler.HandleFunc("/problems/move-problems", editor(problemsHandler.LoadMoveProblems))
	muxHandler.HandleFunc("/move-problems", editor(problemsHandler.MoveProblems))

	tagsHandler := appComponentPtr.TagsHandler
	muxHandler.HandleFunc("/api/tags", tagsHandler.GetTags)

	examsHandler := appComponentPtr.ExamsHandler
	muxHandler.HandleFunc("/api/exams", examsHandler.GetExams)
}
