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

// ServeHTTP handles the request
func (middleware *HTMXMiddleware) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	middleware.lock.RLock()
	defer middleware.lock.RUnlock()

	// Check if the request is from HTMX
	if request.Header.Get("HX-Request") == "" {
		// If the request is NOT from HTMX, redirect to the home page
		http.Redirect(responseWriter, request, "/chapters", http.StatusSeeOther)
		return
	}

	// Call the next handler if set
	if middleware.handler != nil {
		middleware.handler.ServeHTTP(responseWriter, request)
	}
}

func RequireHTMX(next http.Handler) http.Handler {
	return &HTMXMiddleware{handler: next}
}
