# PROJECT_DEEP_DIVE: ShardX (Fragmented Obfuscation Storage)

Welcome to the deep dive of **ShardX**. This document provides a comprehensive technical breakdown of the system, its architecture, data flows, and design philosophy.

---

## 1. 🧠 Project Overview
**ShardX** is a specialized cloud storage aggregator and security tool designed for **fragmented, obfuscated storage**. Instead of storing a sensitive file as a single entity on a single cloud provider, ShardX splits the file into multiple chunks, injects deterministic noise (obfuscation), and distributes these chunks across multiple Google Drive accounts.

### Problem it Solves
- **Single Point of Failure**: Prevents data loss if a single cloud account is compromised or banned.
- **Privacy/Obfuscation**: Even if a provider scans your "files," they only see a fragment of a file filled with noise, making reconstruction impossible without the **Key File**.
- **Storage Aggregation**: Combines free storage quotas from multiple accounts into one large virtual volume.

### Key Features
- **Deterministic Obfuscation**: Uses ChaCha20-DRBG to inject "noise" into the file before splitting.
- **Multiple Chunking Strategies**: Greedy, Balanced, Proportional, and Manual.
- **Key-File Sovereignty**: The decryption/reconstruction metadata (Key File) is never stored on the server; the user is solely responsible for it.
- **OAuth Aggregation**: Link and manage multiple Google Drive accounts in one session.

---

## 2. 🏗️ Architecture Overview

The system follows a classic **client-server decoupling** model, with a Go-based backend and a React-based frontend.

### High-Level System Architecture
```text
[ apps/web ] <---(REST API / JWT)---> [ apps/api ]
      |                                              |
      |                                      [ MongoDB (User/Sessions) ]
      |                                              |
      +------------(Direct Google OAuth)-------------+
      |                                              |
      +----------(Chunked File Reconstruction)-------+----[ Google Drive Accounts ]
```

### Separation of Concerns
- **Frontend (React)**: Handles user interaction, OAuth initiation, file chunking (for upload), and visualization of storage space.
- **Backend (Go)**: Orchestrates auth, manages MongoDB state, performs the heavy lifting of obfuscation/splitting, and interacts with the Google Drive API for chunk distribution.
- **Storage (MongoDB)**: Stores user profiles, hashed passwords, and encrypted OAuth tokens.
- **Key File (.2xpfm.key)**: The "soul" of the data; it stays with the user.

---

## 3. 🔄 End-to-End Data Flow (Upload Flow)

A typical file upload and distribution request follows these steps:

1.  **Initiation**:
    - Frontend sends filename and size to `/api/files/upload/initiate`.
    - Backend creates a `UploadSession` in MongoDB and returns a `session_id`.
2.  **Chunked Upload**:
    - Frontend uploads the original file to the backend in chunks (e.g., 5MB each) to `/api/files/upload/chunk`.
    - Backend stores these chunks in a temporary location on the server.
3.  **Strategy Selection**:
    - User selects a distribution strategy (e.g., "Balanced").
    - Frontend calls `/api/files/upload/finalize`.
4.  **Core Processing (The "Magic")**:
    - **Obfuscation**: Backend generates a 32-byte seed. It uses ChaCha20-DRBG to inject noise blocks at deterministic offsets.
    - **Splitting**: The obfuscated file is split into fragments according to the selected strategy.
    - **Distribution**: Backend parallel-uploads fragments to the various linked Google Drive accounts.
5.  **Completion**:
    - Backend generates a `.2xpfm.key` (JSON) containing the seed and chunk map.
    - User downloads the Key File.
    - Backend scrubs all temporary files from its disk.

---

## 4. ⚙️ Backend Deep Dive (apps/api)

### Tech Stack
- **Language**: Go 1.24.0
- **Database**: MongoDB (v1.17 driver)
- **Auth**: JWT (v5), bcrypt, AES-GCM (for token encryption)
- **Crypto**: `golang.org/x/crypto/chacha20`

### Folder Structure
- `cmd/server/`: Entrance point (`main.go`).
- `internal/auth/`: JWT handling and standard Login/Signup.
- `internal/fileprocessor/`:
    - `obfuscator.go`: Implementation of the ChaCha20-DRBG noise injection.
    - `chunker.go`: Logic for splitting files based on strategies.
    - `keyfile.go`: Generation of the JSON metadata file.
- `internal/oauth/`: Google OAuth2 flow and state management.
- `internal/store/`: MongoDB repository layer.

### Core Logic: Deterministic Obfuscation
The `obfuscator.go` uses a **ChaCha20 stream cipher** as a Deterministic Random Bit Generator (DRBG). 
- It calculates "injection points" based on the file size and a `defaultOverheadPct` (standard ~8%).
- It streams the file, and at each deterministic offset, it injects a block of noise.
- This creates a processed file larger than the original, where the "true" data is hidden between noise blocks.

---

## 5. 🎨 Frontend Deep Dive (apps/web)

### Tech Stack
- **Framework**: React 18 + Vite
- **Styling**: Tailwind CSS + `shadcn/ui` (Radix) + `lucide-react`
- **Data Fetching**: TanStack Query (React Query) v5
- **Forms**: React Hook Form + Zod

