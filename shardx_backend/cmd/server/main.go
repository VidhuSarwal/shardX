package main

import (
	"SE/internal/auth"
	"SE/internal/authprovider"
	"SE/internal/filehandlers"
	"SE/internal/fileprocessor"
	"SE/internal/handlers"
	"SE/internal/metadatastore"
	"SE/internal/middleware"
	"SE/internal/oauth"
	"SE/internal/storage"
	"SE/internal/store"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load env vars
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Check required env vars
	required := []string{"MONGO_URI", "JWT_SECRET", "TOKEN_ENC_KEY", "GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "BASE_URL"}
	for _, k := range required {
		if os.Getenv(k) == "" {
			log.Fatalf("env %s is required", k)
		}
	}

	// Initialize store (Mongo)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := store.InitStore(ctx); err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.DisconnectStore(ctx); err != nil {
			log.Printf("disconnect store: %v", err)
		}
	}()

	// Initialize oauth config
	oauth.InitOAuthConfig()

	// Initialize file processor config
	fileprocessor.InitFileConfig()

	// Select provider implementations based on env vars. Group 0 of the
	// provider-interface migration: only "drive"/"mongo"/"custom" have real
	// implementations today (all thin wrappers around the existing
	// concrete packages, unchanged behavior). Additional providers (e.g.
	// S3, DynamoDB, Cognito) will be added in later phases.
	metaStore := selectMetadataStore(os.Getenv("DB_PROVIDER"))
	storageProvider := selectStorageProvider(os.Getenv("STORAGE_PROVIDER"))
	authProv := selectAuthProvider(os.Getenv("AUTH_PROVIDER"))

	// NOTE: storageProvider and authProv are constructed here to prove the
	// providers are selectable and usable, but most existing handlers
	// (internal/filehandlers, internal/oauth, internal/fileprocessor,
	// mux route registration for signup/login/AuthMiddleware below) still
	// call the concrete internal/drivemanager, internal/store, and
	// internal/auth packages directly. Migrating every call site to the
	// interfaces was judged too risky for this pure-refactor phase; it is
	// deferred to when real alternative implementations (S3/DynamoDB/
	// Cognito) land and there is a concrete reason to route through the
	// interface everywhere. handlers.DriveAccountsHandler has been
	// migrated to demonstrate the pattern end-to-end.
	_ = storageProvider
	_ = authProv

	driveAccountsHandler := handlers.NewDriveAccountsHandler(metaStore)

	// Setup routes
	mux := http.NewServeMux()

	// Health check route
	mux.HandleFunc("/health", requireMethod("GET", healthCheckHandler))

	// Authentication routes
	mux.HandleFunc("/api/signup", requireMethod("POST", auth.SignupHandler))
	mux.HandleFunc("/api/login", requireMethod("POST", auth.LoginHandler))

	// Drive OAuth routes
	mux.HandleFunc("/api/drive/link", auth.AuthMiddleware(requireMethod("GET", oauth.DriveLinkHandler)))
	mux.HandleFunc("/api/drive/accounts", auth.AuthMiddleware(requireMethod("GET", driveAccountsHandler.ListDriveAccounts)))
	mux.HandleFunc("/api/drive/space", auth.AuthMiddleware(requireMethod("GET", filehandlers.GetDriveSpacesHandler)))

	// File upload routes
	mux.HandleFunc("/api/files/upload/initiate", auth.AuthMiddleware(requireMethod("POST", filehandlers.InitiateUploadHandler)))
	mux.HandleFunc("/api/files/upload/chunk", auth.AuthMiddleware(requireMethod("POST", filehandlers.UploadChunkHandler)))
	mux.HandleFunc("/api/files/upload/finalize", auth.AuthMiddleware(requireMethod("POST", filehandlers.FinalizeUploadHandler)))
	mux.HandleFunc("/api/files/upload/status/", auth.AuthMiddleware(requireMethod("GET", filehandlers.GetUploadStatusHandler)))
	mux.HandleFunc("/api/files/chunking/calculate", auth.AuthMiddleware(requireMethod("POST", filehandlers.CalculateChunkingHandler)))
	mux.HandleFunc("/api/files/download-key/", auth.AuthMiddleware(requireMethod("GET", filehandlers.DownloadKeyFileHandler)))

	// OAuth callback (no auth header; state validated via DB)
	mux.HandleFunc("/oauth2/callback", requireMethod("GET", oauth.OauthCallbackHandler))

	// OAuth completion page
	mux.HandleFunc("/oauth/finished", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>OAuth flow completed</h1><p>You can close this window and return to the application.</p>"))
	})

	addr := ":5555"
	fmt.Printf("Starting server on %s\n", addr)
	// Apply middlewares: CORS (allow all for now) then Logger
	handler := middleware.CORS([]string{"*"})(mux)
	if err := http.ListenAndServe(addr, middleware.Logger(handler)); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"status": "healthy",
		"message": "Server is running",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	json.NewEncoder(w).Encode(response)
}

func requireMethod(verb string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != verb {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h(w, r)
	}
}

// selectMetadataStore picks a metadatastore.MetadataStore implementation
// based on the DB_PROVIDER env var. Defaults to "mongo" if unset.
func selectMetadataStore(provider string) metadatastore.MetadataStore {
	if provider == "" {
		provider = "mongo"
	}
	switch provider {
	case "mongo":
		return metadatastore.NewMongoStore()
	default:
		log.Fatalf("DB_PROVIDER %q not yet implemented, coming in a later phase", provider)
		return nil
	}
}

// selectStorageProvider picks a storage.StorageProvider implementation
// based on the STORAGE_PROVIDER env var. Defaults to "drive" if unset.
func selectStorageProvider(provider string) storage.StorageProvider {
	if provider == "" {
		provider = "drive"
	}
	switch provider {
	case "drive":
		return storage.NewDriveProvider()
	default:
		log.Fatalf("STORAGE_PROVIDER %q not yet implemented, coming in a later phase", provider)
		return nil
	}
}

// selectAuthProvider picks an authprovider.AuthProvider implementation
// based on the AUTH_PROVIDER env var. Defaults to "custom" if unset.
func selectAuthProvider(provider string) authprovider.AuthProvider {
	if provider == "" {
		provider = "custom"
	}
	switch provider {
	case "custom":
		return authprovider.NewCustomProvider()
	case "cognito":
		userPoolID := os.Getenv("COGNITO_USER_POOL_ID")
		clientID := os.Getenv("COGNITO_CLIENT_ID")
		region := os.Getenv("AWS_REGION")
		if userPoolID == "" || clientID == "" || region == "" {
			log.Fatalf("AUTH_PROVIDER=cognito requires COGNITO_USER_POOL_ID, COGNITO_CLIENT_ID, and AWS_REGION")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := authprovider.NewCognitoProvider(ctx, userPoolID, clientID, region)
		if err != nil {
			log.Fatalf("init cognito auth provider: %v", err)
		}
		return p
	default:
		log.Fatalf("AUTH_PROVIDER %q not yet implemented, coming in a later phase", provider)
		return nil
	}
}
