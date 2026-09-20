// s3_provider.go implements StorageProvider on top of Amazon S3. It mirrors
// the target-string conventions established by DriveProvider (see
// provider.go's doc comment) but reinterprets "target" as an S3 key prefix
// (a logical shard-group / tenant path within a single configured bucket)
// rather than a Mongo ObjectID, because S3 has no notion of "accounts" the
// way Drive does. See NewS3Provider and the per-method docs below for the
// exact mapping.
package storage

import (
	"SE/internal/models"
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// multipartThreshold is the file-size cutoff above which uploads are sent
// via the multipart manager.Uploader instead of a single PutObject call.
// This mirrors the simple-vs-resumable split in the existing Drive upload
// path (internal/drivemanager), which switches to a resumable session
// above a similar-sized threshold. 5 MiB is also the minimum part size S3
// itself allows for multipart uploads, so it is a natural boundary.
const multipartThreshold = 5 * 1024 * 1024 // 5 MiB

// s3API is the subset of the S3 SDK client surface that S3Provider calls
// directly (i.e. not through manager.Uploader, which handles all uploads
// -- see UploadChunk). Defining it as a narrow interface covering only
// the methods actually called (DeleteObject, HeadBucket) lets tests
// substitute a fake implementation and assert on exactly the parameters
// sent, without making any network call. manager.NewUploader itself
// accepts the concrete *s3.Client and is constructed separately in
// production; its own surface is captured by the narrower s3Uploader
// interface below.
type s3API interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
}

// s3Uploader is the subset of manager.Uploader's surface used by
// S3Provider, narrowed for testability the same way as s3API.
type s3Uploader interface {
	Upload(ctx context.Context, input *s3.PutObjectInput, opts ...func(*manager.Uploader)) (*manager.UploadOutput, error)
}

// S3Provider implements StorageProvider backed by a single Amazon S3
// bucket, with all objects encrypted using a customer-managed KMS key
// (matching the infrastructure/terraform/modules/storage module, which
// provisions exactly one CMK-encrypted bucket per environment).
//
// Target semantics: for UploadChunk and DeleteChunk, target is used as the
// S3 key *prefix* under which the chunk is stored/looked up (e.g. a hex
// drive-account-style ID, a user ID, or any other logical shard-group
// identifier the caller wants to segregate objects by within the bucket).
// This keeps the mapping simple and bucket-per-environment rather than
// bucket-per-account, which would not scale to S3's account bucket-count
// limits. The resulting object key is target + "/" + filename, and the
// locationID returned by UploadChunk (and required by DeleteChunk) is that
// full key.
type S3Provider struct {
	client   s3API
	uploader s3Uploader
	bucket   string
	kmsKeyID string
}

// compile-time interface compliance assertion
var _ StorageProvider = (*S3Provider)(nil)

// NewS3Provider constructs an S3Provider for the given bucket, encrypting
// all uploads with the given KMS key ID (a key ID, alias, or ARN -- passed
// straight through to SSEKMSKeyId). AWS credentials and region are
// resolved exclusively via the standard SDK default credential chain and
// environment (AWS_PROFILE, AWS_REGION, shared config/credentials files,
// EC2/ECS/EKS instance roles, etc.) through config.LoadDefaultConfig; no
// credentials are read from a custom source or hardcoded here.
//
// If the S3_ENDPOINT_URL environment variable is set, it is used as the
// client's BaseEndpoint. This is intended for local testing against
// LocalStack and is not consulted in production deployments (where the
// variable is simply unset).
func NewS3Provider(ctx context.Context, bucket, kmsKeyID string) (*S3Provider, error) {
	if bucket == "" {
		return nil, fmt.Errorf("s3 bucket name must not be empty")
	}
	if kmsKeyID == "" {
		return nil, fmt.Errorf("s3 kms key id must not be empty")
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint := os.Getenv("S3_ENDPOINT_URL"); endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true // required by LocalStack
		}
	})

	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = multipartThreshold
	})

	return &S3Provider{
		client:   client,
		uploader: uploader,
		bucket:   bucket,
		kmsKeyID: kmsKeyID,
	}, nil
}

// newS3ProviderForTest builds an S3Provider around fake client/uploader
// implementations, bypassing NewS3Provider's AWS config loading. Used only
// from tests in this package.
func newS3ProviderForTest(client s3API, uploader s3Uploader, bucket, kmsKeyID string) *S3Provider {
	return &S3Provider{client: client, uploader: uploader, bucket: bucket, kmsKeyID: kmsKeyID}
}

