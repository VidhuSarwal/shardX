// DynamoDB-backed implementation of MetadataStore.
//
// Table / index schema assumptions (to be provisioned by a future Terraform
// task -- not written here, just documented so that task can match):
//
//   {prefix}-users
//     PK: id (S)               -- hex string of a primitive.ObjectID
//     GSI: email-index          PK: email (S), Projection: ALL
//       Needed by FindUserByEmail, which in Mongo looks users up by a
//       unique index on "email" rather than by _id. Projection MUST be ALL
//       (not KEYS_ONLY/INCLUDE-partial): FindUserByEmail reads the full
//       item straight off the GSI result (passwords_hash, created_at) --
//       if the GSI only projects keys, login will silently see an empty
//       password hash.
//
//   {prefix}-oauth-states
//     PK: state (S)             -- the OAuth state token is already the
//                                  natural unique key Mongo queries on
//                                  (FindAndDeleteState does bson.M{"state": state}),
//                                  so it is used directly as the partition
//                                  key instead of introducing a synthetic id.
//     TTL attribute: created_at is used purely informationally here; Mongo
//       relies on a TTL index (10 minutes) to expire abandoned OAuth state
//       docs. DynamoDB TTL requires a Unix-epoch-seconds numeric attribute,
//       so the Terraform table should add an `expires_at_unix` (N) attribute
//       with a TTL spec if automatic expiry is desired. Not implemented here
//       since InitStore doesn't create indexes at runtime for Dynamo (see
//       below) -- flagged for the Terraform task.
//
//   {prefix}-upload-sessions
//     PK: id (S)                -- hex string of a primitive.ObjectID
//     GSI: user_id-index         PK: user_id (S), Projection: ALL
//       Needed by CountActiveUserSessions (Mongo: sessionsCol.CountDocuments
//       filtered by user_id + status). We Query the GSI and filter on
//       status via a FilterExpression -- Projection MUST be ALL (or at
//       minimum INCLUDE "status") or the FilterExpression on #status
//       cannot evaluate and the count silently comes back wrong.
//     GetExpiredSessions has no good key -- Mongo scans sessionsCol filtered
//       by expires_at + status with no index assumption either. We do a
//       Scan with a FilterExpression on expires_at/status here, which is
//       the faithful (if not most efficient) DynamoDB equivalent.
//
//   {prefix}-drive-accounts
//     PK: id (S)                -- hex string of a primitive.ObjectID
//     GSI: user_id-index         PK: user_id (S), Projection: ALL
//       ListUserDriveAccounts reads full items off this GSI (provider,
//       display_name, encrypted_token) -- Projection MUST be ALL or those
//       fields silently come back empty.
//     Mongo has no separate drive-accounts collection: DriveAccount is an
//     embedded array field (`drive_accounts`) on the User document. We
//     intentionally break this out into its own DynamoDB table so that
//     AddDriveAccountToUser / ListUserDriveAccounts / GetDriveAccountByID
//     map to clean PutItem / Query(user_id-index) / GetItem operations
//     instead of requiring a full table Scan for GetDriveAccountByID (which
//     looks an account up by its own id with no user context) or read-
//     modify-write races for AddDriveAccountToUser. This is invisible to
//     callers through the MetadataStore interface.
//
//   {prefix}-shard-metadata
//     PK: file_id (S)            -- the upload session ID (hex string),
//                                   reused as the "file id" since this
//                                   system has no separate file identifier.
//     SK: shard_id (N)           -- the chunk/shard index within the file.
//       A single partition key alone cannot hold N shard rows per file (each
//       PutItem would overwrite the previous one) so this table uses a
//       composite primary key (PK+SK), matching the natural
//       "every shard has a record" access pattern: GetShardMetadata does a
//       Query on file_id and gets all shard rows back in one call; no GSI is
//       needed since the only read path is "all shards for a file".
//
// ID format decision: the MetadataStore interface signatures are hard-fixed
// to primitive.ObjectID (from go.mongodb.org/mongo-driver/bson/primitive),
// inherited unchanged from internal/store. A UUID string cannot satisfy
// that type without changing the interface, which the task requires
// mirroring exactly. primitive.NewObjectID() is just a 12-byte identifier
// generator -- it has no dependency on a running MongoDB -- so we reuse it
// here for parity with Mongo's ID shape/format instead of adding a
// google/uuid dependency. IDs are stored in DynamoDB as their .Hex() string
// form (a DynamoDB item's partition key must be a scalar S/N/B; a raw
// primitive.ObjectID marshals via attributevalue as a byte-list, not a
// friendly key type, so we go through hex strings explicitly).
//
// Not-found semantics are replicated per-method to match internal/store
// exactly (this is a deliberate behavior-preserving detail, not an
// oversight):
//   - FindUserByEmail, GetUploadSession, FindAndDeleteState: not found -> (nil, nil)
//   - GetDriveAccountByID: not found -> (nil, error) (Mongo does not
//     special-case mongo.ErrNoDocuments in this one function)
//
// Known, deliberate behavior divergences from Mongo (both a consequence of
// splitting drive-accounts into their own table -- see above):
//   - FindUserByEmail/CreateUser always return/store DriveAccounts as an
//     empty slice rather than Mongo's embedded, possibly-populated array.
//     No current caller of FindUserByEmail depends on a populated
//     DriveAccounts field (the login path only checks the password hash),
//     but a future caller that does would need to separately call
//     ListUserDriveAccounts.
//   - CountActiveUserSessions counts a single Query page (out.Items) off
//     the user_id-index GSI rather than Mongo's exhaustive
//     CountDocuments. Fine at the expected scale (a user's own concurrent
//     upload sessions, not a global count), but would need pagination
//     (LastEvaluatedKey) if a user could plausibly exceed 1MB of GSI
//     result items in "uploading"/"processing" status.
//
// InitStore/DisconnectStore are no-ops for DynamoDB: unlike Mongo's
// InitStore (which opens a connection and creates indexes at runtime),
// DynamoDB tables/GSIs/TTL specs are expected to be provisioned ahead of
// time (by Terraform); there is no persistent connection to open or close.
package metadatastore

