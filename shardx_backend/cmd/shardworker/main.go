// shardworker consumes shard-upload retry jobs from SQS (S3 mode). Run it
// next to cmd/server with the same .env: it needs the same STORAGE_PROVIDER,
// DB_PROVIDER, S3_*, SQS_QUEUE_URL and UPLOAD_TEMP_DIR so it can read the
// chunk files the API left on disk.
package main

import (
	"SE/internal/bootstrap"
	"SE/internal/filehandlers"
	"SE/internal/worker"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}
	if os.Getenv("SQS_QUEUE_URL") == "" {
		log.Fatal("SQS_QUEUE_URL is required")
	}

	metaStore := bootstrap.SelectMetadataStore(os.Getenv("DB_PROVIDER"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := metaStore.InitStore(ctx); err != nil {
		log.Fatalf("init store: %v", err)
	}
	cancel()
	defer metaStore.DisconnectStore(context.Background())

	// Not strictly needed by the worker, but keeps the health handler's
	// package state consistent if this binary ever serves it.
	filehandlers.InitMetadataStore(metaStore)

	w := &worker.Worker{
		Queue:   bootstrap.SelectRetryQueue(os.Getenv("SQS_QUEUE_URL")),
		Store:   metaStore,
		Storage: bootstrap.SelectStorageProvider(os.Getenv("STORAGE_PROVIDER")),
		Events:  bootstrap.SelectEventEmitter(os.Getenv("EVENT_BUS_NAME")),
	}

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Println("shardworker: polling for retry jobs")
	w.Run(runCtx)
	log.Println("shardworker: stopped")
}
