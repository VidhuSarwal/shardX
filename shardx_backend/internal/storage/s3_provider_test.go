package storage

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// fakeS3Client is a minimal in-memory fake satisfying s3API, used to test
// S3Provider's business logic (key derivation, encryption params, error
// propagation) with no network calls.
type fakeS3Client struct {
	deleteObjectCalls []*s3.DeleteObjectInput
	deleteObjectErr   error

	headBucketErr error

	putObjectCalls []*s3.PutObjectInput
	putObjectErr   error
}

func (f *fakeS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	f.putObjectCalls = append(f.putObjectCalls, params)
	if f.putObjectErr != nil {
		return nil, f.putObjectErr
	}
	return &s3.PutObjectOutput{}, nil
}

func (f *fakeS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.deleteObjectCalls = append(f.deleteObjectCalls, params)
	if f.deleteObjectErr != nil {
		return nil, f.deleteObjectErr
	}
	return &s3.DeleteObjectOutput{}, nil
}

func (f *fakeS3Client) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	if f.headBucketErr != nil {
		return nil, f.headBucketErr
	}
	return &s3.HeadBucketOutput{}, nil
}

// fakeUploader is a minimal fake satisfying s3Uploader, recording the
// PutObjectInput it was called with (including encryption fields) without
// doing any network I/O. It fully drains the Body reader, mimicking real
// uploader behavior closely enough for our tests (which only assert on
// input fields, not persisted bytes).
type fakeUploader struct {
	calls []*s3.PutObjectInput
	err   error
}

func (u *fakeUploader) Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error) {
	u.calls = append(u.calls, input)
	if u.err != nil {
		return nil, u.err
	}
	if input.Body != nil {
		buf := make([]byte, 32*1024)
		for {
			n, err := input.Body.Read(buf)
			_ = n
			if err != nil {
				break
			}
		}
	}
	return &manager.UploadOutput{Location: "fake://" + *input.Bucket + "/" + *input.Key}, nil
}

func TestS3Provider_InterfaceCompliance(t *testing.T) {
	var _ StorageProvider = (*S3Provider)(nil)
}

func mustTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "chunk_*.bin")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return f.Name()
}

func TestS3Provider_UploadChunk_Success(t *testing.T) {
	uploader := &fakeUploader{}
	client := &fakeS3Client{}
	p := newS3ProviderForTest(client, uploader, "my-bucket", "arn:aws:kms:us-east-1:123456789012:key/test-key")

	chunkPath := mustTempFile(t, "hello chunk contents")

	locationID, err := p.UploadChunk(context.Background(), "account123", chunkPath, "chunk_000.bin")
	if err != nil {
		t.Fatalf("UploadChunk returned error: %v", err)
	}

	wantKey := "account123/chunk_000.bin"
	if locationID != wantKey {
		t.Fatalf("locationID = %q, want %q", locationID, wantKey)
	}

	if len(uploader.calls) != 1 {
		t.Fatalf("expected 1 upload call, got %d", len(uploader.calls))
	}
	call := uploader.calls[0]
	if *call.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q, want %q", *call.Bucket, "my-bucket")
	}
	if *call.Key != wantKey {
		t.Errorf("Key = %q, want %q", *call.Key, wantKey)
	}
	if call.ServerSideEncryption != types.ServerSideEncryptionAwsKms {
		t.Errorf("ServerSideEncryption = %v, want %v", call.ServerSideEncryption, types.ServerSideEncryptionAwsKms)
	}
	if call.SSEKMSKeyId == nil || *call.SSEKMSKeyId != "arn:aws:kms:us-east-1:123456789012:key/test-key" {
		t.Errorf("SSEKMSKeyId = %v, want the configured KMS key ARN", call.SSEKMSKeyId)
	}
}

func TestS3Provider_UploadChunk_EmptyTarget(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "bucket", "key-id")
	chunkPath := mustTempFile(t, "data")

	_, err := p.UploadChunk(context.Background(), "", chunkPath, "file.bin")
	if err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
}

func TestS3Provider_UploadChunk_PathTraversalRejected(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "bucket", "key-id")
	chunkPath := mustTempFile(t, "data")

	_, err := p.UploadChunk(context.Background(), "../etc", chunkPath, "file.bin")
	if err == nil {
		t.Fatal("expected error for target containing '..', got nil")
	}
}

func TestS3Provider_UploadChunk_MissingFile(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "bucket", "key-id")

	_, err := p.UploadChunk(context.Background(), "target", "/no/such/path/chunk.bin", "chunk.bin")
	if err == nil {
		t.Fatal("expected error for missing chunk file, got nil")
	}
}

