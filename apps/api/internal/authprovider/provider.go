// Package authprovider defines a provider-agnostic interface for
// authentication, so that a future alternative backend (e.g. AWS Cognito)
// can be introduced without touching callers.
//
// Group 0 constraint: this is a pure refactor. AuthProvider mirrors the
// real, current exported surface of internal/auth: SignupHandler,
// LoginHandler (both http.HandlerFunc-shaped, as they are today) and
// AuthMiddleware. ValidateToken is included because it was added to
// internal/auth as a thin exported wrapper around the pre-existing
// unexported parseJWT (no new logic, no behavior change) specifically to
// give this interface something to call for the "validate a token" use
// case, since AuthMiddleware itself is a middleware wrapper rather than a
// standalone validation function.
package authprovider

import "net/http"

// AuthProvider mirrors internal/auth's current exported surface.
type AuthProvider interface {
	// Signup handles a signup HTTP request (same shape as
	// auth.SignupHandler).
	Signup(w http.ResponseWriter, r *http.Request)

	// Login handles a login HTTP request (same shape as auth.LoginHandler).
	Login(w http.ResponseWriter, r *http.Request)

	// ValidateToken parses and validates a bearer token string, returning
	// the userID (sub claim) it was issued for.
	ValidateToken(token string) (string, error)

	// AuthMiddleware wraps an http.HandlerFunc requiring a valid bearer
	// token, same as auth.AuthMiddleware.
	AuthMiddleware(next http.HandlerFunc) http.HandlerFunc
}
