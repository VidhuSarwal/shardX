package metadatastore

import (
	"SE/internal/models"
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestDynamoDBStore_LocalStack exercises DynamoDBStore against a real
// DynamoDB API surface provided by LocalStack (localhost:4566). It is
// skipped unless LOCALSTACK=1 is set, since CI/dev environments without
// Docker (or without `docker compose -f docker-compose.localstack.yml up`
// already running) have nothing listening on that port.
//
// To run locally:
//
//	docker compose -f docker-compose.localstack.yml up -d
//	LOCALSTACK=1 go test ./internal/metadatastore/... -run LocalStack -v
func TestDynamoDBStore_LocalStack(t *testing.T) {
	if os.Getenv("LOCALSTACK") != "1" {
		t.Skip("set LOCALSTACK=1 (with LocalStack running on :4566) to run this test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String("http://localhost:4566")
	})

	prefix := "shardx-test"
	tables := map[string]struct {
		pk  string
		gsi string
		gpk string
	}{
		prefix + "-users":           {pk: "id", gsi: usersEmailIndex, gpk: "email"},
		prefix + "-oauth-states":    {pk: "state"},
		prefix + "-upload-sessions": {pk: "id", gsi: uploadSessionsUserIDIndex, gpk: "user_id"},
		prefix + "-drive-accounts":  {pk: "id", gsi: driveAccountsUserIDIndex, gpk: "user_id"},
	}

	for name, schema := range tables {
		createTestTable(ctx, t, client, name, schema.pk, schema.gsi, schema.gpk)
	}

	shardMetadataTable := prefix + "-shard-metadata"
	createCompositeKeyTestTable(ctx, t, client, shardMetadataTable, "file_id", "shard_id")

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for name := range tables {
			_, _ = client.DeleteTable(cleanupCtx, &dynamodb.DeleteTableInput{TableName: aws.String(name)})
		}
		_, _ = client.DeleteTable(cleanupCtx, &dynamodb.DeleteTableInput{TableName: aws.String(shardMetadataTable)})
	})

	store := newDynamoDBStoreWithClient(client, prefix)

	t.Run("CreateUser and FindUserByEmail", func(t *testing.T) {
		u := &models.User{Email: "localstack-user@example.com", PasswordsHash: []byte("hash")}
		if err := store.CreateUser(ctx, u); err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		if u.ID.IsZero() {
			t.Fatal("expected assigned ID")
		}

		found, err := store.FindUserByEmail(ctx, "localstack-user@example.com")
		if err != nil {
			t.Fatalf("FindUserByEmail: %v", err)
		}
		if found == nil {
			t.Fatal("expected to find user")
		}
		if found.ID != u.ID {
			t.Errorf("expected id %s, got %s", u.ID.Hex(), found.ID.Hex())
		}

		missing, err := store.FindUserByEmail(ctx, "nobody@example.com")
		if err != nil {
			t.Fatalf("FindUserByEmail (missing): %v", err)
		}
		if missing != nil {
			t.Errorf("expected nil for missing user, got %+v", missing)
		}
	})

	t.Run("InsertOAuthState and FindAndDeleteState", func(t *testing.T) {
		st := &models.OAuthState{State: "state-xyz", UserID: primitive.NewObjectID(), Provider: "google"}
		if err := store.InsertOAuthState(ctx, st); err != nil {
			t.Fatalf("InsertOAuthState: %v", err)
		}

		got, err := store.FindAndDeleteState(ctx, "state-xyz")
		if err != nil {
			t.Fatalf("FindAndDeleteState: %v", err)
		}
		if got == nil {
			t.Fatal("expected to find and delete state")
		}
		if got.UserID != st.UserID {
			t.Errorf("expected user id %s, got %s", st.UserID.Hex(), got.UserID.Hex())
		}

		// Second call should now find nothing (already deleted).
		gone, err := store.FindAndDeleteState(ctx, "state-xyz")
		if err != nil {
			t.Fatalf("FindAndDeleteState (second): %v", err)
		}
		if gone != nil {
			t.Errorf("expected nil after deletion, got %+v", gone)
		}
	})

	t.Run("Drive accounts", func(t *testing.T) {
		userID := primitive.NewObjectID()
		acct := models.DriveAccount{Provider: "google", DisplayName: "Personal", EncryptedToken: []byte("tok")}
		if err := store.AddDriveAccountToUser(ctx, userID, acct); err != nil {
			t.Fatalf("AddDriveAccountToUser: %v", err)
		}

		accounts, err := store.ListUserDriveAccounts(ctx, userID)
		if err != nil {
			t.Fatalf("ListUserDriveAccounts: %v", err)
		}
		if len(accounts) != 1 {
			t.Fatalf("expected 1 account, got %d", len(accounts))
		}

		fetched, err := store.GetDriveAccountByID(ctx, accounts[0].ID)
		if err != nil {
			t.Fatalf("GetDriveAccountByID: %v", err)
		}
		if fetched.DisplayName != "Personal" {
			t.Errorf("expected display name Personal, got %s", fetched.DisplayName)
		}

		_, err = store.GetDriveAccountByID(ctx, primitive.NewObjectID())
		if err == nil {
			t.Error("expected error for nonexistent drive account")
		}
	})

	t.Run("Upload session lifecycle", func(t *testing.T) {
		session := &models.UploadSession{
			UserID:           primitive.NewObjectID(),
			OriginalFilename: "file.bin",
			TotalSize:        1000,
			Status:           "uploading",
			CreatedAt:        time.Now().UTC(),
			ExpiresAt:        time.Now().Add(-time.Minute), // already expired for GetExpiredSessions test
		}
		if err := store.CreateUploadSession(ctx, session); err != nil {
			t.Fatalf("CreateUploadSession: %v", err)
		}

		got, err := store.GetUploadSession(ctx, session.ID)
		if err != nil || got == nil {
			t.Fatalf("GetUploadSession: %v, got=%+v", err, got)
		}

		if err := store.UpdateSessionUploadProgress(ctx, session.ID, 500); err != nil {
			t.Fatalf("UpdateSessionUploadProgress: %v", err)
		}
		if err := store.UpdateSessionStatus(ctx, session.ID, "processing", 42.0, ""); err != nil {
			t.Fatalf("UpdateSessionStatus: %v", err)
		}
		if err := store.UpdateSessionKeyFile(ctx, session.ID, "/tmp/key.json"); err != nil {
			t.Fatalf("UpdateSessionKeyFile: %v", err)
		}

		count, err := store.CountActiveUserSessions(ctx, session.UserID)
		if err != nil {
			t.Fatalf("CountActiveUserSessions: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 active session, got %d", count)
		}

		expired, err := store.GetExpiredSessions(ctx)
		if err != nil {
			t.Fatalf("GetExpiredSessions: %v", err)
		}
		found := false
		for _, s := range expired {
			if s.ID == session.ID {
				found = true
			}
		}
		if !found {
			t.Error("expected session to appear in GetExpiredSessions")
		}

		now := time.Now().UTC()
		if err := store.CompleteSession(ctx, session.ID, &now); err != nil {
			t.Fatalf("CompleteSession: %v", err)
		}

		if err := store.DeleteUploadSession(ctx, session.ID); err != nil {
			t.Fatalf("DeleteUploadSession: %v", err)
		}
		afterDelete, err := store.GetUploadSession(ctx, session.ID)
		if err != nil {
			t.Fatalf("GetUploadSession (after delete): %v", err)
		}
		if afterDelete != nil {
			t.Error("expected nil session after delete")
		}
	})

	t.Run("Shard metadata", func(t *testing.T) {
		sessionID := primitive.NewObjectID().Hex()
		shards := []models.ShardRecord{
			{ShardID: 0, SHA256: "aaa", Size: 100, Bucket: "acct-1", Status: "verified", CreatedAt: time.Now().UTC()},
			{ShardID: 1, SHA256: "bbb", Size: 200, Bucket: "acct-2", Status: "verified", CreatedAt: time.Now().UTC()},
		}
		if err := store.SaveShardMetadata(ctx, sessionID, shards); err != nil {
			t.Fatalf("SaveShardMetadata: %v", err)
		}

		got, err := store.GetShardMetadata(ctx, sessionID)
		if err != nil {
			t.Fatalf("GetShardMetadata: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 shard records, got %d", len(got))
		}

		otherSession := primitive.NewObjectID().Hex()
		none, err := store.GetShardMetadata(ctx, otherSession)
		if err != nil {
			t.Fatalf("GetShardMetadata (other session): %v", err)
		}
		if len(none) != 0 {
			t.Errorf("expected 0 shard records for unrelated session, got %d", len(none))
		}
	})
}