import (
	"SE/internal/models"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// dynamoDBAPI is the subset of *dynamodb.Client that DynamoDBStore calls.
// Keeping it as a small interface (rather than depending on the concrete
// SDK client type) lets unit tests supply a fake implementation without
// needing a live AWS account or LocalStack.
type dynamoDBAPI interface {
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// DynamoDBStore implements MetadataStore backed by Amazon DynamoDB.
type DynamoDBStore struct {
	client              dynamoDBAPI
	usersTable          string
	oauthStatesTable    string
	uploadSessionsTable string
	driveAccountsTable  string
	shardMetadataTable  string
}

// compile-time interface compliance assertion
var _ MetadataStore = (*DynamoDBStore)(nil)

// NewDynamoDBStore constructs a DynamoDBStore, loading AWS configuration via
// the standard credential/region chain (AWS_PROFILE, AWS_REGION, shared
// config/credentials files, instance/task roles, etc). No credentials are
// hardcoded.
func NewDynamoDBStore(ctx context.Context, tablePrefix string) (*DynamoDBStore, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	client := dynamodb.NewFromConfig(cfg)
	return &DynamoDBStore{
		client:              client,
		usersTable:          tablePrefix + "-users",
		oauthStatesTable:    tablePrefix + "-oauth-states",
		uploadSessionsTable: tablePrefix + "-upload-sessions",
		driveAccountsTable:  tablePrefix + "-drive-accounts",
		shardMetadataTable:  tablePrefix + "-shard-metadata",
	}, nil
}

// newDynamoDBStoreWithClient is used by tests to inject a fake client.
func newDynamoDBStoreWithClient(client dynamoDBAPI, tablePrefix string) *DynamoDBStore {
	return &DynamoDBStore{
		client:              client,
		usersTable:          tablePrefix + "-users",
		oauthStatesTable:    tablePrefix + "-oauth-states",
		uploadSessionsTable: tablePrefix + "-upload-sessions",
		driveAccountsTable:  tablePrefix + "-drive-accounts",
		shardMetadataTable:  tablePrefix + "-shard-metadata",
	}
}

// ---- lifecycle ----

// InitStore is a no-op for DynamoDB: tables/GSIs/TTL are provisioned ahead
// of time (Terraform), there is no connection to open, and the SDK client
// is already constructed in NewDynamoDBStore.
func (d *DynamoDBStore) InitStore(ctx context.Context) error {
	return nil
}

// DisconnectStore is a no-op for DynamoDB: the SDK client has no persistent
// connection/session that needs explicit teardown.
func (d *DynamoDBStore) DisconnectStore(ctx context.Context) error {
	return nil
}

// ---- record shapes for DynamoDB marshaling ----
//
// We do not attributevalue.MarshalMap the models package structs directly:
// primitive.ObjectID is a [12]byte array that attributevalue would encode
// as a numeric list rather than a readable key, and the bson struct tags
// on models.* are meaningless to attributevalue (which reads `dynamodbav`
// tags). These record types are the DynamoDB-native shapes, translated
// to/from models.* in each method.

type userRecord struct {
	ID            string    `dynamodbav:"id"`
	Email         string    `dynamodbav:"email"`
	PasswordsHash []byte    `dynamodbav:"passwords_hash"`
	CreatedAt     time.Time `dynamodbav:"created_at"`
}

type oauthStateRecord struct {
	State     string    `dynamodbav:"state"`
	UserID    string    `dynamodbav:"user_id"`
	Provider  string    `dynamodbav:"provider"`
	CreatedAt time.Time `dynamodbav:"created_at"`
}

type driveAccountRecord struct {
	ID             string    `dynamodbav:"id"`
	UserID         string    `dynamodbav:"user_id"`
	Provider       string    `dynamodbav:"provider"`
	DisplayName    string    `dynamodbav:"display_name"`
	EncryptedToken []byte    `dynamodbav:"encrypted_token"`
	CreatedAt      time.Time `dynamodbav:"created_at"`
}

type uploadSessionRecord struct {
	ID                 string     `dynamodbav:"id"`
	UserID             string     `dynamodbav:"user_id"`
	OriginalFilename   string     `dynamodbav:"original_filename"`
	TempFilePath       string     `dynamodbav:"temp_file_path"`
	KeyFilePath        string     `dynamodbav:"key_file_path,omitempty"`
	TotalSize          int64      `dynamodbav:"total_size"`
	UploadedSize       int64      `dynamodbav:"uploaded_size"`
	Status             string     `dynamodbav:"status"`
	ProcessingProgress float64    `dynamodbav:"processing_progress"`
	ErrorMessage       string     `dynamodbav:"error_message,omitempty"`
	CreatedAt          time.Time  `dynamodbav:"created_at"`
	ExpiresAt          time.Time  `dynamodbav:"expires_at"`
	CompletedAt        *time.Time `dynamodbav:"completed_at,omitempty"`
}

func userToRecord(u *models.User) userRecord {
	return userRecord{
		ID:            u.ID.Hex(),
		Email:         u.Email,
		PasswordsHash: u.PasswordsHash,
		CreatedAt:     u.CreatedAt,
	}
}

func recordToUser(r userRecord) (*models.User, error) {
	id, err := primitive.ObjectIDFromHex(r.ID)
	if err != nil {
		return nil, fmt.Errorf("decode user id %q: %w", r.ID, err)
	}
	return &models.User{
		ID:            id,
		Email:         r.Email,
		PasswordsHash: r.PasswordsHash,
		DriveAccounts: []models.DriveAccount{},
		CreatedAt:     r.CreatedAt,
	}, nil
}

func driveAccountToRecord(userID primitive.ObjectID, acct models.DriveAccount) driveAccountRecord {
	return driveAccountRecord{
		ID:             acct.ID.Hex(),
		UserID:         userID.Hex(),
		Provider:       acct.Provider,
		DisplayName:    acct.DisplayName,
		EncryptedToken: acct.EncryptedToken,
		CreatedAt:      acct.CreatedAt,
	}
}

func recordToDriveAccount(r driveAccountRecord) (*models.DriveAccount, error) {
	id, err := primitive.ObjectIDFromHex(r.ID)
	if err != nil {
		return nil, fmt.Errorf("decode drive account id %q: %w", r.ID, err)
	}
	return &models.DriveAccount{
		ID:             id,
		Provider:       r.Provider,
		DisplayName:    r.DisplayName,
		EncryptedToken: r.EncryptedToken,
		CreatedAt:      r.CreatedAt,
	}, nil
}

type shardMetadataRecord struct {
	FileID    string    `dynamodbav:"file_id"`
	ShardID   int       `dynamodbav:"shard_id"`
	SHA256    string    `dynamodbav:"sha256"`
	Size      int64     `dynamodbav:"size"`
	Bucket    string    `dynamodbav:"bucket,omitempty"`
	Region    string    `dynamodbav:"region,omitempty"`
	CreatedAt time.Time `dynamodbav:"created_at"`
	Status    string    `dynamodbav:"status"`
}

func shardRecordToRecord(sessionID string, s models.ShardRecord) shardMetadataRecord {
	return shardMetadataRecord{
		FileID:    sessionID,
		ShardID:   s.ShardID,
		SHA256:    s.SHA256,
		Size:      s.Size,
		Bucket:    s.Bucket,
		Region:    s.Region,
		CreatedAt: s.CreatedAt,
		Status:    s.Status,
	}
}

func recordToShardRecord(r shardMetadataRecord) models.ShardRecord {
	return models.ShardRecord{
		FileID:    r.FileID,
		ShardID:   r.ShardID,
		SHA256:    r.SHA256,
		Size:      r.Size,
		Bucket:    r.Bucket,
		Region:    r.Region,
		CreatedAt: r.CreatedAt,
		Status:    r.Status,
	}
}

func sessionToRecord(s *models.UploadSession) uploadSessionRecord {
	return uploadSessionRecord{
		ID:                 s.ID.Hex(),
		UserID:             s.UserID.Hex(),
		OriginalFilename:   s.OriginalFilename,
		TempFilePath:       s.TempFilePath,
		KeyFilePath:        s.KeyFilePath,
		TotalSize:          s.TotalSize,
		UploadedSize:       s.UploadedSize,
		Status:             s.Status,
		ProcessingProgress: s.ProcessingProgress,
		ErrorMessage:       s.ErrorMessage,
		CreatedAt:          s.CreatedAt,
		ExpiresAt:          s.ExpiresAt,
		CompletedAt:        s.CompletedAt,
	}
}

func recordToSession(r uploadSessionRecord) (*models.UploadSession, error) {
	id, err := primitive.ObjectIDFromHex(r.ID)
	if err != nil {
		return nil, fmt.Errorf("decode session id %q: %w", r.ID, err)
	}
	userID, err := primitive.ObjectIDFromHex(r.UserID)
	if err != nil {
		return nil, fmt.Errorf("decode session user_id %q: %w", r.UserID, err)
	}
	return &models.UploadSession{
		ID:                 id,
		UserID:             userID,
		OriginalFilename:   r.OriginalFilename,
		TempFilePath:       r.TempFilePath,
		KeyFilePath:        r.KeyFilePath,
		TotalSize:          r.TotalSize,
		UploadedSize:       r.UploadedSize,
		Status:             r.Status,
		ProcessingProgress: r.ProcessingProgress,
		ErrorMessage:       r.ErrorMessage,
		CreatedAt:          r.CreatedAt,
		ExpiresAt:          r.ExpiresAt,
		CompletedAt:        r.CompletedAt,
	}, nil
}

// ---- users ----

const usersEmailIndex = "email-index"

func (d *DynamoDBStore) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	out, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.usersTable),
		IndexName:              aws.String(usersEmailIndex),
		KeyConditionExpression: aws.String("email = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: email},
		},
		Limit: aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("query users by email: %w", err)
	}
	if len(out.Items) == 0 {
		return nil, nil
	}
	var rec userRecord
	if err := attributevalue.UnmarshalMap(out.Items[0], &rec); err != nil {
		return nil, fmt.Errorf("unmarshal user: %w", err)
	}
	return recordToUser(rec)
}

