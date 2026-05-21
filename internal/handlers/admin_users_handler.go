package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/auth"
	"github.com/avantifellows/nex-gen-cms/internal/repositories/db"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const (
	adminUsersTemplate   = "admin_users.html"
	adminUserRowTemplate = "admin_user_row.html"
)

type AdminUsersHandler struct {
	users *db.CmsUserRepo
}

func NewAdminUsersHandler(users *db.CmsUserRepo) *AdminUsersHandler {
	return &AdminUsersHandler{users: users}
}

// List renders the admin users page (full page via base template).
func (h *AdminUsersHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		log.Printf("admin users list: %v", err)
		http.Error(w, "Could not load users", http.StatusInternalServerError)
		return
	}
	data := map[string]interface{}{
		"Users": users,
		"Roles": []string{auth.RoleViewer, auth.RoleEditor, auth.RoleAdmin},
	}
	views.ExecuteTemplates(w, data, nil, baseTemplate, adminUsersTemplate)
}

// Create adds a new user. HTMX POST returns the new row.
func (h *AdminUsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.FormValue("email"))
	role := r.FormValue("role")
	fullName := strings.TrimSpace(r.FormValue("full_name"))

	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if !auth.ValidRole(role) {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}
	if !strings.HasSuffix(strings.ToLower(email), "@avantifellows.org") {
		http.Error(w, "Only @avantifellows.org emails can be added", http.StatusBadRequest)
		return
	}

	var fullNamePtr *string
	if fullName != "" {
		fullNamePtr = &fullName
	}

	id, err := h.users.Create(r.Context(), email, role, fullNamePtr)
	if err != nil {
		log.Printf("admin users create: %v", err)
		http.Error(w, "Could not create user (email may already exist)", http.StatusBadRequest)
		return
	}

	created, err := h.users.GetByEmail(r.Context(), email)
	if err != nil {
		log.Printf("admin users lookup after create id=%d: %v", id, err)
		http.Error(w, "Created but could not reload", http.StatusInternalServerError)
		return
	}
	views.ExecuteTemplate(adminUserRowTemplate, w, created, nil)
}

// SetActive toggles is_active. HTMX returns the updated row.
func (h *AdminUsersHandler) SetActive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	active := r.URL.Query().Get("active") == "true"

	// Don't let an admin deactivate themselves — they'd lock themselves out.
	claims := auth.FromContext(r.Context())
	if claims != nil && claims.UserID == id && !active {
		http.Error(w, "You cannot deactivate your own account", http.StatusBadRequest)
		return
	}

	if err := h.users.SetActive(r.Context(), id, active); err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("admin users set active id=%d: %v", id, err)
		http.Error(w, "Could not update user", http.StatusInternalServerError)
		return
	}
	h.renderRowByID(w, r, id)
}

// UpdateRole changes a user's role. HTMX returns the updated row.
func (h *AdminUsersHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	role := r.FormValue("role")
	if !auth.ValidRole(role) {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Prevent an admin from demoting themselves to a non-admin role (lockout protection).
	claims := auth.FromContext(r.Context())
	if claims != nil && claims.UserID == id && role != auth.RoleAdmin {
		http.Error(w, "You cannot demote your own admin role", http.StatusBadRequest)
		return
	}

	if err := h.users.UpdateRole(r.Context(), id, role); err != nil {
		log.Printf("admin users update role id=%d: %v", id, err)
		http.Error(w, "Could not update role", http.StatusInternalServerError)
		return
	}
	h.renderRowByID(w, r, id)
}

func (h *AdminUsersHandler) renderRowByID(w http.ResponseWriter, r *http.Request, id int64) {
	users, err := h.users.List(r.Context())
	if err != nil {
		log.Printf("admin users reload: %v", err)
		http.Error(w, "Could not reload user", http.StatusInternalServerError)
		return
	}
	for _, u := range users {
		if u.ID == id {
			views.ExecuteTemplate(adminUserRowTemplate, w, u, nil)
			return
		}
	}
	http.NotFound(w, r)
}
