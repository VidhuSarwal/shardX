package metadatastore

import (
	"context"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestMongoStore_InterfaceCompliance ensures MongoStore satisfies
// MetadataStore at compile time.
func TestMongoStore_InterfaceCompliance(t *testing.T) {
	var _ MetadataStore = (*MongoStore)(nil)
	var s MetadataStore = NewMongoStore()
	if s == nil {
		t.Fatal("NewMongoStore() returned nil")
	}
}

// TestMongoStore_SessionMethods_NotInitialized exercises the one class of
// internal/store functions that are safely callable without a live
// MongoDB connection: the upload-session functions nil-check their
// package-level sessionsCol and return an explicit error ("sessions
// collection not initialized") instead of panicking, when InitStore has
// not been called. This lets us prove the MongoStore wrapper delegates
// correctly (same error, unmodified) without needing live Mongo.
//
// NOTE: user/drive-account functions (CreateUser, FindUserByEmail,
// GetDriveAccountByID, ListUserDriveAccounts, AddDriveAccountToUser) do
// NOT nil-check their collections and will panic if called without
// InitStore -- this is a pre-existing property of internal/store, not
// something introduced by this wrapper, so those methods are not
// exercised here without a live Mongo connection.
func TestMongoStore_SessionMethods_NotInitialized(t *testing.T) {
	s := NewMongoStore()
	ctx := context.Background()
	sessionID := primitive.NewObjectID()

	if err := s.CreateUploadSession(ctx, nil); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("CreateUploadSession: expected 'not initialized' error, got %v", err)
	}

	if _, err := s.GetUploadSession(ctx, sessionID); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("GetUploadSession: expected 'not initialized' error, got %v", err)
	}

	if err := s.UpdateSessionUploadProgress(ctx, sessionID, 100); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("UpdateSessionUploadProgress: expected 'not initialized' error, got %v", err)
	}

	if err := s.UpdateSessionStatus(ctx, sessionID, "processing", 10, ""); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("UpdateSessionStatus: expected 'not initialized' error, got %v", err)
	}

	if err := s.CompleteSession(ctx, sessionID, nil); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("CompleteSession: expected 'not initialized' error, got %v", err)
	}

	if _, err := s.CountActiveUserSessions(ctx, sessionID); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("CountActiveUserSessions: expected 'not initialized' error, got %v", err)
	}

	if _, err := s.GetExpiredSessions(ctx); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("GetExpiredSessions: expected 'not initialized' error, got %v", err)
	}

	if err := s.DeleteUploadSession(ctx, sessionID); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("DeleteUploadSession: expected 'not initialized' error, got %v", err)
	}

	if err := s.UpdateSessionKeyFile(ctx, sessionID, "/tmp/foo.key"); err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("UpdateSessionKeyFile: expected 'not initialized' error, got %v", err)
	}
}

// TestMongoStore_DisconnectStore_NoOpWhenNeverConnected verifies that
// DisconnectStore is safe to call even if InitStore was never called
// (mirrors internal/store.DisconnectStore's nil-client guard).
func TestMongoStore_DisconnectStore_NoOpWhenNeverConnected(t *testing.T) {
	s := NewMongoStore()
	if err := s.DisconnectStore(context.Background()); err != nil {
		t.Errorf("expected nil error disconnecting a never-initialized store, got %v", err)
	}
}
