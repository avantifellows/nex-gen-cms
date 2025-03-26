package local_repo

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/avantifellows/nex-gen-cms/internal/constants"
	"github.com/patrickmn/go-cache"
)

// CacheRepository wraps go-cache functionality
type CacheRepository struct {
	cache *cache.Cache
}

// NewCacheRepository creates a new cache repository
func NewCacheRepository(defaultExpiration, cleanupInterval time.Duration) *CacheRepository {
	return &CacheRepository{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

// Set sets a value in the cache
func (r *CacheRepository) Set(key string, value any) {
	r.cache.Set(key, value, cache.DefaultExpiration)
}

// Get retrieves a value from the cache
func (r *CacheRepository) Get(key string) (any, bool) {
	return r.cache.Get(key)
}

// Delete removes an item from the cache
func (r *CacheRepository) Delete(key string) {
	r.cache.Delete(key)
}

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

func ExecuteTemplates(baseFileName string, contentFileName string, responseWriter http.ResponseWriter, data any,
	funcMap template.FuncMap) {
	htmlFolderPath := constants.GetHtmlFolderPath()
	baseTmplPath := filepath.Join(htmlFolderPath, baseFileName)
	contentTmplPath := filepath.Join(htmlFolderPath, contentFileName)
	var tmpl *template.Template

	if funcMap != nil {
		tmpl = template.Must(template.New("base").Funcs(funcMap).ParseFiles(baseTmplPath, contentTmplPath))
	} else {
		// Direct parsing if no FuncMap
		tmpl = template.Must(template.ParseFiles(baseTmplPath, contentTmplPath))
	}

	err := tmpl.ExecuteTemplate(responseWriter, "base", data)
	if err != nil {
		log.Println("Template Execution Error:", err)
		http.Error(responseWriter, "Internal Server Error", http.StatusInternalServerError)
	}
}