// objectKey builds the S3 key for a chunk given its target prefix and
// filename, per the target semantics documented on S3Provider.
func objectKey(target, filename string) (string, error) {
	if target == "" {
		return "", fmt.Errorf("target (key prefix) must not be empty")
	}
	if filename == "" {
		return "", fmt.Errorf("filename must not be empty")
	}
	// Guard against accidental path traversal / prefix escape via either
	// component, since both may originate from user-controlled input
	// upstream.
	if strings.Contains(target, "..") || strings.Contains(filename, "..") {
		return "", fmt.Errorf("target/filename must not contain '..'")
	}
	target = strings.Trim(target, "/")
	filename = strings.TrimPrefix(filename, "/")
	return target + "/" + filename, nil
}

// UploadChunk uploads the chunk file at chunkPath (named filename) to this
// provider's bucket under the key derived from target (see S3Provider's
// target semantics doc). All uploads -- regardless of size -- go through
// manager.Uploader, which transparently performs a single PutObject for
// files at or below multipartThreshold and a multipart upload above it;
// this satisfies the "simple vs resumable split" requirement while keeping
// one code path. Every upload is server-side encrypted with the
// provider's configured KMS key. The returned locationID is the full S3
// object key, which DeleteChunk expects back unchanged.
func (p *S3Provider) UploadChunk(ctx context.Context, target string, chunkPath, filename string) (string, error) {
	key, err := objectKey(target, filename)
	if err != nil {
		return "", err
	}

	f, err := os.Open(chunkPath)
	if err != nil {
		return "", fmt.Errorf("opening chunk file %q: %w", chunkPath, err)
	}
	defer f.Close()

	_, err = p.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(p.bucket),
		Key:                  aws.String(key),
		Body:                 f,
		ServerSideEncryption: types.ServerSideEncryptionAwsKms,
		SSEKMSKeyId:          aws.String(p.kmsKeyID),
	})
	if err != nil {
		return "", fmt.Errorf("uploading chunk to s3://%s/%s: %w", p.bucket, key, err)
	}

	return key, nil
}

// DeleteChunk deletes the object identified by locationID (the full S3 key
// returned by a prior UploadChunk) from this provider's bucket. target is
// accepted for StorageProvider interface conformance and defensive
// validation (it is checked but not required to reconstruct the key --
// unlike Drive, S3 does not need a separate account handle to address an
// object once the key is known), consistent with the interface doc's
// existing precedent of preserving real call-site quirks rather than
// "fixing" them.
func (p *S3Provider) DeleteChunk(ctx context.Context, target string, locationID string) error {
	if target == "" {
		return fmt.Errorf("target must not be empty")
	}
	if locationID == "" {
		return fmt.Errorf("locationID must not be empty")
	}

	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(locationID),
	})
	if err != nil {
		return fmt.Errorf("deleting s3://%s/%s: %w", p.bucket, locationID, err)
	}
	return nil
}

// GetSpace returns space information for the S3 backend.
//
// Interface-design compromise: unlike Google Drive, S3 has no per-account
// storage quota API -- buckets are effectively unlimited in size (subject
// only to account-level soft limits AWS will raise on request), and
// deriving actual "used" bytes cheaply would require either an expensive
// ListObjectsV2 walk of the whole prefix or S3 Storage Lens / CloudWatch
// storage metrics (which lag by up to 24-48h and are bucket-wide, not
// prefix-scoped). Neither gives an honest, real-time per-target number
// comparable to Drive's quota API.
//
// Rather than fabricate a number, GetSpace here returns a single
// synthetic models.DriveSpaceInfo record per bucket with TotalSpace and
// FreeSpace set to math.MaxInt64 (an explicit "unlimited" sentinel,
// documented as such) and UsedSpace set to 0. AccountID is the zero
// primitive.ObjectID (S3 has no account concept to populate it with) and
// DisplayName is set to the bucket name so callers can still identify
// which backend answered. This keeps HeadBucket as a live
// existence/permissions check (so GetSpace still fails meaningfully if
// the bucket is missing or inaccessible) while being explicit that no
// real usage accounting is performed here. Real per-tenant usage
// tracking is expected to be superseded by DynamoDB-tracked logical
// usage in a later phase (see infrastructure/terraform/modules/storage's
// TODO(dynamodb) comment), at which point this method should be updated
// to read from that source instead of returning a sentinel.
func (p *S3Provider) GetSpace(ctx context.Context, target string) ([]models.DriveSpaceInfo, error) {
	if target == "" {
		return nil, fmt.Errorf("target must not be empty")
	}

	if _, err := p.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(p.bucket)}); err != nil {
		return nil, fmt.Errorf("checking bucket s3://%s: %w", p.bucket, err)
	}

	const unlimited = int64(math.MaxInt64)
	return []models.DriveSpaceInfo{
		{
			DisplayName: p.bucket,
			TotalSpace:  unlimited,
			UsedSpace:   0,
			FreeSpace:   unlimited,
			Available:   true,
		},
	}, nil
}
