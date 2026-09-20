<div align="center">

# ShardX

**Your Google Drive storage limit is a lie. You have much more.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![MongoDB](https://img.shields.io/badge/MongoDB-Driver_1.17-47A248?logo=mongodb&logoColor=white)](https://mongodb.com/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

</div>

---

Every Google account comes with 15 GB free. Most people have three. ShardX pools them into a single private drive, splits your files into fragments, injects deterministic noise using ChaCha20-DRBG, and distributes the pieces across your accounts. Google only ever sees meaningless fragments. Reconstruction is only possible with your **Key File** — which never touches the server.

---

## How It Works

```
Upload file → Obfuscate (ChaCha20-DRBG) → Split into fragments → Distribute across drives / S3
                                                                          ↓
                                              Download .2xpfm.key ← Verify shards (Integrity Engine)
```

1. **Upload** — The browser streams the file in 5 MB chunks to an upload session, then picks a distribution strategy (with a plan preview) and finalizes.
2. **Obfuscate** — A 32-byte seed initializes a ChaCha20-DRBG that injects deterministic noise at calculated offsets throughout the file.
3. **Split & Distribute** — The obfuscated file is broken into fragments using your chosen strategy and uploaded across your linked Google Drive accounts (or an S3 bucket in AWS mode). Each shard's SHA-256 is recorded so health, shard placement and an audit timeline can be shown per file.
4. **Key File** — Download the `.2xpfm.key` that holds the seed and chunk map. Reconstruction from the key file is not implemented yet (see [Roadmap](#roadmap)).

---

## Features

- **Storage aggregation** — 3 accounts × 15 GB = 45 GB free. 5 accounts = 75 GB free.
- **Zero-knowledge server** — Key File, seed, and chunk map never leave your hands.
- **ChaCha20-DRBG obfuscation** — Fragments are indistinguishable from random noise.
- **Multiple chunking strategies** — Greedy, Balanced, Proportional, or Manual distribution.
- **OAuth aggregation** — Link and manage multiple Google accounts in one session.
- **AES-256-GCM token encryption** — OAuth tokens encrypted at rest; useless without the server ENV key.
- **Integrity Engine** — Per-file health score, shard map and audit timeline (`/api/files/{id}/health|shards|timeline`).
- **Resumable, async uploads** — Chunked upload with pause/cancel; finalize returns immediately and the UI polls processing status.

---

## Tech Stack

| Layer | Technology |
| :--- | :--- |
| Backend | Go 1.24, MongoDB, JWT |
| Frontend | React 18, TypeScript, Vite, TanStack Query, React Router |
| Styling | Tailwind CSS, shadcn/ui |
| Cryptography | ChaCha20-DRBG, AES-256-GCM, Bcrypt |
| Cloud | Google Drive API v3 · optional AWS mode: S3 + KMS, DynamoDB, Cognito, SQS, EventBridge, Step Functions (Terraform in `infrastructure/`) |

---

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 18+
- MongoDB (local or [Atlas](https://mongodb.com/atlas) free tier)
- Google Cloud project with Drive API + OAuth credentials ([guide](https://console.cloud.google.com))

### Backend

```bash
cd apps/api
cp .env.example .env
# Required: MONGO_URI, JWT_SECRET, TOKEN_ENC_KEY (base64 of 32 bytes; see generate_key.sh),
#           GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, BASE_URL (this server, e.g. http://localhost:5555)
# Recommended: FRONTEND_URL=http://localhost:5173 so the OAuth callback lands on the web app
go mod tidy
go run cmd/server/main.go
# Runs on :5555
```

Register `BASE_URL/oauth2/callback` as an authorized redirect URI in your Google OAuth client.

### Frontend

```bash
cd apps/web
cp .env.example .env     # VITE_API_BASE_URL (defaults to http://localhost:5555)
npm install
npm run dev              # http://localhost:5173
npm test && npm run lint
```

Routes: `/login`, `/signup`, `/files` (upload), `/files/:sessionId` (health, shard map, timeline), `/profile` (linked drives). See `apps/web/README.md` for the page → endpoint map and `apps/api/API_REFERENCE.md` for the HTTP contract.

---

## Project Structure

```
shardx/
├── apps/
│   ├── api/                        # Go backend
│   │   ├── cmd/server/             # HTTP API entry point
│   │   ├── cmd/shardworker/        # SQS retry worker (S3 mode)
│   │   └── internal/
│   │       ├── auth/, oauth/       # JWT + Google OAuth2
│   │       ├── filehandlers/       # Upload, finalize, status, key file, health routes
│   │       ├── fileprocessor/      # Obfuscation, chunking, key file
│   │       ├── drivemanager/       # Google Drive uploads + quota
│   │       ├── store/              # MongoDB repository layer
│   │       ├── storage/            # StorageProvider: drive | s3
│   │       ├── metadatastore/      # MetadataStore: mongo | dynamodb
│   │       ├── authprovider/       # AuthProvider: custom | cognito
│   │       ├── events/, orchestration/, queue/, worker/, integrity/
│   │       └── bootstrap/          # env → provider wiring
│   └── web/                        # React app (Vite + shadcn/ui)
│       └── src/{pages,components,lib}   # lib/api.ts is the only place that talks to the API
├── infrastructure/terraform/       # AWS Phase 1 IaC (+ standalone search/)
├── docs/                           # PLAN.md, TODO.md, ARCHITECTURE.md
└── UPDATE.md                       # What changed from Vcrypt
```

---

## Security Model

The server stores only two things: hashed passwords (bcrypt) and encrypted OAuth tokens (AES-256-GCM). That's it.

The server **never** stores the Key File, the obfuscation seed, the chunk-to-drive mapping, or the original file (temp files are scrubbed immediately after distribution).

> **A full database breach exposes nothing useful.** OAuth tokens are encrypted with a server-side ENV key the attacker does not have. And without the Key File, the fragments stored on Google Drive are permanently unreadable.

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
      "drive_account_id": "<linked account id, or S3 bucket>",
      "drive_file_id": "<Drive file id, or S3 object key>",
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

- The full file is written to the server disk temporarily before splitting, capping the max file size to available server storage (`MAX_FILE_SIZE_GB`).
- No download/reconstruct endpoint yet — the key file is produced, but restoring a file from it is a manual/offline step for now.
- In S3 mode, drive space is reported as "Unlimited" (no usage accounting), and users/sessions still live in MongoDB even with `DB_PROVIDER=dynamodb`. See `docs/ARCHITECTURE.md` → Known gaps.

---

## Roadmap

- [ ] Download / reconstruct endpoint (fetch shards → strip noise → original file)
- [ ] Unlink a drive account
- [ ] Browser-side obfuscation via WebAssembly
- [ ] OneDrive support
- [ ] Dropbox support
- [x] S3 support (`STORAGE_PROVIDER=s3`, see `docs/ARCHITECTURE.md`) with SQS retry worker (`cmd/shardworker`)
- [x] Integrity Engine (health, shard map, audit timeline)

---

## Contributing

PRs are welcome. For significant changes, open an issue first to discuss what you'd like to change.

```bash
# Backend
cd apps/api && go test ./... && bash test_routes.sh   # unit tests + HTTP smoke test

# Frontend
cd apps/web && npm test && npm run lint && npm run build
```

---

## License

MIT — free to use, self-host, and modify.

---

<div align="center">
<sub>Built with the belief that your free storage limit shouldn't be 15 GB.</sub>
</div>