### Key UI Flows
- **Dashboard**: Visualizes storage usage across all linked drives.
- **Upload Wizard**: A multi-step process (Selection -> Chunking Strategy -> Processing -> Key Download).
- **Download/Reconstruct**: Accepts a `.key` file, communicates with the backend to fetch fragments, and reconstructs the file locally (or streams it).

### State Management
- **React Query**: Centralized state for API data (user profile, drive spaces, upload sessions).
- **Zustand/Context**: (Not explicitly found in large use, standard React state and hooks are preferred).

---

## 6. 🔌 Integration Layer

### API Contracts
The frontend communicates with the backend via JSON over HTTP.

#### Example: Initiate Upload
**POST** `/api/files/upload/initiate`
```json
// Request
{
  "filename": "confidential.pdf",
  "file_size": 10485760
}

// Response
{
  "session_id": "67a...",
  "upload_url": "/api/files/upload/chunk?session_id=67a...",
  "drive_spaces": [ ... ],
  "max_file_size": 107374182400
}
```

#### The Key File (.2xpfm.key) Format
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

## 7. 📦 Tech Stack Summary

| Layer | Technology |
| :--- | :--- |
| **Backend** | Go 1.24, MongoDB, JWT |
| **Frontend** | React, TypeScript, Vite |
| **Styling** | Tailwind CSS, shadcn/ui |
| **API** | REST (JSON) |
| **Cryptography** | ChaCha20-DRBG, AES-256-GCM, Bcrypt |
| **Cloud** | Google Drive API (v3) |

---

## 8. 🚀 Setup & Execution

### Prerequisites
- Go 1.24+
- Node.js 18+
- MongoDB instance (local or Atlas)
- Google Cloud Console Project (for OAuth credentials)

### Backend Setup
1.  Navigate to `apps/api/`.
2.  Copy `.env.example` to `.env` and fill in:
    - `MONGO_URI`, `JWT_SECRET`, `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`.
3.  Run: `go mod tidy`.
4.  Launch: `go run cmd/server/main.go`. (Starts on port 5555).

### Frontend Setup
1.  Navigate to `apps/web/`.
2.  Run: `npm install`.
3.  Launch: `npm run dev`. (Starts via Vite, usually port 5173).

---

## 9. 🧩 Key Design Decisions

1.  **Deterministic Noise vs. Random Encryption**:
    - *Decision*: Inject noise deterministically rather than encrypting the whole file.
    - *Rationale*: Allows for simpler streaming reconstruction if the user has the seed. It also makes the file look like "corrupted data" or "random noise" rather than a ciphertext block that invites brute-force attempts.
2.  **Key File Sovereignty**:
    - *Decision*: Never store the mapping of file-chunks-to-drives on the server.
    - *Rationale*: Maximum security. If the ShardX database is hacked, the hacker only gets encrypted OAuth tokens (useless without the ENV key) and no knowledge of what files exist or where their chunks are.
3.  **State-Based OAuth Validation**:
    - Use a `OAuthState` collection in Mongo to track users during the Google redirect loop.

---

## 10. ⚠️ Limitations & Improvements

- **Limitations**:
    - **Temp Disk Usage**: The backend currently stores the full original file temporarily before splitting. This limits max file size to the server's available disk.
    - **Synchronous Processing**: For very large files, the HTTP connection might time out during "finalize" (splitting/uploading).
- **Suggested Improvements**:
    - **Worker Queue**: Implement a worker pattern (Redis/RabbitMQ) for the "Finalize" step.
    - **Browser-Side Obfuscation**: Move the noise injection to the frontend (WebAssembly) to avoid sending "raw" data to the server.
    - **Multi-Cloud Support**: Add OneDrive, Dropbox, or S3 as alternate fragment destinations.

---

## 11. 🧪 Testing & Quality

- **Route Testing**: `apps/api/test_routes.sh` contains a structured suite of `curl` commands to verify the end-to-end API logic.
- **Linting**: 
    - Frontend: ESLint configuration present.
    - Backend: Go standard formatting.

---

## 12. 📚 Glossary / Concepts

- **Obfuscation Seed**: A 256-bit value used to initialize the ChaCha20-DRBG for reproducible noise injection.
- **Chunking Strategy**: The algorithm (Balanced, Greedy, etc.) that determines which fragment of the file goes to which Google Drive account.
- **Fragment (.2xpfm)**: A piece of the obfuscated file stored on Google Drive.
- **Key File**: The JSON metadata required to find and de-obfuscate fragments.

---

## 13. 🧭 Developer Notes

- **Modifying Crypto**: If you change the `OBFUSCATION_BLOCK_SIZE` or algorithm in `internal/fileprocessor`, older Key Files will become incompatible.
- **Google API Scopes**: Ensure your Google Project has `https://www.googleapis.com/auth/drive.file` scope enabled.
- **Adding Providers**: To add a new storage provider, implement a new manager in `internal/drivemanager` (or similar) and update the `ChunkPlan` logic in `filehandlers`.

---
*Created by Antigravity Documentation Engineer*
