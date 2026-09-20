# ShardX Immersive Frontend — Design Spec

Date: 2026-09-20
Status: approved

## Goal

Replace the current utilitarian React frontend (`apps/web`) with an award-level, dark-cinematic experience: a scroll-driven Three.js landing page that explains ShardX end to end, a written `/guide`, revamped auth pages, a revamped app shell (Files / FileDetail / Profile), and a first-login in-app tour. All existing API behaviour is preserved; only presentation changes.

## Non-goals

- No Next.js migration. Stay on React 18 + Vite.
- No light theme. Dark only; `next-themes` and `ThemeToggle` are removed.
- No i18n, analytics, CMS, or backend changes.
- No WebGL on `/files` and `/profile` — only Landing, Auth, and FileDetail carry a canvas.

## Stack additions

| Package | Purpose |
|---|---|
| `three`, `@react-three/fiber`, `@react-three/drei` | 3D scenes |
| `@react-three/postprocessing` | Bloom on landing/auth only |
| `gsap`, `@gsap/react` | ScrollTrigger timelines, text reveals, tour tweens |
| `lenis` | Smooth scroll; synced to ScrollTrigger via `lenis.on('scroll', ScrollTrigger.update)` |

Fonts (Google Fonts, `font-display: swap`): **Space Grotesk** (display/body), **JetBrains Mono** (hashes, ids, code).

Removed: `next-themes`, `ThemeToggle.tsx`, `theme-provider.tsx`.

## Design tokens (`src/index.css`)

Dark-only HSL tokens on `:root`. Names kept compatible with existing shadcn components (`--background`, `--foreground`, `--card`, `--primary`, `--accent`, `--muted`, `--border`, `--success`, `--warning`, `--destructive`, `--radius`).

- `--background: 228 20% 4%` (near-black, slight blue)
- `--foreground: 220 15% 92%`
- `--card: 228 18% 7%`
- `--primary: 190 100% 55%` (cyan — "signal")
- `--accent: 275 90% 65%` (violet — "noise")
- `--success: 150 80% 50%`, `--warning: 35 100% 60%`, `--destructive: 0 85% 62%`
- `--muted: 228 14% 12%`, `--muted-foreground: 220 10% 55%`, `--border: 228 14% 16%`
- Additional: `--glow-primary`, `--glow-accent` (rgba for box-shadows), `--grain` (noise overlay opacity).

Global: subtle film-grain overlay (`::after` on body, CSS-only SVG noise, `opacity: var(--grain)`), custom scrollbar, selection colour = primary.

## Routes

| Path | Component | Auth | Canvas |
|---|---|---|---|
| `/` | `Landing` | public | yes (lazy) |
| `/guide` | `Guide` | public | no |
| `/login`, `/signup` | `Login`, `Signup` | public | yes (small, lazy) |
| `/files` | `Files` | protected | no |
| `/files/:sessionId` | `FileDetail` | protected | yes (shard constellation, lazy) |
| `/profile` | `Profile` | protected | no |
| `/oauth/finished` | `OAuthFinished` | — | no |
| `*` | `NotFound` | — | no |

`/` no longer redirects to `/files`. When authenticated, Landing's nav shows "Open vault → /files" instead of "Sign in".

All canvases are `React.lazy` + `Suspense` so `/files` and `/profile` never load `three`.

## Shared design-system components (`src/components/ds/`)

| Component | Props | Behaviour |
|---|---|---|
| `MagneticButton` | `Button` props + `strength?: number` | Translates toward cursor within 40px radius (GSAP quickTo); resets on leave. No-op when reduced motion. |
| `Reveal` | `children`, `delay?`, `as?` | Fades + rises 24px into view via ScrollTrigger once. |
| `SplitText` | `text`, `by: 'char'|'word'`, `stagger?` | Wraps chars/words in spans for stagger reveals. |
| `PageTransition` | `children` | Route-change fade/blur (GSAP) wrapped around `<Routes>`. |
| `GlowCard` | `Card` props | Card with gradient border + hover glow. |
| `Loader` | — | Branded full-screen loader used as Suspense fallback (logo shard pulsing). |
| `Grain` | — | Film-grain overlay. |

Hooks (`src/hooks/`): `useReducedMotion()`, `useWebGL()` (returns `false` if `WebGLRenderingContext` unavailable), `useLenis()` (creates/destroys Lenis, syncs ScrollTrigger).

## Landing page (`src/pages/Landing.tsx`)

### Structure

```
<Landing>
  <Nav />                          fixed, glass, logo + Guide + Sign in / Open vault
  <Suspense fallback={<Loader/>}>
    <LandingScene progress={p} chapter={c} />   fixed full-viewport canvas behind DOM
  </Suspense>
  <ChapterRail chapters current={c} />          fixed right rail, dots + labels
  <main>  9 sections, each 100–150vh </main>
  <Footer />
</Landing>
```

