// Package authprovider: CognitoProvider is an AuthProvider implementation
// backed by Amazon Cognito User Pools.
//
// Phase-1 simplifications (deliberate, documented here rather than hidden):
//
//   - Signup calls Cognito's SignUp API and then immediately calls
//     AdminConfirmSignUp so that a newly-signed-up user can log in right
//     away, mirroring the existing CustomProvider UX where there is no
//     email-confirmation step. In a real production rollout, email/SMS
//     verification should be required before AdminConfirmSignUp is skipped
//     like this. There is no confirmation-code UX in the frontend yet, so
//     matching the existing synchronous signup->login flow was prioritized.
//
//   - Login uses the USER_PASSWORD_AUTH auth flow via InitiateAuth. This
//     requires the Cognito User Pool App Client to have USER_PASSWORD_AUTH
//     enabled as an explicit auth flow (Terraform hand-off item, see
//     README/task notes -- not yet wired in any Terraform module in this
//     repo).
//
//   - Login also requires the App Client to be created WITHOUT a client
//     secret. If the Terraform `identity` module creates the client with a
//     secret, SignUp/InitiateAuth calls must additionally compute and pass
//     a SECRET_HASH (HMAC-SHA256 of username+clientID keyed by the client
//     secret) -- not implemented here since there is no client secret to
//     test against yet. Another Terraform hand-off item.
//
//   - ValidateToken/AuthMiddleware validate the Cognito ID token (not the
//     access token): only ID tokens carry an `aud` claim equal to the App
//     Client ID, which requirement (5) in the task asks us to check.
//     Access tokens carry `client_id` instead of `aud` and would never
//     satisfy an audience check. Login therefore returns the ID token as
//     the provider's "token" string, matching the shape CustomProvider
//     returns (a single opaque bearer token string).
//
//   - AuthMiddleware here stores the validated Cognito `sub` (a UUID
//     string) in the request context under the same "userID" key that
//     auth.AuthMiddleware uses. IMPORTANT: every current consumer of that
//     context key (internal/oauth, internal/filehandlers,
//     internal/handlers) does a hard type assertion to
//     primitive.ObjectID, which a Cognito sub UUID string is NOT and can
//     never safely be coerced into. Those call sites are unmigrated
//     (main.go currently discards the constructed CognitoProvider/
//     authProv value entirely -- see `_ = authProv`), so this is not yet a
//     live bug, but it IS an explicit unresolved hand-off: mapping a
//     Cognito identity to a Mongo user document (and therefore a
//     primitive.ObjectID) needs a real decision (e.g. look up/create a
//     Mongo user keyed by Cognito sub, and store that Mongo ObjectID in
//     the context instead of the raw sub) before any handler can be
//     migrated to run behind CognitoProvider.AuthMiddleware. This
//     provider stores the raw Cognito sub string under "userID" for now,
//     which is a deliberate, documented, non-panicking placeholder -- NOT
//     a drop-in replacement for auth.AuthMiddleware's context contract.
package authprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
)

// minPasswordLength mirrors the client-side rule already enforced by
// internal/auth.SignupHandler (see internal/auth/auth.go), so error
// messages from CognitoProvider.Signup stay consistent with CustomProvider
// even though Cognito itself also enforces its own (typically stricter)
// password policy server-side.
const minPasswordLength = 6

// cognitoClient is the minimal subset of
// *cognitoidentityprovider.Client that CognitoProvider calls. Declaring it
// as an interface lets tests substitute a fake without hitting real AWS.
type cognitoClient interface {
	SignUp(ctx context.Context, params *cognitoidentityprovider.SignUpInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.SignUpOutput, error)
	AdminConfirmSignUp(ctx context.Context, params *cognitoidentityprovider.AdminConfirmSignUpInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminConfirmSignUpOutput, error)
	InitiateAuth(ctx context.Context, params *cognitoidentityprovider.InitiateAuthInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.InitiateAuthOutput, error)
}

