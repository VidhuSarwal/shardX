# Vcrypt → ShardX: what changed

ShardX is Vcrypt renamed and extended with an optional AWS-native mode.
The original Google-Drive/MongoDB/JWT path is untouched and remains the
default; everything below is opt-in via env vars.

## Added

- **Provider interfaces**: `StorageProvider`, `MetadataStore`, `AuthProvider`
  with the existing Drive/Mongo/JWT code wrapped as the default
  implementations (`internal/storage`, `internal/metadatastore`,
  `internal/authprovider`).
- **AWS implementations**: S3 (multipart, SSE-KMS), DynamoDB, Cognito (JWKS).
- **Orchestration & eventing**: EventBridge domain events, Step Functions
  execution per upload, SQS + DLQ shard-upload retries with a standalone
  `cmd/shardworker` consumer.
- **Integrity Engine**: per-shard SHA256 records persisted at upload time;
  `GET /api/files/{id}/health`, `/shards`, `/timeline`.
- **Frontend**: `/files/:sessionId` page with File Health card, Shard Map
  and Audit Timeline.
- **Infrastructure as code**: Terraform for budget, S3+KMS, DynamoDB,
  Cognito+IAM, EventBridge/SQS/Step Functions, CloudWatch/CloudTrail, and a
  destroy-between-sessions OpenSearch root.
- **Docs**: `docs/PLAN.md`, `docs/TODO.md`, `docs/ARCHITECTURE.md`.

## Changed

- Repo layout: `shardx_backend/` → `apps/api/`, `shardx_frontend/` → `apps/web/`.
- `filehandlers` reads drive spaces through `StorageProvider.GetSpace`
  (identical behavior in Drive mode) and routes S3-mode chunk uploads
  through the provider.
- `MetadataStore` gained `SaveShardMetadata`, `GetShardMetadata`,
  `UpdateShardStatus`; Mongo got a `shard_metadata` collection.

## Not changed

- Obfuscation (ChaCha20-DRBG), chunking strategies, key-file format
  (`drive_file_id` now holds an S3 object key in S3 mode), OAuth flow,
  JWT auth, and every existing route's request/response shape.
