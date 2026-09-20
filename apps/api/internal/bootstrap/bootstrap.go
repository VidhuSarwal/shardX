// Package bootstrap holds the env-var -> provider selection shared by
// cmd/server and cmd/shardworker, so both binaries wire the same
// backends (STORAGE_PROVIDER, DB_PROVIDER, AUTH_PROVIDER, EVENT_BUS_NAME,
// STATE_MACHINE_ARN, SQS_QUEUE_URL) identically. Every selector defaults
// to the pre-migration behavior when its env var is unset.
package bootstrap

import (
	"SE/internal/authprovider"
	"SE/internal/events"
	"SE/internal/metadatastore"
	"SE/internal/orchestration"
	"SE/internal/queue"
	"SE/internal/storage"
	"context"
	"log"
	"os"
	"time"
)

// SelectMetadataStore picks a metadatastore.MetadataStore implementation
// based on the DB_PROVIDER env var. Defaults to "mongo" if unset.
func SelectMetadataStore(provider string) metadatastore.MetadataStore {
	if provider == "" {
		provider = "mongo"
	}
	switch provider {
	case "mongo":
		metadatastore.Active = metadatastore.NewMongoStore()
		return metadatastore.Active
	case "dynamodb":
		tablePrefix := os.Getenv("DYNAMODB_TABLE_PREFIX")
		if tablePrefix == "" {
			log.Fatalf("DB_PROVIDER=dynamodb requires DYNAMODB_TABLE_PREFIX")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s, err := metadatastore.NewDynamoDBStore(ctx, tablePrefix)
		if err != nil {
			log.Fatalf("init dynamodb store: %v", err)
		}
		metadatastore.Active = s
		return s
	default:
		log.Fatalf("DB_PROVIDER %q not yet implemented, coming in a later phase", provider)
		return nil
	}
}

// SelectStorageProvider picks a storage.StorageProvider implementation
// based on the STORAGE_PROVIDER env var. Defaults to "drive" if unset.
func SelectStorageProvider(provider string) storage.StorageProvider {
	if provider == "" {
		provider = "drive"
	}
	switch provider {
	case "drive":
		return storage.NewDriveProvider()
	case "s3":
		bucket := os.Getenv("S3_BUCKET")
		kmsKeyID := os.Getenv("S3_KMS_KEY_ID")
		if bucket == "" || kmsKeyID == "" {
			log.Fatalf("STORAGE_PROVIDER=s3 requires S3_BUCKET and S3_KMS_KEY_ID env vars to be set")
		}
		p, err := storage.NewS3Provider(context.Background(), bucket, kmsKeyID)
		if err != nil {
			log.Fatalf("init s3 storage provider: %v", err)
		}
		return p
	default:
		log.Fatalf("STORAGE_PROVIDER %q not yet implemented, coming in a later phase", provider)
		return nil
	}
}

// SelectAuthProvider picks an authprovider.AuthProvider implementation
// based on the AUTH_PROVIDER env var. Defaults to "custom" if unset.
func SelectAuthProvider(provider string) authprovider.AuthProvider {
	if provider == "" {
		provider = "custom"
	}
	switch provider {
	case "custom":
		return authprovider.NewCustomProvider()
	case "cognito":
		userPoolID := os.Getenv("COGNITO_USER_POOL_ID")
		clientID := os.Getenv("COGNITO_CLIENT_ID")
		region := os.Getenv("AWS_REGION")
		if userPoolID == "" || clientID == "" || region == "" {
			log.Fatalf("AUTH_PROVIDER=cognito requires COGNITO_USER_POOL_ID, COGNITO_CLIENT_ID, and AWS_REGION")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := authprovider.NewCognitoProvider(ctx, userPoolID, clientID, region)
		if err != nil {
			log.Fatalf("init cognito auth provider: %v", err)
		}
		return p
	default:
		log.Fatalf("AUTH_PROVIDER %q not yet implemented, coming in a later phase", provider)
		return nil
	}
}

// SelectEventEmitter picks an events.Emitter based on the
// EVENT_BUS_NAME env var. Unset means no event bus configured: return
// a no-op emitter so local dev / Drive-mode runs without AWS access
// see zero behavior change.
func SelectEventEmitter(busName string) events.Emitter {
	if busName == "" {
		return events.NoopEmitter{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e, err := events.NewEventBridgeEmitter(ctx, busName)
	if err != nil {
		log.Fatalf("init eventbridge emitter: %v", err)
	}
	return e
}

// SelectOrchestrator picks an orchestration.Orchestrator based on the
// STATE_MACHINE_ARN env var. Unset means no state machine configured:
// return a no-op orchestrator so local dev / Drive-mode runs without
// AWS access see zero behavior change.
func SelectOrchestrator(stateMachineARN string) orchestration.Orchestrator {
	if stateMachineARN == "" {
		return orchestration.NoopOrchestrator{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	o, err := orchestration.NewSFNOrchestrator(ctx, stateMachineARN)
	if err != nil {
		log.Fatalf("init step functions orchestrator: %v", err)
	}
	return o
}

// SelectRetryQueue picks a queue.Queue based on the SQS_QUEUE_URL env
// var. Unset means no queue: return NoopQueue so the upload pipeline
// keeps its fail-fast behavior.
func SelectRetryQueue(queueURL string) queue.Queue {
	if queueURL == "" {
		return queue.NoopQueue{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	q, err := queue.NewSQSQueue(ctx, queueURL)
	if err != nil {
		log.Fatalf("init sqs retry queue: %v", err)
	}
	return q
}
