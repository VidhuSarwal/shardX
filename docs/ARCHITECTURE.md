# ShardX Phase 1 Architecture (as built)

This describes what actually exists on `main`, not the full target spec.
Every AWS piece is **additive and env-selected**; with no AWS env vars set
the system is byte-for-byte the original Drive + MongoDB + custom-JWT app.

## Two runtime modes

| Concern | Default (`drive`/`mongo`/`custom`) | AWS mode |
|---|---|---|
| Shard storage | Google Drive accounts (`internal/drivemanager`) | One S3 bucket, SSE-KMS, `<session_id>/chunk_NNN.2xpfm` keys (`internal/storage/s3_provider.go`) |
| Metadata | MongoDB (`internal/store`) | DynamoDB, 5 on-demand tables (`internal/metadatastore/dynamodb_store.go`) |
| Auth | bcrypt + HS256 JWT (`internal/auth`) | Cognito User Pool, JWKS validation (`internal/authprovider/cognito_provider.go`) |
| Domain events | no-op | EventBridge bus `shardx-events` (`internal/events`) |
| Lifecycle tracking | no-op | One Step Functions execution per upload — placeholder Pass states (`internal/orchestration`) |
| Shard retry | fail whole session | SQS queue + DLQ, `cmd/shardworker` consumer (`internal/queue`, `internal/worker`) |

Selection lives in `internal/bootstrap` and is shared by `cmd/server` and
`cmd/shardworker`. Env vars: `STORAGE_PROVIDER`, `DB_PROVIDER`,
`AUTH_PROVIDER`, `EVENT_BUS_NAME`, `STATE_MACHINE_ARN`, `SQS_QUEUE_URL`
(see `apps/api/.env.example`).

## Upload pipeline

```
client ──chunks──▶ POST /upload/chunk ──▶ temp file
                   POST /upload/finalize ──▶ go processAndUploadFile()
                                              │  FILE_CREATED event, StartExecution
                                              ├─ obfuscate (ChaCha20-DRBG)
                                              ├─ GetSpace() ─▶ chunk plan
                                              ├─ split
                                              ├─ upload each chunk
                                              │    drive: drivemanager.UploadChunksToDrivers (unchanged, fail-fast)
                                              │    s3:    provider.UploadChunk; on error → SQS RetryJob, shard = pending
                                              ├─ SaveShardMetadata (sha256, size, bucket/region, status)
                                              ├─ key file (.2xpfm.key)
                                              └─ complete, or "processing" until shardworker drains pending shards
```

`cmd/shardworker` (S3 mode only): `Receive` → `UploadChunk` → patch
`drive_file_id` in the key file → `UpdateShardStatus(verified)` → delete
chunk file → if no shard is `pending`, `CompleteSession`. A job that errors
is not deleted, so SQS redelivers it and dead-letters after
`max_receive_count` (Terraform, default 5). The worker reads chunk files
from the API's `UPLOAD_TEMP_DIR`, so it must share that filesystem.

## Integrity Engine & read endpoints

All under `GET /api/files/{session_id}/…`, owner-checked against the session:

- `/health` — `HealthReport` (X/Y shards healthy). Presence/status based,
  **not** a re-hash of bytes.
- `/shards` — `ShardRecord[]` sorted by `shard_id` (placement for the Shard Map).
- `/timeline` — audit events derived from the session timestamps + shard
  records (`FILE_CREATED`, `SHARD_VERIFIED`, `SHARD_QUEUED_FOR_RETRY`,
  `SHARD_CORRUPTED`, `FILE_HEALTHY`, `UPLOAD_FAILED`). No separate event
  store; EventBridge events are emitted but not read back.

Frontend: `/files/:sessionId` (`apps/web/src/pages/FileDetail.tsx`) renders
`FileHealthCard`, `ShardMap`, `AuditTimeline`, polling every 5 s while any
shard is `pending`.

## Infrastructure (`infrastructure/terraform`)

Root module applies, in order: `budget` (AWS Budget + SNS), `storage`
(S3 + KMS CMK + DynamoDB tables), `orchestration` (EventBridge bus,
SQS + DLQ, Step Functions placeholder), `identity` (Cognito pool/client,
`api-role`, `shard-worker-role`, `security-worker-role`, scoped to the
bucket/tables/queue), `observability` (CloudWatch log group + DLQ alarm,
CloudTrail). `compute` (added after Phase 1) provisions the runtime: an EC2
`t3.small` in the default VPC running `deploy/docker-compose.yml` (api +
shardworker, shared `/data/uploads` volume; no database container) with an instance role
that unions the api/worker policies plus Cognito/EventBridge/Step Functions,
port 80 open only to CloudFront's origin-facing prefix list, no SSH (SSM);
a private S3 web bucket with OAC; and one CloudFront distribution — default
behavior → S3 (SPA fallback to `index.html`), `/api/*`, `/oauth2/*`,
`/health` → the host. The app `.env` is an SSM SecureString
(`/shardx/app-env`) rendered by Terraform, which the host polls for at boot
because it embeds the CloudFront domain. `deploy/deploy.sh` runs apply,
builds/syncs the web app, and triggers `shardx-redeploy` over SSM.
`search/` is a separate root (OpenSearch `t3.small.search`)
meant to be applied/destroyed per dev session. Runbook:
`infrastructure/terraform/README.md`.

## Known gaps (carried from `docs/TODO.md`)

- `DB_PROVIDER=dynamodb` is only half-wired: `internal/oauth`,
  `internal/fileprocessor` and `internal/auth` still call `internal/store`
  (Mongo) directly, so sessions/users live in Mongo while shard metadata
  goes to DynamoDB. `MONGO_URI` is therefore still required in every mode.
- Cognito `AuthMiddleware` derives the context `ObjectID` from
  `sha256(sub)[:12]`; no `users` document exists for Cognito users, so
  Drive linking (`oauth.DriveLinkHandler`) finds nothing for them.
- S3 `GetSpace` returns an "unlimited" sentinel; no usage accounting.
- No download/reconstruct endpoint exists on the backend; the web client
  has no download page either (only the key file can be fetched).
- Applied to a real AWS account on 2026-09-20; S3 upload/retry, Cognito
  signup/login and the DynamoDB store were verified live (`docs/TODO.md`).
