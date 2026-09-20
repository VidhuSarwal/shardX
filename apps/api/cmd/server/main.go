package main

import (
	"SE/internal/bootstrap"
	"SE/internal/filehandlers"
	"SE/internal/fileprocessor"
	"SE/internal/handlers"
	"SE/internal/middleware"
	"SE/internal/oauth"
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
	metaStore := bootstrap.SelectMetadataStore(os.Getenv("DB_PROVIDER"))
	storageProvider := bootstrap.SelectStorageProvider(os.Getenv("STORAGE_PROVIDER"))
	authProv := bootstrap.SelectAuthProvider(os.Getenv("AUTH_PROVIDER"))
	filehandlers.InitEvents(bootstrap.SelectEventEmitter(os.Getenv("EVENT_BUS_NAME")))
	filehandlers.InitOrchestrator(bootstrap.SelectOrchestrator(os.Getenv("STATE_MACHINE_ARN")))
	filehandlers.InitMetadataStore(metaStore)
	filehandlers.InitStorageProvider(storageProvider)
	filehandlers.InitRetryQueue(bootstrap.SelectRetryQueue(os.Getenv("SQS_QUEUE_URL")))

	driveAccountsHandler := handlers.NewDriveAccountsHandler(metaStore)

	// Setup routes
	mux := http.NewServeMux()

	// Health check route
	mux.HandleFunc("/health", requireMethod("GET", healthCheckHandler))

	// Authentication routes
	mux.HandleFunc("/api/signup", requireMethod("POST", authProv.Signup))
	mux.HandleFunc("/api/login", requireMethod("POST", authProv.Login))

	// Drive OAuth routes
	mux.HandleFunc("/api/drive/link", authProv.AuthMiddleware(requireMethod("GET", oauth.DriveLinkHandler)))
	mux.HandleFunc("/api/drive/accounts", authProv.AuthMiddleware(requireMethod("GET", driveAccountsHandler.ListDriveAccounts)))
	mux.HandleFunc("/api/drive/space", authProv.AuthMiddleware(requireMethod("GET", filehandlers.GetDriveSpacesHandler)))

	// File upload routes
	mux.HandleFunc("/api/files/upload/initiate", authProv.AuthMiddleware(requireMethod("POST", filehandlers.InitiateUploadHandler)))
	mux.HandleFunc("/api/files/upload/chunk", authProv.AuthMiddleware(requireMethod("POST", filehandlers.UploadChunkHandler)))
	mux.HandleFunc("/api/files/upload/finalize", authProv.AuthMiddleware(requireMethod("POST", filehandlers.FinalizeUploadHandler)))
	mux.HandleFunc("/api/files/upload/status/", authProv.AuthMiddleware(requireMethod("GET", filehandlers.GetUploadStatusHandler)))
	mux.HandleFunc("/api/files/chunking/calculate", authProv.AuthMiddleware(requireMethod("POST", filehandlers.CalculateChunkingHandler)))
	mux.HandleFunc("/api/files/download-key/", authProv.AuthMiddleware(requireMethod("GET", filehandlers.DownloadKeyFileHandler)))

	// Integrity Engine: shard health score, shard placement, audit
	// timeline ({session_id}/health, /shards, /timeline). Registered against the
	// "/api/files/" prefix since the session ID sits in the middle of the
	// path ("/api/files/{session_id}/health"); ServeMux's longest-prefix
	// match means the more specific routes above still take precedence.
	mux.HandleFunc("/api/files/", authProv.AuthMiddleware(requireMethod("GET", filehandlers.GetFileHealthHandler)))

	// OAuth callback (no auth header; state validated via DB)
	mux.HandleFunc("/oauth2/callback", requireMethod("GET", oauth.OauthCallbackHandler))

	// OAuth completion page
	mux.HandleFunc("/oauth/finished", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<h1>OAuth flow completed</h1><p>You can close this window and return to the application.</p>"))
	})

	addr := ":5555"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
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
		"status":    "healthy",
		"message":   "Server is running",
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
