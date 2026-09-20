# shardX web

React 18 + Vite + TypeScript + Tailwind/shadcn client for `apps/api`. Landing page and file-detail visuals use Three.js (`@react-three/fiber`) and GSAP; the app pages stay WebGL-free.

```sh
cp .env.example .env   # VITE_API_BASE_URL defaults to http://localhost:5555
npm install
npm run dev            # http://localhost:5173
npm test               # vitest (src/lib/__tests__)
npm run lint
npm run e2e            # playwright smoke (needs: npx playwright install chromium)
npm run build
```

## Pages → backend routes

| Page | Endpoints |
|---|---|
| `/` | none — scroll-driven landing (Three.js + GSAP); falls back to SVG chapters without WebGL and to static frames under `prefers-reduced-motion` |
| `/guide` | none — written user manual |
| `/login`, `/signup` | `POST /api/login`, `POST /api/signup` |
| `/files` | `POST /api/files/upload/initiate` → `POST /api/files/upload/chunk?session_id=` → `POST /api/files/chunking/calculate` (preview) → `POST /api/files/upload/finalize` → poll `GET /api/files/upload/status/{id}` → `GET /api/files/download-key/{id}` |
| `/files/:sessionId` | `GET /api/files/{id}/health`, `/shards`, `/timeline`, `GET /api/drive/accounts` |
| `/profile` | `GET /api/drive/accounts`, `GET /api/drive/space`, `GET /api/drive/link` (opens Google consent in a popup) |
| `/oauth/finished` | Landing page after `GET /oauth2/callback`; posts `oauth_finished` to the opener. The API must have `FRONTEND_URL` set to this app's origin for that redirect to land here. |

## Frontend notes

- Design tokens live in `src/index.css` (dark only); shared primitives in `src/components/ds/`; 3D scenes in `src/three/` (all lazy-loaded).
- `/files` shows a one-time spotlight tour (flag `shardx_tour_done` in localStorage); replay it from the sidebar "Show me around".
- Reduced motion disables smooth scroll, scroll-scrubbed 3D, the custom cursor and magnetic buttons; browsers without WebGL get CSS/SVG fallbacks.
- Bundle boundaries: `/files` and `/profile` never load `three`; the landing/three chunks are separate (check with `npm run build`).

Every call goes through `src/lib/api.ts`; non-2xx responses throw `ApiError` with the backend's plain-text message.

There is no download/reconstruct flow yet: the backend has no such endpoint (see `docs/ARCHITECTURE.md`, "Known gaps").
