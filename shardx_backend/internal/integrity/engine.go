// Package integrity implements the Shard Health Score feature: aggregating
// per-shard integrity metadata (persisted at upload time by
// internal/metadatastore) into a health report for a given upload session.
//
// Scope of this pass (read this before assuming more than it does):
//
// VerifyShardHealth does a LIGHTWEIGHT check. It does NOT re-download and
// re-hash each shard from its storage backend. It reports whether each shard
// has a persisted integrity record on file and whether that record's status
// (captured at upload time, when the chunk's SHA256 was actually computed
// against the bytes that were uploaded -- see internal/drivemanager/uploader.go
// calculateFileChecksum) was "verified".
//
// This is the "was verified at upload time and is that record still on
// file" signal -- a real, honest feature -- not full re-verification. Full
// re-verification (re-fetching each shard's bytes and recomputing its
// SHA256, or at minimum confirming the object still exists at its storage
// location) is a deeper feature this pass does not implement, because the
// storage.StorageProvider interface (internal/storage/provider.go) does not
// currently expose an existence/HeadObject-equivalent method, and that
// interface is owned by other in-flight work and must not be changed here.
// A future pass could add e.g. `HeadChunk(ctx, target, locationID) error` to
// StorageProvider and use it here to detect shards that were verified at
// upload time but have since disappeared from storage.
//
// The storageProvider parameter is accepted (and threaded through) so that
// future re-verification logic has a natural place to live without another
// signature change; it is not used by the current lightweight check and may
// be nil.
package integrity

import (
	"SE/internal/metadatastore"
	"SE/internal/models"
	"SE/internal/storage"
	"context"
	"fmt"
)

// StatusVerified is the ShardRecord.Status value written at upload time once
// a chunk's checksum has been computed and the chunk successfully uploaded.
const StatusVerified = "verified"

// HealthReport is the JSON-shaped result of a shard health check, per the
// Integrity Engine product spec.
type HealthReport struct {
	FileID                string  `json:"file_id"`
	HealthPercentage      float64 `json:"health_percentage"`
	ShardsAvailable       int     `json:"shards_available"`
	ShardsTotal           int     `json:"shards_total"`
	IntegrityChecksPassed int     `json:"integrity_checks_passed"`
	IntegrityChecksTotal  int     `json:"integrity_checks_total"`
	CorruptionEvents      int     `json:"corruption_events"`
}

// VerifyShardHealth loads the persisted ShardRecords for sessionID and
// aggregates them into a HealthReport.
//
// storageProvider is currently unused (see package doc) but is accepted so
// a future deeper check can be added without another signature change. It
// may be nil.
func VerifyShardHealth(ctx context.Context, sessionID string, store metadatastore.MetadataStore, storageProvider storage.StorageProvider) (*HealthReport, error) {
	if store == nil {
		return nil, fmt.Errorf("metadata store is required")
	}

	shards, err := store.GetShardMetadata(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load shard metadata: %w", err)
	}

	report := &HealthReport{
		FileID:      sessionID,
		ShardsTotal: len(shards),
	}
	report.IntegrityChecksTotal = len(shards)

	for _, s := range shards {
		if isShardHealthy(s) {
			report.ShardsAvailable++
			report.IntegrityChecksPassed++
		} else {
			report.CorruptionEvents++
		}
	}

	if report.ShardsTotal == 0 {
		// No persisted records: nothing to report as healthy, but also
		// nothing to divide by. Report 0%, not NaN.
		report.HealthPercentage = 0
		return report, nil
	}

	report.HealthPercentage = 100 * float64(report.ShardsAvailable) / float64(report.ShardsTotal)
	return report, nil
}

// isShardHealthy reports whether a persisted ShardRecord represents a shard
// currently considered healthy: it has a non-empty checksum (proof a
// checksum was actually computed at upload time, not a zero-value record)
// and its persisted status is "verified".
func isShardHealthy(s models.ShardRecord) bool {
	return s.SHA256 != "" && s.Status == StatusVerified
}
