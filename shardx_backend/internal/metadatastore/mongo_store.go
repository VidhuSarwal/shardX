package metadatastore

import (
	"SE/internal/models"
	"SE/internal/store"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoStore implements MetadataStore by thin-wrapping the existing
// internal/store package (unchanged behavior, unchanged MongoDB access).
type MongoStore struct{}

// compile-time interface compliance assertion
var _ MetadataStore = (*MongoStore)(nil)

// NewMongoStore constructs a MongoStore.
func NewMongoStore() *MongoStore {
	return &MongoStore{}
}

func (m *MongoStore) InitStore(ctx context.Context) error {
	return store.InitStore(ctx)
}

func (m *MongoStore) DisconnectStore(ctx context.Context) error {
	return store.DisconnectStore(ctx)
}

func (m *MongoStore) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return store.FindUserByEmail(ctx, email)
}

func (m *MongoStore) CreateUser(ctx context.Context, u *models.User) error {
	return store.CreateUser(ctx, u)
}

func (m *MongoStore) InsertOAuthState(ctx context.Context, state *models.OAuthState) error {
	return store.InsertOAuthState(ctx, state)
}

func (m *MongoStore) FindAndDeleteState(ctx context.Context, state string) (*models.OAuthState, error) {
	return store.FindAndDeleteState(ctx, state)
}

func (m *MongoStore) AddDriveAccountToUser(ctx context.Context, userID primitive.ObjectID, acct models.DriveAccount) error {
	return store.AddDriveAccountToUser(ctx, userID, acct)
}

func (m *MongoStore) ListUserDriveAccounts(ctx context.Context, userID primitive.ObjectID) ([]models.DriveAccount, error) {
	return store.ListUserDriveAccounts(ctx, userID)
}

func (m *MongoStore) GetDriveAccountByID(ctx context.Context, accountID primitive.ObjectID) (*models.DriveAccount, error) {
	return store.GetDriveAccountByID(ctx, accountID)
}

func (m *MongoStore) CreateUploadSession(ctx context.Context, session *models.UploadSession) error {
	return store.CreateUploadSession(ctx, session)
}

func (m *MongoStore) GetUploadSession(ctx context.Context, sessionID primitive.ObjectID) (*models.UploadSession, error) {
	return store.GetUploadSession(ctx, sessionID)
}

func (m *MongoStore) UpdateSessionUploadProgress(ctx context.Context, sessionID primitive.ObjectID, uploadedSize int64) error {
	return store.UpdateSessionUploadProgress(ctx, sessionID, uploadedSize)
}

func (m *MongoStore) UpdateSessionStatus(ctx context.Context, sessionID primitive.ObjectID, status string, progress float64, errorMsg string) error {
	return store.UpdateSessionStatus(ctx, sessionID, status, progress, errorMsg)
}

func (m *MongoStore) CompleteSession(ctx context.Context, sessionID primitive.ObjectID, completedAt *time.Time) error {
	return store.CompleteSession(ctx, sessionID, completedAt)
}

func (m *MongoStore) CountActiveUserSessions(ctx context.Context, userID primitive.ObjectID) (int, error) {
	return store.CountActiveUserSessions(ctx, userID)
}

func (m *MongoStore) GetExpiredSessions(ctx context.Context) ([]*models.UploadSession, error) {
	return store.GetExpiredSessions(ctx)
}

func (m *MongoStore) DeleteUploadSession(ctx context.Context, sessionID primitive.ObjectID) error {
	return store.DeleteUploadSession(ctx, sessionID)
}

func (m *MongoStore) UpdateSessionKeyFile(ctx context.Context, sessionID primitive.ObjectID, keyFilePath string) error {
	return store.UpdateSessionKeyFile(ctx, sessionID, keyFilePath)
}

func (m *MongoStore) SaveShardMetadata(ctx context.Context, sessionID string, shards []models.ShardRecord) error {
	return store.SaveShardMetadata(ctx, sessionID, shards)
}

func (m *MongoStore) GetShardMetadata(ctx context.Context, sessionID string) ([]models.ShardRecord, error) {
	return store.GetShardMetadata(ctx, sessionID)
}
