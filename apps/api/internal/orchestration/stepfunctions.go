// Package orchestration provides a thin abstraction over AWS Step
// Functions so the upload pipeline can start a tracked execution that
// mirrors its lifecycle without coupling internal/filehandlers directly
// to the AWS SDK.
//
// Today the Step Functions state machine (see
// infrastructure/terraform/modules/orchestration/step_functions.tf) is
// built entirely from placeholder Pass states (VALIDATE -> FRAGMENTED ->
// UPLOAD_SHARDS -> VERIFY_CHECKSUMS -> REGISTER_METADATA -> HEALTHY).
// Pass states auto-advance the moment the execution starts; there are no
// Task states, so no task tokens are ever emitted and there is nothing
// for Go to call SendTaskSuccess/SendTaskFailure against yet. The
// honest, non-fictional integration available today is: start one
// execution per upload for visibility/tracking (giving every upload a
// correlated execution ARN visible in the Step Functions console/CLI),
// and log the resulting ARN. Real per-state task-token integration
// (SendTaskSuccess/SendTaskFailure as the Go pipeline completes each
// step) can only be wired up once the ASL is updated to have `.sync` or
// `waitForTaskToken` Task states that name a real resource - that is
// deferred to when the ASL gets real Lambda/ECS ARNs.
package orchestration

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

// Orchestrator starts a tracked Step Functions execution representing an
// upload's lifecycle. Implementations must not block or fail the
// caller's critical path on a slow or unavailable Step Functions
// service - failures should be logged, not propagated as request/upload
// errors.
type Orchestrator interface {
	// StartUploadExecution starts (or simulates starting, for
	// no-op implementations) an execution tracking the upload
	// identified by sessionID. input is marshaled to JSON and used as
	// the execution's input payload. Returns the execution ARN (empty
	// string for no-op implementations) and any error encountered.
	StartUploadExecution(ctx context.Context, sessionID string, input map[string]any) (executionARN string, err error)
}

// NoopOrchestrator starts nothing and returns an empty execution ARN.
// It's the default when no state machine is configured (e.g.
// STATE_MACHINE_ARN unset), so local/dev/Drive-mode runs behave exactly
// as before Step Functions orchestration existed.
type NoopOrchestrator struct{}

func (NoopOrchestrator) StartUploadExecution(context.Context, string, map[string]any) (string, error) {
	return "", nil
}

// sfnAPI is the subset of the Step Functions SDK client this package
// calls, so tests can substitute a fake without a live state machine.
type sfnAPI interface {
	StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
}

// SFNOrchestrator starts Step Functions executions against a fixed
// state machine ARN.
type SFNOrchestrator struct {
	client          sfnAPI
	stateMachineARN string
}

// NewSFNOrchestrator loads AWS config via the standard credential chain
// (AWS_PROFILE/AWS_REGION, no custom credential source) and returns an
// orchestrator targeting stateMachineARN.
func NewSFNOrchestrator(ctx context.Context, stateMachineARN string) (*SFNOrchestrator, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &SFNOrchestrator{
		client:          sfn.NewFromConfig(cfg),
		stateMachineARN: stateMachineARN,
	}, nil
}

// StartUploadExecution starts one Step Functions execution per upload,
// using sessionID directly as the execution name and input as the
// JSON-encoded execution input.
//
// Step Functions requires execution names to be unique per state
// machine within a 90-day window. Using the upload session ID as the
// name means StartExecution is idempotent for retries of the *same*
// session with the *same* input (AWS returns the existing execution's
// ARN with no error) - but if FinalizeUploadHandler is ever re-invoked
// for the same session with different input (e.g. a different chunking
// strategy on retry), AWS returns ExecutionAlreadyExists, which this
// method logs and surfaces as an error. That is a known, accepted
// limitation: it does not fail the Go upload pipeline (the error is
// only logged by the caller), and a real per-attempt unique name was
// judged unnecessary until there's a concrete case for re-finalizing a
// session with different parameters.
//
// This establishes a tracked execution ARN immediately; it does not
// attempt to drive the execution through its states from Go, since
// every state in the current ASL is a Pass state that advances on its
// own the instant the execution starts.
func (o *SFNOrchestrator) StartUploadExecution(ctx context.Context, sessionID string, input map[string]any) (string, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		log.Printf("orchestration: failed to marshal input for session %s: %v", sessionID, err)
		return "", err
	}

	out, err := o.client.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: aws.String(o.stateMachineARN),
		Name:            aws.String(sessionID),
		Input:           aws.String(string(inputJSON)),
	})
	if err != nil {
		log.Printf("orchestration: failed to start execution for session %s: %v", sessionID, err)
		return "", err
	}

	executionARN := aws.ToString(out.ExecutionArn)
	log.Printf("orchestration: started execution %s for session %s", executionARN, sessionID)
	return executionARN, nil
}
