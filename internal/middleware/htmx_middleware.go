package middleware

import (
	"net/http"
	"sync"
)

// Singleton structure to hold the middleware
type HTMXMiddleware struct {
	handler http.Handler
	lock    sync.RWMutex
}

// Set the next handler
func (middleware *HTMXMiddleware) SetNext(next http.Handler) {
	middleware.lock.Lock()
	defer middleware.lock.Unlock()
	middleware.handler = next
}

// ServeHTTP handles the request
func (middleware *HTMXMiddleware) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	middleware.lock.RLock()
	defer middleware.lock.RUnlock()

	// Check if the request is from HTMX
	if request.Header.Get("HX-Request") == "" {
		// If the request is NOT from HTMX, redirect to the home page
		http.Redirect(responseWriter, request, "/", http.StatusSeeOther)
		return
	}

	// Call the next handler if set
	if middleware.handler != nil {
		middleware.handler.ServeHTTP(responseWriter, request)
	}
}

// RequireHTMX returns the same middleware instance
var instance *HTMXMiddleware
var once sync.Once

func RequireHTMX(next http.Handler) http.Handler {
	once.Do(func() {
		instance = &HTMXMiddleware{}
	})
	instance.SetNext(next)
	return instance
}
