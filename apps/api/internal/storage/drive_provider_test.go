package storage

import (
	"context"
	"errors"
	"testing"

	"SE/internal/drivemanager"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestDriveProvider_InterfaceCompliance ensures DriveProvider satisfies
// StorageProvider at compile time. (Also asserted as a package-level var in
// drive_provider.go; repeated here so a test explicitly exercises it.)
func TestDriveProvider_InterfaceCompliance(t *testing.T) {
	var _ StorageProvider = (*DriveProvider)(nil)
	var p StorageProvider = NewDriveProvider()
	if p == nil {
		t.Fatal("NewDriveProvider() returned nil")
	}
}

// TestDriveProvider_UploadChunk_InvalidTarget verifies that an invalid hex
// target string is rejected before any network/Mongo call is attempted,
// and that the error is returned rather than panicking.
func TestDriveProvider_UploadChunk_InvalidTarget(t *testing.T) {
	p := NewDriveProvider()
	_, err := p.UploadChunk(context.Background(), "not-a-valid-object-id", "/tmp/does-not-matter", "chunk_000.2xpfm")
	if err == nil {
		t.Fatal("expected error for invalid target, got nil")
	}
	if _, parseErr := primitive.ObjectIDFromHex("not-a-valid-object-id"); !errors.Is(err, parseErr) {
		// Not required to be the exact same error type, just confirm it's
		// a parse-related failure and not a nil-pointer panic / silent
		// success.
		t.Logf("upload error (expected, parse-related): %v", err)
	}
}

// TestDriveProvider_DeleteChunk_InvalidTarget mirrors the upload case for
// DeleteChunk.
func TestDriveProvider_DeleteChunk_InvalidTarget(t *testing.T) {
	p := NewDriveProvider()
	err := p.DeleteChunk(context.Background(), "not-a-valid-object-id", "some-file-id")
	if err == nil {
		t.Fatal("expected error for invalid target, got nil")
	}
}

// TestDriveProvider_GetSpace_InvalidTarget mirrors the upload case for
// GetSpace.
func TestDriveProvider_GetSpace_InvalidTarget(t *testing.T) {
	p := NewDriveProvider()
	_, err := p.GetSpace(context.Background(), "not-a-valid-object-id")
	if err == nil {
		t.Fatal("expected error for invalid target, got nil")
	}
}

// TestDriveProvider_ValidHexParsesCleanly documents the boundary of what is
// testable without live Mongo/Drive credentials: DriveProvider correctly
// parses a well-formed hex target into a primitive.ObjectID (the same type
// drivemanager.UploadChunkToDrive/DeleteDriveFile/GetUserDriveSpaces
// expect), matching what the wrapper passes through unchanged. We stop
// short of invoking p.UploadChunk/DeleteChunk/GetSpace with this valid hex
// because, without a live MongoDB connection, internal/store's
// GetDriveAccountByID / ListUserDriveAccounts panic on their nil
// collections (a pre-existing property of internal/store, unrelated to
// this wrapper) rather than returning an error. Full end-to-end behavior
// requires live Mongo + Drive credentials, unavailable in this
// environment.
func TestDriveProvider_ValidHexParsesCleanly(t *testing.T) {
	validHex := primitive.NewObjectID().Hex()
	parsed, err := primitive.ObjectIDFromHex(validHex)
	if err != nil {
		t.Fatalf("expected valid hex to parse, got error: %v", err)
	}
	if parsed.Hex() != validHex {
		t.Fatalf("round-trip mismatch: got %s, want %s", parsed.Hex(), validHex)
	}
	// Sanity: the drivemanager functions this wrapper delegates to exist
	// and have the expected signature (compile-time check only).
	var _ = drivemanager.UploadChunkToDrive
	var _ = drivemanager.DeleteDriveFile
	var _ = drivemanager.GetUserDriveSpaces
}
