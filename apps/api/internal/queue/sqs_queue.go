package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// sqsAPI is the subset of the SQS client this package calls, so tests can
// substitute a fake without a live queue.
type sqsAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// SQSQueue implements Queue on top of one SQS queue URL. Jobs are sent as
// JSON message bodies.
type SQSQueue struct {
	client   sqsAPI
	queueURL string
}

var _ Queue = (*SQSQueue)(nil)

// NewSQSQueue resolves AWS credentials/region via the default SDK chain
// (AWS_PROFILE / AWS_REGION). SQS_ENDPOINT_URL, if set, points the client
// at LocalStack, mirroring S3_ENDPOINT_URL in internal/storage.
func NewSQSQueue(ctx context.Context, queueURL string) (*SQSQueue, error) {
	if queueURL == "" {
		return nil, fmt.Errorf("sqs queue url must not be empty")
	}
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if endpoint := os.Getenv("SQS_ENDPOINT_URL"); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &SQSQueue{client: client, queueURL: queueURL}, nil
}

func newSQSQueueForTest(client sqsAPI, queueURL string) *SQSQueue {
	return &SQSQueue{client: client, queueURL: queueURL}
}

func (q *SQSQueue) Enqueue(ctx context.Context, job RetryJob) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal retry job: %w", err)
	}
	_, err = q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("send retry job for session %s chunk %d: %w", job.SessionID, job.ChunkID, err)
	}
	return nil
}

func (q *SQSQueue) Receive(ctx context.Context, max int32, waitSeconds int32) ([]Message, error) {
	out, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.queueURL),
		MaxNumberOfMessages: max,
		WaitTimeSeconds:     waitSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("receive retry jobs: %w", err)
	}
	msgs := make([]Message, 0, len(out.Messages))
	for _, m := range out.Messages {
		var job RetryJob
		if err := json.Unmarshal([]byte(aws.ToString(m.Body)), &job); err != nil {
			// A malformed body can never succeed; let SQS redrive it to the
			// DLQ rather than dropping it silently here.
			return nil, fmt.Errorf("unmarshal retry job %s: %w", aws.ToString(m.MessageId), err)
		}
		msgs = append(msgs, Message{Job: job, ReceiptHandle: aws.ToString(m.ReceiptHandle)})
	}
	return msgs, nil
}

func (q *SQSQueue) Delete(ctx context.Context, receiptHandle string) error {
	_, err := q.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	if err != nil {
		return fmt.Errorf("delete retry job: %w", err)
	}
	return nil
}
