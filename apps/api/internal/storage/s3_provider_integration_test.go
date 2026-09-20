// s3_provider_integration_test.go exercises S3Provider against a real S3
// API surface provided by LocalStack (https://localstack.cloud/), running
// locally in Docker. It performs actual network calls (no mocking) to
// validate UploadChunk/DeleteChunk/GetSpace end-to-end: object PUT with
// SSE-KMS headers, object existence, and DELETE.
//
// This test is opt-in and skipped by default so `go test ./...` does not
// require Docker/LocalStack in CI or on developer machines that don't have
// it running. To run it:
//
//	docker compose -f docker-compose.localstack.yml up -d
//	LOCALSTACK=1 go test ./internal/storage/... -run Integration -v
//
// (LocalStack's S3 implementation accepts SSE-KMS parameters but does not
// perform real KMS encryption/validation, so this test uses a fake KMS key
// ID; it verifies the SDK call succeeds and the object round-trips, not
// that LocalStack actually encrypts anything.)
package storage

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const localstackEndpoint = "http://localhost:4566"

func skipUnlessLocalstack(t *testing.T) {
	t.Helper()
	if os.Getenv("LOCALSTACK") != "1" {
		t.Skip("skipping LocalStack integration test: set LOCALSTACK=1 and run docker-compose.localstack.yml to enable")
	}
}

// newLocalstackS3Provider builds an S3Provider pointed at a local
// LocalStack container using throwaway static credentials (LocalStack does
// not validate them), bypassing NewS3Provider's real credential-chain
// loading only insofar as it supplies fake-but-well-formed static creds so
// config.LoadDefaultConfig succeeds without real AWS access.
func newLocalstackS3Provider(t *testing.T, bucket string) *S3Provider {
	t.Helper()
	ctx := context.Background()

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		t.Fatalf("loading test AWS config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(localstackEndpoint)
		o.UsePathStyle = true
	})

	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatalf("creating test bucket %q on localstack: %v", bucket, err)
	}
	t.Cleanup(func() {
		// Best-effort cleanup; LocalStack state doesn't persist across
		// container restarts anyway.
		emptyAndDeleteBucket(context.Background(), client, bucket)
	})

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = multipartThreshold
	})

	return newS3ProviderForTest(client, uploader, bucket, "alias/test-key")
}

func emptyAndDeleteBucket(ctx context.Context, client *s3.Client, bucket string) {
	out, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err == nil {
		for _, obj := range out.Contents {
			_, _ = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: obj.Key})
		}
	}
	_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
}

func TestIntegration_S3Provider_UploadDeleteRoundTrip(t *testing.T) {
	skipUnlessLocalstack(t)

	bucket := "shardx-test-bucket"
	p := newLocalstackS3Provider(t, bucket)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	content := []byte("integration test chunk contents")
	chunkPath := mustWriteTempFile(t, content)

	locationID, err := p.UploadChunk(ctx, "test-account", chunkPath, "chunk_000.bin")
	if err != nil {
		t.Fatalf("UploadChunk failed: %v", err)
	}
	if locationID != "test-account/chunk_000.bin" {
		t.Fatalf("locationID = %q, want %q", locationID, "test-account/chunk_000.bin")
	}

	// Verify the object actually exists in LocalStack with the expected
	// content (real network GetObject, no mocking).
	underlying := p.client.(*s3.Client)
	getOut, err := underlying.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(locationID),
	})
	if err != nil {
		t.Fatalf("GetObject failed after upload: %v", err)
	}
	defer getOut.Body.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(getOut.Body); err != nil {
		t.Fatalf("reading uploaded object body: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), content) {
		t.Fatalf("uploaded content mismatch: got %q, want %q", buf.String(), string(content))
	}

	if err := p.DeleteChunk(ctx, "test-account", locationID); err != nil {
		t.Fatalf("DeleteChunk failed: %v", err)
	}

	_, err = underlying.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(locationID),
	})
	if err == nil {
		t.Fatal("expected GetObject to fail after DeleteChunk, but object still exists")
	}
}

func TestIntegration_S3Provider_GetSpace(t *testing.T) {
	skipUnlessLocalstack(t)

	bucket := "shardx-test-bucket-space"
	p := newLocalstackS3Provider(t, bucket)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	spaces, err := p.GetSpace(ctx, "any-target")
	if err != nil {
		t.Fatalf("GetSpace failed: %v", err)
	}
	if len(spaces) != 1 || !spaces[0].Available {
		t.Fatalf("unexpected GetSpace result: %+v", spaces)
	}
}

func TestIntegration_S3Provider_LargeFileMultipartUpload(t *testing.T) {
	skipUnlessLocalstack(t)

	bucket := "shardx-test-bucket-multipart"
	p := newLocalstackS3Provider(t, bucket)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Larger than multipartThreshold (5 MiB) to force the uploader down
	// the multipart path.
	content := bytes.Repeat([]byte("x"), multipartThreshold+1024*1024)
	chunkPath := mustWriteTempFile(t, content)

	locationID, err := p.UploadChunk(ctx, "big-account", chunkPath, "big_chunk.bin")
	if err != nil {
		t.Fatalf("UploadChunk (multipart) failed: %v", err)
	}

	underlying := p.client.(*s3.Client)
	headOut, err := underlying.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(locationID),
	})
	if err != nil {
		t.Fatalf("HeadObject failed after multipart upload: %v", err)
	}
	if headOut.ContentLength == nil || *headOut.ContentLength != int64(len(content)) {
		t.Fatalf("uploaded object size mismatch: got %v, want %d", headOut.ContentLength, len(content))
	}
}

func mustWriteTempFile(t *testing.T, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "chunk_*.bin")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return f.Name()
}
