package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/services"
	"github.com/avantifellows/nex-gen-cms/internal/views"
	"github.com/avantifellows/nex-gen-cms/utils"
)

const baseTemplate = "home.html"

/*
Handles loading html template files having same name as that of path passed
in request.
*/
func GenericHandler(responseWriter http.ResponseWriter, request *http.Request) {
	// Extract the requested path
	path := request.URL.Path
	// All urls contain -, which are replaced by _ in file names, hence replace hyphens by underscores
	filename := strings.ReplaceAll(path, "-", "_")
	// Define the template file path
	filePath := filepath.Join(constants.GetHtmlFolderPath(), filename+".html")

	// Parse the template
	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.NotFound(responseWriter, request)
		log.Printf("Template not found: %s", filePath)
		return
	}

	// Render the template
	if err := tmpl.Execute(responseWriter, nil); err != nil {
		http.Error(responseWriter, "Error rendering template", http.StatusInternalServerError)
		log.Printf("Error executing template: %s", err)
	}
}

// renderEntityList fetches a cached list from the given endpoint and renders it
// with the given template. On failure it writes a 500 referencing label (e.g.
// "grades", "exams"). It backs the otherwise-identical simple list handlers.
func renderEntityList[T any](responseWriter http.ResponseWriter, service *services.Service[T],
	endpoint, cacheKey, tmpl, label string) {
	items, err := service.GetList(endpoint, cacheKey, false, false)
	if err != nil {
		http.Error(responseWriter, fmt.Sprintf("Error fetching %s: %v", label, err), http.StatusInternalServerError)
		return
	}
	views.ExecuteTemplate(tmpl, responseWriter, items, nil)
}

// parseCodeAndName trims the "code" and "name" form values and writes a 400
// error if either is blank. ok is false when validation failed, in which case
// the caller should return without proceeding.
func parseCodeAndName(responseWriter http.ResponseWriter, request *http.Request, entityLabel string) (code, name string, ok bool) {
	code = strings.TrimSpace(request.FormValue("code"))
	name = strings.TrimSpace(request.FormValue("name"))
	if code == "" || name == "" {
		http.Error(responseWriter, fmt.Sprintf("%s code and name cannot be blank", entityLabel), http.StatusBadRequest)
		return code, name, false
	}
	return code, name, true
}

func getCurriculumGradeSubjectIds(urlValues url.Values) (int16, int8, int8) {
	// these query parameters can be queried by element names only, not ids
	curriculumId, _ := utils.StringToIntType[int16](urlValues.Get(CURRICULUM_DROPDOWN_NAME))
	gradeId, _ := utils.StringToIntType[int8](urlValues.Get(GRADE_DROPDOWN_NAME))
	subjectId, _ := utils.StringToIntType[int8](urlValues.Get(SUBJECT_DROPDOWN_NAME))
	return curriculumId, gradeId, subjectId
}
