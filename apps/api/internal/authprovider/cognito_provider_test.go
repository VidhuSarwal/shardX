package authprovider

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
)

// ---------------------------------------------------------------------
// fake Cognito client for Signup/Login unit tests (no real AWS access)
// ---------------------------------------------------------------------

type fakeCognitoClient struct {
	signUpFunc             func(ctx context.Context, in *cognitoidentityprovider.SignUpInput) (*cognitoidentityprovider.SignUpOutput, error)
	adminConfirmSignUpFunc func(ctx context.Context, in *cognitoidentityprovider.AdminConfirmSignUpInput) (*cognitoidentityprovider.AdminConfirmSignUpOutput, error)
	initiateAuthFunc       func(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput) (*cognitoidentityprovider.InitiateAuthOutput, error)

	lastSignUpInput             *cognitoidentityprovider.SignUpInput
	lastAdminConfirmSignUpInput *cognitoidentityprovider.AdminConfirmSignUpInput
	lastInitiateAuthInput       *cognitoidentityprovider.InitiateAuthInput
}

func (f *fakeCognitoClient) SignUp(ctx context.Context, in *cognitoidentityprovider.SignUpInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.SignUpOutput, error) {
	f.lastSignUpInput = in
	if f.signUpFunc != nil {
		return f.signUpFunc(ctx, in)
	}
	return &cognitoidentityprovider.SignUpOutput{}, nil
}

func (f *fakeCognitoClient) AdminConfirmSignUp(ctx context.Context, in *cognitoidentityprovider.AdminConfirmSignUpInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminConfirmSignUpOutput, error) {
	f.lastAdminConfirmSignUpInput = in
	if f.adminConfirmSignUpFunc != nil {
		return f.adminConfirmSignUpFunc(ctx, in)
	}
	return &cognitoidentityprovider.AdminConfirmSignUpOutput{}, nil
}

func (f *fakeCognitoClient) InitiateAuth(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput, _ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.InitiateAuthOutput, error) {
	f.lastInitiateAuthInput = in
	if f.initiateAuthFunc != nil {
		return f.initiateAuthFunc(ctx, in)
	}
	return &cognitoidentityprovider.InitiateAuthOutput{}, nil
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	return bytes.NewReader(b)
}

// ---------------------------------------------------------------------
// interface compliance
// ---------------------------------------------------------------------

func TestCognitoProvider_InterfaceCompliance(t *testing.T) {
	var _ AuthProvider = (*CognitoProvider)(nil)
}

// ---------------------------------------------------------------------
// Signup
// ---------------------------------------------------------------------

func TestCognitoProvider_Signup_BadRequest(t *testing.T) {
	fake := &fakeCognitoClient{}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", nil) // no body -> decode error
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCognitoProvider_Signup_MissingFields(t *testing.T) {
	fake := &fakeCognitoClient{}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, cognitoLoginReq{Email: "", Password: ""}))
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCognitoProvider_Signup_PasswordTooShort(t *testing.T) {
	fake := &fakeCognitoClient{}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, cognitoLoginReq{Email: "a@b.com", Password: "abc"}))
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if fake.lastSignUpInput != nil {
		t.Fatal("expected SignUp not to be called for a too-short password")
	}
}

func TestCognitoProvider_Signup_Success_CallsSignUpThenAdminConfirm(t *testing.T) {
	fake := &fakeCognitoClient{}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, cognitoLoginReq{Email: "User@Example.com ", Password: "correcthorse"}))
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if fake.lastSignUpInput == nil {
		t.Fatal("expected SignUp to be called")
	}
	if got := *fake.lastSignUpInput.Username; got != "user@example.com" {
		t.Fatalf("expected lowercased/trimmed email as username, got %q", got)
	}
	if *fake.lastSignUpInput.ClientId != "client-id" {
		t.Fatalf("expected ClientId to be passed through, got %q", *fake.lastSignUpInput.ClientId)
	}

	if fake.lastAdminConfirmSignUpInput == nil {
		t.Fatal("expected AdminConfirmSignUp to be called immediately after SignUp (Phase-1 simplification)")
	}
	if *fake.lastAdminConfirmSignUpInput.UserPoolId != "pool-id" {
		t.Fatalf("expected UserPoolId to be passed through, got %q", *fake.lastAdminConfirmSignUpInput.UserPoolId)
	}
	if *fake.lastAdminConfirmSignUpInput.Username != "user@example.com" {
		t.Fatalf("expected same username passed to AdminConfirmSignUp, got %q", *fake.lastAdminConfirmSignUpInput.Username)
	}
}

