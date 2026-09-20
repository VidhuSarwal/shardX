package metadatastore

import (
	"SE/internal/models"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeDynamoClient is a minimal in-memory-ish stand-in for the dynamoDBAPI
// interface. Each test wires up only the behavior it needs via function
// fields; unset fields cause a test failure if invoked (nil deref caught by
// panic->t.Fatal pattern is intentionally avoided by explicit nil checks).
type fakeDynamoClient struct {
	getItemFn    func(ctx context.Context, in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
	putItemFn    func(ctx context.Context, in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
	deleteItemFn func(ctx context.Context, in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
	updateItemFn func(ctx context.Context, in *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
	queryFn      func(ctx context.Context, in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
	scanFn       func(ctx context.Context, in *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
}

func (f *fakeDynamoClient) GetItem(ctx context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if f.getItemFn == nil {
		return nil, errors.New("GetItem not configured")
	}
	return f.getItemFn(ctx, in)
}

func (f *fakeDynamoClient) PutItem(ctx context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if f.putItemFn == nil {
		return nil, errors.New("PutItem not configured")
	}
	return f.putItemFn(ctx, in)
}

func (f *fakeDynamoClient) DeleteItem(ctx context.Context, in *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if f.deleteItemFn == nil {
		return nil, errors.New("DeleteItem not configured")
	}
	return f.deleteItemFn(ctx, in)
}

func (f *fakeDynamoClient) UpdateItem(ctx context.Context, in *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	if f.updateItemFn == nil {
		return nil, errors.New("UpdateItem not configured")
	}
	return f.updateItemFn(ctx, in)
}

func (f *fakeDynamoClient) Query(ctx context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if f.queryFn == nil {
		return nil, errors.New("Query not configured")
	}
	return f.queryFn(ctx, in)
}

func (f *fakeDynamoClient) Scan(ctx context.Context, in *dynamodb.ScanInput, _ ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	if f.scanFn == nil {
		return nil, errors.New("Scan not configured")
	}
	return f.scanFn(ctx, in)
}

func TestDynamoDBStore_InterfaceCompliance(t *testing.T) {
	var _ MetadataStore = (*DynamoDBStore)(nil)
}

func TestDynamoDBStore_InitAndDisconnectAreNoops(t *testing.T) {
	s := newDynamoDBStoreWithClient(&fakeDynamoClient{}, "test")
	if err := s.InitStore(context.Background()); err != nil {
		t.Fatalf("InitStore: %v", err)
	}
	if err := s.DisconnectStore(context.Background()); err != nil {
		t.Fatalf("DisconnectStore: %v", err)
	}
}

func TestDynamoDBStore_CreateUser_MarshalsIDAndEmailAsStrings(t *testing.T) {
	var captured map[string]types.AttributeValue
	fake := &fakeDynamoClient{
		putItemFn: func(ctx context.Context, in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			captured = in.Item
			if *in.TableName != "test-users" {
				t.Errorf("expected table test-users, got %s", *in.TableName)
			}
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	u := &models.User{Email: "alice@example.com", PasswordsHash: []byte("hash")}
	if err := s.CreateUser(context.Background(), u); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if u.ID.IsZero() {
		t.Fatal("expected CreateUser to assign an ObjectID")
	}

	// Assert the id/email are stored as exact scalar strings, not the
	// numeric-list encoding attributevalue would produce for a raw
	// primitive.ObjectID or a struct without dynamodbav tags.
	idAttr, ok := captured["id"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("expected id to be stored as S, got %T", captured["id"])
	}
	if idAttr.Value != u.ID.Hex() {
		t.Errorf("expected stored id %q, got %q", u.ID.Hex(), idAttr.Value)
	}

	emailAttr, ok := captured["email"].(*types.AttributeValueMemberS)
	if !ok {
		t.Fatalf("expected email to be stored as S, got %T", captured["email"])
	}
	if emailAttr.Value != "alice@example.com" {
		t.Errorf("expected stored email %q, got %q", "alice@example.com", emailAttr.Value)
	}
}

func TestDynamoDBStore_FindUserByEmail_NotFoundReturnsNilNil(t *testing.T) {
	fake := &fakeDynamoClient{
		queryFn: func(ctx context.Context, in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			if *in.IndexName != usersEmailIndex {
				t.Errorf("expected index %s, got %s", usersEmailIndex, *in.IndexName)
			}
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	u, err := s.FindUserByEmail(context.Background(), "missing@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if u != nil {
		t.Fatalf("expected nil user, got %+v", u)
	}
}

func TestDynamoDBStore_FindUserByEmail_Found(t *testing.T) {
	id := primitive.NewObjectID()
	createdAt := time.Now().UTC().Truncate(time.Second)
	rec := userRecord{ID: id.Hex(), Email: "bob@example.com", CreatedAt: createdAt}
	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}

	fake := &fakeDynamoClient{
		queryFn: func(ctx context.Context, in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{item}}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	u, err := s.FindUserByEmail(context.Background(), "bob@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.ID != id {
		t.Errorf("expected id %s, got %s", id.Hex(), u.ID.Hex())
	}
	if u.Email != "bob@example.com" {
		t.Errorf("expected email bob@example.com, got %s", u.Email)
	}
}

func TestDynamoDBStore_FindUserByEmail_QueryErrorPropagates(t *testing.T) {
	fake := &fakeDynamoClient{
		queryFn: func(ctx context.Context, in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			return nil, errors.New("boom")
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	_, err := s.FindUserByEmail(context.Background(), "x@example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDynamoDBStore_GetDriveAccountByID_NotFoundReturnsError(t *testing.T) {
	fake := &fakeDynamoClient{
		getItemFn: func(ctx context.Context, in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	acct, err := s.GetDriveAccountByID(context.Background(), primitive.NewObjectID())
	// This is the key not-found-semantics divergence from GetUploadSession /
	// FindUserByEmail: Mongo's GetDriveAccountByID does not special-case
	// ErrNoDocuments, so DynamoDB must synthesize an error here too rather
	// than returning (nil, nil).
	if err == nil {
		t.Fatal("expected error for missing drive account, got nil")
	}
	if acct != nil {
		t.Fatalf("expected nil account, got %+v", acct)
	}
}

func TestDynamoDBStore_GetUploadSession_NotFoundReturnsNilNil(t *testing.T) {
	fake := &fakeDynamoClient{
		getItemFn: func(ctx context.Context, in *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
			return &dynamodb.GetItemOutput{Item: nil}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	session, err := s.GetUploadSession(context.Background(), primitive.NewObjectID())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if session != nil {
		t.Fatalf("expected nil session, got %+v", session)
	}
}

func TestDynamoDBStore_CreateUploadSession_GeneratesIDWhenZero(t *testing.T) {
	var captured map[string]types.AttributeValue
	fake := &fakeDynamoClient{
		putItemFn: func(ctx context.Context, in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			captured = in.Item
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	session := &models.UploadSession{
		UserID:           primitive.NewObjectID(),
		OriginalFilename: "foo.txt",
		Status:           "uploading",
	}
	if err := s.CreateUploadSession(context.Background(), session); err != nil {
		t.Fatalf("CreateUploadSession: %v", err)
	}
	if session.ID.IsZero() {
		t.Fatal("expected CreateUploadSession to assign an ObjectID when zero")
	}
	idAttr, ok := captured["id"].(*types.AttributeValueMemberS)
	if !ok || idAttr.Value != session.ID.Hex() {
		t.Fatalf("expected stored id to match session.ID.Hex(), got %+v", captured["id"])
	}
}

func TestDynamoDBStore_CreateUploadSession_PreservesExistingID(t *testing.T) {
	existingID := primitive.NewObjectID()
	fake := &fakeDynamoClient{
		putItemFn: func(ctx context.Context, in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	session := &models.UploadSession{ID: existingID, UserID: primitive.NewObjectID()}
	if err := s.CreateUploadSession(context.Background(), session); err != nil {
		t.Fatalf("CreateUploadSession: %v", err)
	}
	if session.ID != existingID {
		t.Fatalf("expected ID to be preserved, got %s want %s", session.ID.Hex(), existingID.Hex())
	}
}

func TestDynamoDBStore_FindAndDeleteState_NotFoundReturnsNilNil(t *testing.T) {
	fake := &fakeDynamoClient{
		deleteItemFn: func(ctx context.Context, in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
			if in.ReturnValues != types.ReturnValueAllOld {
				t.Errorf("expected ReturnValues=ALL_OLD, got %s", in.ReturnValues)
			}
			return &dynamodb.DeleteItemOutput{Attributes: nil}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	state, err := s.FindAndDeleteState(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state, got %+v", state)
	}
}

func TestDynamoDBStore_FindAndDeleteState_Found(t *testing.T) {
	userID := primitive.NewObjectID()
	rec := oauthStateRecord{State: "abc123", UserID: userID.Hex(), Provider: "google", CreatedAt: time.Now().UTC()}
	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	fake := &fakeDynamoClient{
		deleteItemFn: func(ctx context.Context, in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
			return &dynamodb.DeleteItemOutput{Attributes: item}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	state, err := s.FindAndDeleteState(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.UserID != userID {
		t.Errorf("expected user id %s, got %s", userID.Hex(), state.UserID.Hex())
	}
	if state.Provider != "google" {
		t.Errorf("expected provider google, got %s", state.Provider)
	}
}

func TestDynamoDBStore_CountActiveUserSessions(t *testing.T) {
	fake := &fakeDynamoClient{
		queryFn: func(ctx context.Context, in *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
			if *in.IndexName != uploadSessionsUserIDIndex {
				t.Errorf("expected index %s, got %s", uploadSessionsUserIDIndex, *in.IndexName)
			}
			return &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{{}, {}}}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	count, err := s.CountActiveUserSessions(context.Background(), primitive.NewObjectID())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestDynamoDBStore_DeleteUploadSession_ErrorPropagates(t *testing.T) {
	fake := &fakeDynamoClient{
		deleteItemFn: func(ctx context.Context, in *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
			return nil, errors.New("dynamo unavailable")
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	err := s.DeleteUploadSession(context.Background(), primitive.NewObjectID())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDynamoDBStore_UpdateSessionStatus_IncludesErrorMessageOnlyWhenPresent(t *testing.T) {
	var captured *dynamodb.UpdateItemInput
	fake := &fakeDynamoClient{
		updateItemFn: func(ctx context.Context, in *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
			captured = in
			return &dynamodb.UpdateItemOutput{}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	if err := s.UpdateSessionStatus(context.Background(), primitive.NewObjectID(), "failed", 50, "disk full"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := captured.ExpressionAttributeValues[":error_message"]; !ok {
		t.Error("expected :error_message to be set when errorMsg is non-empty")
	}

	if err := s.UpdateSessionStatus(context.Background(), primitive.NewObjectID(), "processing", 10, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := captured.ExpressionAttributeValues[":error_message"]; ok {
		t.Error("expected :error_message to be absent when errorMsg is empty")
	}
}

func TestDynamoDBStore_GetExpiredSessions_UnmarshalsItems(t *testing.T) {
	id := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	rec := uploadSessionRecord{
		ID:        id.Hex(),
		UserID:    userID.Hex(),
		Status:    "uploading",
		ExpiresAt: time.Now().Add(-time.Hour),
		CreatedAt: time.Now(),
	}
	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	fake := &fakeDynamoClient{
		scanFn: func(ctx context.Context, in *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
			return &dynamodb.ScanOutput{Items: []map[string]types.AttributeValue{item}}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	sessions, err := s.GetExpiredSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != id {
		t.Errorf("expected id %s, got %s", id.Hex(), sessions[0].ID.Hex())
	}
}

func TestDynamoDBStore_AddDriveAccountToUser_StoresUserIDForGSI(t *testing.T) {
	var captured map[string]types.AttributeValue
	fake := &fakeDynamoClient{
		putItemFn: func(ctx context.Context, in *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
			captured = in.Item
			return &dynamodb.PutItemOutput{}, nil
		},
	}
	s := newDynamoDBStoreWithClient(fake, "test")

	userID := primitive.NewObjectID()
	acct := models.DriveAccount{Provider: "google", DisplayName: "Work"}
	if err := s.AddDriveAccountToUser(context.Background(), userID, acct); err != nil {
		t.Fatalf("AddDriveAccountToUser: %v", err)
	}

	userIDAttr, ok := captured["user_id"].(*types.AttributeValueMemberS)
	if !ok || userIDAttr.Value != userID.Hex() {
		t.Fatalf("expected stored user_id %q, got %+v", userID.Hex(), captured["user_id"])
	}
}