// CognitoProvider implements AuthProvider backed by an Amazon Cognito User
// Pool.
type CognitoProvider struct {
	client     cognitoClient
	userPoolID string
	clientID   string
	region     string

	jwksURL string

	mu        sync.Mutex
	keyfunc   keyfunc.Keyfunc
	keyfuncAt time.Time
}

// compile-time interface compliance assertion
var _ AuthProvider = (*CognitoProvider)(nil)

// NewCognitoProvider constructs a CognitoProvider backed by a real AWS
// Cognito client, loading AWS credentials/region via the standard AWS SDK
// v2 default credential chain (env vars, shared config/credentials file,
// AWS_PROFILE, EC2/ECS/EKS instance roles, etc.) -- no credentials are
// hardcoded here.
func NewCognitoProvider(ctx context.Context, userPoolID, clientID, region string) (*CognitoProvider, error) {
	if userPoolID == "" || clientID == "" || region == "" {
		return nil, errors.New("userPoolID, clientID, and region are all required")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := cognitoidentityprovider.NewFromConfig(cfg)

	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID)

	return &CognitoProvider{
		client:     client,
		userPoolID: userPoolID,
		clientID:   clientID,
		region:     region,
		jwksURL:    jwksURL,
	}, nil
}

// newCognitoProviderForTest builds a CognitoProvider around an injected
// fake cognitoClient, bypassing NewCognitoProvider's AWS config loading
// entirely, so unit tests never need real AWS credentials.
func newCognitoProviderForTest(client cognitoClient, userPoolID, clientID, region string) *CognitoProvider {
	return &CognitoProvider{
		client:     client,
		userPoolID: userPoolID,
		clientID:   clientID,
		region:     region,
		jwksURL:    fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", region, userPoolID),
	}
}

// newCognitoProviderWithKeyfunc builds a CognitoProvider with a
// pre-supplied keyfunc.Keyfunc (e.g. one built from an in-memory JWKS
// document via keyfunc.NewJWKSetJSON), so ValidateToken's signature/claims
// logic can be exercised fully offline in tests, without ever making an
// HTTP call to fetch a JWKS.
func newCognitoProviderWithKeyfunc(kf keyfunc.Keyfunc, userPoolID, clientID, region string) *CognitoProvider {
	return &CognitoProvider{
		userPoolID: userPoolID,
		clientID:   clientID,
		region:     region,
		keyfunc:    kf,
	}
}

type cognitoLoginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type cognitoLoginResp struct {
	Token string `json:"token"`
}

