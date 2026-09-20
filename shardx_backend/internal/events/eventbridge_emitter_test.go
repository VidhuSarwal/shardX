package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type fakeEventBridgeAPI struct {
	lastInput *eventbridge.PutEventsInput
	err       error
	calls     int
}

func (f *fakeEventBridgeAPI) PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	f.calls++
	f.lastInput = params
	if f.err != nil {
		return nil, f.err
	}
	return &eventbridge.PutEventsOutput{}, nil
}

func TestEventBridgeEmitter_Emit_PublishesExpectedEntry(t *testing.T) {
	fake := &fakeEventBridgeAPI{}
	e := &EventBridgeEmitter{client: fake, busName: "shardx-events"}

	e.Emit(context.Background(), Event{
		Type:   ShardUploaded,
		Detail: map[string]any{"session_id": "abc123", "chunk_id": 3},
	})

	if fake.calls != 1 {
		t.Fatalf("expected 1 PutEvents call, got %d", fake.calls)
	}
	entries := fake.lastInput.Entries
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	entry := entries[0]
	if *entry.EventBusName != "shardx-events" {
		t.Errorf("bus name = %q", *entry.EventBusName)
	}
	if *entry.Source != "shardx.control-plane" {
		t.Errorf("source = %q", *entry.Source)
	}
	if *entry.DetailType != string(ShardUploaded) {
		t.Errorf("detail type = %q", *entry.DetailType)
	}
	var detail map[string]any
	if err := json.Unmarshal([]byte(*entry.Detail), &detail); err != nil {
		t.Fatalf("detail not valid JSON: %v", err)
	}
	if detail["session_id"] != "abc123" {
		t.Errorf("detail session_id = %v", detail["session_id"])
	}
}

func TestEventBridgeEmitter_Emit_SwallowsPutEventsError(t *testing.T) {
	fake := &fakeEventBridgeAPI{err: errors.New("bus unavailable")}
	e := &EventBridgeEmitter{client: fake, busName: "shardx-events"}

	// Must not panic and must not return an error - there's nothing to
	// return to, Emit's signature has no error return.
	e.Emit(context.Background(), Event{Type: FileCreated, Detail: map[string]any{}})

	if fake.calls != 1 {
		t.Fatalf("expected PutEvents to still be attempted once, got %d calls", fake.calls)
	}
}

func TestNoopEmitter_DoesNothing(t *testing.T) {
	var e Emitter = NoopEmitter{}
	// Should not panic on nil detail or any input.
	e.Emit(context.Background(), Event{Type: FileCreated})
}
