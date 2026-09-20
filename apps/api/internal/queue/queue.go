// Package queue defines the retry queue used by the S3-mode upload
// pipeline. When a shard upload fails, the pipeline enqueues a RetryJob
// instead of failing the whole session; cmd/shardworker consumes the
// queue and retries. SQS's redrive policy (see
// infrastructure/terraform/modules/orchestration/sqs.tf) moves a job to
// the DLQ after max_receive_count failed attempts -- the worker never
// tracks attempts itself.
package queue

import (
	"context"
	"errors"
)

// RetryJob is one shard upload to retry. ChunkPath points at the local
// chunk file, which the pipeline leaves in place for the worker.
type RetryJob struct {
	SessionID string `json:"session_id"`
	ChunkID   int    `json:"chunk_id"`
	ChunkPath string `json:"chunk_path"`
	Target    string `json:"target"`
	Filename  string `json:"filename"`
}

// Message is a received job plus the handle needed to acknowledge it.
type Message struct {
	Job           RetryJob
	ReceiptHandle string
}

// ErrNotConfigured is returned by NoopQueue: no queue is wired up, so the
// caller must fall back to its non-queued (fail-fast) behavior.
var ErrNotConfigured = errors.New("queue: no retry queue configured")

// Queue is the producer/consumer surface used by the API and the worker.
type Queue interface {
	Enqueue(ctx context.Context, job RetryJob) error
	// Receive long-polls for up to waitSeconds and returns at most max jobs.
	Receive(ctx context.Context, max int32, waitSeconds int32) ([]Message, error)
	// Delete acknowledges a job so SQS won't redeliver it.
	Delete(ctx context.Context, receiptHandle string) error
}

// NoopQueue is the default when SQS_QUEUE_URL is unset. Every call returns
// ErrNotConfigured so Drive-mode / local runs keep their existing
// fail-fast behavior.
type NoopQueue struct{}

func (NoopQueue) Enqueue(context.Context, RetryJob) error { return ErrNotConfigured }
func (NoopQueue) Receive(context.Context, int32, int32) ([]Message, error) {
	return nil, ErrNotConfigured
}
func (NoopQueue) Delete(context.Context, string) error { return ErrNotConfigured }