func TestCognitoProvider_Signup_UsernameExists(t *testing.T) {
	fake := &fakeCognitoClient{
		signUpFunc: func(ctx context.Context, in *cognitoidentityprovider.SignUpInput) (*cognitoidentityprovider.SignUpOutput, error) {
			return nil, &cognitotypes.UsernameExistsException{Message: strPtr("already exists")}
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, cognitoLoginReq{Email: "dup@example.com", Password: "correcthorse"}))
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if fake.lastAdminConfirmSignUpInput != nil {
		t.Fatal("expected AdminConfirmSignUp NOT to be called when SignUp fails")
	}
}

func TestCognitoProvider_Signup_InvalidPassword(t *testing.T) {
	fake := &fakeCognitoClient{
		signUpFunc: func(ctx context.Context, in *cognitoidentityprovider.SignUpInput) (*cognitoidentityprovider.SignUpOutput, error) {
			return nil, &cognitotypes.InvalidPasswordException{Message: strPtr("too weak")}
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, cognitoLoginReq{Email: "weak@example.com", Password: "abcdef"}))
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCognitoProvider_Signup_ConfirmFails(t *testing.T) {
	fake := &fakeCognitoClient{
		adminConfirmSignUpFunc: func(ctx context.Context, in *cognitoidentityprovider.AdminConfirmSignUpInput) (*cognitoidentityprovider.AdminConfirmSignUpOutput, error) {
			return nil, errFake("boom")
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/signup", jsonBody(t, cognitoLoginReq{Email: "a@example.com", Password: "correcthorse"}))
	rec := httptest.NewRecorder()
	p.Signup(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------

func TestCognitoProvider_Login_BadRequest(t *testing.T) {
	fake := &fakeCognitoClient{}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/login", nil)
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCognitoProvider_Login_Success(t *testing.T) {
	fake := &fakeCognitoClient{
		initiateAuthFunc: func(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput) (*cognitoidentityprovider.InitiateAuthOutput, error) {
			if in.AuthFlow != cognitotypes.AuthFlowTypeUserPasswordAuth {
				t.Fatalf("expected USER_PASSWORD_AUTH flow, got %v", in.AuthFlow)
			}
			return &cognitoidentityprovider.InitiateAuthOutput{
				AuthenticationResult: &cognitotypes.AuthenticationResultType{
					IdToken: strPtr("fake-id-token"),
				},
			}, nil
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/login", jsonBody(t, cognitoLoginReq{Email: "a@example.com", Password: "correcthorse"}))
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp cognitoLoginResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token != "fake-id-token" {
		t.Fatalf("expected the ID token to be returned as the bearer token, got %q", resp.Token)
	}
}

func TestCognitoProvider_Login_InvalidCredentials(t *testing.T) {
	fake := &fakeCognitoClient{
		initiateAuthFunc: func(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput) (*cognitoidentityprovider.InitiateAuthOutput, error) {
			return nil, &cognitotypes.NotAuthorizedException{Message: strPtr("bad creds")}
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/login", jsonBody(t, cognitoLoginReq{Email: "a@example.com", Password: "wrong"}))
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCognitoProvider_Login_UserNotFound(t *testing.T) {
	fake := &fakeCognitoClient{
		initiateAuthFunc: func(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput) (*cognitoidentityprovider.InitiateAuthOutput, error) {
			return nil, &cognitotypes.UserNotFoundException{Message: strPtr("no such user")}
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/login", jsonBody(t, cognitoLoginReq{Email: "nope@example.com", Password: "whatever"}))
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCognitoProvider_Login_UnexpectedChallenge(t *testing.T) {
	fake := &fakeCognitoClient{
		initiateAuthFunc: func(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput) (*cognitoidentityprovider.InitiateAuthOutput, error) {
			return &cognitoidentityprovider.InitiateAuthOutput{
				ChallengeName: cognitotypes.ChallengeNameTypeNewPasswordRequired,
			}, nil
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/login", jsonBody(t, cognitoLoginReq{Email: "a@example.com", Password: "correcthorse"}))
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an unhandled challenge, got %d", rec.Code)
	}
}

func TestCognitoProvider_Login_ServerError(t *testing.T) {
	fake := &fakeCognitoClient{
		initiateAuthFunc: func(ctx context.Context, in *cognitoidentityprovider.InitiateAuthInput) (*cognitoidentityprovider.InitiateAuthOutput, error) {
			return nil, errFake("throttled or some other AWS error")
		},
	}
	p := newCognitoProviderForTest(fake, "pool-id", "client-id", "us-east-1")

	req := httptest.NewRequest(http.MethodPost, "/api/login", jsonBody(t, cognitoLoginReq{Email: "a@example.com", Password: "correcthorse"}))
	rec := httptest.NewRecorder()
	p.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

// ---------------------------------------------------------------------
// ValidateToken / AuthMiddleware: fully offline JWKS + JWT validation.
//
// This is the core security logic (signature, issuer, audience, expiry,
// token_use), so it is tested thoroughly against a self-generated RSA
// keypair and hand-built JWKS document -- no network access, no real
// Cognito User Pool required.
// ---------------------------------------------------------------------

const (
	testUserPoolID = "us-east-1_TESTPOOL"
	testClientID   = "test-client-id"
	testRegion     = "us-east-1"
	testKID        = "test-key-id"
)

func testIssuer() string {
	return "https://cognito-idp." + testRegion + ".amazonaws.com/" + testUserPoolID
}

// buildTestProvider generates an RSA keypair, builds a JWKS document from
// the public key, and returns a CognitoProvider whose ValidateToken will
// verify against that JWKS entirely in memory, plus a signing function for
// building test tokens with the matching private key.
func buildTestProvider(t *testing.T) (*CognitoProvider, func(claims jwt.MapClaims) string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	jwks := buildJWKS(t, testKID, &priv.PublicKey)

	kf, err := keyfunc.NewJWKSetJSON(jwks)
	if err != nil {
		t.Fatalf("build keyfunc from JWKS: %v", err)
	}

	p := newCognitoProviderWithKeyfunc(kf, testUserPoolID, testClientID, testRegion)

	sign := func(claims jwt.MapClaims) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tok.Header["kid"] = testKID
		s, err := tok.SignedString(priv)
		if err != nil {
			t.Fatalf("sign test token: %v", err)
		}
		return s
	}

	return p, sign
}

// buildJWKS constructs a minimal RFC 7517 JWK Set JSON document from an
// RSA public key, mimicking the shape Cognito publishes at
// .well-known/jwks.json.
func buildJWKS(t *testing.T, kid string, pub *rsa.PublicKey) []byte {
	t.Helper()

	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBytes := bigIntToBytes(pub.E)
	e := base64.RawURLEncoding.EncodeToString(eBytes)

	doc := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"alg": "RS256",
				"n":   n,
				"e":   e,
			},
		},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	return b
}

func bigIntToBytes(v int) []byte {
	// RSA public exponent is almost always 65537 (0x010001), 3 bytes.
	b := make([]byte, 0, 4)
	started := false
	for i := 3; i >= 0; i-- {
		byteVal := byte(v >> (8 * i))
		if byteVal != 0 {
			started = true
		}
		if started {
			b = append(b, byteVal)
		}
	}
	if len(b) == 0 {
		b = []byte{0}
	}
	return b
}

func validIDTokenClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":        testIssuer(),
		"aud":        testClientID,
		"sub":        "11111111-2222-3333-4444-555555555555",
		"token_use":  "id",
		"email":      "user@example.com",
		"exp":        now.Add(1 * time.Hour).Unix(),
		"iat":        now.Unix(),
		"auth_time":  now.Unix(),
		"token_type": "id",
	}
}

func TestCognitoProvider_ValidateToken_Valid(t *testing.T) {
	p, sign := buildTestProvider(t)
	tok := sign(validIDTokenClaims())

	sub, err := p.ValidateToken(tok)
	if err != nil {
		t.Fatalf("expected valid token to be accepted, got error: %v", err)
	}
	if sub != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("unexpected sub: %q", sub)
	}
}

func TestCognitoProvider_ValidateToken_Expired(t *testing.T) {
	p, sign := buildTestProvider(t)
	claims := validIDTokenClaims()
	claims["exp"] = time.Now().Add(-1 * time.Hour).Unix()
	claims["iat"] = time.Now().Add(-2 * time.Hour).Unix()
	tok := sign(claims)

	if _, err := p.ValidateToken(tok); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_WrongAudience(t *testing.T) {
	p, sign := buildTestProvider(t)
	claims := validIDTokenClaims()
	claims["aud"] = "some-other-client-id"
	tok := sign(claims)

	if _, err := p.ValidateToken(tok); err == nil {
		t.Fatal("expected wrong-audience token to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_WrongIssuer(t *testing.T) {
	p, sign := buildTestProvider(t)
	claims := validIDTokenClaims()
	claims["iss"] = "https://cognito-idp.us-east-1.amazonaws.com/us-east-1_SOMEOTHERPOOL"
	tok := sign(claims)

	if _, err := p.ValidateToken(tok); err == nil {
		t.Fatal("expected wrong-issuer token to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_WrongTokenUse(t *testing.T) {
	p, sign := buildTestProvider(t)
	claims := validIDTokenClaims()
	claims["token_use"] = "access" // access tokens must not pass ID-token validation
	tok := sign(claims)

	if _, err := p.ValidateToken(tok); err == nil {
		t.Fatal("expected non-ID token_use to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_BadSignature(t *testing.T) {
	p, _ := buildTestProvider(t)

	// Sign with a DIFFERENT RSA key than the one whose public half is in
	// the provider's JWKS, but keep the same kid so it reaches signature
	// verification.
	otherPriv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, validIDTokenClaims())
	tok.Header["kid"] = testKID
	signed, err := tok.SignedString(otherPriv)
	if err != nil {
		t.Fatalf("sign with wrong key: %v", err)
	}

	if _, err := p.ValidateToken(signed); err == nil {
		t.Fatal("expected bad-signature token to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_MissingSub(t *testing.T) {
	p, sign := buildTestProvider(t)
	claims := validIDTokenClaims()
	delete(claims, "sub")
	tok := sign(claims)

	if _, err := p.ValidateToken(tok); err == nil {
		t.Fatal("expected token without sub claim to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_EmptyToken(t *testing.T) {
	p, _ := buildTestProvider(t)
	if _, err := p.ValidateToken(""); err == nil {
		t.Fatal("expected empty token to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_Malformed(t *testing.T) {
	p, _ := buildTestProvider(t)
	if _, err := p.ValidateToken("not-a-real-jwt"); err == nil {
		t.Fatal("expected malformed token to be rejected")
	}
}

func TestCognitoProvider_ValidateToken_WrongAlg(t *testing.T) {
	p, _ := buildTestProvider(t)

	// HS256-signed token, using the client ID as a "secret" -- should be
	// rejected outright since only RS256 is an accepted signing method.
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, validIDTokenClaims())
	signed, err := tok.SignedString([]byte("some-shared-secret"))
	if err != nil {
		t.Fatalf("sign HS256 token: %v", err)
	}

	if _, err := p.ValidateToken(signed); err == nil {
		t.Fatal("expected non-RS256 token to be rejected")
	}
}

// ---------------------------------------------------------------------
// AuthMiddleware
// ---------------------------------------------------------------------

func TestCognitoProvider_AuthMiddleware_NoAuthHeader(t *testing.T) {
	p, _ := buildTestProvider(t)
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

func TestCognitoProvider_AuthMiddleware_MalformedBearer(t *testing.T) {
	p, _ := buildTestProvider(t)
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

func TestCognitoProvider_AuthMiddleware_InvalidToken(t *testing.T) {
	p, _ := buildTestProvider(t)
	called := false
	handler := p.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if called {
		t.Fatal("expected inner handler NOT to be called with an invalid token")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCognitoProvider_AuthMiddleware_ValidToken(t *testing.T) {
	p, sign := buildTestProvider(t)
	tok := sign(validIDTokenClaims())

	var gotUserID any
	handler := p.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = r.Context().Value("userID")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	sub, ok := gotUserID.(string)
	if !ok || sub != "11111111-2222-3333-4444-555555555555" {
		t.Fatalf("expected the Cognito sub string to be set on context under \"userID\", got %#v", gotUserID)
	}
}

// ---------------------------------------------------------------------
// small helpers
// ---------------------------------------------------------------------

func strPtr(s string) *string { return &s }

type errFake string

func (e errFake) Error() string { return string(e) }
