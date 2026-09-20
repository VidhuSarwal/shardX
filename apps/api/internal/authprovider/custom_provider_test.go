package authprovider

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCustomProvider_InterfaceCompliance ensures CustomProvider satisfies
// AuthProvider at compile time.
func TestCustomProvider_InterfaceCompliance(t *testing.T) {
	var _ AuthProvider = (*CustomProvider)(nil)
	var p AuthProvider = NewCustomProvider()
	if p == nil {
		t.Fatal("NewCustomProvider() returned nil")
	}
}

// TestCustomProvider_ValidateToken_InvalidToken verifies that an
// obviously-malformed token is rejected, delegating to
// auth.ValidateToken -> the pre-existing unexported parseJWT, unchanged.
func TestCustomProvider_ValidateToken_InvalidToken(t *testing.T) {
	p := NewCustomProvider()
	_, err := p.ValidateToken("not-a-real-jwt")
	if err == nil {
		t.Fatal("expected error validating a malformed token, got nil")
	}
}

// TestCustomProvider_ValidateToken_EmptyToken mirrors the invalid-token
// case for an empty string.
func TestCustomProvider_ValidateToken_EmptyToken(t *testing.T) {
	p := NewCustomProvider()
	_, err := p.ValidateToken("")
	if err == nil {
		t.Fatal("expected error validating an empty token, got nil")
	}
}

// TestCustomProvider_AuthMiddleware_NoAuthHeader verifies the wrapped
// AuthMiddleware rejects requests with no Authorization header, exactly
// as auth.AuthMiddleware does today (401 Unauthorized), without requiring
// any live Mongo connection since this path never reaches the store.
func TestCustomProvider_AuthMiddleware_NoAuthHeader(t *testing.T) {
	p := NewCustomProvider()
	called := false
	handler := p.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if called {
		t.Fatal("expected inner handler NOT to be called without auth header")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// TestCustomProvider_AuthMiddleware_MalformedBearer verifies a malformed
// "Authorization" header (not matching "Bearer <token>") is rejected.
func TestCustomProvider_AuthMiddleware_MalformedBearer(t *testing.T) {
	p := NewCustomProvider()
	called := false
	handler := p.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "garbage")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if called {
		t.Fatal("expected inner handler NOT to be called with malformed auth header")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// TestCustomProvider_Signup_BadRequest verifies Signup delegates to
// auth.SignupHandler for the request-validation path that doesn't touch
// Mongo (invalid JSON body -> 400 Bad Request).
func TestCustomProvider_Signup_BadRequest(t *testing.T) {
	p := NewCustomProvider()
	req := httptest.NewRequest(http.MethodPost, "/api/signup", nil) // no body -> JSON decode error
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// TestCustomProvider_Login_BadRequest mirrors the Signup case for Login.
func TestCustomProvider_Login_BadRequest(t *testing.T) {
	p := NewCustomProvider()
	req := httptest.NewRequest(http.MethodPost, "/api/login", nil) // no body -> JSON decode error
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
