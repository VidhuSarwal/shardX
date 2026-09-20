package authprovider

import (
	"SE/internal/auth"
	"net/http"
)

// CustomProvider implements AuthProvider by thin-wrapping the existing
// internal/auth package (custom JWT + bcrypt), unchanged.
type CustomProvider struct{}

// compile-time interface compliance assertion
var _ AuthProvider = (*CustomProvider)(nil)

// NewCustomProvider constructs a CustomProvider.
func NewCustomProvider() *CustomProvider {
	return &CustomProvider{}
}

func (p *CustomProvider) Signup(w http.ResponseWriter, r *http.Request) {
	auth.SignupHandler(w, r)
}

func (p *CustomProvider) Login(w http.ResponseWriter, r *http.Request) {
	auth.LoginHandler(w, r)
}

func (p *CustomProvider) ValidateToken(token string) (string, error) {
	return auth.ValidateToken(token)
}

func (p *CustomProvider) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return auth.AuthMiddleware(next)
}
