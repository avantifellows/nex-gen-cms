package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/avantifellows/nex-gen-cms/config"
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

// RequireServiceToken guards server-to-server API routes with a shared bearer token
// (CMS_SERVICE_TOKEN). It is the inbound counterpart to the outbound DB_SERVICE_TOKEN
// bearer that api_repository uses when calling db-service. Routes wrapped with this are
// listed in RequireLogin's exceptions so they bypass the Google-OIDC session check
// entirely. Fails closed: an unset CMS_SERVICE_TOKEN rejects every request.
func RequireServiceToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := config.GetEnv("CMS_SERVICE_TOKEN", "")
		got, hasBearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if want == "" || !hasBearer || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireServiceTokenFunc is the http.HandlerFunc-flavored convenience wrapper.
func RequireServiceTokenFunc(h http.HandlerFunc) http.HandlerFunc {
	return RequireServiceToken(h).ServeHTTP
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
