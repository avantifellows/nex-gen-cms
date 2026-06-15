package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
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

func getCurriculumGradeSubjectIds(urlValues url.Values) (int16, int8, int8) {
	// these query parameters can be queried by element names only, not ids
	curriculumId, _ := utils.StringToIntType[int16](urlValues.Get(CURRICULUM_DROPDOWN_NAME))
	gradeId, _ := utils.StringToIntType[int8](urlValues.Get(GRADE_DROPDOWN_NAME))
	subjectId, _ := utils.StringToIntType[int8](urlValues.Get(SUBJECT_DROPDOWN_NAME))
	return curriculumId, gradeId, subjectId
}