func (d *DynamoDBStore) CreateUser(ctx context.Context, u *models.User) error {
	u.CreatedAt = time.Now().UTC()
	if u.ID.IsZero() {
		u.ID = primitive.NewObjectID()
	}

	item, err := attributevalue.MarshalMap(userToRecord(u))
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.usersTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put user: %w", err)
	}
	return nil
}

// ---- oauth state ----

func (d *DynamoDBStore) InsertOAuthState(ctx context.Context, state *models.OAuthState) error {
	state.CreatedAt = time.Now().UTC()
	rec := oauthStateRecord{
		State:     state.State,
		UserID:    state.UserID.Hex(),
		Provider:  state.Provider,
		CreatedAt: state.CreatedAt,
	}
	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return fmt.Errorf("marshal oauth state: %w", err)
	}
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.oauthStatesTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put oauth state: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) FindAndDeleteState(ctx context.Context, state string) (*models.OAuthState, error) {
	out, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(d.oauthStatesTable),
		Key: map[string]types.AttributeValue{
			"state": &types.AttributeValueMemberS{Value: state},
		},
		ReturnValues: types.ReturnValueAllOld,
	})
	if err != nil {
		return nil, fmt.Errorf("delete oauth state: %w", err)
	}
	if len(out.Attributes) == 0 {
		return nil, nil
	}
	var rec oauthStateRecord
	if err := attributevalue.UnmarshalMap(out.Attributes, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal oauth state: %w", err)
	}
	userID, err := primitive.ObjectIDFromHex(rec.UserID)
	if err != nil {
		return nil, fmt.Errorf("decode oauth state user_id %q: %w", rec.UserID, err)
	}
	return &models.OAuthState{
		CreatedAt: rec.CreatedAt,
		State:     rec.State,
		UserID:    userID,
		Provider:  rec.Provider,
	}, nil
}

