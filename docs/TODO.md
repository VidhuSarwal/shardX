# ShardX AWS Migration — Status & Handoff

This file tracks exactly what's done and what's left on the Phase 1 AWS
migration described in `docs/PLAN.md`. Update it whenever you finish or
start a task, so any agent/session can pick up cold without re-deriving
context. Treat this as the source of truth over memory/chat history.

## How to resume

1. Read `docs/PLAN.md` for the full context/architecture decisions.
2. Read the checklist below — anything not `[x]` is remaining work.
3. `cd apps/api && go build ./... && go vet ./... && go test ./...`
   should always be clean on `main`. If it isn't, something got merged
   wrong — fix that before adding new work.
4. If Docker is available, LocalStack-gated integration tests exist for
   S3/DynamoDB (`LOCALSTACK=1 go test ./...`) — see
   `apps/api/docker-compose.localstack.yml`. Always `docker compose
   down` when finished so nothing idles on the machine.
5. Keep committing after each discrete unit of work, not in one giant
   batch at the end.
6. AWS credentials: `AWS_PROFILE=shardx` (account 694282668761, user `shardx-deployer`) is configured locally and Terraform is applied. Everything before 2026-09-20 was built against mocks + LocalStack; live E2E status is tracked under "Remaining" below.

## Running locally (verified 2026-09-20)

```
brew services start mongodb-community      # DB name is hardcoded to "complete" in internal/store
cd apps/api && cp .env.example .env         # fill JWT_SECRET, TOKEN_ENC_KEY (openssl rand -base64 32)
go run ./cmd/server                         # :5555 (override with PORT=…)
cd apps/web && npm install && npm run dev   # :5173
```

Drive-mode uploads need real `GOOGLE_CLIENT_ID/SECRET`; S3-mode uploads need
`terraform apply` + `AWS_PROFILE` (see `apps/api/.env.example`). Everything
else (signup/login, `/api/files/{id}/health|shards|timeline`, the
`/files/:sessionId` page) was verified locally against MongoDB with seeded
shard records.

## Checklist

### Group 0 — Foundational interfaces ✅ done
- [x] `StorageProvider` interface + Drive wrapper (`internal/storage/`)
- [x] `MetadataStore` interface + Mongo wrapper (`internal/metadatastore/`)
- [x] `AuthProvider` interface + custom-JWT wrapper (`internal/authprovider/`)
- [x] Env switches in `cmd/server/main.go` (`STORAGE_PROVIDER`, `DB_PROVIDER`, `AUTH_PROVIDER`)
- [x] Regression tests proving zero behavior change

### Group 1 — AWS provider implementations ✅ done
- [x] S3 `StorageProvider` (`internal/storage/s3_provider.go`) — multipart upload, SSE-KMS, GetSpace returns an "unlimited" sentinel (S3 has no quota API)
- [x] DynamoDB `MetadataStore` (`internal/metadatastore/dynamodb_store.go`) — see file for table/GSI schema handoff to Terraform
- [x] Cognito `AuthProvider` (`internal/authprovider/cognito_provider.go`) — needs a real User Pool to test signup/login end-to-end; JWKS validation is fully tested offline

### Group 2 — Orchestration & eventing
- [x] EventBridge domain events (`internal/events/`) — FILE_CREATED, SHARD_UPLOADED, SHARD_VERIFIED wired into the upload pipeline
- [x] Step Functions execution tracking (`internal/orchestration/`) — starts an execution per upload for visibility; the ASL is still placeholder Pass states (see `infrastructure/terraform/modules/orchestration/step_functions.tf`), so no per-state task-token integration yet
- [x] Integrity engine + health endpoint (`internal/integrity/`, `GET /api/files/{session_id}/health`) — persisted-status check, not a full re-hash (documented as an honest scope limit)
- [x] SQS + DLQ for shard upload retries — `internal/queue/` (Queue + SQSQueue + NoopQueue), `internal/worker/` (retry consumer), `cmd/shardworker/main.go`. In S3 mode a failed chunk upload is enqueued (chunk file kept on disk, shard record `pending`, session stays `processing`); the worker re-uploads, patches the key file, marks the shard `verified`, and completes the session when nothing is pending. `SQS_QUEUE_URL` unset → `NoopQueue` → original fail-fast. Drive-mode path untouched. Added `MetadataStore.UpdateShardStatus` and `internal/bootstrap` (provider selectors shared by both binaries).

### Group 3 — Infrastructure as Code ✅ done (applied 2026-09-20)
- [x] Terraform modules: budget, storage (S3+KMS), identity (Cognito+IAM), orchestration (EventBridge/SQS/Step Functions), observability (CloudWatch/CloudTrail) — `infrastructure/terraform/`
- [x] Standalone OpenSearch module (`infrastructure/terraform/search/`) — apply/destroy independently per dev session for cost control
- [x] Apply/destroy runbook in `infrastructure/terraform/README.md`
- [x] DynamoDB table definitions — `infrastructure/terraform/modules/storage/dynamodb.tf` (5 on-demand tables, GSIs w/ ALL projection, oauth-states TTL, CMK encryption). Table ARNs + retry queue ARN now feed the identity module so api-role/shard-worker-role get scoped DynamoDB + SQS grants. `terraform validate` passes.
- [x] Applied to account 694282668761 on 2026-09-20 (53 resources, `terraform plan` clean). Bucket overridden via `terraform.tfvars` (`shardx-694282668761-shards`); state is local + gitignored.

