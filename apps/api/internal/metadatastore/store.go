// Package metadatastore defines a provider-agnostic interface for the
// metadata persistence operations that internal/store currently performs
// directly against MongoDB. It exists so that a future alternative backend
// (e.g. DynamoDB) can be introduced without touching callers.
//
// Group 0 constraint: this is a pure refactor. MetadataStore mirrors the
// real, current exported surface of internal/store exactly (same names,
// same signatures) -- no new methods, no renamed methods.
package metadatastore

import (
	"SE/internal/models"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MetadataStore mirrors the exported functions of internal/store.
type MetadataStore interface {
	// Lifecycle
	InitStore(ctx context.Context) error
	DisconnectStore(ctx context.Context) error

	// Users
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, u *models.User) error

	// OAuth state
	InsertOAuthState(ctx context.Context, state *models.OAuthState) error
	FindAndDeleteState(ctx context.Context, state string) (*models.OAuthState, error)

	// Drive accounts
	AddDriveAccountToUser(ctx context.Context, userID primitive.ObjectID, acct models.DriveAccount) error
	ListUserDriveAccounts(ctx context.Context, userID primitive.ObjectID) ([]models.DriveAccount, error)
	GetDriveAccountByID(ctx context.Context, accountID primitive.ObjectID) (*models.DriveAccount, error)

	// Upload sessions
	CreateUploadSession(ctx context.Context, session *models.UploadSession) error
	GetUploadSession(ctx context.Context, sessionID primitive.ObjectID) (*models.UploadSession, error)
	UpdateSessionUploadProgress(ctx context.Context, sessionID primitive.ObjectID, uploadedSize int64) error
	UpdateSessionStatus(ctx context.Context, sessionID primitive.ObjectID, status string, progress float64, errorMsg string) error
	CompleteSession(ctx context.Context, sessionID primitive.ObjectID, completedAt *time.Time) error
	CountActiveUserSessions(ctx context.Context, userID primitive.ObjectID) (int, error)
	GetExpiredSessions(ctx context.Context) ([]*models.UploadSession, error)
	DeleteUploadSession(ctx context.Context, sessionID primitive.ObjectID) error
	UpdateSessionKeyFile(ctx context.Context, sessionID primitive.ObjectID, keyFilePath string) error

	// Shard metadata (Integrity Engine)
	SaveShardMetadata(ctx context.Context, sessionID string, shards []models.ShardRecord) error
	GetShardMetadata(ctx context.Context, sessionID string) ([]models.ShardRecord, error)
	UpdateShardStatus(ctx context.Context, sessionID string, shardID int, status string) error
}
