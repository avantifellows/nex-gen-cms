package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/auth"
	"github.com/avantifellows/nex-gen-cms/internal/repositories/db"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const loginTemplate = "login.html"

type LoginHandler struct {
	google *auth.GoogleAuth
	users  *db.CmsUserRepo
}

func NewLoginHandler(google *auth.GoogleAuth, users *db.CmsUserRepo) *LoginHandler {
	return &LoginHandler{google: google, users: users}
}

// Login renders the login page (GET) or shows an error message after a failed OAuth callback.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	if auth.ReadSession(r) != nil {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

	data := map[string]interface{}{
		"DevLoginEmail": config.GetEnv("DEV_LOGIN_EMAIL", ""),
	}
	if msg := r.URL.Query().Get("error"); msg != "" {
		data["Error"] = msg
	}
	views.ExecuteTemplate(loginTemplate, w, data, nil)
}

// StartGoogleAuth redirects to Google's OAuth consent screen.
func (h *LoginHandler) StartGoogleAuth(w http.ResponseWriter, r *http.Request) {
	url, err := h.google.AuthCodeURL(w)
	if err != nil {
		log.Printf("oauth start: %v", err)
		http.Redirect(w, r, "/login?error=Could+not+start+sign-in", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// GoogleCallback handles Google's OAuth redirect: validates the ID token, looks up the user in
// cms_user_permission, and issues a session cookie. Rejects unknown or deactivated users.
func (h *LoginHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	defer auth.ClearStateCookie(w)

	claims, err := h.google.Exchange(r.Context(), r)
	if err != nil {
		log.Printf("oauth callback: %v", err)
		http.Redirect(w, r, "/login?error=Sign-in+failed", http.StatusSeeOther)
		return
	}

	user, err := h.users.GetByEmail(r.Context(), claims.Email)
	if errors.Is(err, db.ErrUserNotFound) {
		http.Redirect(w, r, "/login?error=Your+account+is+not+authorized.+Ask+an+admin+to+add+you.", http.StatusSeeOther)
		return
	}
	if err != nil {
		log.Printf("oauth lookup: %v", err)
		http.Redirect(w, r, "/login?error=Sign-in+failed", http.StatusSeeOther)
		return
	}
	if !user.IsActive {
		http.Redirect(w, r, "/login?error=Your+access+has+been+revoked", http.StatusSeeOther)
		return
	}

	if err := auth.IssueSession(w, user.ID, user.Email, user.Role); err != nil {
		log.Printf("issue session: %v", err)
		http.Redirect(w, r, "/login?error=Sign-in+failed", http.StatusSeeOther)
		return
	}
	_ = h.users.UpdateLastLogin(r.Context(), user.ID)

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

// DevLogin is a non-production bypass: POST /dev-login signs in as the email named in DEV_LOGIN_EMAIL.
// Only mounted by cmd/main.go when APP_ENV != "production".
func (h *LoginHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	email := config.GetEnv("DEV_LOGIN_EMAIL", "")
	if email == "" {
		http.Error(w, "dev login disabled", http.StatusForbidden)
		return
	}
	user, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "dev login user not found in cms_user_permission", http.StatusForbidden)
		return
	}
	if !user.IsActive {
		http.Error(w, "dev login user is deactivated", http.StatusForbidden)
		return
	}
	if err := auth.IssueSession(w, user.ID, user.Email, user.Role); err != nil {
		http.Error(w, "could not issue session", http.StatusInternalServerError)
		return
	}
	_ = h.users.UpdateLastLogin(r.Context(), user.ID)

	w.Header().Set("HX-Redirect", "/home")
	w.WriteHeader(http.StatusOK)
}

// Logout clears the session and redirects to /login.
func (h *LoginHandler) Logout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSession(w)
	w.Header().Set("HX-Redirect", "/login")
	w.WriteHeader(http.StatusOK)
}
