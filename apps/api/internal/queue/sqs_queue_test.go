package queue

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type fakeSQS struct {
	sent     []*sqs.SendMessageInput
	received *sqs.ReceiveMessageOutput
	deleted  []string
	err      error
}

func (f *fakeSQS) SendMessage(_ context.Context, in *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.sent = append(f.sent, in)
	return &sqs.SendMessageOutput{}, f.err
}
func (f *fakeSQS) ReceiveMessage(_ context.Context, in *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return f.received, f.err
}
func (f *fakeSQS) DeleteMessage(_ context.Context, in *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	f.deleted = append(f.deleted, aws.ToString(in.ReceiptHandle))
	return &sqs.DeleteMessageOutput{}, f.err
}

func TestSQSQueue_RoundTrip(t *testing.T) {
	f := &fakeSQS{}
	q := newSQSQueueForTest(f, "https://sqs/q")
	job := RetryJob{SessionID: "s1", ChunkID: 3, ChunkPath: "/tmp/c", Target: "s1", Filename: "chunk_003.2xpfm"}

	if err := q.Enqueue(context.Background(), job); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if got := aws.ToString(f.sent[0].QueueUrl); got != "https://sqs/q" {
		t.Fatalf("queue url = %q", got)
	}

	f.received = &sqs.ReceiveMessageOutput{Messages: []types.Message{
		{Body: f.sent[0].MessageBody, ReceiptHandle: aws.String("rh-1")},
	}}
	msgs, err := q.Receive(context.Background(), 1, 0)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("receive: %v %d", err, len(msgs))
	}
	if msgs[0].Job != job || msgs[0].ReceiptHandle != "rh-1" {
		t.Fatalf("round-trip mismatch: %+v", msgs[0])
	}

	if err := q.Delete(context.Background(), "rh-1"); err != nil || f.deleted[0] != "rh-1" {
		t.Fatalf("delete: %v %v", err, f.deleted)
	}
}

func TestSQSQueue_MalformedBodyIsError(t *testing.T) {
	f := &fakeSQS{received: &sqs.ReceiveMessageOutput{Messages: []types.Message{{Body: aws.String("{nope"), ReceiptHandle: aws.String("x")}}}}
	if _, err := newSQSQueueForTest(f, "u").Receive(context.Background(), 1, 0); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestSQSQueue_SendErrorPropagates(t *testing.T) {
	f := &fakeSQS{err: errors.New("boom")}
	if err := newSQSQueueForTest(f, "u").Enqueue(context.Background(), RetryJob{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNoopQueue_NotConfigured(t *testing.T) {
	if !errors.Is(NoopQueue{}.Enqueue(context.Background(), RetryJob{}), ErrNotConfigured) {
		t.Fatal("NoopQueue must return ErrNotConfigured")
	}
}
