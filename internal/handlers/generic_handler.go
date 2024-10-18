package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

/*
Handles loading html template files having same name as that of path passed
in request. Path containing only '/' is considered as "/home" resulting in
loading web/html/home.html file
*/
func GenericHandler(w http.ResponseWriter, r *http.Request) {

	// Extract the requested path
	path := r.URL.Path
	var data HomeChapterData
	if initialLoad := path == "/"; initialLoad {
		data = HomeChapterData{
			true,
			nil,
		}
	}

	if path == "/" {
		path = "/home"
	}
	// All urls contain -, which are replaced by _ in file names, hence replace hyphens by underscores
	filename := strings.ReplaceAll(path, "-", "_")
	// Define the template file path
	filePath := filepath.Join("web/html", filename+".html")

	// Parse the template
	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		http.NotFound(w, r)
		log.Printf("Template not found: %s", filePath)
		return
	}

	// Render the template
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		log.Printf("Error executing template: %s", err)
	}
}
