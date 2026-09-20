<div align="center">

# ShardX

**Your Google Drive storage limit is a lie. You have much more.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev/)
[![MongoDB](https://img.shields.io/badge/MongoDB-Driver_1.17-47A248?logo=mongodb&logoColor=white)](https://mongodb.com/)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

---

Every Google account comes with 15 GB free. Most people have three. ShardX pools them into a single private drive, splits your files into fragments, injects deterministic noise using ChaCha20-DRBG, and distributes the pieces across your accounts. Google only ever sees meaningless fragments. Reconstruction is only possible with your **Key File** — which never touches the server.

---

## How It Works

```
Upload file → Obfuscate (ChaCha20-DRBG) → Split into fragments → Distribute across drives
                                                                          ↓
Reconstruct ← Strip noise ← Fetch fragments ← Provide .2xpfm.key file
```

1. **Obfuscate** — A 32-byte seed initializes a ChaCha20-DRBG that injects deterministic noise at calculated offsets throughout the file.
2. **Split & Distribute** — The obfuscated file is broken into fragments using your chosen strategy and uploaded in parallel across your linked Google Drive accounts.
3. **Reconstruct** — Provide your `.2xpfm.key` file. ShardX fetches all fragments, strips the noise, and restores your original file exactly.

---

## Features

- **Storage aggregation** — 3 accounts × 15 GB = 45 GB free. 5 accounts = 75 GB free.
- **Zero-knowledge server** — Key File, seed, and chunk map never leave your hands.
- **ChaCha20-DRBG obfuscation** — Fragments are indistinguishable from random noise.
- **Multiple chunking strategies** — Greedy, Balanced, Proportional, or Manual distribution.
- **OAuth aggregation** — Link and manage multiple Google accounts in one session.
- **AES-256-GCM token encryption** — OAuth tokens encrypted at rest; useless without the server ENV key.

---

## Tech Stack

| Layer | Technology |
| :--- | :--- |
| Backend | Go 1.24, MongoDB, JWT |
| Frontend | React 18, TypeScript, Vite |
| Styling | Tailwind CSS, shadcn/ui |
| Cryptography | ChaCha20-DRBG, AES-256-GCM, Bcrypt |
| Cloud | Google Drive API v3 |

---

## Getting Started

### Prerequisites

- Go 1.24+
- Node.js 18+
- MongoDB (local or [Atlas](https://mongodb.com/atlas) free tier)
- Google Cloud project with Drive API + OAuth credentials ([guide](https://console.cloud.google.com))

### Backend

```bash
cd shardx_backend
cp .env.example .env
# Fill in: MONGO_URI, JWT_SECRET, GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET
go mod tidy
go run cmd/server/main.go
# Runs on :5555
```

### Frontend

```bash
cd shardx_frontend
npm install
npm run dev
# Runs on :5173
```

---

## Project Structure

```
shardx/
├── shardx_backend/
│   ├── cmd/server/             # Entry point
│   └── internal/
│       ├── auth/               # JWT, login, signup
│       ├── fileprocessor/      # Obfuscation, chunking, key file generation
│       ├── oauth/              # Google OAuth2 flow
│       └── store/              # MongoDB repository layer
├── shardx_frontend/
│   └── src/                   # React app
└── README.md
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
  "obfuscation": {
    "algorithm": "ChaCha20-DRBG",
    "seed": "...",
    "block_size": 256
  },
  "chunks": [
    {
      "chunk_id": 1,
      "drive_file_id": "google_drive_id_here",
      "start_offset": 0,
      "end_offset": 5242880
    }
  ]
}
```

---

## Limitations

- The full file is written to the server disk temporarily before splitting, capping the max file size to available server storage.
- Very large files may cause the `/finalize` request to time out during processing. A worker queue (Redis/RabbitMQ) is the recommended fix.

---

## Roadmap

- [ ] Worker queue for async finalize processing
- [ ] Browser-side obfuscation via WebAssembly
- [ ] OneDrive support
- [ ] Dropbox support
- [ ] S3 support

---

## Contributing

PRs are welcome. For significant changes, open an issue first to discuss what you'd like to change.

```bash
# Run API test suite
cd shardx_backend
bash test_routes.sh
```

---

## License

[MIT](LICENSE) — free to use, self-host, and modify.

---

<div align="center">
<sub>Built with the belief that your free storage limit shouldn't be 15 GB.</sub>
</div>
