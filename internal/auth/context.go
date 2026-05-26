package auth

import "context"

type ctxKey struct{}

var sessionKey ctxKey

// WithSession returns a context carrying the session claims.
func WithSession(ctx context.Context, c *SessionClaims) context.Context {
	return context.WithValue(ctx, sessionKey, c)
}

// FromContext returns the session claims attached to ctx, or nil.
func FromContext(ctx context.Context) *SessionClaims {
	c, _ := ctx.Value(sessionKey).(*SessionClaims)
	return c
}