// ---- drive accounts ----
//
// See package-level doc comment: drive accounts get their own table (not
// embedded in the user item like Mongo) with a user_id-index GSI.

const driveAccountsUserIDIndex = "user_id-index"

func (d *DynamoDBStore) AddDriveAccountToUser(ctx context.Context, userID primitive.ObjectID, acct models.DriveAccount) error {
	acct.CreatedAt = time.Now().UTC()
	acct.ID = primitive.NewObjectID()

	item, err := attributevalue.MarshalMap(driveAccountToRecord(userID, acct))
	if err != nil {
		return fmt.Errorf("marshal drive account: %w", err)
	}
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.driveAccountsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put drive account: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) ListUserDriveAccounts(ctx context.Context, userID primitive.ObjectID) ([]models.DriveAccount, error) {
	out, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.driveAccountsTable),
		IndexName:              aws.String(driveAccountsUserIDIndex),
		KeyConditionExpression: aws.String("user_id = :user_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":user_id": &types.AttributeValueMemberS{Value: userID.Hex()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query drive accounts by user: %w", err)
	}
	accounts := make([]models.DriveAccount, 0, len(out.Items))
	for _, item := range out.Items {
		var rec driveAccountRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			return nil, fmt.Errorf("unmarshal drive account: %w", err)
		}
		acct, err := recordToDriveAccount(rec)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *acct)
	}
	return accounts, nil
}