// Signup handles a signup HTTP request against Cognito: SignUp followed by
// AdminConfirmSignUp (see package doc for why). Request/response JSON
// shapes match auth.SignupHandler exactly so callers don't need to branch
// on which provider is active.
func (p *CognitoProvider) Signup(w http.ResponseWriter, r *http.Request) {
	var req cognitoLoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "email and password required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < minPasswordLength {
		http.Error(w, fmt.Sprintf("password must be at least %d characters", minPasswordLength), http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	ctx := r.Context()

	_, err := p.client.SignUp(ctx, &cognitoidentityprovider.SignUpInput{
		ClientId: aws.String(p.clientID),
		Username: aws.String(email),
		Password: aws.String(req.Password),
		UserAttributes: []cognitotypes.AttributeType{
			{Name: aws.String("email"), Value: aws.String(email)},
		},
	})
	if err != nil {
		if isUsernameExistsErr(err) {
			http.Error(w, "email exists", http.StatusBadRequest)
			return
		}
		if isInvalidPasswordErr(err) {
			http.Error(w, "password does not meet requirements", http.StatusBadRequest)
			return
		}
		http.Error(w, "create user failed", http.StatusInternalServerError)
		return
	}

	// Phase-1 simplification: auto-confirm immediately so login works
	// without a frontend confirmation-code flow. See package doc.
	_, err = p.client.AdminConfirmSignUp(ctx, &cognitoidentityprovider.AdminConfirmSignUpInput{
		UserPoolId: aws.String(p.userPoolID),
		Username:   aws.String(email),
	})
	if err != nil {
		http.Error(w, "confirm user failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "user created"})
}

// Login handles a login HTTP request against Cognito using the
// USER_PASSWORD_AUTH flow, returning the ID token (not the access token --
// see package doc for why) as the opaque bearer token string, in the same
// {"token": "..."} JSON shape auth.LoginHandler uses.
func (p *CognitoProvider) Login(w http.ResponseWriter, r *http.Request) {
	var req cognitoLoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	ctx := r.Context()

	out, err := p.client.InitiateAuth(ctx, &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: cognitotypes.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(p.clientID),
		AuthParameters: map[string]string{
			"USERNAME": email,
			"PASSWORD": req.Password,
		},
	})
	if err != nil {
		if isNotAuthorizedErr(err) || isUserNotFoundErr(err) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	if out.AuthenticationResult == nil || out.AuthenticationResult.IdToken == nil {
		// e.g. an unsupported ChallengeName came back instead (MFA, new
		// password required, etc.) -- Phase 1 doesn't implement challenge
		// flows, matching CustomProvider's lack of MFA support.
		http.Error(w, "unsupported authentication challenge", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cognitoLoginResp{Token: *out.AuthenticationResult.IdToken})
}

// ValidateToken verifies a Cognito-issued ID token's signature (via the
// User Pool's JWKS), issuer, audience (App Client ID), token_use, and
// expiry, returning the `sub` claim (the Cognito user's stable UUID) on
// success.
func (p *CognitoProvider) ValidateToken(token string) (string, error) {
	if token == "" {
		return "", errors.New("empty token")
	}

	kf, err := p.getKeyfunc()
	if err != nil {
		return "", fmt.Errorf("load jwks: %w", err)
	}

	expectedIssuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", p.region, p.userPoolID)

	claims := jwt.MapClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, kf.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(expectedIssuer),
		jwt.WithAudience(p.clientID),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !parsed.Valid {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	tokenUse, _ := claims["token_use"].(string)
	if tokenUse != "id" {
		return "", errors.New("invalid token: not an ID token")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("invalid token: missing sub claim")
	}

	return sub, nil
}

// AuthMiddleware wraps an http.HandlerFunc, requiring a valid Cognito ID
// token bearer header, mirroring auth.AuthMiddleware's header extraction
// and 401 response shape. See package doc for the important caveat about
// what gets stored under the "userID" context key.
func (p *CognitoProvider) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if h == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var tok string
		_, err := fmt.Sscanf(h, "Bearer %s", &tok)
		if err != nil || tok == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		sub, err := p.ValidateToken(tok)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// NOTE: stores the raw Cognito sub (UUID string), NOT a
		// primitive.ObjectID -- see package doc. Existing handlers that
		// type-assert this context value to primitive.ObjectID are not
		// compatible with this provider yet.
		ctx := context.WithValue(r.Context(), "userID", sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// getKeyfunc lazily fetches and caches the Cognito JWKS as a
// keyfunc.Keyfunc, refreshing it periodically. keyfunc.NewDefaultCtx
// itself performs background auto-refresh, so we only need to build it
// once per provider instance.
func (p *CognitoProvider) getKeyfunc() (keyfunc.Keyfunc, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.keyfunc != nil {
		return p.keyfunc, nil
	}

	kf, err := keyfunc.NewDefaultCtx(context.Background(), []string{p.jwksURL})
	if err != nil {
		return nil, err
	}

	p.keyfunc = kf
	p.keyfuncAt = time.Now()
	return kf, nil
}

func isUsernameExistsErr(err error) bool {
	var e *cognitotypes.UsernameExistsException
	return errors.As(err, &e)
}

func isInvalidPasswordErr(err error) bool {
	var e *cognitotypes.InvalidPasswordException
	return errors.As(err, &e)
}

func isNotAuthorizedErr(err error) bool {
	var e *cognitotypes.NotAuthorizedException
	return errors.As(err, &e)
}

func isUserNotFoundErr(err error) bool {
	var e *cognitotypes.UserNotFoundException
	return errors.As(err, &e)
}
