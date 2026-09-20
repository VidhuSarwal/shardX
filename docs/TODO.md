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
6. AWS credentials are intentionally **not** wired up yet — the user
   wants to hand those over only at the very end, right before Terraform
   actually gets applied / before live Cognito-pool testing. Don't block
   on needing them; everything so far has been built against mocks and
   LocalStack.

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

### Group 3 — Infrastructure as Code ✅ done (not yet applied to real AWS)
- [x] Terraform modules: budget, storage (S3+KMS), identity (Cognito+IAM), orchestration (EventBridge/SQS/Step Functions), observability (CloudWatch/CloudTrail) — `infrastructure/terraform/`
- [x] Standalone OpenSearch module (`infrastructure/terraform/search/`) — apply/destroy independently per dev session for cost control
- [x] Apply/destroy runbook in `infrastructure/terraform/README.md`
- [x] DynamoDB table definitions — `infrastructure/terraform/modules/storage/dynamodb.tf` (5 on-demand tables, GSIs w/ ALL projection, oauth-states TTL, CMK encryption). Table ARNs + retry queue ARN now feed the identity module so api-role/shard-worker-role get scoped DynamoDB + SQS grants. `terraform validate` passes.
- [ ] **Not yet applied to real AWS at all.** No `terraform apply` has been run. The default S3 bucket name (`shardx-prod-shards`) will collide globally — must be overridden via tfvars before first apply.

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

## Known gaps / honest limitations to carry forward

- Cognito `AuthMiddleware` puts the Cognito `sub` (UUID) in the same context key handlers expect to be a Mongo `ObjectID` — unresolved until Cognito auth is actually wired into a live route.
- S3 `GetSpace` returns an unlimited sentinel, not real usage — real usage should come from DynamoDB-tracked logical size later.
- Step Functions execution ARic is logged, not persisted on `UploadSession` (no field exists for it yet).
- Integrity health check is presence/status-based, not a re-hash of shard bytes.
- No end-to-end test against real AWS has been run anywhere in this migration — only LocalStack + mocks. First real integration will surface real IAM/bucket-naming/Cognito-flow issues; that's expected, not a regression.