func (d *DynamoDBStore) GetDriveAccountByID(ctx context.Context, accountID primitive.ObjectID) (*models.DriveAccount, error) {
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.driveAccountsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: accountID.Hex()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get drive account: %w", err)
	}
	if len(out.Item) == 0 {
		// Match Mongo's GetDriveAccountByID, which does NOT special-case
		// "not found" -- it propagates an error rather than returning
		// (nil, nil).
		return nil, errors.New("account not found")
	}
	var rec driveAccountRecord
	if err := attributevalue.UnmarshalMap(out.Item, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal drive account: %w", err)
	}
	return recordToDriveAccount(rec)
}

// ---- upload sessions ----

const uploadSessionsUserIDIndex = "user_id-index"

func (d *DynamoDBStore) CreateUploadSession(ctx context.Context, session *models.UploadSession) error {
	// Mongo relies on the server to assign _id on InsertOne; there is no
	// server-side ID assignment in DynamoDB, so generate one here if the
	// caller hasn't already set one (mirrors CreateUser/AddDriveAccountToUser).
	if session.ID.IsZero() {
		session.ID = primitive.NewObjectID()
	}
	item, err := attributevalue.MarshalMap(sessionToRecord(session))
	if err != nil {
		return fmt.Errorf("marshal upload session: %w", err)
	}
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put upload session: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) GetUploadSession(ctx context.Context, sessionID primitive.ObjectID) (*models.UploadSession, error) {
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: sessionID.Hex()},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get upload session: %w", err)
	}
	if len(out.Item) == 0 {
		return nil, nil
	}
	var rec uploadSessionRecord
	if err := attributevalue.UnmarshalMap(out.Item, &rec); err != nil {
		return nil, fmt.Errorf("unmarshal upload session: %w", err)
	}
	return recordToSession(rec)
}

