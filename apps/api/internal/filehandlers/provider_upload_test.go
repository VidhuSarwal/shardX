package filehandlers

import (
	"SE/internal/integrity"
	"SE/internal/models"
	"SE/internal/queue"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeProvider fails UploadChunk for the chunk filenames listed in failOn.
type fakeProvider struct {
	failOn   map[string]bool
	uploaded []string
	deleted  []string
}

func (f *fakeProvider) UploadChunk(_ context.Context, target, _, filename string) (string, error) {
	if f.failOn[filename] {
		return "", errors.New("s3 down")
	}
	f.uploaded = append(f.uploaded, filename)
	return target + "/" + filename, nil
}
func (f *fakeProvider) DeleteChunk(_ context.Context, _, locationID string) error {
	f.deleted = append(f.deleted, locationID)
	return nil
}
func (f *fakeProvider) GetSpace(context.Context, string) ([]models.DriveSpaceInfo, error) {
	return nil, nil
}
func (f *fakeProvider) Bucket() string { return "bkt" }
func (f *fakeProvider) Region() string { return "us-east-1" }

type fakeQueue struct {
	queue.NoopQueue
	jobs []queue.RetryJob
}

func (f *fakeQueue) Enqueue(_ context.Context, j queue.RetryJob) error {
	f.jobs = append(f.jobs, j)
	return nil
}

func writeChunks(t *testing.T, n int) ([]string, []models.ChunkPlan) {
	t.Helper()
	dir := t.TempDir()
	var paths []string
	var plan []models.ChunkPlan
	for i := 1; i <= n; i++ {
		p := filepath.Join(dir, "c"+string(rune('0'+i)))
		if err := os.WriteFile(p, []byte{byte(i)}, 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
		plan = append(plan, models.ChunkPlan{ChunkID: i, DriveAccountID: primitive.NilObjectID, Size: 1})
	}
	return paths, plan
}

func TestUploadChunksViaProvider_FailedChunkIsQueuedNotFatal(t *testing.T) {
	origP, origQ := activeStorageProvider, retryQueue
	defer func() { activeStorageProvider, retryQueue = origP, origQ }()

	fp := &fakeProvider{failOn: map[string]bool{"chunk_002.2xpfm": true}}
	fq := &fakeQueue{}
	activeStorageProvider, retryQueue = fp, fq

	paths, plan := writeChunks(t, 3)
	meta, pending, err := uploadChunksViaProvider(context.Background(), "sess", paths, plan, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(meta) != 3 || meta[1].DriveFileID != "" || meta[0].DriveFileID != "sess/chunk_001.2xpfm" {
		t.Fatalf("metadata = %+v", meta)
	}
	if meta[1].Checksum == "" {
		t.Fatal("checksum must be computed even for queued chunk")
	}
	if len(pending) != 1 || pending[0] != paths[1] {
		t.Fatalf("pending = %v", pending)
	}
	if len(fq.jobs) != 1 || fq.jobs[0].ChunkID != 2 || fq.jobs[0].ChunkPath != paths[1] || fq.jobs[0].Target != "sess" {
		t.Fatalf("jobs = %+v", fq.jobs)
	}

	recs := chunkMetadataToShardRecords(meta)
	if recs[1].Status != integrity.StatusPending || recs[0].Status != integrity.StatusVerified {
		t.Fatalf("statuses = %s %s", recs[0].Status, recs[1].Status)
	}
	if recs[0].Bucket != "bkt" || recs[0].Region != "us-east-1" {
		t.Fatalf("placement = %s/%s", recs[0].Bucket, recs[0].Region)
	}
}

func TestUploadChunksViaProvider_NoQueueFallsBackToFailFast(t *testing.T) {
	origP, origQ := activeStorageProvider, retryQueue
	defer func() { activeStorageProvider, retryQueue = origP, origQ }()

	fp := &fakeProvider{failOn: map[string]bool{"chunk_002.2xpfm": true}}
	activeStorageProvider, retryQueue = fp, queue.NoopQueue{}

	paths, plan := writeChunks(t, 3)
	_, _, err := uploadChunksViaProvider(context.Background(), "sess", paths, plan, nil)
	if err == nil {
		t.Fatal("expected fail-fast error")
	}
	if len(fp.deleted) != 1 || fp.deleted[0] != "sess/chunk_001.2xpfm" {
		t.Fatalf("expected cleanup of chunk 1, got %v", fp.deleted)
	}
}