func TestS3Provider_UploadChunk_UploaderError(t *testing.T) {
	wantErr := errors.New("simulated network failure")
	uploader := &fakeUploader{err: wantErr}
	p := newS3ProviderForTest(&fakeS3Client{}, uploader, "bucket", "key-id")
	chunkPath := mustTempFile(t, "data")

	_, err := p.UploadChunk(context.Background(), "target", chunkPath, "chunk.bin")
	if err == nil {
		t.Fatal("expected error to propagate from uploader, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error to satisfy errors.Is(%v), got %v", wantErr, err)
	}
}

func TestS3Provider_DeleteChunk_Success(t *testing.T) {
	client := &fakeS3Client{}
	p := newS3ProviderForTest(client, &fakeUploader{}, "my-bucket", "key-id")

	err := p.DeleteChunk(context.Background(), "account123", "account123/chunk_000.bin")
	if err != nil {
		t.Fatalf("DeleteChunk returned error: %v", err)
	}

	if len(client.deleteObjectCalls) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(client.deleteObjectCalls))
	}
	call := client.deleteObjectCalls[0]
	if *call.Bucket != "my-bucket" {
		t.Errorf("Bucket = %q, want %q", *call.Bucket, "my-bucket")
	}
	if *call.Key != "account123/chunk_000.bin" {
		t.Errorf("Key = %q, want %q", *call.Key, "account123/chunk_000.bin")
	}
}

func TestS3Provider_DeleteChunk_EmptyLocationID(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "bucket", "key-id")
	err := p.DeleteChunk(context.Background(), "target", "")
	if err == nil {
		t.Fatal("expected error for empty locationID, got nil")
	}
}

func TestS3Provider_DeleteChunk_EmptyTarget(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "bucket", "key-id")
	err := p.DeleteChunk(context.Background(), "", "some/key.bin")
	if err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
}

func TestS3Provider_DeleteChunk_ClientError(t *testing.T) {
	wantErr := errors.New("simulated delete failure")
	client := &fakeS3Client{deleteObjectErr: wantErr}
	p := newS3ProviderForTest(client, &fakeUploader{}, "bucket", "key-id")

	err := p.DeleteChunk(context.Background(), "target", "target/key.bin")
	if err == nil {
		t.Fatal("expected error to propagate from client, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error to satisfy errors.Is(%v), got %v", wantErr, err)
	}
}

func TestS3Provider_GetSpace_ReturnsUnlimitedSentinel(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "my-bucket", "key-id")

	spaces, err := p.GetSpace(context.Background(), "some-target")
	if err != nil {
		t.Fatalf("GetSpace returned error: %v", err)
	}
	if len(spaces) != 1 {
		t.Fatalf("expected 1 space entry, got %d", len(spaces))
	}
	got := spaces[0]
	if got.DisplayName != "my-bucket" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "my-bucket")
	}
	if !got.Available {
		t.Error("expected Available = true")
	}
	if got.UsedSpace != 0 {
		t.Errorf("UsedSpace = %d, want 0 (not tracked by this provider)", got.UsedSpace)
	}
	if got.TotalSpace <= 0 || got.FreeSpace <= 0 {
		t.Errorf("expected positive sentinel Total/FreeSpace, got Total=%d Free=%d", got.TotalSpace, got.FreeSpace)
	}
}

func TestS3Provider_GetSpace_EmptyTarget(t *testing.T) {
	p := newS3ProviderForTest(&fakeS3Client{}, &fakeUploader{}, "bucket", "key-id")
	_, err := p.GetSpace(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty target, got nil")
	}
}

func TestS3Provider_GetSpace_HeadBucketError(t *testing.T) {
	wantErr := errors.New("bucket not found")
	client := &fakeS3Client{headBucketErr: wantErr}
	p := newS3ProviderForTest(client, &fakeUploader{}, "bucket", "key-id")

	_, err := p.GetSpace(context.Background(), "target")
	if err == nil {
		t.Fatal("expected error when HeadBucket fails, got nil")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error to satisfy errors.Is(%v), got %v", wantErr, err)
	}
}

func TestObjectKey_TrimsSlashes(t *testing.T) {
	key, err := objectKey("/account123/", "/chunk.bin")
	if err != nil {
		t.Fatalf("objectKey returned error: %v", err)
	}
	if key != "account123/chunk.bin" {
		t.Errorf("objectKey = %q, want %q", key, "account123/chunk.bin")
	}
}

func TestObjectKey_RejectsEmptyFilename(t *testing.T) {
	_, err := objectKey("target", "")
	if err == nil {
		t.Fatal("expected error for empty filename, got nil")
	}
}

func TestNewS3Provider_RejectsEmptyBucket(t *testing.T) {
	_, err := NewS3Provider(context.Background(), "", "key-id")
	if err == nil {
		t.Fatal("expected error for empty bucket, got nil")
	}
}

func TestNewS3Provider_RejectsEmptyKMSKeyID(t *testing.T) {
	_, err := NewS3Provider(context.Background(), "bucket", "")
	if err == nil {
		t.Fatal("expected error for empty kms key id, got nil")
	}
}