func (d *DynamoDBStore) UpdateSessionUploadProgress(ctx context.Context, sessionID primitive.ObjectID, uploadedSize int64) error {
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: sessionID.Hex()},
		},
		UpdateExpression: aws.String("SET uploaded_size = :uploaded_size"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uploaded_size": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", uploadedSize)},
		},
	})
	if err != nil {
		return fmt.Errorf("update session upload progress: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) UpdateSessionStatus(ctx context.Context, sessionID primitive.ObjectID, status string, progress float64, errorMsg string) error {
	updateExpr := "SET #status = :status, processing_progress = :progress"
	exprNames := map[string]string{"#status": "status"}
	exprValues := map[string]types.AttributeValue{
		":status":   &types.AttributeValueMemberS{Value: status},
		":progress": &types.AttributeValueMemberN{Value: fmt.Sprintf("%v", progress)},
	}
	if errorMsg != "" {
		updateExpr += ", error_message = :error_message"
		exprValues[":error_message"] = &types.AttributeValueMemberS{Value: errorMsg}
	}
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: sessionID.Hex()},
		},
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
	})
	if err != nil {
		return fmt.Errorf("update session status: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) CompleteSession(ctx context.Context, sessionID primitive.ObjectID, completedAt *time.Time) error {
	var completedAtAV types.AttributeValue
	if completedAt != nil {
		completedAtAV = &types.AttributeValueMemberS{Value: completedAt.Format(time.RFC3339Nano)}
	} else {
		completedAtAV = &types.AttributeValueMemberNULL{Value: true}
	}
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: sessionID.Hex()},
		},
		UpdateExpression: aws.String("SET #status = :status, completed_at = :completed_at"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":       &types.AttributeValueMemberS{Value: "complete"},
			":completed_at": completedAtAV,
		},
	})
	if err != nil {
		return fmt.Errorf("complete session: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) CountActiveUserSessions(ctx context.Context, userID primitive.ObjectID) (int, error) {
	out, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.uploadSessionsTable),
		IndexName:              aws.String(uploadSessionsUserIDIndex),
		KeyConditionExpression: aws.String("user_id = :user_id"),
		FilterExpression:       aws.String("#status IN (:uploading, :processing)"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":user_id":    &types.AttributeValueMemberS{Value: userID.Hex()},
			":uploading":  &types.AttributeValueMemberS{Value: "uploading"},
			":processing": &types.AttributeValueMemberS{Value: "processing"},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("count active user sessions: %w", err)
	}
	return len(out.Items), nil
}