// createCompositeKeyTestTable creates a LocalStack DynamoDB table with a
// composite primary key (partition key + sort key), used by the
// shard-metadata table which needs one row per shard per file.
func createCompositeKeyTestTable(ctx context.Context, t *testing.T, client *dynamodb.Client, tableName, pk, sk string) {
	t.Helper()

	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String(pk), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String(sk), AttributeType: types.ScalarAttributeTypeN},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(pk), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String(sk), KeyType: types.KeyTypeRange},
		},
		BillingMode: types.BillingModePayPerRequest,
	}

	if _, err := client.CreateTable(ctx, input); err != nil {
		t.Fatalf("create table %s: %v", tableName, err)
	}

	waiter := dynamodb.NewTableExistsWaiter(client)
	if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tableName)}, 30*time.Second); err != nil {
		t.Fatalf("wait for table %s: %v", tableName, err)
	}
}

func createTestTable(ctx context.Context, t *testing.T, client *dynamodb.Client, tableName, pk, gsiName, gsiPK string) {
	t.Helper()

	attrDefs := []types.AttributeDefinition{
		{AttributeName: aws.String(pk), AttributeType: types.ScalarAttributeTypeS},
	}
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String(pk), KeyType: types.KeyTypeHash},
		},
		BillingMode: types.BillingModePayPerRequest,
	}

	if gsiName != "" {
		attrDefs = append(attrDefs, types.AttributeDefinition{
			AttributeName: aws.String(gsiPK), AttributeType: types.ScalarAttributeTypeS,
		})
		input.GlobalSecondaryIndexes = []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String(gsiName),
				KeySchema: []types.KeySchemaElement{
					{AttributeName: aws.String(gsiPK), KeyType: types.KeyTypeHash},
				},
				Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
			},
		}
	}
	input.AttributeDefinitions = attrDefs

	if _, err := client.CreateTable(ctx, input); err != nil {
		t.Fatalf("create table %s: %v", tableName, err)
	}

	waiter := dynamodb.NewTableExistsWaiter(client)
	if err := waiter.Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tableName)}, 30*time.Second); err != nil {
		t.Fatalf("wait for table %s: %v", tableName, err)
	}
}