### Group 4 — Frontend ✅ done
- [x] Backend: `GET /api/files/{session_id}/shards` and `/timeline` added next to `/health` (timeline is derived from the persisted session + shard records — no separate event store)
- [x] API client additions in `apps/web/src/lib/api.ts` (`getFileHealth`, `getFileShards`, `getFileTimeline` + types)
- [x] Shard Map component (`src/components/ShardMap.tsx`)
- [x] File Health card (`src/components/FileHealthCard.tsx`)
- [x] Audit Timeline view (`src/components/AuditTimeline.tsx`)
- [x] `/files/:sessionId` page (`src/pages/FileDetail.tsx`), linked from completed uploads; polls while shards are `pending`
- [x] Vitest tests for the render-free logic in `src/lib/fileDetail.ts` (`src/lib/__tests__/fileDetail.test.ts`), same pattern as `oauth.test.ts`

### Group 5 — Repo restructuring & docs ✅ done
- [x] `docs/PLAN.md` (this migration's plan, copied from the planning session)
- [x] `docs/TODO.md` (this file)
- [x] Move `shardx_backend/` → `apps/api/`, `shardx_frontend/` → `apps/web/` (`git mv`; README/PROJECT_DEEP_DIVE/TODO paths updated; no script had a hardcoded cross-dir path)
- [x] `docs/ARCHITECTURE.md` describing what was actually built
- [x] `UPDATE.md` summarizing Vcrypt → ShardX changes

## Live AWS verification (2026-09-20, account 694282668761)

- [x] S3 mode: `cmd/server` + `cmd/shardworker` with `STORAGE_PROVIDER=s3 SQS_QUEUE_URL=…`. Real upload → `complete`, `/health` 100%, `/shards` `verified`, object in S3 with SSE-KMS, Step Functions execution `SUCCEEDED`. Retry path verified by pointing the server at a bogus `S3_KMS_KEY_ID`: chunk enqueued (`pending`, `SHARD_QUEUED_FOR_RETRY`), worker re-uploaded, patched key file, session `complete`, queue + DLQ empty.
- [x] Cognito mode: `AUTH_PROVIDER=cognito` now actually routes signup/login/middleware through the provider (it was constructed and discarded before). Live signup → CONFIRMED user (in-code `AdminConfirmSignUp`, no manual step), login → RS256 ID token, authed routes 200, tampered token 401. Middleware maps `sub` → deterministic `ObjectID` (`subToObjectID`), see known gaps.
- [x] DynamoDB store: all 5 tables match the code's schema; all 20 `MetadataStore` methods pass against the real tables. `DB_PROVIDER=dynamodb` through HTTP is only partially effective — see known gaps.
- [ ] Drive-mode regression run (`tester2.sh`) — blocked on real `GOOGLE_CLIENT_ID/SECRET` (only placeholders in `.env`).

## Known gaps / honest limitations to carry forward

- Cognito users have no `users` document: `AuthMiddleware` derives the context `ObjectID` from `sha256(sub)[:12]`, which is enough for uploads/listing but `oauth.DriveLinkHandler` (update by `_id`) finds nothing. Proper fix: `FindOrCreateUserByCognitoSub` on `MetadataStore` and put the real `_id` in the context. Also `email_verified` stays `false` after `AdminConfirmSignUp`, so Cognito forgot-password won't work for these users.
- `DB_PROVIDER=dynamodb` is half-wired: `internal/auth` (users), `internal/fileprocessor/session.go` (upload sessions), `internal/oauth` (states/drive accounts) and `filehandlers.GetUploadStatusHandler` still call `internal/store` (Mongo) directly — 18 call sites in 4 files. Consequence: in dynamodb mode shard rows land in DynamoDB but sessions/users land in Mongo, so `/api/files/{id}/health|shards|timeline` 404 ("session not found"). Routing those call sites through `metaStore` is the remaining work for a real DynamoDB cutover.
- Retry worker race: the retry job is enqueued before the key file is written, so the first SQS delivery always fails ("session has no key file yet") and burns one of 5 receives + a 60 s visibility delay. Fix: enqueue after `UpdateSessionKeyFile` in `processAndUploadFile`.
- `UPLOAD_TEMP_DIR` is not session-scoped: chunk/key files for two concurrent sessions with the same filename clobber each other (pre-existing in `internal/fileprocessor`).
- With one S3 "drive", every strategy yields exactly one shard per file.
- `shardx-deployer` lacks `dynamodb:ListTables` (per-table ops work).
- S3 `GetSpace` returns an unlimited sentinel, not real usage — real usage should come from DynamoDB-tracked logical size later.
- Step Functions execution ARic is logged, not persisted on `UploadSession` (no field exists for it yet).
- Integrity health check is presence/status-based, not a re-hash of shard bytes.
- Only the S3 storage path has been exercised against real AWS (Mongo + custom JWT still in front of it). DynamoDB store and Cognito auth have not been run live.