func (d *DynamoDBStore) GetExpiredSessions(ctx context.Context) ([]*models.UploadSession, error) {
	// No good key for this access pattern in either backend: Mongo does an
	// unindexed collection scan filtered by expires_at + status, so a
	// DynamoDB Scan with an equivalent FilterExpression is the faithful
	// translation (not a regression -- Mongo's version has the same
	// O(collection size) cost profile absent a supporting index).
	now := time.Now()
	out, err := d.client.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(d.uploadSessionsTable),
		FilterExpression: aws.String("expires_at < :now AND #status IN (:uploading, :processing)"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now":        &types.AttributeValueMemberS{Value: now.Format(time.RFC3339Nano)},
			":uploading":  &types.AttributeValueMemberS{Value: "uploading"},
			":processing": &types.AttributeValueMemberS{Value: "processing"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("scan expired sessions: %w", err)
	}
	sessions := make([]*models.UploadSession, 0, len(out.Items))
	for _, item := range out.Items {
		var rec uploadSessionRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			return nil, fmt.Errorf("unmarshal upload session: %w", err)
		}
		s, err := recordToSession(rec)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (d *DynamoDBStore) DeleteUploadSession(ctx context.Context, sessionID primitive.ObjectID) error {
	_, err := d.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: sessionID.Hex()},
		},
	})
	if err != nil {
		return fmt.Errorf("delete upload session: %w", err)
	}
	return nil
}

func (d *DynamoDBStore) UpdateSessionKeyFile(ctx context.Context, sessionID primitive.ObjectID, keyFilePath string) error {
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.uploadSessionsTable),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: sessionID.Hex()},
		},
		UpdateExpression: aws.String("SET key_file_path = :key_file_path"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":key_file_path": &types.AttributeValueMemberS{Value: keyFilePath},
		},
	})
	if err != nil {
		return fmt.Errorf("update session key file: %w", err)
	}
	return nil
}

// ---- shard metadata (Integrity Engine) ----
//
// See package-level doc comment: this table uses a composite key
// (file_id, shard_id) since a file has many shard rows. There is no
// BatchWriteItem in the dynamoDBAPI interface (kept intentionally small for
// testability), so SaveShardMetadata issues one PutItem per shard -- fine at
// expected shard counts (tens, not thousands, per file).

func (d *DynamoDBStore) SaveShardMetadata(ctx context.Context, sessionID string, shards []models.ShardRecord) error {
	for _, s := range shards {
		item, err := attributevalue.MarshalMap(shardRecordToRecord(sessionID, s))
		if err != nil {
			return fmt.Errorf("marshal shard metadata: %w", err)
		}
		if _, err := d.client.PutItem(ctx, &dynamodb.PutItemInput{
			TableName: aws.String(d.shardMetadataTable),
			Item:      item,
		}); err != nil {
			return fmt.Errorf("put shard metadata (shard %d): %w", s.ShardID, err)
		}
	}
	return nil
}

func (d *DynamoDBStore) GetShardMetadata(ctx context.Context, sessionID string) ([]models.ShardRecord, error) {
	out, err := d.client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(d.shardMetadataTable),
		KeyConditionExpression: aws.String("file_id = :file_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":file_id": &types.AttributeValueMemberS{Value: sessionID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("query shard metadata: %w", err)
	}
	shards := make([]models.ShardRecord, 0, len(out.Items))
	for _, item := range out.Items {
		var rec shardMetadataRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			return nil, fmt.Errorf("unmarshal shard metadata: %w", err)
		}
		shards = append(shards, recordToShardRecord(rec))
	}
	return shards, nil
}

func (d *DynamoDBStore) UpdateShardStatus(ctx context.Context, sessionID string, shardID int, status string) error {
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.shardMetadataTable),
		Key: map[string]types.AttributeValue{
			"file_id":  &types.AttributeValueMemberS{Value: sessionID},
			"shard_id": &types.AttributeValueMemberN{Value: strconv.Itoa(shardID)},
		},
		UpdateExpression:         aws.String("SET #status = :status"),
		ExpressionAttributeNames: map[string]string{"#status": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: status},
		},
	})
	if err != nil {
		return fmt.Errorf("update shard status (shard %d): %w", shardID, err)
	}
	return nil
}
