package filehandlers

import (
	"SE/internal/metadatastore"
	"SE/internal/models"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeHealthStore is a minimal MetadataStore stand-in for testing
// GetFileHealthHandler without a live database. It embeds the interface so
// only GetUploadSession/GetShardMetadata need overriding.
type fakeHealthStore struct {
	metadatastore.MetadataStore
	session    *models.UploadSession
	sessionErr error
	shards     []models.ShardRecord
	shardsErr  error
}

func (f *fakeHealthStore) GetUploadSession(ctx context.Context, sessionID primitive.ObjectID) (*models.UploadSession, error) {
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}
	return f.session, nil
}

func (f *fakeHealthStore) GetShardMetadata(ctx context.Context, sessionID string) ([]models.ShardRecord, error) {
	if f.shardsErr != nil {
		return nil, f.shardsErr
	}
	return f.shards, nil
}

func withUserID(r *http.Request, userID primitive.ObjectID) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "userID", userID))
}

func TestGetFileHealthHandler_ReturnsHealthReportForOwner(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()

	userID := primitive.NewObjectID()
	sessionID := primitive.NewObjectID()
	metaStore = &fakeHealthStore{
		session: &models.UploadSession{ID: sessionID, UserID: userID, Status: "complete"},
		shards: []models.ShardRecord{
			{ShardID: 0, SHA256: "aaa", Status: "verified", CreatedAt: time.Now()},
			{ShardID: 1, SHA256: "bbb", Status: "verified", CreatedAt: time.Now()},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/files/"+sessionID.Hex()+"/health", nil)
	req = withUserID(req, userID)
	rw := httptest.NewRecorder()

	GetFileHealthHandler(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rw.Code, rw.Body.String())
	}

	var body struct {
		FileID                string  `json:"file_id"`
		HealthPercentage      float64 `json:"health_percentage"`
		ShardsAvailable       int     `json:"shards_available"`
		ShardsTotal           int     `json:"shards_total"`
		IntegrityChecksPassed int     `json:"integrity_checks_passed"`
		IntegrityChecksTotal  int     `json:"integrity_checks_total"`
		CorruptionEvents      int     `json:"corruption_events"`
	}
	if err := json.Unmarshal(rw.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body.FileID != sessionID.Hex() {
		t.Errorf("expected file_id %s, got %s", sessionID.Hex(), body.FileID)
	}
	if body.ShardsTotal != 2 || body.ShardsAvailable != 2 {
		t.Errorf("expected 2/2 shards, got %d/%d", body.ShardsAvailable, body.ShardsTotal)
	}
	if body.HealthPercentage != 100.0 {
		t.Errorf("expected 100%% health, got %v", body.HealthPercentage)
	}
	if body.CorruptionEvents != 0 {
		t.Errorf("expected 0 corruption events, got %d", body.CorruptionEvents)
	}
}

func TestGetFileHealthHandler_RejectsNonOwner(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()

	ownerID := primitive.NewObjectID()
	requesterID := primitive.NewObjectID()
	sessionID := primitive.NewObjectID()
	metaStore = &fakeHealthStore{
		session: &models.UploadSession{ID: sessionID, UserID: ownerID, Status: "complete"},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/files/"+sessionID.Hex()+"/health", nil)
	req = withUserID(req, requesterID)
	rw := httptest.NewRecorder()

	GetFileHealthHandler(rw, req)

	if rw.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rw.Code, rw.Body.String())
	}
}

func TestGetFileHealthHandler_SessionNotFound(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()

	userID := primitive.NewObjectID()
	sessionID := primitive.NewObjectID()
	metaStore = &fakeHealthStore{session: nil}

	req := httptest.NewRequest(http.MethodGet, "/api/files/"+sessionID.Hex()+"/health", nil)
	req = withUserID(req, userID)
	rw := httptest.NewRecorder()

	GetFileHealthHandler(rw, req)

	if rw.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rw.Code, rw.Body.String())
	}
}

func TestGetFileHealthHandler_InvalidSessionID(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()
	metaStore = &fakeHealthStore{}

	req := httptest.NewRequest(http.MethodGet, "/api/files/not-a-valid-id/health", nil)
	req = withUserID(req, primitive.NewObjectID())
	rw := httptest.NewRecorder()

	GetFileHealthHandler(rw, req)

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rw.Code, rw.Body.String())
	}
}

func TestGetFileHealthHandler_WrongPathShapeNotFound(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()
	metaStore = &fakeHealthStore{}

	req := httptest.NewRequest(http.MethodGet, "/api/files/upload/status/", nil)
	req = withUserID(req, primitive.NewObjectID())
	rw := httptest.NewRecorder()

	GetFileHealthHandler(rw, req)

	if rw.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rw.Code, rw.Body.String())
	}
}

func TestGetFileHealthHandler_NilMetaStoreReturns500(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()
	metaStore = nil

	req := httptest.NewRequest(http.MethodGet, "/api/files/"+primitive.NewObjectID().Hex()+"/health", nil)
	req = withUserID(req, primitive.NewObjectID())
	rw := httptest.NewRecorder()

	GetFileHealthHandler(rw, req)

	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rw.Code, rw.Body.String())
	}
}

func TestGetFileHealthHandler_ShardsAndTimeline(t *testing.T) {
	origStore := metaStore
	defer func() { metaStore = origStore }()

	userID := primitive.NewObjectID()
	sessionID := primitive.NewObjectID()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Minute)
	metaStore = &fakeHealthStore{
		session: &models.UploadSession{ID: sessionID, UserID: userID, Status: "complete", OriginalFilename: "a.bin", CreatedAt: t0, CompletedAt: &t1},
		shards: []models.ShardRecord{
			{ShardID: 2, SHA256: "bbb", Status: "pending", CreatedAt: t0.Add(20 * time.Second), Bucket: "bkt"},
			{ShardID: 1, SHA256: "aaa", Status: "verified", CreatedAt: t0.Add(10 * time.Second), Bucket: "bkt"},
		},
	}

	req := withUserID(httptest.NewRequest(http.MethodGet, "/api/files/"+sessionID.Hex()+"/shards", nil), userID)
	rr := httptest.NewRecorder()
	GetFileHealthHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("shards status = %d: %s", rr.Code, rr.Body.String())
	}
	var shardsResp struct {
		Shards []models.ShardRecord `json:"shards"`
	}
	json.Unmarshal(rr.Body.Bytes(), &shardsResp)
	if len(shardsResp.Shards) != 2 || shardsResp.Shards[0].ShardID != 1 {
		t.Fatalf("shards not sorted by id: %+v", shardsResp.Shards)
	}

	req = withUserID(httptest.NewRequest(http.MethodGet, "/api/files/"+sessionID.Hex()+"/timeline", nil), userID)
	rr = httptest.NewRecorder()
	GetFileHealthHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("timeline status = %d: %s", rr.Code, rr.Body.String())
	}
	var tl struct {
		Events []TimelineEvent `json:"events"`
	}
	json.Unmarshal(rr.Body.Bytes(), &tl)
	got := make([]string, 0, len(tl.Events))
	for _, e := range tl.Events {
		got = append(got, e.Type)
	}
	want := []string{"FILE_CREATED", "SHARD_VERIFIED", "SHARD_QUEUED_FOR_RETRY", "FILE_HEALTHY"}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}

	req = withUserID(httptest.NewRequest(http.MethodGet, "/api/files/"+sessionID.Hex()+"/nope", nil), userID)
	rr = httptest.NewRecorder()
	GetFileHealthHandler(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("unknown action status = %d", rr.Code)
	}
}
