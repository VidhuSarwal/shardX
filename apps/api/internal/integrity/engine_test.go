package integrity

import (
	"SE/internal/metadatastore"
	"SE/internal/models"
	"context"
	"errors"
	"testing"
	"time"
)

// fakeStore is a minimal MetadataStore stand-in for testing VerifyShardHealth
// without a live database. It embeds the interface so only the methods
// actually exercised need overriding; any other method call panics loudly
// (nil pointer deref) rather than silently succeeding, which is fine since
// VerifyShardHealth only calls GetShardMetadata.
type fakeStore struct {
	metadatastore.MetadataStore
	shards    []models.ShardRecord
	shardsErr error
}

func (f *fakeStore) GetShardMetadata(ctx context.Context, sessionID string) ([]models.ShardRecord, error) {
	if f.shardsErr != nil {
		return nil, f.shardsErr
	}
	return f.shards, nil
}

func TestVerifyShardHealth_AllVerified(t *testing.T) {
	now := time.Now().UTC()
	store := &fakeStore{shards: []models.ShardRecord{
		{ShardID: 0, SHA256: "aaa", Size: 100, Status: StatusVerified, CreatedAt: now},
		{ShardID: 1, SHA256: "bbb", Size: 200, Status: StatusVerified, CreatedAt: now},
		{ShardID: 2, SHA256: "ccc", Size: 300, Status: StatusVerified, CreatedAt: now},
	}}

	report, err := VerifyShardHealth(context.Background(), "session-1", store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.FileID != "session-1" {
		t.Errorf("expected file_id session-1, got %s", report.FileID)
	}
	if report.ShardsTotal != 3 || report.ShardsAvailable != 3 {
		t.Errorf("expected 3/3 shards available, got %d/%d", report.ShardsAvailable, report.ShardsTotal)
	}
	if report.IntegrityChecksTotal != 3 || report.IntegrityChecksPassed != 3 {
		t.Errorf("expected 3/3 integrity checks passed, got %d/%d", report.IntegrityChecksPassed, report.IntegrityChecksTotal)
	}
	if report.CorruptionEvents != 0 {
		t.Errorf("expected 0 corruption events, got %d", report.CorruptionEvents)
	}
	if report.HealthPercentage != 100.0 {
		t.Errorf("expected 100%% health, got %v", report.HealthPercentage)
	}
}

func TestVerifyShardHealth_PartialCorruption(t *testing.T) {
	store := &fakeStore{shards: []models.ShardRecord{
		{ShardID: 0, SHA256: "aaa", Status: StatusVerified},
		{ShardID: 1, SHA256: "", Status: StatusVerified},      // missing checksum -> unhealthy
		{ShardID: 2, SHA256: "ccc", Status: "corrupted"},      // bad status -> unhealthy
		{ShardID: 3, SHA256: "ddd", Status: StatusVerified},
	}}

	report, err := VerifyShardHealth(context.Background(), "session-2", store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.ShardsTotal != 4 {
		t.Errorf("expected 4 total shards, got %d", report.ShardsTotal)
	}
	if report.ShardsAvailable != 2 {
		t.Errorf("expected 2 available shards, got %d", report.ShardsAvailable)
	}
	if report.CorruptionEvents != 2 {
		t.Errorf("expected 2 corruption events, got %d", report.CorruptionEvents)
	}
	if report.HealthPercentage != 50.0 {
		t.Errorf("expected 50%% health, got %v", report.HealthPercentage)
	}
}

func TestVerifyShardHealth_NoShards_ReportsZeroNotNaN(t *testing.T) {
	store := &fakeStore{shards: nil}

	report, err := VerifyShardHealth(context.Background(), "session-3", store, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.ShardsTotal != 0 || report.HealthPercentage != 0 {
		t.Errorf("expected 0 total shards and 0%% health, got total=%d health=%v", report.ShardsTotal, report.HealthPercentage)
	}
}

func TestVerifyShardHealth_StoreErrorPropagates(t *testing.T) {
	store := &fakeStore{shardsErr: errors.New("boom")}

	_, err := VerifyShardHealth(context.Background(), "session-4", store, nil)
	if err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestVerifyShardHealth_NilStoreErrors(t *testing.T) {
	_, err := VerifyShardHealth(context.Background(), "session-5", nil, nil)
	if err == nil {
		t.Fatal("expected error for nil store")
	}
}
