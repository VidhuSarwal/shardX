package filehandlers

import (
	"SE/internal/integrity"
	"encoding/json"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetFileHealthHandler - GET /api/files/{session_id}/health
//
// Registered against the "/api/files/" prefix in cmd/server/main.go (there
// is no static suffix to slice on, since the session ID sits in the middle
// of the path). This handler is responsible for rejecting any other
// "/api/files/..." path that doesn't match "<session_id>/health", so it
// doesn't shadow the other, more specific routes registered under
// "/api/files/upload/...", "/api/files/chunking/...", and
// "/api/files/download-key/..." (Go's ServeMux picks the longest matching
// registered pattern, so those still win over this catch-all).
func GetFileHealthHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)

	const prefix = "/api/files/"
	const suffix = "/health"

	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.NotFound(w, r)
		return
	}

	sessionIDStr := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if sessionIDStr == "" || strings.Contains(sessionIDStr, "/") {
		http.NotFound(w, r)
		return
	}

	sessionID, err := primitive.ObjectIDFromHex(sessionIDStr)
	if err != nil {
		http.Error(w, "invalid session_id", http.StatusBadRequest)
		return
	}

	if metaStore == nil {
		http.Error(w, "metadata store not configured", http.StatusInternalServerError)
		return
	}

	// Verify ownership the same way GetUploadStatusHandler does: look up
	// the session and compare its UserID to the authenticated caller.
	session, err := metaStore.GetUploadSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "failed to get session", http.StatusInternalServerError)
		return
	}
	if session == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if session.UserID != userID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	report, err := integrity.VerifyShardHealth(r.Context(), sessionID.Hex(), metaStore, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
