// Package events defines ShardX's domain event vocabulary and an
// Emitter abstraction so upload/download lifecycle code can announce
// what happened without depending on EventBridge directly.
package events

import "context"

// Type is one of the domain events defined in the ShardX AWS migration spec.
type Type string

const (
	FileCreated        Type = "FILE_CREATED"
	ShardUploaded      Type = "SHARD_UPLOADED"
	ShardVerified      Type = "SHARD_VERIFIED"
	ShardCorrupted     Type = "SHARD_CORRUPTED"
	RecoveryStarted    Type = "RECOVERY_STARTED"
	RecoveryCompleted  Type = "RECOVERY_COMPLETED"
	MalwareDetected    Type = "MALWARE_DETECTED"
	MacieFinding       Type = "MACIE_FINDING"
	GuardDutyFinding   Type = "GUARDDUTY_FINDING"
	PolicyViolation    Type = "POLICY_VIOLATION"
	ReplicationFailed  Type = "REPLICATION_FAILED"
)

// Event is a single domain event. Detail is arbitrary, JSON-marshalable
// data specific to the event Type (e.g. session/shard/file identifiers).
type Event struct {
	Type   Type
	Detail map[string]any
}

// Emitter publishes domain events. Implementations must not block the
// caller's critical path on a slow or unavailable event bus - emit
// failures should be logged, not propagated as request errors.
type Emitter interface {
	Emit(ctx context.Context, event Event)
}

// NoopEmitter discards every event. It's the default when no event bus
// is configured (e.g. EVENT_BUS_NAME unset), so local/dev/Drive-mode
// runs behave exactly as before events existed.
type NoopEmitter struct{}

func (NoopEmitter) Emit(context.Context, Event) {}