A single GSAP master ScrollTrigger (`scrub: 1`, `start: top top`, `end: bottom bottom` on `<main>`) yields `progress ∈ [0,1]`. `chapterProgress(progress, n=9)` → `{ chapter: 0..8, t: 0..1 }` (pure function, unit-tested). Both values go into a Zustand-free store: a `useRef` + `useFrame` read in R3F (no React re-render per frame). DOM text animations use their own per-section ScrollTriggers.

### Chapters and 3D states

| # | Section | Copy headline | 3D state (`LandingScene`) |
|---|---|---|---|
| 0 | Hero | "Your files become noise." | `Slab` floating, cursor parallax, idle rotation. Bloom on edges. |
| 1 | Upload | "Streamed in 5 MB chunks." | Slab splits into a grid of chunk tiles; a progress ring around them fills with `t`. |
| 2 | Obfuscate | "ChaCha20 noise, everywhere." | Tiles dissolve into a 20k-point particle field (custom shader, `uProgress`, `uSeed`). Colour lerps cyan → violet. |
| 3 | Shard | "Split by strategy." | Particles converge into N=6 shard prisms, each with a floating label `chunk_00N.2xpfm` (drei `<Html>` or `<Text>`). |
| 4 | Store | "S3, under your KMS key." | Three nodes appear (S3, KMS, DynamoDB); shards fly along curves into S3. One shard bounces back (SQS retry) then lands. |
| 5 | Key File | "The server never sees the seed." | A `.2xpfm.key` card materialises in front; shards dim. |
| 6 | Integrity | "Every shard, verified." | Health ring (0→100%) and timeline ticks; shard prisms pulse green. |
| 7 | Drive mode | "No AWS? Use Drive." | Nodes swap to 3 Drive discs; shards redistribute. |
| 8 | CTA | "Create your vault." | Everything recomposes back into the slab; CTA buttons. |

`LandingScene` exposes a single imperative ref API: `setProgress(p: number)`; internally maps to chapter state and uses `MathUtils.damp` for smoothing. It contains: `Slab`, `ChunkGrid`, `NoiseField` (shader), `ShardPrisms`, `StorageNodes`, `KeyCard`, `HealthRing`. Each accepts `t` (0–1) and `active` (bool).

### DOM per chapter

Each section: eyebrow label (mono, e.g. `01 / UPLOAD`), headline (SplitText char reveal), 1–2 sentence body, an optional "detail" pill list (e.g. "5 MB chunks · pause/resume · retry with backoff"). Sections are `min-h-[120vh]` with content pinned via `position: sticky; top: 30vh` so text lingers while the 3D transitions.

### Hero extras

Magnetic CTAs ("Create your vault" → `/signup`, "See how it works" → scrolls to chapter 1). Scroll hint (animated line). Cursor: a custom small dot + ring cursor on pointer devices, blends with `mix-blend-mode: difference`.

## Guide page (`src/pages/Guide.tsx`)

Long-form written manual, dark editorial layout, two columns on desktop: sticky left TOC (chapter list, active highlighting via IntersectionObserver), right content. Sections:

1. What ShardX is (zero-knowledge, sharded)
2. Creating an account (Cognito; password rules)
3. Uploading a file (chunks, pause/resume/cancel, strategies: greedy/balanced/proportional/manual with a table explaining each)
4. What happens after Finalize (async processing, polling)
5. Understanding File Detail (health score, shard map colours, timeline events)
6. The Key File — why it matters, keep it safe, what's inside (seed + chunk map), what the server can and cannot see
7. Google Drive mode — linking accounts, pooled space
8. Reliability — SQS retry, DLQ, what "pending" means
9. FAQ (6–8 questions)

Illustrations: small inline SVG/CSS diagrams (no WebGL). Code/ids in JetBrains Mono. Each section uses `Reveal`.

## Auth pages (`Login.tsx`, `Signup.tsx`)

Split layout (`lg:grid-cols-2`): left panel = `AuthScene` (lazy canvas: a loose cloud of ~200 small shard prisms drifting, parallax to cursor, bloom) with a rotating quote/claim; right panel = form on a `GlowCard`. Mobile: scene collapses to a 30vh header band.

Form behaviour unchanged (same `api.login` / `api.signup`, same validation, same toasts). Add: password visibility toggle, inline field error state (border → destructive), submit button shows spinner, `Enter` submits. Successful login triggers `PageTransition` to `/files`.

## App shell (`src/components/Layout.tsx` → `AppShell`)

Sidebar layout (desktop): 240px left rail with logo, nav (Files, Profile, Guide), "?" tour button, user block with logout at the bottom. Mobile: top bar + slide-in drawer. Main content has max-width 1200px, 32px padding. Active nav item has a sliding indicator (GSAP `flip`-free — a single absolutely positioned pill tweened with `gsap.to`).

### Files page

- Drop zone becomes the hero of the page: dashed border, on drag-over it glows and a ripple animates; idle state shows a faint animated grid.
- Upload cards: glass cards with a segmented progress bar (chunks as segments), status pill, pause/resume/cancel icon buttons, strategy picker as a segmented control (unchanged options).
- Plan preview: horizontal stacked bar coloured per target, with byte labels.
- Completed uploads: "View detail" and "Download key file" actions. Empty state: illustrated (CSS) "No files yet" with tour launch link.

