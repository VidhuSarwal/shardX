# ShardX — AWS-Native Phase 1 Migration

## Context

ShardX (freshly renamed from Vcrypt, repo: `/Users/vidhu/Documents/shardX`) currently fragments an obfuscated file across multiple Google Drive accounts, using MongoDB for metadata and a custom JWT/OAuth auth system. The target spec ("SHARDX — AWS-Native Product Update") re-architects this into an AWS-native distributed storage control plane: S3 as the storage fabric, KMS for encryption, DynamoDB for metadata, Cognito for auth, Step Functions/SQS/EventBridge for orchestration, plus later phases of security intelligence (GuardDuty/Macie), search (OpenSearch), and AI (Bedrock).

Given hackathon time/cost constraints, this plan targets **Phase 1 MVP only**: a working AWS-native core added *alongside* the existing Drive/Mongo/JWT system via env-driven switches (no destructive rip-out), so the app never breaks and can demo either mode. Later phases (GuardDuty, Macie, Bedrock, Athena, QuickSight, full OpenSearch vector search) are explicitly deferred and listed at the end as future work, not built now.

**Cost reality (confirmed with user):** this is for hackathon-style intermittent use, not always-on production.
- S3, KMS, DynamoDB (on-demand), Cognito, Step Functions, SQS, EventBridge, CloudWatch, CloudTrail, API Gateway at dev/test volume: **~$5-10/month total**, mostly from the $1/mo flat KMS CMK charge prorated by day (~$0.03/day) — everything else is pennies or free-tier.
- OpenSearch: **use a small managed instance-based domain (`t3.small.search`, ~$0.036/hr), not Serverless.** Serverless has a ~$175-350/mo *minimum floor billed even when idle*, which is disastrous for a "spin up only when developing" workflow. A small instance-based domain can be `terraform apply`'d before a work session and `terraform destroy`'d after — cost is purely instance-hours. Even 40 hours of active dev spread over 2 weeks ≈ **$1.50**.
- **Net estimate for the whole hackathon build/demo period, provisioning only when actively working and destroying between sessions: roughly $10-20 total**, assuming you remember to `terraform destroy` the OpenSearch domain when not using it (it's the only component with a meaningful per-hour cost).
- **Action before provisioning anything:** set an AWS Budget alert (e.g. $25 threshold) — this will be one of the Terraform/setup tasks.

**Credentials:** Do NOT paste AWS access keys into this chat — `aws configure` does not mask secret input and it would land in the transcript permanently. Instead:
1. In the AWS Console, create a dedicated IAM user (not root) scoped to this project (S3, KMS, DynamoDB, Cognito, Step Functions, SQS, EventBridge, CloudWatch, CloudTrail, API Gateway, IAM-for-role-creation, Budgets — least privilege, not AdministratorAccess).
2. Open a **separate terminal outside this terminal session** and run `aws configure --profile shardx`, pasting the access key/secret there.
3. Tell me the profile name (`shardx`); I'll use `AWS_PROFILE=shardx` for all AWS/Terraform calls. The secret itself never enters my context.

**Scope decision carried forward:** "no hard changes" (your instruction) is applied not just to Google Drive but to Mongo/JWT too — DynamoDB and Cognito are added as new, env-selectable providers alongside the existing ones, not replacements. Flag if you actually meant Drive-only.

**API hosting decision:** Keep the existing Go HTTP server process as-is (no Lambda rewrite of handlers). Front it with API Gateway (HTTP API) + Cognito JWT authorizer, proxying to the Go server running on a small EC2 instance (or reachable via a tunnel during local dev). This is the low-risk path confirmed with the user.

---

## Foundational constraint (must happen first, serially)

Both the Go backend and Mongo store are **tightly coupled with no existing interface** (confirmed via exploration: `internal/drivemanager` hardcodes Google Drive REST calls; `internal/store` is Mongo-specific; `internal/auth` is custom JWT-only). Every fan-out task below (S3, DynamoDB, Cognito implementations) depends on these interfaces existing first. This is the critical path — everything else can parallelize only after it lands.

---

## Task List

### Group 0 — Foundational interfaces (serial, blocks everything else)
- [ ] Define `StorageProvider` interface (`UploadChunk`, `DownloadChunk`, `DeleteChunk`, `GetSpace`) in `internal/storage/provider.go`; wrap existing Drive code (`internal/drivemanager`) as the `drive` implementation behind it — zero behavior change.
- [ ] Define `MetadataStore` interface covering the operations `internal/store` currently exposes (users, drive/storage accounts, upload sessions); wrap existing Mongo code as the `mongo` implementation.
- [ ] Define `AuthProvider` interface (`IssueToken`, `ValidateToken`, `Signup`, `Login`) wrapping the existing JWT/bcrypt code as the `custom` implementation.
- [ ] Add env switches: `STORAGE_PROVIDER=drive|s3`, `DB_PROVIDER=mongo|dynamodb`, `AUTH_PROVIDER=custom|cognito` (default to existing behavior if unset) in `cmd/server/main.go` wiring.
- [ ] Add Go unit tests for the wrapped Drive/Mongo/Custom implementations against the new interfaces (behavior-preserving regression tests) — this is also the safety net proving Group 0 didn't break anything.

### Group 1 — AWS provider implementations (parallel, after Group 0)
- [ ] **S3 storage provider**: implement `StorageProvider` using `aws-sdk-go-v2/s3`, with multipart upload for large chunks and SSE-KMS on PutObject. Add LocalStack-backed integration test (docker-compose LocalStack S3) plus unit tests with a mocked S3 client.
- [ ] **DynamoDB metadata store**: implement `MetadataStore` using `aws-sdk-go-v2/dynamodb` for Users/UploadSessions/ShardMetadata tables. Add LocalStack-backed integration test + unit tests with a mocked client.
- [ ] **Cognito auth provider**: implement `AuthProvider` using Cognito User Pool (signup/login via `InitiateAuth`, token validation via JWKS). Unit tests with mocked Cognito client; manual end-to-end test against the real User Pool once provisioned.

### Group 2 — Orchestration & eventing (parallel, after Group 0; can start once S3/DynamoDB shapes are known)
- [ ] EventBridge: emit domain events (`FILE_CREATED`, `SHARD_UPLOADED`, `SHARD_VERIFIED`, `RECOVERY_STARTED`, `RECOVERY_COMPLETED`) from the existing upload/finalize/download handlers — additive, behind a no-op emitter if `EVENT_BUS_NAME` unset so local dev without AWS doesn't break.
- [ ] SQS + DLQ: move shard-upload retries and checksum verification into an SQS-backed worker (only active when `STORAGE_PROVIDER=s3`); DLQ after N failed attempts.
- [ ] Step Functions: model the upload lifecycle (`VALIDATE → FRAGMENTED → UPLOAD_SHARDS → VERIFY_CHECKSUMS → REGISTER_METADATA → HEALTHY`) as a state machine that the Go server starts via `StartExecution` for S3-mode uploads; keep the existing synchronous path for `drive` mode untouched.
- [ ] Integrity engine: compute/store SHA256 per shard (already partially done — keyfile has checksums) into DynamoDB `ShardMetadata`, expose a `/api/files/{id}/health` endpoint returning the Shard Health Score JSON shape from the spec.

### Group 3 — Infrastructure as Code (parallel, after Group 0, informed by Group 1 resource names)
- [ ] Terraform modules: `networking` (VPC/SG for EC2), `identity` (Cognito User Pool + IAM roles per workload — API role, shard-worker role, per §24 least privilege), `storage` (S3 buckets with versioning + KMS CMK), `orchestration` (Step Functions, SQS+DLQ, EventBridge bus), `search` (OpenSearch — see below), `observability` (CloudWatch log groups/alarms, CloudTrail trail), plus an **AWS Budget alert module** (deploy this one first, before anything else).
- [ ] `terraform apply`/`destroy` runbook documented in `infrastructure/terraform/README.md`, explicitly calling out which resources are safe to destroy between sessions (OpenSearch domain) vs. cheap to leave running (S3/DynamoDB/KMS/Cognito).
- [ ] OpenSearch: single `t3.small.search` instance-based domain (not Serverless), indexing file metadata/shard health/events only (no plaintext) — provisioned via its own Terraform module so it can be destroyed independently between dev sessions.

### Group 4 — Frontend additions (parallel, after Group 1 API shapes stabilize)
- [ ] New API client functions in `src/lib/api.ts` for: shard placement/location (extends existing chunk model which already has `chunk_id`/size/offsets — generalize `drive_account_id` to a `storage_target` union of Drive account or S3 bucket/region), file health endpoint, audit timeline (from EventBridge-sourced log).
- [ ] **Shard Map** page/component: visualize chunk → storage target (bucket/region) using existing chunking plan data.
- [ ] **File Health** card: consume `/api/files/{id}/health`, render the "X/X shards healthy" score from the spec.
- [ ] **Audit Timeline** view: simple chronological list from stored events (DynamoDB or CloudWatch Logs Insights query — no OpenSearch dependency required for this one, per deferred-search decision below).
- [ ] Vitest component tests for the three new pieces above, following the existing pattern in `src/lib/__tests__/`.

### Group 5 — Repo restructuring (light touch, do last, low risk)
- [ ] Move Go backend into `apps/api/`, frontend into `apps/web/` per spec §34 (skip the `services/*` micro-split and multi-account org structure — not warranted for Phase 1 scope); update root README and any hardcoded relative paths/scripts (`test_routes.sh`, `tester2.sh`, CI if any).
- [ ] Add `docs/ARCHITECTURE.md` describing the Phase 1 AWS architecture actually built (not the full 38-section spec) and `UPDATE.md` summarizing what changed from Vcrypt.

### Explicitly deferred (not built this pass — listed for traceability)
- GuardDuty, GuardDuty Malware Protection for S3, Macie, SNS/SES notifications, S3 Object Lock/Intelligent-Tiering/Lifecycle/Replication, Bedrock copilot, OpenSearch vector search, Athena/Glue/QuickSight analytics, CloudFront/Route53/ACM/WAF edge, multi-account org, ECR/containerization.

---

## Verification plan

- **Unit tests (Go):** `go test ./...` in `apps/api` (or current backend path) — must pass for wrapped Drive/Mongo/Custom implementations (Group 0) and for S3/DynamoDB/Cognito mocked-client tests (Group 1).
- **Integration tests (LocalStack):** `docker-compose up localstack`, run S3 + DynamoDB integration test suite against it — validates real SDK wiring without touching real AWS or incurring cost.
- **Manual smoke test, Drive/Mongo/Custom mode (regression):** run `tester2.sh` end-to-end (upload → chunk → finalize → status → download) with env vars unset/defaulted — must behave identically to today, proving Group 0 didn't regress the working system.
- **Manual smoke test, AWS mode:** with `STORAGE_PROVIDER=s3 DB_PROVIDER=dynamodb AUTH_PROVIDER=cognito` and real (budget-alerted) AWS resources from `terraform apply`, repeat the same upload→health-check→download cycle against S3/DynamoDB/Cognito.
- **Failure-injection test:** manually delete/corrupt one S3 shard, confirm `/api/files/{id}/health` reports the failure and (if Step Functions recovery is wired) that recovery triggers.
- **Frontend:** `npm run test` (vitest) for new components; manual click-through of Shard Map / File Health / Audit Timeline pages against a running AWS-mode backend.
- **Cost control:** after each work session, `terraform destroy` the OpenSearch module (and any EC2 instance if not needed); confirm AWS Budget alert is active before the first `terraform apply`.
