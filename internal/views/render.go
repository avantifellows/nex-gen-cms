package views

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
)

func ExecuteTemplate(filename string, responseWriter http.ResponseWriter, data any, funcMap template.FuncMap) {
	tmplPath := filepath.Join(constants.GetHtmlFolderPath(), filename)
	var tmpl *template.Template
	if funcMap != nil {
		tmpl = template.Must(template.New(filename).Funcs(funcMap).ParseFiles(tmplPath))
	} else {
		tmpl = template.Must(template.ParseFiles(tmplPath))
	}
	tmpl.Execute(responseWriter, data)
}

func ExecuteTemplates(responseWriter http.ResponseWriter, data any, funcMap template.FuncMap, templateFiles ...string) {
	if len(templateFiles) == 0 {
		http.Error(responseWriter, "No template files provided", http.StatusInternalServerError)
		return
	}

	htmlFolderPath := constants.GetHtmlFolderPath()
	var fullPaths []string
	for _, file := range templateFiles {
		fullPaths = append(fullPaths, filepath.Join(htmlFolderPath, file))
	}

	var tmpl *template.Template
	var err error

	if funcMap != nil {
		tmpl, err = template.New(templateFiles[0]).Funcs(funcMap).ParseFiles(fullPaths...)
	} else {
		tmpl, err = template.ParseFiles(fullPaths...)
	}

	if err != nil {
		log.Println("Template Parsing Error:", err)
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(responseWriter, templateFiles[0], data)
	if err != nil {
		log.Println("Template Execution Error:", err)
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
	}
}
