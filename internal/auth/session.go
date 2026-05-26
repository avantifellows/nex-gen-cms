package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/avantifellows/nex-gen-cms/config"
	"github.com/golang-jwt/jwt/v5"
)

const (
	SessionCookieName = "cms_session"
	// RoleCookieName is a non-HttpOnly mirror of the session's role claim. JS uses it to gate UI elements
	// (e.g., the Admin nav link). Real authorization always happens server-side against the signed session.
	RoleCookieName = "cms_role"
	sessionMaxAge  = 12 * time.Hour
)

type SessionClaims struct {
	UserID int64  `json:"uid"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func signingKey() []byte {
	return []byte(config.GetEnv("SESSION_SECRET", ""))
}

// IssueSession signs a JWT and sets it as an HttpOnly cookie.
func IssueSession(w http.ResponseWriter, userID int64, email, role string) error {
	key := signingKey()
	if len(key) == 0 {
		return errors.New("SESSION_SECRET is not set")
	}

	now := time.Now()
	claims := SessionClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   email,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(sessionMaxAge)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    signed,
		Path:     "/",
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RoleCookieName,
		Value:    role,
		Path:     "/",
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: false,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// ClearSession removes the session cookie.
func ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RoleCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		Secure:   isSecureCookie(),
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadSession parses and verifies the session cookie. Returns nil if absent/invalid.
func ReadSession(r *http.Request) *SessionClaims {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}
	key := signingKey()
	if len(key) == 0 {
		return nil
	}

	parsed, err := jwt.ParseWithClaims(cookie.Value, &SessionClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return key, nil
	})
	if err != nil || !parsed.Valid {
		return nil
	}
	claims, ok := parsed.Claims.(*SessionClaims)
	if !ok {
		return nil
	}
	return claims
}

func isSecureCookie() bool {
	// "production" is the explicit gate; local dev runs over HTTP and would silently drop a Secure cookie.
	return config.GetEnv("APP_ENV", "") == "production"
}
