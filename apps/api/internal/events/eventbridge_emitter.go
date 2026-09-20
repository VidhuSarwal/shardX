package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

// eventBridgeAPI is the subset of the EventBridge SDK client this
// package calls, so tests can substitute a fake without a live bus.
type eventBridgeAPI interface {
	PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

// EventBridgeEmitter publishes domain events onto a custom EventBridge
// bus. Source is fixed to "shardx.control-plane"; DetailType matches
// the event's Type.
type EventBridgeEmitter struct {
	client   eventBridgeAPI
	busName  string
}

// NewEventBridgeEmitter loads AWS config via the standard credential
// chain (AWS_PROFILE/AWS_REGION, no custom credential source) and
// returns an emitter targeting busName.
func NewEventBridgeEmitter(ctx context.Context, busName string) (*EventBridgeEmitter, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &EventBridgeEmitter{
		client:  eventbridge.NewFromConfig(cfg),
		busName: busName,
	}, nil
}

// Emit publishes the event. Failures are logged, never returned or
// panicked on - a broken event bus must not fail the upload/download
// request that triggered the event.
func (e *EventBridgeEmitter) Emit(ctx context.Context, event Event) {
	detailJSON, err := json.Marshal(event.Detail)
	if err != nil {
		log.Printf("events: failed to marshal detail for %s: %v", event.Type, err)
		return
	}

	_, err = e.client.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(e.busName),
				Source:       aws.String("shardx.control-plane"),
				DetailType:   aws.String(string(event.Type)),
				Detail:       aws.String(string(detailJSON)),
			},
		},
	})
	if err != nil {
		log.Printf("events: failed to publish %s: %v", event.Type, err)
	}
}
