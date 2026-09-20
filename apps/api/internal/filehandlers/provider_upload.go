package filehandlers

import (
	"SE/internal/integrity"
	"SE/internal/models"
	"SE/internal/queue"
	"context"
	"fmt"
	"log"
)

// uploadChunksViaProvider uploads every chunk through activeStorageProvider
// using the session ID as the target (an S3 key prefix). It mirrors
// drivemanager.UploadChunksToDrivers, with one difference: a failed chunk
// is returned as a RetryJob instead of aborting the session. Its
// ChunkMetadata comes back with an empty DriveFileID (location) and the
// caller keeps its file on disk and enqueues the job only once the key
// file exists (cmd/shardworker refuses jobs for sessions without one).
//
// Only when no queue is configured (NoopQueue) does this fall back to
// Drive-style fail-fast: best-effort delete of already-uploaded chunks and
// an error.
func uploadChunksViaProvider(ctx context.Context, sessionID string, chunkPaths []string, plan []models.ChunkPlan, progressCallback func(int, int)) ([]models.ChunkMetadata, []queue.RetryJob, error) {
	if len(chunkPaths) != len(plan) {
		return nil, nil, fmt.Errorf("mismatch: %d chunk files but %d planned chunks", len(chunkPaths), len(plan))
	}
	_, noQueue := retryQueue.(queue.NoopQueue)

	chunkMetadata := make([]models.ChunkMetadata, 0, len(plan))
	var pendingJobs []queue.RetryJob

	for i, chunkPath := range chunkPaths {
		if progressCallback != nil {
			progressCallback(i+1, len(chunkPaths))
		}

		chunk := plan[i]
		filename := fmt.Sprintf("chunk_%03d.2xpfm", chunk.ChunkID)

		checksum, err := integrity.ChecksumFile(chunkPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to calculate checksum for chunk %d: %w", chunk.ChunkID, err)
		}

		locationID, err := activeStorageProvider.UploadChunk(ctx, sessionID, chunkPath, filename)
		if err != nil {
			if noQueue {
				for _, m := range chunkMetadata {
					if m.DriveFileID != "" {
						activeStorageProvider.DeleteChunk(ctx, sessionID, m.DriveFileID) // best effort
					}
				}
				return nil, nil, fmt.Errorf("failed to upload chunk %d: %w (%v)", chunk.ChunkID, err, queue.ErrNotConfigured)
			}
			log.Printf("Chunk %d upload failed for session %s, deferring to retry queue: %v", chunk.ChunkID, sessionID, err)
			pendingJobs = append(pendingJobs, queue.RetryJob{
				SessionID: sessionID,
				ChunkID:   chunk.ChunkID,
				ChunkPath: chunkPath,
				Target:    sessionID,
				Filename:  filename,
			})
		}

		chunkMetadata = append(chunkMetadata, models.ChunkMetadata{
			ChunkID:        chunk.ChunkID,
			DriveAccountID: sessionID,
			DriveFileID:    locationID,
			Filename:       filename,
			StartOffset:    chunk.StartOffset,
			EndOffset:      chunk.EndOffset,
			Size:           chunk.Size,
			Checksum:       checksum,
		})
	}

	return chunkMetadata, pendingJobs, nil
}
