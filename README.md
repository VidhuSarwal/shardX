<div align="center">

# ShardX

**Zero-knowledge, sharded object storage on AWS. Your files become noise; only your Key File can put them back together.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![AWS](https://img.shields.io/badge/AWS-S3%20·%20KMS%20·%20DynamoDB%20·%20Cognito%20·%20SQS%20·%20EventBridge%20·%20Step%20Functions-FF9900?logo=amazonaws&logoColor=white)](infrastructure/terraform)
[![Terraform](https://img.shields.io/badge/IaC-Terraform-7B42BC?logo=terraform&logoColor=white)](infrastructure/terraform)
[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

</div>

---

ShardX is a Go + React storage service built on AWS. Every upload is obfuscated with a ChaCha20-DRBG noise stream, split into shards, and written to S3 under a KMS customer-managed key. Shard integrity metadata lives in DynamoDB, users in Cognito, failed shard uploads are retried through SQS, and every lifecycle step emits to EventBridge and is tracked by Step Functions — all provisioned by Terraform in `infrastructure/`.

The server never holds the obfuscation seed or the chunk map. They live in a `.2xpfm.key` file that only you download. A full compromise of the bucket *and* the database yields nothing readable.

A **Google Drive backend** is also included for running without an AWS account: the same pipeline can shard a file across several linked Drive accounts (3 × 15 GB free → 45 GB).

---

## Architecture

```
                         ┌────────────────────────────── AWS ──────────────────────────────┐
  React (Vite)           │                                                                  │
  ┌────────────┐ chunks  │  Go API ──▶ obfuscate ──▶ split ──▶ S3 (SSE-KMS CMK)             │
  │ /  /guide  │         │    │                       │                                    │
  │ /files     │────────▶│    │                       │         <session>/chunk_NNN.2xpfm  │
  │ /files/:id │  poll   │    │                       ├──▶ DynamoDB  shard sha256/size/status│
  │ /profile   │◀────────│    │                       ├──▶ EventBridge  FILE_CREATED,         │
  └────────────┘         │    │                       │                 SHARD_UPLOADED, …     │
        ▲                │    │                       ├──▶ Step Functions  execution / upload │
        │ JWT            │    │                       └──▶ SQS ──▶ cmd/shardworker (retry,    │
        ▼                │    │                                    DLQ + CloudWatch alarm)    │
     Cognito ◀───────────│────┘  auth                                                         │
                         │  CloudTrail · CloudWatch Logs · AWS Budgets · (OpenSearch, opt-in) │
                         └────────────────────────────────────────────────────────────────────┘
                                          ▼
                              .2xpfm.key  (seed + chunk map — client only)
```

1. **Upload** — The browser streams the file in 5 MB chunks to an upload session, previews a distribution plan, and finalizes. Finalize returns immediately; processing is async and the UI polls status.
2. **Obfuscate** — A 32-byte seed initializes a ChaCha20-DRBG that injects deterministic noise at calculated offsets throughout the file.
3. **Shard & store** — The obfuscated stream is split by the chosen strategy and each shard is `PutObject`-ed to S3 with SSE-KMS. Its SHA-256, size, bucket and region are persisted to DynamoDB as a `ShardRecord`.
4. **Retry & track** — A shard that fails to upload is queued to SQS and marked `pending`; `cmd/shardworker` drains the queue, patches the key file, and completes the session. Every step emits an EventBridge event and is tracked by a Step Functions execution.
5. **Integrity Engine** — `GET /api/files/{id}/health | shards | timeline` serve a health score, a shard placement map, and an audit timeline straight from DynamoDB.
6. **Key File** — Download the `.2xpfm.key`. Reconstruction from it is on the [Roadmap](#roadmap).

Full as-built detail: [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

---

## AWS Services

| Service | Role in ShardX | Code |
| :--- | :--- | :--- |
| **S3 + KMS** | Shard storage; one bucket, SSE-KMS with a customer-managed key, keys `<session_id>/chunk_NNN.2xpfm` | `internal/storage/s3_provider.go` |
| **DynamoDB** | Users, drive accounts, sessions, OAuth state, shard integrity records (5 on-demand tables; `DB_PROVIDER=dynamodb` needs no MongoDB) | `internal/metadatastore/dynamodb_store.go` |
| **Cognito** | User pool + app client; API validates JWTs against the pool's JWKS | `internal/authprovider/cognito_provider.go` |
| **SQS (+ DLQ)** | Shard-upload retry queue consumed by `cmd/shardworker`; dead-letters after 5 receives | `internal/queue`, `internal/worker` |
| **EventBridge** | Domain events on bus `shardx-events`: `FILE_CREATED`, `SHARD_UPLOADED`, `SHARD_VERIFIED`, … | `internal/events` |
| **Step Functions** | One execution per upload for lifecycle tracking | `internal/orchestration` |
| **CloudWatch + CloudTrail** | Log group, DLQ-depth alarm, API audit trail | `modules/observability` |
| **AWS Budgets + SNS** | Spend guard-rail with e-mail alert | `modules/budget` |
| **OpenSearch** | Opt-in metadata search (`t3.small.search`), separate Terraform root | `infrastructure/terraform/search` |
| **IAM** | Least-privilege `api-role`, `shard-worker-role`, `security-worker-role` scoped to the bucket/tables/queue; deployer policies in `infrastructure/iam` | `modules/identity` |
| **EC2 + CloudFront** | One host (api + shardworker via docker compose) behind a CloudFront distribution that also serves the web app from a private S3 bucket (OAC); env in SSM Parameter Store, ops via SSM | `modules/compute`, `deploy/` |

Each integration sits behind an interface (`StorageProvider`, `MetadataStore`, `AuthProvider`, `Emitter`, `Orchestrator`, `Queue`) chosen at boot by `internal/bootstrap` from env vars, so the API and worker share one wiring path.

---

## Features

- **Zero-knowledge** — Seed and chunk map exist only in the Key File; server-side data is useless without it.
- **ChaCha20-DRBG obfuscation** — Shards are indistinguishable from random noise.
- **Encrypted at rest, twice** — SSE-KMS on every S3 object; AES-256-GCM on any stored OAuth token.
- **Self-healing uploads** — SQS retry with DLQ and alarm; sessions stay `processing` until every shard is verified.
- **Integrity Engine** — Per-file health %, shard map with placement (bucket/region), and audit timeline.
- **Chunking strategies** — Greedy, Balanced, Proportional, or Manual sizes, with a plan preview before finalize.
- **Infrastructure as code** — One `terraform apply` provisions budget → storage → orchestration → identity → observability.
- **Drive fallback** — Pool multiple Google Drive accounts when running without AWS.

---

## Tech Stack

| Layer | Technology |
| :--- | :--- |
| Cloud | AWS: S3, KMS, DynamoDB, Cognito, SQS, EventBridge, Step Functions, CloudWatch, CloudTrail, Budgets, OpenSearch · Terraform |
| Backend | Go 1.24, AWS SDK for Go v2, `net/http` |
| Frontend | React 18, TypeScript, Vite, TanStack Query, React Router, Tailwind CSS, shadcn/ui |
| Cryptography | ChaCha20-DRBG, AES-256-GCM, SHA-256, bcrypt |
| Fallback mode | Google Drive API v3, MongoDB, HS256 JWT |

---

## Getting Started

### 1. Provision AWS

```bash
cd infrastructure/terraform
# edit terraform.tfvars: region, budget e-mail, name suffix
terraform init && terraform apply   # ~32 resources
terraform output                    # bucket, table prefix, queue URL, bus, state machine ARN, Cognito ids
```

IAM policies for the deployer user are in `infrastructure/iam/` (`ShardXInfra`, `ShardXIAM`, `ShardXOps`). Runbook: `infrastructure/terraform/README.md`. For local development without an account, `docker-compose.localstack.yml` in `apps/api` stands in for S3 and SQS.

### 2. Backend

```bash
cd apps/api
cp .env.example .env
```

Point the API at what Terraform created:

```dotenv
AWS_PROFILE=shardx
AWS_REGION=us-east-1
STORAGE_PROVIDER=s3          S3_BUCKET=<output>   S3_KMS_KEY_ID=alias/shardx-shards
DB_PROVIDER=dynamodb         DYNAMODB_TABLE_PREFIX=shardx
AUTH_PROVIDER=cognito        COGNITO_USER_POOL_ID=<output>   COGNITO_CLIENT_ID=<output>
EVENT_BUS_NAME=shardx-events STATE_MACHINE_ARN=<output>      SQS_QUEUE_URL=<output>

BASE_URL=http://localhost:5555       # this server
FRONTEND_URL=http://localhost:5173   # web app; OAuth callback redirects here
JWT_SECRET=…  TOKEN_ENC_KEY=…        # TOKEN_ENC_KEY: base64 of 32 bytes, see generate_key.sh
```

```bash
go mod tidy
go run cmd/server/main.go        # API on :5555
go run cmd/shardworker/main.go   # SQS retry worker (shares UPLOAD_TEMP_DIR with the API)
```

### 3. Frontend

```bash
cd apps/web
cp .env.example .env     # VITE_API_BASE_URL (defaults to http://localhost:5555)
npm install
npm run dev              # http://localhost:5173
npm test && npm run lint
```

Routes: `/login`, `/signup`, `/files` (upload), `/files/:sessionId` (health, shard map, timeline), `/profile` (storage targets). Page → endpoint map in `apps/web/README.md`; HTTP contract in `apps/api/API_REFERENCE.md`.

### 4. Deploy to AWS

`deploy/deploy.sh` is the whole pipeline: `terraform apply` (adds an EC2 host + CloudFront on top of the base infra) → build the web app → sync to the private S3 web bucket → invalidate CloudFront → rebuild the API/worker on the host over SSM → `/health` smoke check.

```bash
# once: attach infrastructure/iam/ShardXCompute.json to the deployer user (EC2, CloudFront, SSM)
AWS_PROFILE=shardx ./deploy/deploy.sh          # full deploy; prints https://<id>.cloudfront.net
AWS_PROFILE=shardx ./deploy/deploy.sh web      # web only
AWS_PROFILE=shardx ./deploy/deploy.sh api      # api + worker only (git pull + docker compose build on the host)
```

Topology: one CloudFront distribution serves the SPA from S3 and proxies `/api/*`, `/oauth2/*`, `/health` to a `t3.small` running `deploy/docker-compose.yml` (api + shardworker; all state is in DynamoDB/S3/Cognito) — same origin, so no CORS and `BASE_URL == FRONTEND_URL`. The host has no SSH; use SSM Session Manager. App env lives in SSM Parameter Store (`/shardx/app-env`); edit it and run `deploy.sh api` to roll.

### Running on Google Drive instead

Leave the `*_PROVIDER` vars unset, set `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET`, and register `BASE_URL/oauth2/callback` as a redirect URI in your Google OAuth client. Link accounts from `/profile`.

---

## Project Structure

```
shardx/
├── apps/
│   ├── api/                        # Go backend
│   │   ├── cmd/server/             # HTTP API entry point
│   │   ├── cmd/shardworker/        # SQS retry worker
│   │   └── internal/
│   │       ├── bootstrap/          # env → provider wiring (shared by server + worker)
│   │       ├── storage/            # StorageProvider: s3 | drive
│   │       ├── metadatastore/      # MetadataStore: dynamodb | mongo
│   │       ├── authprovider/       # AuthProvider: cognito | custom
│   │       ├── events/             # EventBridge emitter
│   │       ├── orchestration/      # Step Functions
│   │       ├── queue/, worker/     # SQS retry pipeline
│   │       ├── integrity/          # Health scoring
│   │       ├── filehandlers/       # Upload, finalize, status, key file, health routes
│   │       ├── fileprocessor/      # Obfuscation, chunking, key file
│   │       ├── auth/, oauth/, drivemanager/, store/   # Drive/Mongo/JWT fallback
│   │       └── middleware/, models/
│   └── web/                        # React app (Vite + shadcn/ui)
│       └── src/{pages,components,lib}   # lib/api.ts is the only place that talks to the API
├── deploy/                         # deploy.sh + production docker-compose (runs on the EC2 host)
├── infrastructure/
│   ├── terraform/                  # Root module + modules/{budget,storage,orchestration,identity,observability,compute}
│   │   └── search/                 # Standalone OpenSearch root
│   └── iam/                        # Deployer IAM policies
├── docs/                           # ARCHITECTURE.md, PLAN.md, TODO.md
└── UPDATE.md                       # What changed from Vcrypt
```

---

## Security Model

- **Client-held secrets** — The Key File (seed + chunk map) is generated server-side, handed to the client once, and never persisted. The server keeps only bcrypt password hashes (custom auth) and AES-256-GCM-encrypted OAuth tokens (Drive mode).
- **Encrypted objects** — Every shard is written with SSE-KMS under a customer-managed key; the API and worker roles get `kms:Decrypt` / `kms:GenerateDataKey*` on that key alone.
- **Least privilege** — Terraform-managed roles are scoped to the single bucket, the `shardx*` tables, and the one queue. Deployer policies are namespaced to `shardx-*` resources.
- **Auditability** — CloudTrail on the account, EventBridge domain events, and a per-file audit timeline served from shard records.
- **Temp files** — The upload is spooled to disk only until sharding completes, then scrubbed (`TEMP_FILE_CLEANUP_MINUTES`).

> **A full breach of S3 + DynamoDB exposes nothing useful.** Objects are noise without the seed; the seed is only in your Key File.

⚠️ **The Key File is the single point of trust. There is no server-side recovery. Losing it means permanent loss of access to that file.**

---

## Key File Format

The `.2xpfm.key` file is a JSON document that stays with you. It contains the obfuscation seed and the chunk map required for reconstruction.

```json
{
  "version": "1.0",
  "original_filename": "backup.tar",
  "original_size": 6291579,
  "processed_size": 6920736,
  "obfuscation": {
    "algorithm": "ChaCha20-DRBG",
    "seed": "<base64, 32 bytes>",
    "block_size": 1024,
    "overhead_pct": 0.1,
    "min_gap": 100
  },
  "chunks": [
    {
      "chunk_id": 1,
      "drive_account_id": "<S3 bucket, or linked Drive account id>",
      "drive_file_id": "<S3 object key, or Drive file id>",
      "filename": "chunk_001.2xpfm",
      "start_offset": 0,
      "end_offset": 5242880,
      "size": 5242880,
      "checksum": "<sha256>"
    }
  ],
  "created_at": "2026-09-20T12:03:19Z"
}
```

---

## Limitations

- No download/reconstruct endpoint yet — the key file is produced, but restoring a file from it is a manual/offline step for now.
- The Step Functions definition is a placeholder (Pass states) used for tracking, not for driving transitions.
- S3 mode reports space as "Unlimited" (no usage accounting).
- The full file is spooled to server disk before sharding, capping file size at available disk (`MAX_FILE_SIZE_GB`).

See `docs/ARCHITECTURE.md` → *Known gaps* for the current list.

---

## Roadmap

- [x] S3 + KMS shard storage, DynamoDB shard records, SQS retry worker
- [x] EventBridge domain events, Step Functions tracking, CloudWatch/CloudTrail/Budgets via Terraform
- [x] Integrity Engine (health, shard map, audit timeline)
- [ ] Download / reconstruct endpoint (fetch shards → strip noise → original file)
- [ ] Real Step Functions state machine with task tokens
- [ ] Byte-level re-verification job (re-hash shards from S3)
- [ ] Unlink a storage account
- [ ] Browser-side obfuscation via WebAssembly

---

## Contributing

PRs are welcome. For significant changes, open an issue first to discuss what you'd like to change.

```bash
# Backend
cd apps/api && go test ./... && bash test_routes.sh   # unit tests + HTTP smoke test

# Frontend
cd apps/web && npm test && npm run lint && npm run build

# Infrastructure
cd infrastructure/terraform && terraform fmt -check && terraform validate
```

---

## License

MIT — free to use, self-host, and modify.

---

<div align="center">
<sub>Sharded, obfuscated, zero-knowledge — on AWS.</sub>
</div>
