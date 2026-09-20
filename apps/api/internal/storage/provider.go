// Package storage defines a provider-agnostic interface for chunk storage
// backends. It exists so that the rest of the codebase (filehandlers,
// fileprocessor, etc.) can eventually depend on this interface instead of
// calling internal/drivemanager directly, making it possible to add
// alternative backends (e.g. S3) in a later phase without touching callers.
//
// Group 0 constraint: this is a pure refactor. The interface below only
// covers the methods actually invoked today by internal/filehandlers and
// internal/drivemanager (UploadChunkToDrive, DeleteDriveFile,
// GetUserDriveSpaces). No new behavior is introduced.
package storage

import (
	"SE/internal/models"
	"context"
)

// StorageProvider generalizes the operations that internal/drivemanager
// performs against Google Drive today.
//
// Note on "target" semantics: for UploadChunk and DeleteChunk, target is the
// drive account ID (hex string form of primitive.ObjectID, e.g.
// chunk.DriveAccountID.Hex()) that the chunk belongs to/was uploaded to. For
// GetSpace, target is the *user* ID (hex string), because the existing
// GetUserDriveSpaces function operates per-user (returning space info for
// all of that user's linked drive accounts), not per-account. This
// inconsistency mirrors the real call sites and is intentionally preserved
// rather than "fixed", to keep this refactor zero-behavior-change.
type StorageProvider interface {
	// UploadChunk uploads a chunk file at chunkPath (named filename) to the
	// drive account identified by target (hex drive account ID). It returns
	// the resulting drive file ID (locationID).
	UploadChunk(ctx context.Context, target string, chunkPath, filename string) (locationID string, err error)

	// DeleteChunk deletes the chunk identified by locationID from the drive
	// account identified by target (hex drive account ID).
	DeleteChunk(ctx context.Context, target string, locationID string) error

	// GetSpace returns space information for all drive accounts linked to
	// the user identified by target (hex user ID).
	GetSpace(ctx context.Context, target string) ([]models.DriveSpaceInfo, error)
}
