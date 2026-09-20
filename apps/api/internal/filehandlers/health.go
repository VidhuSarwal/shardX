package filehandlers

import (
	"SE/internal/integrity"
	"SE/internal/models"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetFileHealthHandler - GET /api/files/{session_id}/health
//
//	GET /api/files/{session_id}/shards
//	GET /api/files/{session_id}/timeline
//
// Registered against the "/api/files/" prefix in cmd/server/main.go (there
// is no static suffix to slice on, since the session ID sits in the middle
// of the path). This handler is responsible for rejecting any other
// "/api/files/..." path that doesn't match "<session_id>/<action>", so it
// doesn't shadow the other, more specific routes registered under
// "/api/files/upload/...", "/api/files/chunking/...", and
// "/api/files/download-key/..." (Go's ServeMux picks the longest matching
// registered pattern, so those still win over this catch-all).
func GetFileHealthHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("userID").(primitive.ObjectID)

	const prefix = "/api/files/"
	rest := strings.TrimPrefix(r.URL.Path, prefix)
	sessionIDStr, action, ok := strings.Cut(rest, "/")
	if !strings.HasPrefix(r.URL.Path, prefix) || !ok || sessionIDStr == "" || strings.Contains(action, "/") {
		http.NotFound(w, r)
		return
	}
	if action != "health" && action != "shards" && action != "timeline" {
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

	var body any
	switch action {
	case "health":
		body, err = integrity.VerifyShardHealth(r.Context(), sessionID.Hex(), metaStore, nil)
	case "shards":
		var shards []models.ShardRecord
		shards, err = metaStore.GetShardMetadata(r.Context(), sessionID.Hex())
		sort.Slice(shards, func(i, j int) bool { return shards[i].ShardID < shards[j].ShardID })
		body = map[string]any{"file_id": sessionID.Hex(), "shards": shards}
	case "timeline":
		var shards []models.ShardRecord
		shards, err = metaStore.GetShardMetadata(r.Context(), sessionID.Hex())
		body = map[string]any{"file_id": sessionID.Hex(), "events": buildTimeline(session, shards)}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

// TimelineEvent is one row of the audit timeline.
type TimelineEvent struct {
	Type   string         `json:"type"`
	At     *time.Time     `json:"at"`
	Detail map[string]any `json:"detail,omitempty"`
}

// buildTimeline derives the audit timeline from what is already persisted
// (session lifecycle timestamps + per-shard records) rather than from a
// separate event log. Same vocabulary as internal/events so a future
// EventBridge-backed log can replace this without changing the client.
func buildTimeline(session *models.UploadSession, shards []models.ShardRecord) []TimelineEvent {
	created := session.CreatedAt
	events := []TimelineEvent{{
		Type: "FILE_CREATED",
		At:   &created,
		Detail: map[string]any{
			"filename":   session.OriginalFilename,
			"total_size": session.TotalSize,
		},
	}}
	for _, s := range shards {
		at := s.CreatedAt
		typ := "SHARD_VERIFIED"
		if s.Status == integrity.StatusPending {
			typ = "SHARD_QUEUED_FOR_RETRY"
		} else if s.Status != integrity.StatusVerified {
			typ = "SHARD_CORRUPTED"
		}
		events = append(events, TimelineEvent{Type: typ, At: &at, Detail: map[string]any{
			"shard_id": s.ShardID, "size": s.Size, "bucket": s.Bucket, "region": s.Region, "sha256": s.SHA256,
		}})
	}
	switch session.Status {
	case "complete":
		events = append(events, TimelineEvent{Type: "FILE_HEALTHY", At: session.CompletedAt, Detail: map[string]any{"shards_total": len(shards)}})
	case "failed":
		events = append(events, TimelineEvent{Type: "UPLOAD_FAILED", At: nil, Detail: map[string]any{"error": session.ErrorMessage}})
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].At == nil || events[j].At == nil {
			return events[j].At == nil && events[i].At != nil
		}
		return events[i].At.Before(*events[j].At)
	})
	return events
}