All upload logic (chunking, retry, polling, pause/cancel refs) is untouched — only JSX/className changes.

### FileDetail page

Top: filename/sessionId, health score as a large animated ring (counts up). Middle: `ShardConstellation` (lazy R3F, drei `<Instances>` of shard prisms grouped by target in orbital rings; colour by status; hover → drei `<Html>` tooltip with id/size/sha256 prefix; pending shards pulse). Below it: the existing 2D `ShardMap` grid retained (restyled) for accessibility and no-WebGL fallback. Bottom: `AuditTimeline` as a vertical line with glowing event dots and mono timestamps.

### Profile page

Drive accounts as cards with a circular usage gauge each; "Link Drive" as a `MagneticButton`; S3 mode shows an "Unlimited" pill. Logic untouched.

## In-app tour (`src/components/Tour.tsx`)

- Steps (each: `selector`, `title`, `body`, `placement`): 1) drop zone, 2) strategy picker (only if an upload is awaiting strategy; otherwise skipped), 3) sidebar "Files" → explains detail/health, 4) "Download key file" (or, if none, the sidebar "?" explaining where to find it).
- Overlay: fixed full-screen `div` with a `box-shadow: 0 0 0 9999px rgba(0,0,0,.7)` spotlight element positioned over the target's `getBoundingClientRect()`; GSAP tweens `x/y/width/height` between steps; the card with copy is positioned by `placement`. Esc/backdrop click = skip. Keyboard: Tab trapped inside card, arrows advance.
- Trigger: on `/files` mount if `localStorage.shardx_tour_done !== '1'`. Sidebar "?" resets and replays. Reducer `tourReducer(state, action)` is pure and unit-tested.

## Performance & robustness (mandatory)

- `prefers-reduced-motion: reduce` → no Lenis, no scrub (chapters show final states), no particles (static tiles), no magnetic/cursor effects.
- `useWebGL() === false` → Landing renders a CSS-only fallback: each chapter shows an inline SVG diagram; auth shows a static gradient; FileDetail shows only 2D ShardMap.
- `<Canvas dpr={[1, 2]} frameloop="always">` on Landing, `frameloop="demand"` on FileDetail (invalidate on hover). `document.visibilitychange` → `frameloop='never'` when hidden.
- Mobile (`<768px`): particle count 20k → 6k; bloom off.
- Route-level code splitting for Landing, Guide, and all canvases. Budget: `/files` initial JS ≤ 250 KB gz; landing chunk ≤ 700 KB gz.
- Fonts preconnected; `size-adjust` fallback to avoid CLS.
- Every interactive element focusable with visible `:focus-visible` ring; canvases `aria-hidden`; sections have `aria-labelledby`.
- No console errors/warnings in dev; ESLint clean; `tsc` clean.

## Testing

- Keep existing vitest tests green (`api`, `oauth`, `fileDetail`).
- New unit tests: `chapterProgress()` boundaries; `tourReducer` (next/prev/skip/done, skipped steps).
- Playwright smoke (`apps/web/e2e/`): landing loads, scroll to bottom reveals CTA, `/guide` TOC navigates, login → files → drop-zone visible, tour appears once and not again after done, reduced-motion mode renders all chapter headlines.
- Manual: Lighthouse on `/` and `/files` (perf ≥ 80 desktop, a11y ≥ 95).

## Parallel execution plan (orchestrated)

Four agents in git worktrees, integrated by the orchestrator:

| Agent | Owns | Must not touch |
|---|---|---|
| A – Scene | `src/three/landing/*` (`LandingScene`, sub-objects, shaders), `src/three/auth/AuthScene.tsx`, `src/three/ShardConstellation.tsx` | pages, layout, tokens |
| B – Landing/Guide | `src/pages/Landing.tsx`, `src/pages/Guide.tsx`, `src/components/landing/*`, `src/lib/chapters.ts`, `useLenis` | app pages, three/ |
| C – App shell | `AppShell`, `Files`, `FileDetail`, `Profile`, `Tour`, `OAuthFinished`, `NotFound` | landing, three/ |
| D – DS + Auth | `index.css` tokens, `ds/*`, hooks (`useReducedMotion`, `useWebGL`), `Login`, `Signup`, `App.tsx` routing, deps in `package.json` | pages other than auth |

Contracts (frozen before agents start):
- `LandingScene` — `forwardRef<{ setProgress(p:number):void }, { reducedMotion:boolean; mobile:boolean }>`
- `AuthScene` — `({ className?: string })`
- `ShardConstellation` — `({ shards: ShardRecord[]; driveNames: Record<string,string> })`
- `chapterProgress(p:number, n:number): { chapter:number; t:number }` in `src/lib/chapters.ts`
- Token names as listed above; `ds/*` component names/props as listed above.
- Agent D lands first (tokens, deps, DS); A, B, C start from that commit.

Integration: orchestrator merges D → A → B → C, runs `tsc`, `eslint`, `vitest`, `vite build`, Playwright smoke, then a code-review pass.
