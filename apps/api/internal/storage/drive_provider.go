package storage

import (
	"SE/internal/drivemanager"
	"SE/internal/models"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DriveProvider implements StorageProvider by thin-wrapping the existing
// internal/drivemanager package. It does not reimplement any Google Drive
// logic; it only adapts string-based target IDs to the primitive.ObjectID
// form that drivemanager/store expect.
type DriveProvider struct{}

// compile-time interface compliance assertion
var _ StorageProvider = (*DriveProvider)(nil)

// NewDriveProvider constructs a DriveProvider.
func NewDriveProvider() *DriveProvider {
	return &DriveProvider{}
}

// UploadChunk wraps drivemanager.UploadChunkToDrive.
func (p *DriveProvider) UploadChunk(ctx context.Context, target string, chunkPath, filename string) (string, error) {
	accountID, err := primitive.ObjectIDFromHex(target)
	if err != nil {
		return "", fmt.Errorf("invalid drive account id %q: %w", target, err)
	}
	return drivemanager.UploadChunkToDrive(ctx, accountID, chunkPath, filename)
}

// DeleteChunk wraps drivemanager.DeleteDriveFile.
func (p *DriveProvider) DeleteChunk(ctx context.Context, target string, locationID string) error {
	accountID, err := primitive.ObjectIDFromHex(target)
	if err != nil {
		return fmt.Errorf("invalid drive account id %q: %w", target, err)
	}
	return drivemanager.DeleteDriveFile(ctx, accountID, locationID)
}

// GetSpace wraps drivemanager.GetUserDriveSpaces.
func (p *DriveProvider) GetSpace(ctx context.Context, target string) ([]models.DriveSpaceInfo, error) {
	userID, err := primitive.ObjectIDFromHex(target)
	if err != nil {
		return nil, fmt.Errorf("invalid user id %q: %w", target, err)
	}
	return drivemanager.GetUserDriveSpaces(ctx, userID)
}
