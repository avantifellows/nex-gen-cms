package handlers

import (
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/avantifellows/nex-gen-cms/internal/middleware"
	"github.com/avantifellows/nex-gen-cms/internal/views"
)

const loginTemplate = "login.html"

type LoginHandler struct {
}

func NewLoginHandler() *LoginHandler {
	return &LoginHandler{}
}

func (h *LoginHandler) Login(responseWriter http.ResponseWriter, request *http.Request) {
	// First time visit by entering /login or just / in browser
	if request.Method == http.MethodGet {
		// If the user is already logged in, redirect to /home
		if middleware.IsLoggedIn(request) {
			http.Redirect(responseWriter, request, "/home", http.StatusSeeOther)
			return
		}

		// Prevent login page from being cached, so that back press after logging in doesn't show the login page
		responseWriter.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")

		views.ExecuteTemplate(loginTemplate, responseWriter, nil, nil)
		return
	}

	username := request.FormValue("username")
	password := request.FormValue("password")

	if username == config.GetEnv("CMS_USERNAME", "") && password == config.GetEnv("CMS_PASSWORD", "") {
		middleware.SetSessionCookie(responseWriter)

		responseWriter.Header().Set("HX-Redirect", "/home")
		responseWriter.WriteHeader(http.StatusOK)

	} else {
		// Pass an error to the login template
		data := map[string]interface{}{
			"Error": "Invalid username or password",
		}
		views.ExecuteTemplate(loginTemplate, responseWriter, data, nil)
	}
}

func (h *LoginHandler) Logout(responseWriter http.ResponseWriter, request *http.Request) {
	// Clear the cookie
	middleware.Logout(responseWriter)

	// Respond with HX-Redirect header
	responseWriter.Header().Set("HX-Redirect", "/login")
	responseWriter.WriteHeader(http.StatusOK)
}
