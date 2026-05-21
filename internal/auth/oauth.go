package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	googleIssuer       = "https://accounts.google.com"
	allowedHostedDomain = "avantifellows.org"
	oauthStateCookie    = "cms_oauth_state"
)

// GoogleAuth bundles the OIDC provider + OAuth2 config for Google sign-in.
type GoogleAuth struct {
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	config   *oauth2.Config
}

// NewGoogleAuth constructs a GoogleAuth from env vars. Returns nil + error if anything is missing.
func NewGoogleAuth(ctx context.Context) (*GoogleAuth, error) {
	clientID := config.GetEnv("GOOGLE_CLIENT_ID", "")
	clientSecret := config.GetEnv("GOOGLE_CLIENT_SECRET", "")
	redirectURL := config.GetEnv("OAUTH_REDIRECT_URL", "")
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, errors.New("GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, OAUTH_REDIRECT_URL must be set")
	}

	provider, err := oidc.NewProvider(ctx, googleIssuer)
	if err != nil {
		return nil, fmt.Errorf("create OIDC provider: %w", err)
	}

	return &GoogleAuth{
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
	}, nil
}

// AuthCodeURL builds the redirect URL to Google's consent screen and stamps a state cookie.
func (g *GoogleAuth) AuthCodeURL(w http.ResponseWriter) (string, error) {
	state, err := randomState()
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
	})

	// "hd" pins Google's account chooser to our workspace. We still verify the hd claim server-side below.
	return g.config.AuthCodeURL(state, oauth2.SetAuthURLParam("hd", allowedHostedDomain)), nil
}

// IDTokenClaims is the subset of Google's ID token we use.
type IDTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HostedDomain  string `json:"hd"`
	Name          string `json:"name"`
}

// Exchange validates state, exchanges the code for tokens, verifies the ID token, and returns the claims.
func (g *GoogleAuth) Exchange(ctx context.Context, r *http.Request) (*IDTokenClaims, error) {
	wantState, err := r.Cookie(oauthStateCookie)
	if err != nil {
		return nil, errors.New("missing oauth state cookie")
	}
	if r.URL.Query().Get("state") != wantState.Value {
		return nil, errors.New("oauth state mismatch")
	}

	tok, err := g.config.Exchange(ctx, r.URL.Query().Get("code"))
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	raw, ok := tok.Extra("id_token").(string)
	if !ok || raw == "" {
		return nil, errors.New("no id_token in response")
	}
	idTok, err := g.verifier.Verify(ctx, raw)
	if err != nil {
		return nil, fmt.Errorf("verify id_token: %w", err)
	}

	var claims IDTokenClaims
	if err := idTok.Claims(&claims); err != nil {
		return nil, fmt.Errorf("decode id_token claims: %w", err)
	}
	if !claims.EmailVerified {
		return nil, errors.New("google email not verified")
	}
	if claims.HostedDomain != allowedHostedDomain {
		return nil, fmt.Errorf("email not in %s domain", allowedHostedDomain)
	}
	return &claims, nil
}

// ClearStateCookie removes the OAuth state cookie after a callback completes.
func ClearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
