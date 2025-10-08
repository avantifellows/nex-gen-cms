package middleware

import "net/http"

const sessionCookieName = "cms_session"

// Set a cookie for a logged-in user
func SetSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "true",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   7200, // 2 hours
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
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
}

// Middleware to restrict access
func RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsLoggedIn(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
