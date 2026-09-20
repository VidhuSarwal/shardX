package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type fakeSFNAPI struct {
	lastInput *sfn.StartExecutionInput
	out       *sfn.StartExecutionOutput
	err       error
	calls     int
}

func (f *fakeSFNAPI) StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	f.calls++
	f.lastInput = params
	if f.err != nil {
		return nil, f.err
	}
	if f.out != nil {
		return f.out, nil
	}
	return &sfn.StartExecutionOutput{}, nil
}

func TestSFNOrchestrator_StartUploadExecution_StartsExpectedExecution(t *testing.T) {
	fake := &fakeSFNAPI{
		out: &sfn.StartExecutionOutput{
			ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:shardx-shard-lifecycle:sess-1"),
		},
	}
	o := &SFNOrchestrator{client: fake, stateMachineARN: "arn:aws:states:us-east-1:123456789012:stateMachine:shardx-shard-lifecycle"}

	arn, err := o.StartUploadExecution(context.Background(), "sess-1", map[string]any{
		"session_id": "sess-1",
		"filename":   "foo.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arn != "arn:aws:states:us-east-1:123456789012:execution:shardx-shard-lifecycle:sess-1" {
		t.Errorf("unexpected execution arn: %q", arn)
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 StartExecution call, got %d", fake.calls)
	}
	if aws.ToString(fake.lastInput.StateMachineArn) != o.stateMachineARN {
		t.Errorf("state machine arn = %q", aws.ToString(fake.lastInput.StateMachineArn))
	}
	if aws.ToString(fake.lastInput.Name) != "sess-1" {
		t.Errorf("execution name = %q", aws.ToString(fake.lastInput.Name))
	}
	var input map[string]any
	if err := json.Unmarshal([]byte(aws.ToString(fake.lastInput.Input)), &input); err != nil {
		t.Fatalf("input not valid JSON: %v", err)
	}
	if input["session_id"] != "sess-1" {
		t.Errorf("input session_id = %v", input["session_id"])
	}
}

func TestSFNOrchestrator_StartUploadExecution_PropagatesError(t *testing.T) {
	fake := &fakeSFNAPI{err: errors.New("state machine unavailable")}
	o := &SFNOrchestrator{client: fake, stateMachineARN: "arn:aws:states:us-east-1:123456789012:stateMachine:shardx-shard-lifecycle"}

	arn, err := o.StartUploadExecution(context.Background(), "sess-2", map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if arn != "" {
		t.Errorf("expected empty arn on error, got %q", arn)
	}
	if fake.calls != 1 {
		t.Fatalf("expected 1 StartExecution call, got %d", fake.calls)
	}
}

func TestNoopOrchestrator_StartUploadExecution_ReturnsEmptyARNAndNoError(t *testing.T) {
	var o Orchestrator = NoopOrchestrator{}

	arn, err := o.StartUploadExecution(context.Background(), "sess-3", map[string]any{"anything": "goes"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if arn != "" {
		t.Errorf("expected empty arn, got %q", arn)
	}
}
