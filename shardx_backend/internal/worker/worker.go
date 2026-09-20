// Package worker consumes shard-upload RetryJobs from the queue (S3 mode).
// For each job it re-uploads the chunk, patches the chunk's location into
// the session's key file, marks the shard "verified", and -- once no shard
// is still pending -- marks the session complete.
//
// ponytail: the worker reads the chunk file from the API's UPLOAD_TEMP_DIR,
// so it must run on the same host / shared volume as cmd/server. Staging
// failed chunks somewhere durable is the upgrade path if they ever split.
package worker

import (
	"SE/internal/events"
	"SE/internal/integrity"
	"SE/internal/metadatastore"
	"SE/internal/models"
	"SE/internal/queue"
	"SE/internal/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Worker struct {
	Queue   queue.Queue
	Store   metadatastore.MetadataStore
	Storage storage.StorageProvider
	Events  events.Emitter
}

// Run polls until ctx is cancelled. A job that fails is left on the queue
// (not deleted) so SQS redelivers it and eventually dead-letters it.
func (w *Worker) Run(ctx context.Context) {
	for ctx.Err() == nil {
		msgs, err := w.Queue.Receive(ctx, 10, 20)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("worker: receive failed: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for _, m := range msgs {
			if err := w.Process(ctx, m.Job); err != nil {
				log.Printf("worker: job session=%s chunk=%d failed (will redeliver): %v", m.Job.SessionID, m.Job.ChunkID, err)
				continue
			}
			if err := w.Queue.Delete(ctx, m.ReceiptHandle); err != nil {
				log.Printf("worker: delete message failed: %v", err)
			}
		}
	}
}

// Process handles one retry job end to end. It is idempotent: re-running
// a job after a partial failure re-uploads (S3 PutObject overwrites) and
// re-applies the same metadata updates.
func (w *Worker) Process(ctx context.Context, job queue.RetryJob) error {
	sessionID, err := primitive.ObjectIDFromHex(job.SessionID)
	if err != nil {
		return fmt.Errorf("invalid session id %q: %w", job.SessionID, err)
	}
	session, err := w.Store.GetUploadSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if session == nil {
		return errors.New("session not found")
	}
	if session.KeyFilePath == "" {
		return errors.New("session has no key file yet")
	}

	locationID, err := w.Storage.UploadChunk(ctx, job.Target, job.ChunkPath, job.Filename)
	if err != nil {
		return fmt.Errorf("upload chunk: %w", err)
	}

	if err := patchKeyFile(session.KeyFilePath, job.ChunkID, locationID); err != nil {
		return fmt.Errorf("patch key file: %w", err)
	}
	if err := w.Store.UpdateShardStatus(ctx, job.SessionID, job.ChunkID, integrity.StatusVerified); err != nil {
		return fmt.Errorf("mark shard verified: %w", err)
	}
	os.Remove(job.ChunkPath) // best effort; the chunk is now in S3

	w.Events.Emit(ctx, events.Event{Type: events.ShardVerified, Detail: map[string]any{
		"session_id": job.SessionID, "chunk_id": job.ChunkID, "retried": true,
	}})

	shards, err := w.Store.GetShardMetadata(ctx, job.SessionID)
	if err != nil {
		return fmt.Errorf("load shard metadata: %w", err)
	}
	for _, s := range shards {
		if s.Status == integrity.StatusPending {
			return nil // more retries outstanding
		}
	}
	now := time.Now()
	if err := w.Store.CompleteSession(ctx, sessionID, &now); err != nil {
		return fmt.Errorf("complete session: %w", err)
	}
	if err := w.Store.UpdateSessionStatus(ctx, sessionID, "complete", 100, ""); err != nil {
		return fmt.Errorf("update session status: %w", err)
	}
	log.Printf("worker: session %s complete after retries", job.SessionID)
	return nil
}

// patchKeyFile sets drive_file_id (the storage location) for one chunk in
// the key file the API already wrote with an empty location.
func patchKeyFile(path string, chunkID int, locationID string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var kf models.KeyFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return err
	}
	found := false
	for i := range kf.Chunks {
		if kf.Chunks[i].ChunkID == chunkID {
			kf.Chunks[i].DriveFileID = locationID
			found = true
		}
	}
	if !found {
		return fmt.Errorf("chunk %d not in key file", chunkID)
	}
	out, err := json.MarshalIndent(kf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o600)
}
