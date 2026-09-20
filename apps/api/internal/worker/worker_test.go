package worker

import (
	"SE/internal/events"
	"SE/internal/integrity"
	"SE/internal/metadatastore"
	"SE/internal/models"
	"SE/internal/queue"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type fakeStore struct {
	metadatastore.MetadataStore
	session   *models.UploadSession
	shards    []models.ShardRecord
	completed bool
	status    string
}

func (f *fakeStore) GetUploadSession(context.Context, primitive.ObjectID) (*models.UploadSession, error) {
	return f.session, nil
}
func (f *fakeStore) GetShardMetadata(context.Context, string) ([]models.ShardRecord, error) {
	return f.shards, nil
}
func (f *fakeStore) UpdateShardStatus(_ context.Context, _ string, shardID int, status string) error {
	for i := range f.shards {
		if f.shards[i].ShardID == shardID {
			f.shards[i].Status = status
		}
	}
	return nil
}
func (f *fakeStore) CompleteSession(context.Context, primitive.ObjectID, *time.Time) error {
	f.completed = true
	return nil
}
func (f *fakeStore) UpdateSessionStatus(_ context.Context, _ primitive.ObjectID, status string, _ float64, _ string) error {
	f.status = status
	return nil
}

type fakeStorage struct{ uploads int }

func (f *fakeStorage) UploadChunk(_ context.Context, target, _, filename string) (string, error) {
	f.uploads++
	return target + "/" + filename, nil
}
func (f *fakeStorage) DeleteChunk(context.Context, string, string) error { return nil }
func (f *fakeStorage) GetSpace(context.Context, string) ([]models.DriveSpaceInfo, error) {
	return nil, nil
}

func setup(t *testing.T, pendingShards []int) (*Worker, *fakeStore, string, string) {
	t.Helper()
	dir := t.TempDir()
	chunk := filepath.Join(dir, "chunk_002.2xpfm")
	os.WriteFile(chunk, []byte("x"), 0o644)
	keyPath := filepath.Join(dir, "f.2xpfm.key")
	kf := models.KeyFile{Chunks: []models.ChunkMetadata{
		{ChunkID: 1, DriveFileID: "sess/chunk_001.2xpfm"},
		{ChunkID: 2, DriveFileID: ""},
		{ChunkID: 3, DriveFileID: ""},
	}}
	b, _ := json.Marshal(kf)
	os.WriteFile(keyPath, b, 0o644)

	sid := primitive.NewObjectID()
	st := &fakeStore{session: &models.UploadSession{ID: sid, KeyFilePath: keyPath}}
	for _, id := range []int{1, 2, 3} {
		status := integrity.StatusVerified
		for _, p := range pendingShards {
			if p == id {
				status = integrity.StatusPending
			}
		}
		st.shards = append(st.shards, models.ShardRecord{ShardID: id, Status: status})
	}
	w := &Worker{Queue: queue.NoopQueue{}, Store: st, Storage: &fakeStorage{}, Events: events.NoopEmitter{}}
	return w, st, chunk, keyPath
}

func TestProcess_LastPendingShardCompletesSession(t *testing.T) {
	w, st, chunk, keyPath := setup(t, []int{2})
	job := queue.RetryJob{SessionID: st.session.ID.Hex(), ChunkID: 2, ChunkPath: chunk, Target: "sess", Filename: "chunk_002.2xpfm"}
	if err := w.Process(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	var kf models.KeyFile
	b, _ := os.ReadFile(keyPath)
	json.Unmarshal(b, &kf)
	if kf.Chunks[1].DriveFileID != "sess/chunk_002.2xpfm" {
		t.Fatalf("key file not patched: %+v", kf.Chunks[1])
	}
	if st.shards[1].Status != integrity.StatusVerified || !st.completed || st.status != "complete" {
		t.Fatalf("session not completed: %+v completed=%v status=%s", st.shards, st.completed, st.status)
	}
	if _, err := os.Stat(chunk); !os.IsNotExist(err) {
		t.Fatal("chunk file should be removed after upload")
	}
}

func TestProcess_OtherShardsStillPendingKeepsProcessing(t *testing.T) {
	w, st, chunk, _ := setup(t, []int{2, 3})
	job := queue.RetryJob{SessionID: st.session.ID.Hex(), ChunkID: 2, ChunkPath: chunk, Target: "sess", Filename: "chunk_002.2xpfm"}
	if err := w.Process(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	if st.completed || st.status == "complete" {
		t.Fatal("session must stay processing while shard 3 is pending")
	}
}

func TestProcess_UnknownChunkFails(t *testing.T) {
	w, st, chunk, _ := setup(t, []int{2})
	job := queue.RetryJob{SessionID: st.session.ID.Hex(), ChunkID: 9, ChunkPath: chunk, Target: "sess", Filename: "chunk_009.2xpfm"}
	if err := w.Process(context.Background(), job); err == nil {
		t.Fatal("expected error for chunk not in key file")
	}
}
