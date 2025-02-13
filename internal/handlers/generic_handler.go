package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/avantifellows/nex-gen-cms/internal/dto"
)

/*
Handles loading html template files having same name as that of path passed
in request. Path containing only '/' is considered as "/home", resulting in
loading web/html/home.html file
*/
func GenericHandler(responseWriter http.ResponseWriter, request *http.Request) {

	// Extract the requested path
	path := request.URL.Path
	var data dto.HomeChapterData
	if initialLoad := path == "/"; initialLoad {
		data = dto.HomeChapterData{
			InitialLoad: true,
			ChapterPtr:  nil,
		}
	}

	if path == "/" {
		path = "/home"
	}
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
	if err := tmpl.Execute(responseWriter, data); err != nil {
		http.Error(responseWriter, "Error rendering template", http.StatusInternalServerError)
		log.Printf("Error executing template: %s", err)
	}
}
