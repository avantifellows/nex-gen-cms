package middleware

import (
	"log"
	"net/http"
)

const sessionCookieName = "cms_session"

// Set a cookie for a logged-in user
func SetSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   sessionCookieName,
		Value:  "true",
		Path:   "/",
		MaxAge: 7200, // 2 hours
	}
	http.SetCookie(w, cookie)
}

// Check if a user is logged in
func IsLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	return cookie.Value == "true"
}

// Logout by clearing the cookie
func Logout(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

// Middleware to restrict access
func RequireLogin(next http.Handler, exceptions ...string) http.Handler {
	exceptionSet := make(map[string]struct{}, len(exceptions))
	for _, e := range exceptions {
		exceptionSet[e] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Middleware: Incoming request: %s %s", r.Method, r.URL.Path)
		if _, ok := exceptionSet[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		if !IsLoggedIn(r) {
			if r.Header.Get("HX-Request") == "true" {
				// For HTMX requests, set HX-Redirect header with StatusUnauthorized; otherwise it will add login
				// screen inside the target element (eg - under tests tab for /api/tests call) instead of
				// moving to separate login screen
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
			} else {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}
