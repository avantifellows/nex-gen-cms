package middleware

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/internal/auth"
)

// RequireLogin verifies the session cookie. Unauthenticated requests are redirected to /login
// (or sent HX-Redirect for htmx requests). Exception paths skip the check entirely.
func RequireLogin(next http.Handler, exceptions ...string) http.Handler {
	exceptionSet := make(map[string]struct{}, len(exceptions))
	for _, e := range exceptions {
		exceptionSet[e] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := exceptionSet[r.URL.Path]; ok {
			next.ServeHTTP(w, r)
			return
		}

		claims := auth.ReadSession(r)
		if claims == nil {
			redirectToLogin(w, r)
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.WithSession(r.Context(), claims)))
	})
}

// RequireRole wraps a handler so that only sessions with role >= need can reach it.
// Lower roles get 403 (or HX-equivalent). Unauthenticated requests fall through to /login.
func RequireRole(need string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.FromContext(r.Context())
		if claims == nil {
			claims = auth.ReadSession(r)
		}
		if claims == nil {
			redirectToLogin(w, r)
			return
		}
		if !auth.AtLeast(claims.Role, need) {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Reswap", "none")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithSession(r.Context(), claims)))
	})
}

// RequireRoleFunc is the http.HandlerFunc-flavored convenience wrapper.
func RequireRoleFunc(need string, h http.HandlerFunc) http.HandlerFunc {
	return RequireRole(need, h).ServeHTTP
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
