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
// is enqueued as a RetryJob instead of aborting the session. Its
// ChunkMetadata comes back with an empty DriveFileID (location) and its
// path is returned in pendingPaths so the caller keeps the file on disk.
//
// Only when the queue itself is unavailable (queue.ErrNotConfigured or a
// send failure) does this fall back to Drive-style fail-fast: best-effort
// delete of already-uploaded chunks and an error.
func uploadChunksViaProvider(ctx context.Context, sessionID string, chunkPaths []string, plan []models.ChunkPlan, progressCallback func(int, int)) ([]models.ChunkMetadata, []string, error) {
	if len(chunkPaths) != len(plan) {
		return nil, nil, fmt.Errorf("mismatch: %d chunk files but %d planned chunks", len(chunkPaths), len(plan))
	}

	chunkMetadata := make([]models.ChunkMetadata, 0, len(plan))
	var pendingPaths []string

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
			log.Printf("Chunk %d upload failed for session %s, enqueueing retry: %v", chunk.ChunkID, sessionID, err)
			qerr := retryQueue.Enqueue(ctx, queue.RetryJob{
				SessionID: sessionID,
				ChunkID:   chunk.ChunkID,
				ChunkPath: chunkPath,
				Target:    sessionID,
				Filename:  filename,
			})
			if qerr != nil {
				for _, m := range chunkMetadata {
					if m.DriveFileID != "" {
						activeStorageProvider.DeleteChunk(ctx, sessionID, m.DriveFileID) // best effort
					}
				}
				return nil, nil, fmt.Errorf("failed to upload chunk %d: %w (retry enqueue: %v)", chunk.ChunkID, err, qerr)
			}
			pendingPaths = append(pendingPaths, chunkPath)
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

	return chunkMetadata, pendingPaths, nil
}
