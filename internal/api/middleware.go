package api

import (
	"net/http"
	"strings"

	"github.com/relentlessworks/contactkit/internal/auth"
)

// Middleware wraps an http.Handler with auth.
type Middleware struct {
	auth *auth.Auth
}

// NewMiddleware creates a new middleware instance.
func NewMiddleware(a *auth.Auth) *Middleware {
	return &Middleware{auth: a}
}

// RequireAuth wraps a handler to require a valid bearer token.
func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeError(w, r, http.StatusUnauthorized,
				"missing auth token",
				"call POST /auth/request with email to get an OTP, then POST /auth/verify to get a bearer token")
			return
		}

		t, ok := m.auth.ValidateToken(token)
		if !ok {
			writeError(w, r, http.StatusUnauthorized,
				"invalid or expired token",
				"call POST /auth/request with email to get a new OTP, then POST /auth/verify to get a bearer token")
			return
		}

		// Add workspace to context via header
		r.Header.Set("X-Workspace", t.Workspace)
		r.Header.Set("X-Email", t.Email)
		next(w, r)
	}
}

// extractToken gets the bearer token from the Authorization header.
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// getWorkspace extracts the workspace handle from the request context.
func getWorkspace(r *http.Request) string {
	return r.Header.Get("X-Workspace")
}

// getEmail extracts the email from the request context.
func getEmail(r *http.Request) string {
	return r.Header.Get("X-Email")
}
