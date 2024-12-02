package local_repo

import (
	"html/template"
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

func ExecuteTemplate(filename string, w http.ResponseWriter, data any) {
	tmplPath := filepath.Join(constants.GetHtmlFolderPath(), filename)
	tmpl := template.Must(template.ParseFiles(tmplPath))
	tmpl.Execute(w, data)
}

func ExecuteTemplates(baseFileName string, contentFileName string, w http.ResponseWriter, data any) {
	htmlFolderPath := constants.GetHtmlFolderPath()
	baseTmplPath := filepath.Join(htmlFolderPath, baseFileName)
	contentTmplPath := filepath.Join(htmlFolderPath, contentFileName)
	tmpl := template.Must(template.ParseFiles(baseTmplPath, contentTmplPath))
	tmpl.Execute(w, data)
}
