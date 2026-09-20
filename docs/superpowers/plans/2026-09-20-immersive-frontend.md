# Immersive Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `apps/web`'s utilitarian UI with a dark-cinematic, Three.js + GSAP scroll-story landing page, a `/guide` manual, revamped auth pages, a revamped app shell, and a first-login tour — without changing any API behaviour.

**Architecture:** One lazy R3F `<Canvas>` fixed behind the landing DOM, driven by a single GSAP ScrollTrigger progress value through an imperative `setProgress()` ref (no React re-render per frame). Auth pages and FileDetail get their own small lazy canvases. App pages (`/files`, `/profile`) never load `three`. Work is split into four lanes (D → A, B, C) with frozen interfaces so agents can run in parallel; the orchestrator integrates.

**Tech Stack:** React 18, Vite 5, TypeScript, Tailwind 3 + shadcn, `three@0.170`, `@react-three/fiber@8`, `@react-three/drei@9`, `@react-three/postprocessing@2`, `gsap@3` + `@gsap/react`, `lenis`, vitest, Playwright.

**Spec:** `docs/superpowers/specs/2026-09-20-immersive-frontend-design.md`

## Global Constraints

- React stays at 18 (`@react-three/fiber` **8.x**, `drei` **9.x**, `postprocessing` **2.x** — 9.x/10.x need React 19).
- No Next.js. No light theme. `next-themes`, `ThemeToggle.tsx`, `theme-provider.tsx` are deleted.
- All canvases are `React.lazy` + `<Suspense fallback={<Loader/>}>`; `/files` and `/profile` must not import `three`.
- `prefers-reduced-motion: reduce` → no Lenis, no scrub, no particles, no magnetic/cursor effects.
- `useWebGL() === false` → CSS/SVG fallback, never a blank area.
- Every canvas is `aria-hidden`. Every interactive element has a visible `:focus-visible` ring.
- Fonts: Space Grotesk (display/body), JetBrains Mono (ids/hashes). `font-display: swap`.
- Tokens keep shadcn names (`--background`, `--foreground`, `--card`, `--primary`, `--accent`, `--muted`, `--border`, `--success`, `--warning`, `--destructive`, `--radius`).
- Existing upload/auth/OAuth logic in `Files.tsx`, `Profile.tsx`, `Login.tsx`, `Signup.tsx` is **not** changed — only JSX/classNames.
- Commit messages: no Claude attribution lines (repo rule).
- Run from `apps/web`: `npm run lint`, `npx tsc -p tsconfig.app.json --noEmit`, `npm test`, `npm run build` must all pass at the end of every task.

## Lanes and ownership

| Lane | Tasks | Owns | Must not touch |
|---|---|---|---|
| D – DS + Auth | 1–6 | `package.json`, `index.css`, `tailwind.config.ts`, `index.html`, `src/components/ds/*`, `src/hooks/useReducedMotion.ts`, `src/hooks/useWebGL.ts`, `src/lib/chapters.ts`, `src/pages/Login.tsx`, `src/pages/Signup.tsx`, `src/App.tsx` | other pages, `src/three/*` |
| A – Scene | 7–10 | `src/three/**` | pages, layout, tokens |
| B – Landing/Guide | 11–14 | `src/pages/Landing.tsx`, `src/pages/Guide.tsx`, `src/components/landing/*`, `src/hooks/useLenis.ts` | app pages, `src/three/*` |
| C – App shell | 15–20 | `src/components/AppShell.tsx`, `src/components/Tour.tsx`, `src/lib/tour.ts`, `src/pages/Files.tsx`, `src/pages/FileDetail.tsx`, `src/pages/Profile.tsx`, `src/pages/OAuthFinished.tsx`, `src/pages/NotFound.tsx`, `src/components/ShardMap.tsx`, `src/components/AuditTimeline.tsx`, `src/components/FileHealthCard.tsx` | landing, `src/three/*` |
| Orchestrator | 21–22 | integration, e2e | — |

Lane D lands first on `main`. A, B, C branch from that commit. While A/B/C are in flight, imports across lanes compile against the **Interfaces** blocks below; a lane may create a *stub* of a file it consumes only inside its own worktree and must not commit it.

---

## Lane D — Design system, tokens, auth

### Task 1: Dependencies, fonts, tokens

**Files:**
- Modify: `apps/web/package.json`
- Modify: `apps/web/index.html`
- Modify: `apps/web/src/index.css` (replace whole file)
- Modify: `apps/web/tailwind.config.ts`
- Delete: `apps/web/src/components/ThemeToggle.tsx`, `apps/web/src/components/theme-provider.tsx`

**Interfaces:**
- Produces: CSS tokens listed in Global Constraints plus `--glow-primary`, `--glow-accent`, `--grain`; Tailwind `font-display`, `font-mono`; utility classes `.text-glow`, `.glass`, `.eyebrow`.

- [ ] **Step 1: Install / remove packages**

```bash
cd apps/web
npm i three@0.170.0 @react-three/fiber@8.18.0 @react-three/drei@9.122.0 @react-three/postprocessing@2.19.1 gsap@3.15.0 @gsap/react@2.1.2 lenis@1.3.26
npm i -D @types/three@0.170.0 @playwright/test@1.49.1
npm rm next-themes
rm src/components/ThemeToggle.tsx src/components/theme-provider.tsx
```

- [ ] **Step 2: Fonts + meta in `index.html`**

Replace `<head>` contents:

```html
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<meta name="theme-color" content="#07080d" />
<title>ShardX — Your files become noise</title>
<meta name="description" content="Zero-knowledge, sharded object storage. Every upload is obfuscated with ChaCha20 noise, split into shards, and stored under your KMS key. Only your Key File can put it back together." />
<meta property="og:title" content="ShardX — Your files become noise" />
<meta property="og:description" content="Zero-knowledge, sharded object storage on AWS." />
<meta property="og:type" content="website" />
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
<link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet" />
```

Also add `class="dark"` to `<html>` (shadcn components read it).

- [ ] **Step 3: Replace `src/index.css`**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 228 20% 4%;
    --foreground: 220 15% 92%;
    --card: 228 18% 7%;
    --card-foreground: 220 15% 92%;
    --popover: 228 18% 8%;
    --popover-foreground: 220 15% 92%;
    --primary: 190 100% 55%;
    --primary-foreground: 228 20% 4%;
    --secondary: 228 14% 12%;
    --secondary-foreground: 220 15% 92%;
    --muted: 228 14% 12%;
    --muted-foreground: 220 10% 55%;
    --accent: 275 90% 65%;
    --accent-foreground: 228 20% 4%;
    --success: 150 80% 50%;
    --success-foreground: 228 20% 4%;
    --warning: 35 100% 60%;
    --warning-foreground: 228 20% 4%;
    --destructive: 0 85% 62%;
    --destructive-foreground: 220 15% 92%;
    --border: 228 14% 16%;
    --input: 228 14% 14%;
    --ring: 190 100% 55%;
    --radius: 0.875rem;
    --glow-primary: 0 0 40px hsl(190 100% 55% / 0.35);
    --glow-accent: 0 0 40px hsl(275 90% 65% / 0.35);
    --grain: 0.05;
  }
  * { @apply border-border; }
  html { color-scheme: dark; scroll-behavior: auto; }
  body {
    @apply bg-background text-foreground font-display antialiased;
    overflow-x: hidden;
  }
  ::selection { background: hsl(var(--primary) / 0.35); }
  :focus-visible { outline: 2px solid hsl(var(--primary)); outline-offset: 2px; border-radius: 4px; }
  ::-webkit-scrollbar { width: 10px; }
  ::-webkit-scrollbar-track { background: hsl(var(--background)); }
  ::-webkit-scrollbar-thumb { background: hsl(var(--muted)); border-radius: 999px; border: 2px solid hsl(var(--background)); }
}

@layer utilities {
  .text-glow { text-shadow: 0 0 24px hsl(var(--primary) / 0.5); }
  .glass { background: hsl(var(--card) / 0.55); backdrop-filter: blur(16px) saturate(140%); -webkit-backdrop-filter: blur(16px) saturate(140%); border: 1px solid hsl(var(--border)); }
  .eyebrow { @apply font-mono text-xs uppercase tracking-[0.25em] text-primary; }
  .gradient-text { background: linear-gradient(120deg, hsl(var(--primary)), hsl(var(--accent))); -webkit-background-clip: text; background-clip: text; color: transparent; }
  .no-scrollbar::-webkit-scrollbar { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after { animation-duration: 0.01ms !important; transition-duration: 0.01ms !important; }
}
```

- [ ] **Step 4: Tailwind fonts**

In `tailwind.config.ts` under `theme.extend` add:

```ts
fontFamily: {
  display: ['"Space Grotesk"', 'system-ui', 'sans-serif'],
  mono: ['"JetBrains Mono"', 'ui-monospace', 'monospace'],
},
boxShadow: {
  'glow-primary': 'var(--glow-primary)',
  'glow-accent': 'var(--glow-accent)',
},
```

Remove the whole `sidebar` colour block (unused after this task).

- [ ] **Step 5: Fix breakages from the removed theme provider**

In `src/App.tsx` remove the `ThemeProvider` import and wrapper. In `src/components/Layout.tsx` remove the `ThemeToggle` import and `<ThemeToggle />` element (Lane C replaces this file later; keep it compiling now).

- [ ] **Step 6: Verify**

Run: `cd apps/web && npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm test && npm run build`
Expected: all pass; build output shows no `three` in the main chunk yet (nothing imports it).

- [ ] **Step 7: Commit**

```bash
git add -A apps/web
git commit -m "feat(web): dark tokens, fonts, 3D/animation deps; drop theme toggle"
```

### Task 2: Hooks — `useReducedMotion`, `useWebGL`

**Files:**
- Create: `apps/web/src/hooks/useReducedMotion.ts`
- Create: `apps/web/src/hooks/useWebGL.ts`
- Test: `apps/web/src/hooks/__tests__/useWebGL.test.ts`

**Interfaces:**
- Produces: `useReducedMotion(): boolean`; `useWebGL(): boolean`; `hasWebGL(): boolean` (pure, testable).

- [ ] **Step 1: Failing test**

```ts
// src/hooks/__tests__/useWebGL.test.ts
import { describe, it, expect, vi } from 'vitest';
import { hasWebGL } from '../useWebGL';

describe('hasWebGL', () => {
  it('is false when canvas has no webgl context', () => {
    const doc = { createElement: () => ({ getContext: () => null }) } as unknown as Document;
    expect(hasWebGL(doc)).toBe(false);
  });
  it('is true when webgl context exists', () => {
    const doc = { createElement: () => ({ getContext: () => ({}) }) } as unknown as Document;
    expect(hasWebGL(doc)).toBe(true);
  });
  it('is false when getContext throws', () => {
    const doc = { createElement: () => ({ getContext: () => { throw new Error('x'); } }) } as unknown as Document;
    expect(hasWebGL(doc)).toBe(false);
  });
});
```

- [ ] **Step 2: Run** `npx vitest run src/hooks` → FAIL (module not found).

- [ ] **Step 3: Implement**

```ts
// src/hooks/useWebGL.ts
import { useMemo } from 'react';

export const hasWebGL = (doc: Document = document): boolean => {
  try {
    const c = doc.createElement('canvas');
    return !!(c.getContext('webgl2') || c.getContext('webgl'));
  } catch {
    return false;
  }
};

/** True when the browser can create a WebGL context. Evaluated once per mount. */
export const useWebGL = (): boolean => useMemo(() => typeof document !== 'undefined' && hasWebGL(), []);
```

```ts
// src/hooks/useReducedMotion.ts
import { useEffect, useState } from 'react';

const QUERY = '(prefers-reduced-motion: reduce)';

export const useReducedMotion = (): boolean => {
  const [reduced, setReduced] = useState(() => typeof window !== 'undefined' && window.matchMedia(QUERY).matches);
  useEffect(() => {
    const mq = window.matchMedia(QUERY);
    const on = (e: MediaQueryListEvent) => setReduced(e.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, []);
  return reduced;
};
```

- [ ] **Step 4: Run** `npx vitest run src/hooks` → PASS.

- [ ] **Step 5: Commit** `git add apps/web/src/hooks && git commit -m "feat(web): useReducedMotion and useWebGL hooks"`

### Task 3: `chapterProgress` math

**Files:**
- Create: `apps/web/src/lib/chapters.ts`
- Test: `apps/web/src/lib/__tests__/chapters.test.ts`

**Interfaces:**
- Produces: `CHAPTERS: readonly { id: string; label: string }[]` (9 entries, ids `hero, upload, obfuscate, shard, store, keyfile, integrity, drive, cta`); `chapterProgress(p: number, n?: number): { chapter: number; t: number }`.

- [ ] **Step 1: Failing test**

```ts
import { describe, it, expect } from 'vitest';
import { chapterProgress, CHAPTERS } from '../chapters';

describe('chapterProgress', () => {
  it('has 9 chapters', () => expect(CHAPTERS).toHaveLength(9));
  it('starts at chapter 0, t 0', () => expect(chapterProgress(0)).toEqual({ chapter: 0, t: 0 }));
  it('ends at last chapter, t 1', () => expect(chapterProgress(1)).toEqual({ chapter: 8, t: 1 }));
  it('maps midpoint of chapter 4', () => {
    const r = chapterProgress(4.5 / 9);
    expect(r.chapter).toBe(4);
    expect(r.t).toBeCloseTo(0.5, 5);
  });
  it('clamps out-of-range', () => {
    expect(chapterProgress(-1)).toEqual({ chapter: 0, t: 0 });
    expect(chapterProgress(2)).toEqual({ chapter: 8, t: 1 });
  });
  it('supports custom n', () => expect(chapterProgress(0.5, 2)).toEqual({ chapter: 1, t: 0 }));
});
```

- [ ] **Step 2: Run** `npx vitest run src/lib/__tests__/chapters.test.ts` → FAIL.

- [ ] **Step 3: Implement**

```ts
// src/lib/chapters.ts
export const CHAPTERS = [
  { id: 'hero', label: 'Noise' },
  { id: 'upload', label: 'Upload' },
  { id: 'obfuscate', label: 'Obfuscate' },
  { id: 'shard', label: 'Shard' },
  { id: 'store', label: 'Store' },
  { id: 'keyfile', label: 'Key File' },
  { id: 'integrity', label: 'Integrity' },
  { id: 'drive', label: 'Drive mode' },
  { id: 'cta', label: 'Begin' },
] as const;

/** Maps global scroll progress p∈[0,1] to { chapter, t∈[0,1] within that chapter }. */
export const chapterProgress = (p: number, n: number = CHAPTERS.length) => {
  const clamped = Math.min(1, Math.max(0, p));
  if (clamped === 1) return { chapter: n - 1, t: 1 };
  const scaled = clamped * n;
  const chapter = Math.floor(scaled);
  return { chapter, t: scaled - chapter };
};
```

- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `git add apps/web/src/lib && git commit -m "feat(web): chapter progress math"`

### Task 4: Design-system components

**Files:**
- Create: `apps/web/src/components/ds/MagneticButton.tsx`, `Reveal.tsx`, `SplitText.tsx`, `PageTransition.tsx`, `GlowCard.tsx`, `Loader.tsx`, `Grain.tsx`, `index.ts`

**Interfaces:**
- Produces (all named exports from `@/components/ds`):
  - `MagneticButton: React.FC<ButtonProps & { strength?: number }>`
  - `Reveal: React.FC<{ children: ReactNode; delay?: number; className?: string; as?: keyof JSX.IntrinsicElements }>`
  - `SplitText: React.FC<{ text: string; by?: 'char' | 'word'; className?: string; stagger?: number; trigger?: boolean }>`
  - `PageTransition: React.FC<{ children: ReactNode }>`
  - `GlowCard: React.FC<React.HTMLAttributes<HTMLDivElement>>`
  - `Loader: React.FC<{ label?: string }>`
  - `Grain: React.FC`

- [ ] **Step 1: Write components**

```tsx
// src/components/ds/MagneticButton.tsx
import { useRef } from 'react';
import gsap from 'gsap';
import { Button, type ButtonProps } from '@/components/ui/button';
import { useReducedMotion } from '@/hooks/useReducedMotion';

export const MagneticButton = ({ strength = 0.35, onMouseMove, onMouseLeave, ...props }: ButtonProps & { strength?: number }) => {
  const ref = useRef<HTMLButtonElement>(null);
  const reduced = useReducedMotion();
  return (
    <Button
      ref={ref}
      {...props}
      onMouseMove={(e) => {
        onMouseMove?.(e);
        if (reduced || !ref.current) return;
        const r = ref.current.getBoundingClientRect();
        const x = (e.clientX - (r.left + r.width / 2)) * strength;
        const y = (e.clientY - (r.top + r.height / 2)) * strength;
        gsap.to(ref.current, { x, y, duration: 0.4, ease: 'power3.out' });
      }}
      onMouseLeave={(e) => {
        onMouseLeave?.(e);
        if (ref.current) gsap.to(ref.current, { x: 0, y: 0, duration: 0.6, ease: 'elastic.out(1, 0.4)' });
      }}
    />
  );
};
```

```tsx
// src/components/ds/Reveal.tsx
import { createElement, useRef, type ReactNode } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useGSAP } from '@gsap/react';
import { useReducedMotion } from '@/hooks/useReducedMotion';

gsap.registerPlugin(ScrollTrigger, useGSAP);

export const Reveal = ({ children, delay = 0, className, as = 'div' }: { children: ReactNode; delay?: number; className?: string; as?: keyof JSX.IntrinsicElements }) => {
  const ref = useRef<HTMLElement>(null);
  const reduced = useReducedMotion();
  useGSAP(() => {
    if (reduced || !ref.current) return;
    gsap.fromTo(ref.current, { opacity: 0, y: 24 }, { opacity: 1, y: 0, duration: 0.9, delay, ease: 'power3.out', scrollTrigger: { trigger: ref.current, start: 'top 85%', once: true } });
  }, { dependencies: [reduced] });
  return createElement(as, { ref, className, style: reduced ? undefined : { opacity: 0 } }, children);
};
```

```tsx
// src/components/ds/SplitText.tsx
import { useRef } from 'react';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useGSAP } from '@gsap/react';
import { useReducedMotion } from '@/hooks/useReducedMotion';

gsap.registerPlugin(ScrollTrigger, useGSAP);

/** Splits text into spans and staggers them in. `trigger` false = animate on mount. */
export const SplitText = ({ text, by = 'char', className, stagger = 0.02, trigger = true }: { text: string; by?: 'char' | 'word'; className?: string; stagger?: number; trigger?: boolean }) => {
  const ref = useRef<HTMLSpanElement>(null);
  const reduced = useReducedMotion();
  const parts = by === 'word' ? text.split(' ') : Array.from(text);
  useGSAP(() => {
    if (reduced || !ref.current) return;
    const targets = ref.current.querySelectorAll('[data-part]');
    gsap.fromTo(targets, { yPercent: 110, opacity: 0 }, { yPercent: 0, opacity: 1, duration: 0.8, stagger, ease: 'power4.out', scrollTrigger: trigger ? { trigger: ref.current, start: 'top 85%', once: true } : undefined });
  }, { dependencies: [reduced, text] });
  return (
    <span ref={ref} className={className} aria-label={text}>
      {parts.map((p, i) => (
        <span key={i} className="inline-block overflow-hidden align-bottom" aria-hidden>
          <span data-part className="inline-block will-change-transform">{p === ' ' ? ' ' : p}{by === 'word' && i < parts.length - 1 ? ' ' : ''}</span>
        </span>
      ))}
    </span>
  );
};
```

```tsx
// src/components/ds/PageTransition.tsx
import { useLayoutEffect, useRef, type ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import gsap from 'gsap';
import { useReducedMotion } from '@/hooks/useReducedMotion';

export const PageTransition = ({ children }: { children: ReactNode }) => {
  const ref = useRef<HTMLDivElement>(null);
  const { pathname } = useLocation();
  const reduced = useReducedMotion();
  useLayoutEffect(() => {
    if (reduced || !ref.current) return;
    gsap.fromTo(ref.current, { opacity: 0, filter: 'blur(8px)' }, { opacity: 1, filter: 'blur(0px)', duration: 0.5, ease: 'power2.out', clearProps: 'filter' });
  }, [pathname, reduced]);
  return <div ref={ref}>{children}</div>;
};
```

```tsx
// src/components/ds/GlowCard.tsx
import { cn } from '@/lib/utils';
import type { HTMLAttributes } from 'react';

export const GlowCard = ({ className, ...props }: HTMLAttributes<HTMLDivElement>) => (
  <div
    className={cn(
      'relative rounded-2xl glass p-6 transition-shadow duration-500 hover:shadow-glow-primary',
      'before:pointer-events-none before:absolute before:inset-0 before:rounded-2xl before:p-px before:bg-gradient-to-br before:from-primary/40 before:via-transparent before:to-accent/40 before:[mask:linear-gradient(#000,#000)_content-box,linear-gradient(#000,#000)] before:[mask-composite:exclude]',
      className,
    )}
    {...props}
  />
);
```

```tsx
// src/components/ds/Loader.tsx
export const Loader = ({ label = 'Loading' }: { label?: string }) => (
  <div role="status" aria-live="polite" className="fixed inset-0 z-[100] grid place-items-center bg-background">
    <div className="flex flex-col items-center gap-4">
      <div className="h-10 w-10 rotate-45 rounded-sm bg-gradient-to-br from-primary to-accent animate-pulse shadow-glow-primary" />
      <span className="eyebrow">{label}</span>
    </div>
  </div>
);
```

```tsx
// src/components/ds/Grain.tsx
const NOISE =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='200'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E\")";

export const Grain = () => (
  <div aria-hidden className="pointer-events-none fixed inset-0 z-[90] mix-blend-overlay" style={{ backgroundImage: NOISE, opacity: 'var(--grain)' as unknown as number }} />
);
```

```ts
// src/components/ds/index.ts
export { MagneticButton } from './MagneticButton';
export { Reveal } from './Reveal';
export { SplitText } from './SplitText';
export { PageTransition } from './PageTransition';
export { GlowCard } from './GlowCard';
export { Loader } from './Loader';
export { Grain } from './Grain';
```

- [ ] **Step 2: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint` → clean. (`Button` must export `ButtonProps` — it does in shadcn's button.tsx; if not, add `export type { ButtonProps }`.)
- [ ] **Step 3: Commit** `git add apps/web/src/components/ds && git commit -m "feat(web): design-system primitives (magnetic, reveal, split text, glow card, loader, grain)"`

### Task 5: Routing + `App.tsx` with lazy pages

**Files:**
- Modify: `apps/web/src/App.tsx` (replace whole file)
- Create (stubs, committed, replaced by lanes B/C): `apps/web/src/pages/Landing.tsx`, `apps/web/src/pages/Guide.tsx`

**Interfaces:**
- Produces: routes per spec. Landing and Guide are default-exported components.

- [ ] **Step 1: Stubs** (so routing compiles; lanes B replaces both)

```tsx
// src/pages/Landing.tsx
const Landing = () => <main className="min-h-screen grid place-items-center"><h1 className="text-4xl font-bold">ShardX</h1></main>;
export default Landing;
```
```tsx
// src/pages/Guide.tsx
const Guide = () => <main className="min-h-screen p-8"><h1 className="text-3xl font-bold">Guide</h1></main>;
export default Guide;
```

- [ ] **Step 2: `App.tsx`**

```tsx
import { lazy, Suspense } from 'react';
import { Toaster as Sonner } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from '@/hooks/useAuth';
import { Loader, Grain, PageTransition } from '@/components/ds';
import Login from './pages/Login';
import Signup from './pages/Signup';
import Files from './pages/Files';
import FileDetail from './pages/FileDetail';
import Profile from './pages/Profile';
import OAuthFinished from './pages/OAuthFinished';
import NotFound from './pages/NotFound';

const Landing = lazy(() => import('./pages/Landing'));
const Guide = lazy(() => import('./pages/Guide'));

const queryClient = new QueryClient();

const App = () => (
  <QueryClientProvider client={queryClient}>
    <TooltipProvider>
      <AuthProvider>
        <Sonner theme="dark" />
        <Grain />
        <BrowserRouter>
          <Suspense fallback={<Loader />}>
            <PageTransition>
              <Routes>
                <Route path="/" element={<Landing />} />
                <Route path="/guide" element={<Guide />} />
                <Route path="/login" element={<Login />} />
                <Route path="/signup" element={<Signup />} />
                <Route path="/files" element={<Files />} />
                <Route path="/files/:sessionId" element={<FileDetail />} />
                <Route path="/profile" element={<Profile />} />
                <Route path="/oauth/finished" element={<OAuthFinished />} />
                <Route path="*" element={<NotFound />} />
              </Routes>
            </PageTransition>
          </Suspense>
        </BrowserRouter>
      </AuthProvider>
    </TooltipProvider>
  </QueryClientProvider>
);

export default App;
```

Check `src/components/ui/sonner.tsx`: it imports `useTheme` from `next-themes`. Replace its body with:

```tsx
import { Toaster as Sonner } from 'sonner';
type ToasterProps = React.ComponentProps<typeof Sonner>;
const Toaster = (props: ToasterProps) => (
  <Sonner theme="dark" className="toaster group" toastOptions={{ classNames: { toast: 'group toast group-[.toaster]:bg-card group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg', description: 'group-[.toast]:text-muted-foreground', actionButton: 'group-[.toast]:bg-primary group-[.toast]:text-primary-foreground', cancelButton: 'group-[.toast]:bg-muted group-[.toast]:text-muted-foreground' } }} {...props} />
);
export { Toaster };
```

- [ ] **Step 3: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm run build` → pass; `dist/assets` contains a separate `Landing-*.js` chunk.
- [ ] **Step 4: Commit** `git add -A apps/web/src && git commit -m "feat(web): landing/guide routes, lazy pages, page transition"`

### Task 6: Auth pages

**Files:**
- Modify: `apps/web/src/pages/Login.tsx`, `apps/web/src/pages/Signup.tsx` (replace JSX only)
- Create: `apps/web/src/components/AuthLayout.tsx`

**Interfaces:**
- Consumes: `AuthScene` from `@/three/auth/AuthScene` (Lane A; default export `({ className?: string }) => JSX.Element`). Until Lane A lands, create an **uncommitted** local stub `src/three/auth/AuthScene.tsx` exporting `export default () => null`.
- Produces: `AuthLayout: React.FC<{ title: string; subtitle: string; children: ReactNode; footer: ReactNode }>`.

- [ ] **Step 1: `AuthLayout`**

```tsx
// src/components/AuthLayout.tsx
import { lazy, Suspense, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { GlowCard, SplitText } from '@/components/ds';
import { useWebGL } from '@/hooks/useWebGL';

const AuthScene = lazy(() => import('@/three/auth/AuthScene'));

const CLAIMS = [
  'The server never sees the seed.',
  'A compromised bucket yields nothing readable.',
  'Every shard is verified. Every step is logged.',
];

export const AuthLayout = ({ title, subtitle, children, footer }: { title: string; subtitle: string; children: ReactNode; footer: ReactNode }) => {
  const webgl = useWebGL();
  const claim = CLAIMS[Math.floor(Date.now() / 10000) % CLAIMS.length];
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-2">
      <aside className="relative h-[30vh] lg:h-auto overflow-hidden bg-gradient-to-br from-background via-[hsl(228_20%_6%)] to-[hsl(275_40%_8%)]">
        {webgl && (
          <Suspense fallback={null}>
            <AuthScene className="absolute inset-0" />
          </Suspense>
        )}
        <div className="relative z-10 flex h-full flex-col justify-between p-8 lg:p-12">
          <Link to="/" className="flex items-center gap-2 text-lg font-semibold">
            <span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent" />
            ShardX
          </Link>
          <p className="hidden lg:block max-w-md text-2xl font-medium leading-snug text-foreground/90">
            <SplitText text={claim} by="word" trigger={false} />
          </p>
        </div>
      </aside>
      <main className="flex items-center justify-center p-6 lg:p-12">
        <GlowCard className="w-full max-w-md p-8">
          <h1 className="text-2xl font-semibold">{title}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
          <div className="mt-8">{children}</div>
          <div className="mt-6 text-center text-sm text-muted-foreground">{footer}</div>
        </GlowCard>
      </main>
    </div>
  );
};
```

- [ ] **Step 2: `Login.tsx` JSX** — keep all state/handlers exactly as they are; replace the `return (...)` with:

```tsx
return (
  <AuthLayout
    title="Welcome back"
    subtitle="Sign in to open your vault."
    footer={<>No account? <Link to="/signup" className="text-primary hover:underline">Create one</Link></>}
  >
    <form onSubmit={handleSubmit} className="space-y-5" noValidate>
      <div className="space-y-2">
        <Label htmlFor="email">Email</Label>
        <Input id="email" type="email" autoComplete="email" placeholder="you@example.com" value={email} onChange={(e) => setEmail(e.target.value)} disabled={isLoading} required />
      </div>
      <div className="space-y-2">
        <Label htmlFor="password">Password</Label>
        <div className="relative">
          <Input id="password" type={showPassword ? 'text' : 'password'} autoComplete="current-password" placeholder="••••••••" value={password} onChange={(e) => setPassword(e.target.value)} disabled={isLoading} required className="pr-10" />
          <button type="button" aria-label={showPassword ? 'Hide password' : 'Show password'} onClick={() => setShowPassword((v) => !v)} className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground">
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
      </div>
      <MagneticButton type="submit" className="w-full" disabled={isLoading}>
        {isLoading ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
        {isLoading ? 'Signing in…' : 'Sign in'}
      </MagneticButton>
    </form>
  </AuthLayout>
);
```

Add `const [showPassword, setShowPassword] = useState(false);` and imports: `AuthLayout`, `MagneticButton` from `@/components/ds`, `Eye, EyeOff, Loader2` from `lucide-react`. Remove the `Card*` and `Cloud` imports.

- [ ] **Step 3: `Signup.tsx` JSX** — same treatment: title "Create your vault", subtitle "Zero-knowledge storage starts with an account.", footer link to `/login`, three fields (email, password with toggle, confirm password), `MagneticButton` submit reading "Creating…" / "Create account". Keep handlers unchanged.

- [ ] **Step 4: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm run build` → pass. `npm run dev`, open `/login`: split layout, form works against the API as before.
- [ ] **Step 5: Commit** (do **not** add the AuthScene stub) `git add apps/web/src/pages/Login.tsx apps/web/src/pages/Signup.tsx apps/web/src/components/AuthLayout.tsx && git commit -m "feat(web): cinematic auth layout"`

Lane D done → push to `main`. Lanes A, B, C branch from here.

---

## Lane A — Three.js scenes

All files under `apps/web/src/three/`. Nothing here imports pages or layout. Every `<Canvas>` gets `aria-hidden`, `dpr={[1, 2]}`, `gl={{ antialias: true, powerPreference: 'high-performance' }}`.

### Task 7: Particle targets (pure) + shared progress store

**Files:**
- Create: `apps/web/src/three/landing/store.ts`
- Create: `apps/web/src/three/landing/targets.ts`
- Test: `apps/web/src/three/landing/__tests__/targets.test.ts`

**Interfaces:**
- Produces: `progressStore: { p: number; chapter: number; t: number; mouse: [number, number] }` (mutable singleton, written by `LandingScene.setProgress`, read in `useFrame`).
- Produces: `TargetName = 'slab'|'grid'|'noise'|'shards'|'nodes'|'key'|'ring'|'drive'`; `buildTargets(n: number, seed?: number): Record<TargetName, Float32Array>`; `CHAPTER_TARGETS: [TargetName, TargetName][]` (9 entries: from/to per chapter); `SHARD_POSITIONS: [number,number,number][]` (6); `NODE_POSITIONS: { s3: [n,n,n]; kms: [n,n,n]; dynamo: [n,n,n] }`; `DRIVE_POSITIONS: [n,n,n][]` (3).

- [ ] **Step 1: Failing test**

```ts
import { describe, it, expect } from 'vitest';
import { buildTargets, CHAPTER_TARGETS } from '../targets';

describe('buildTargets', () => {
  const t = buildTargets(1000, 7);
  it('every target has n*3 floats', () => {
    for (const arr of Object.values(t)) expect(arr.length).toBe(3000);
  });
  it('is deterministic for a seed', () => {
    expect(buildTargets(50, 3).noise).toEqual(buildTargets(50, 3).noise);
  });
  it('slab points lie inside the slab box', () => {
    for (let i = 0; i < 1000; i++) {
      expect(Math.abs(t.slab[i * 3])).toBeLessThanOrEqual(1.5);
      expect(Math.abs(t.slab[i * 3 + 1])).toBeLessThanOrEqual(1.0);
      expect(Math.abs(t.slab[i * 3 + 2])).toBeLessThanOrEqual(0.08);
    }
  });
  it('chapter map covers 9 chapters and chains', () => {
    expect(CHAPTER_TARGETS).toHaveLength(9);
    for (let i = 1; i < 9; i++) expect(CHAPTER_TARGETS[i][0]).toBe(CHAPTER_TARGETS[i - 1][1]);
  });
});
```

- [ ] **Step 2: Run** `npx vitest run src/three` → FAIL.

- [ ] **Step 3: Implement**

```ts
// src/three/landing/store.ts
/** Mutable, render-free store. Written by the scroll driver, read in useFrame. */
export const progressStore = { p: 0, chapter: 0, t: 0, mouse: [0, 0] as [number, number] };
```

```ts
// src/three/landing/targets.ts
export type TargetName = 'slab' | 'grid' | 'noise' | 'shards' | 'nodes' | 'key' | 'ring' | 'drive';

export const SHARD_POSITIONS: [number, number, number][] = [
  [-2.2, 1.0, 0], [0, 1.4, -0.5], [2.2, 1.0, 0], [-2.2, -1.0, 0], [0, -1.4, -0.5], [2.2, -1.0, 0],
];
export const NODE_POSITIONS = { s3: [0, 0, 0] as [number, number, number], kms: [-2.8, 1.6, -1] as [number, number, number], dynamo: [2.8, 1.6, -1] as [number, number, number] };
export const DRIVE_POSITIONS: [number, number, number][] = [[-2.4, 0, 0], [0, 0.6, -0.4], [2.4, 0, 0]];

/** from → to per chapter (index = chapter). */
export const CHAPTER_TARGETS: [TargetName, TargetName][] = [
  ['slab', 'slab'], ['slab', 'grid'], ['grid', 'noise'], ['noise', 'shards'], ['shards', 'nodes'],
  ['nodes', 'key'], ['key', 'ring'], ['ring', 'drive'], ['drive', 'slab'],
];

// mulberry32 — tiny seeded PRNG so tests are deterministic.
const rng = (seed: number) => () => {
  seed |= 0; seed = (seed + 0x6d2b79f5) | 0;
  let t = Math.imul(seed ^ (seed >>> 15), 1 | seed);
  t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
};

const fill = (n: number, f: (i: number, out: Float32Array) => void) => {
  const out = new Float32Array(n * 3);
  for (let i = 0; i < n; i++) f(i * 3, out);
  return out;
};

const cluster = (n: number, r: () => number, centers: [number, number, number][], radius: number) =>
  fill(n, (o, out) => {
    const c = centers[Math.floor(r() * centers.length)];
    const u = r() * 2 - 1, phi = r() * Math.PI * 2, rad = Math.cbrt(r()) * radius;
    const s = Math.sqrt(1 - u * u);
    out[o] = c[0] + rad * s * Math.cos(phi); out[o + 1] = c[1] + rad * s * Math.sin(phi); out[o + 2] = c[2] + rad * u;
  });

export const buildTargets = (n: number, seed = 1): Record<TargetName, Float32Array> => {
  const r = rng(seed);
  const slab = fill(n, (o, out) => { out[o] = (r() - 0.5) * 3; out[o + 1] = (r() - 0.5) * 2; out[o + 2] = (r() - 0.5) * 0.15; });
  const grid = fill(n, (o, out) => {
    const col = Math.floor(r() * 5), row = Math.floor(r() * 4);
    out[o] = -1.6 + col * 0.8 + (r() - 0.5) * 0.6; out[o + 1] = 1.2 - row * 0.8 + (r() - 0.5) * 0.6; out[o + 2] = (r() - 0.5) * 0.1;
  });
  const noise = fill(n, (o, out) => {
    const u = r() * 2 - 1, phi = r() * Math.PI * 2, rad = 1.5 + r() * 3;
    const s = Math.sqrt(1 - u * u);
    out[o] = rad * s * Math.cos(phi); out[o + 1] = rad * s * Math.sin(phi); out[o + 2] = rad * u;
  });
  const shards = cluster(n, r, SHARD_POSITIONS, 0.45);
  const nodes = cluster(n, r, [NODE_POSITIONS.s3, NODE_POSITIONS.s3, NODE_POSITIONS.s3, NODE_POSITIONS.kms, NODE_POSITIONS.dynamo], 0.7);
  const key = fill(n, (o, out) => { out[o] = (r() - 0.5) * 2.2; out[o + 1] = (r() - 0.5) * 1.4; out[o + 2] = 0.8 + (r() - 0.5) * 0.05; });
  const ring = fill(n, (o, out) => {
    const a = r() * Math.PI * 2, b = r() * Math.PI * 2, R = 1.8, tube = 0.25;
    out[o] = (R + tube * Math.cos(b)) * Math.cos(a); out[o + 1] = (R + tube * Math.cos(b)) * Math.sin(a); out[o + 2] = tube * Math.sin(b);
  });
  const drive = cluster(n, r, DRIVE_POSITIONS, 0.8);
  return { slab, grid, noise, shards, nodes, key, ring, drive };
};
```

- [ ] **Step 4: Run** → PASS.
- [ ] **Step 5: Commit** `git add apps/web/src/three && git commit -m "feat(web/three): particle targets and progress store"`

### Task 8: `LandingScene` — particle morph, accents, bloom

**Files:**
- Create: `apps/web/src/three/landing/NoiseField.tsx`
- Create: `apps/web/src/three/landing/Accents.tsx`
- Create: `apps/web/src/three/landing/LandingScene.tsx`

**Interfaces:**
- Consumes: `progressStore`, `buildTargets`, `CHAPTER_TARGETS`, positions from Task 7; `chapterProgress` from `@/lib/chapters` (Lane D).
- Produces: `export type LandingSceneHandle = { setProgress(p: number): void; setMouse(x: number, y: number): void }`; `export default forwardRef<LandingSceneHandle, { reducedMotion: boolean; mobile: boolean; className?: string }>(LandingScene)`.

- [ ] **Step 1: `NoiseField.tsx`**

```tsx
import { useMemo, useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';
import { progressStore } from './store';
import { buildTargets, CHAPTER_TARGETS } from './targets';

const vert = /* glsl */ `
  attribute vec3 aFrom;
  attribute vec3 aTo;
  attribute float aSeed;
  uniform float uT;
  uniform float uTime;
  uniform float uSwirl;
  uniform float uSize;
  varying float vSeed;
  varying float vT;
  void main() {
    // per-particle delay so the morph ripples instead of snapping
    float d = clamp((uT - aSeed * 0.35) / 0.65, 0.0, 1.0);
    float e = d * d * (3.0 - 2.0 * d);
    vec3 p = mix(aFrom, aTo, e);
    // swirl during transit, strongest mid-morph
    float mid = 4.0 * e * (1.0 - e);
    float a = uTime * 0.6 + aSeed * 6.2831;
    p += uSwirl * mid * vec3(cos(a), sin(a * 1.3), sin(a)) * 0.6;
    vSeed = aSeed; vT = e;
    vec4 mv = modelViewMatrix * vec4(p, 1.0);
    gl_PointSize = uSize * (1.0 + 0.5 * aSeed) * (300.0 / -mv.z);
    gl_Position = projectionMatrix * mv;
  }
`;
const frag = /* glsl */ `
  uniform vec3 uColorA;
  uniform vec3 uColorB;
  uniform float uMix;
  varying float vSeed;
  varying float vT;
  void main() {
    vec2 c = gl_PointCoord - 0.5;
    float r = dot(c, c);
    if (r > 0.25) discard;
    float alpha = smoothstep(0.25, 0.0, r) * (0.55 + 0.45 * vSeed);
    vec3 col = mix(uColorA, uColorB, clamp(uMix + (vSeed - 0.5) * 0.3, 0.0, 1.0));
    gl_FragColor = vec4(col, alpha);
  }
`;

const CYAN = new THREE.Color('#33e0ff');
const VIOLET = new THREE.Color('#b56bff');
const GREEN = new THREE.Color('#3dffa0');
/** Colour blend per chapter (0 = cyan, 1 = violet). Integrity chapter goes green via uColorB swap. */
const MIX_BY_CHAPTER = [0, 0.1, 1, 0.7, 0.4, 0.2, 0, 0.5, 0];

export const NoiseField = ({ count, reduced }: { count: number; reduced: boolean }) => {
  const targets = useMemo(() => buildTargets(count), [count]);
  const geo = useMemo(() => {
    const g = new THREE.BufferGeometry();
    const seeds = new Float32Array(count);
    for (let i = 0; i < count; i++) seeds[i] = Math.random();
    g.setAttribute('position', new THREE.BufferAttribute(targets.slab.slice(), 3));
    g.setAttribute('aFrom', new THREE.BufferAttribute(targets.slab.slice(), 3));
    g.setAttribute('aTo', new THREE.BufferAttribute(targets.slab.slice(), 3));
    g.setAttribute('aSeed', new THREE.BufferAttribute(seeds, 1));
    g.boundingSphere = new THREE.Sphere(new THREE.Vector3(), 10);
    return g;
  }, [count, targets]);
  const mat = useMemo(() => new THREE.ShaderMaterial({
    vertexShader: vert, fragmentShader: frag, transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
    uniforms: { uT: { value: 0 }, uTime: { value: 0 }, uSwirl: { value: 0 }, uSize: { value: 2.2 }, uColorA: { value: CYAN.clone() }, uColorB: { value: VIOLET.clone() }, uMix: { value: 0 } },
  }), []);
  const lastChapter = useRef(-1);
  const tmpB = useMemo(() => new THREE.Color(), []);

  useFrame((_, dt) => {
    const { chapter, t } = progressStore;
    if (chapter !== lastChapter.current) {
      const [from, to] = CHAPTER_TARGETS[chapter];
      (geo.getAttribute('aFrom') as THREE.BufferAttribute).copyArray(targets[from]).needsUpdate = true;
      (geo.getAttribute('aTo') as THREE.BufferAttribute).copyArray(targets[to]).needsUpdate = true;
      lastChapter.current = chapter;
    }
    const u = mat.uniforms;
    u.uT.value = reduced ? 1 : THREE.MathUtils.damp(u.uT.value, t, 6, dt);
    u.uTime.value += dt;
    u.uSwirl.value = chapter === 2 ? 1 : chapter === 4 ? 0.5 : 0.15;
    u.uMix.value = THREE.MathUtils.damp(u.uMix.value, MIX_BY_CHAPTER[chapter], 4, dt);
    tmpB.copy(chapter === 6 ? GREEN : VIOLET);
    (u.uColorB.value as THREE.Color).lerp(tmpB, Math.min(1, dt * 3));
  });

  return <points geometry={geo} material={mat} frustumCulled={false} />;
};
```

- [ ] **Step 2: `Accents.tsx`** — labelled meshes per chapter. Visibility via `useFrame` damping a group's scale/opacity from the store (no React state per frame).

```tsx
import { useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import { Html, RoundedBox } from '@react-three/drei';
import * as THREE from 'three';
import { progressStore } from './store';
import { SHARD_POSITIONS, NODE_POSITIONS, DRIVE_POSITIONS } from './targets';

/** Scales a group 0↔1 depending on whether the current chapter is in `chapters`. */
const useChapterGroup = (chapters: number[], enter = 0.15) => {
  const ref = useRef<THREE.Group>(null);
  useFrame((_, dt) => {
    if (!ref.current) return;
    const { chapter, t } = progressStore;
    const on = chapters.includes(chapter) && (chapters[0] !== chapter || t > enter);
    const s = THREE.MathUtils.damp(ref.current.scale.x, on ? 1 : 0.0001, 5, dt);
    ref.current.scale.setScalar(s);
    ref.current.visible = s > 0.01;
  });
  return ref;
};

const Label = ({ children, position }: { children: string; position: [number, number, number] }) => (
  <Html position={position} center distanceFactor={8} className="pointer-events-none select-none whitespace-nowrap rounded-md border border-border bg-background/70 px-2 py-1 font-mono text-[11px] text-foreground/90 backdrop-blur">
    {children}
  </Html>
);

const Prism = ({ position, color = '#33e0ff' }: { position: [number, number, number]; color?: string }) => (
  <mesh position={position}>
    <octahedronGeometry args={[0.32, 0]} />
    <meshStandardMaterial color={color} emissive={color} emissiveIntensity={0.6} metalness={0.2} roughness={0.3} />
  </mesh>
);

export const Accents = () => {
  const shards = useChapterGroup([3, 4]);
  const nodes = useChapterGroup([4]);
  const key = useChapterGroup([5]);
  const ring = useChapterGroup([6]);
  const drive = useChapterGroup([7]);
  const retry = useRef<THREE.Mesh>(null);
  const ringMat = useRef<THREE.MeshStandardMaterial>(null);

  useFrame(({ clock }) => {
    // Chapter 4: one shard "fails", arcs back out, then re-lands (SQS retry).
    if (retry.current) {
      const { chapter, t } = progressStore;
      const k = chapter === 4 ? THREE.MathUtils.smoothstep(t, 0.3, 0.9) : 0;
      const bounce = Math.sin(k * Math.PI);
      retry.current.position.set(2.2 * (1 - k) + 0.4 * bounce, -1.0 * (1 - k) + 1.6 * bounce, 0.5 * bounce);
      retry.current.rotation.y = clock.elapsedTime * 2;
    }
    if (ringMat.current) {
      const { chapter, t } = progressStore;
      ringMat.current.emissiveIntensity = chapter === 6 ? 0.4 + t * 1.2 : 0.4;
    }
  });

  return (
    <>
      <group ref={shards}>
        {SHARD_POSITIONS.map((p, i) => (
          <group key={i}>
            <Prism position={p} />
            <Label position={[p[0], p[1] - 0.6, p[2]]}>{`chunk_00${i + 1}.2xpfm`}</Label>
          </group>
        ))}
      </group>
      <group ref={nodes}>
        <mesh position={NODE_POSITIONS.s3}><cylinderGeometry args={[0.9, 0.9, 0.5, 48]} /><meshStandardMaterial color="#0b1a22" emissive="#33e0ff" emissiveIntensity={0.25} metalness={0.6} roughness={0.35} /></mesh>
        <Label position={[0, -0.75, 0]}>S3 · SSE-KMS</Label>
        <mesh position={NODE_POSITIONS.kms}><icosahedronGeometry args={[0.45, 1]} /><meshStandardMaterial color="#1a0b22" emissive="#b56bff" emissiveIntensity={0.5} /></mesh>
        <Label position={[NODE_POSITIONS.kms[0], NODE_POSITIONS.kms[1] - 0.7, NODE_POSITIONS.kms[2]]}>KMS CMK</Label>
        <mesh position={NODE_POSITIONS.dynamo}><boxGeometry args={[0.7, 0.7, 0.7]} /><meshStandardMaterial color="#0b1a22" emissive="#33e0ff" emissiveIntensity={0.35} /></mesh>
        <Label position={[NODE_POSITIONS.dynamo[0], NODE_POSITIONS.dynamo[1] - 0.7, NODE_POSITIONS.dynamo[2]]}>DynamoDB · sha256</Label>
        <mesh ref={retry}><octahedronGeometry args={[0.28, 0]} /><meshStandardMaterial color="#ffb02e" emissive="#ffb02e" emissiveIntensity={0.9} /></mesh>
        <Label position={[1.4, 1.2, 0.5]}>SQS retry</Label>
      </group>
      <group ref={key}>
        <RoundedBox args={[2.4, 1.5, 0.06]} radius={0.06} position={[0, 0, 0.8]}>
          <meshStandardMaterial color="#0a0c14" emissive="#33e0ff" emissiveIntensity={0.15} metalness={0.7} roughness={0.25} />
        </RoundedBox>
        <Label position={[0, 0.45, 0.85]}>.2xpfm.key</Label>
        <Label position={[0, -0.1, 0.85]}>seed · chunk map · never uploaded</Label>
      </group>
      <group ref={ring}>
        <mesh rotation={[Math.PI / 2, 0, 0]}><torusGeometry args={[1.8, 0.06, 16, 96]} /><meshStandardMaterial ref={ringMat} color="#062015" emissive="#3dffa0" emissiveIntensity={0.4} /></mesh>
        <Label position={[0, 0, 0]}>100% · all shards verified</Label>
      </group>
      <group ref={drive}>
        {DRIVE_POSITIONS.map((p, i) => (
          <group key={i}>
            <mesh position={p} rotation={[Math.PI / 2, 0, 0]}><cylinderGeometry args={[0.8, 0.8, 0.12, 48]} /><meshStandardMaterial color="#0b1a22" emissive="#33e0ff" emissiveIntensity={0.3} /></mesh>
            <Label position={[p[0], p[1] - 1.0, p[2]]}>{`Drive ${i + 1} · 15 GB`}</Label>
          </group>
        ))}
      </group>
    </>
  );
};
```

- [ ] **Step 3: `LandingScene.tsx`**

```tsx
import { forwardRef, useImperativeHandle, useRef, useEffect, useState } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { EffectComposer, Bloom, Vignette } from '@react-three/postprocessing';
import * as THREE from 'three';
import { chapterProgress } from '@/lib/chapters';
import { progressStore } from './store';
import { NoiseField } from './NoiseField';
import { Accents } from './Accents';

export type LandingSceneHandle = { setProgress(p: number): void; setMouse(x: number, y: number): void };

const CameraRig = ({ reduced }: { reduced: boolean }) => {
  const { camera } = useThree();
  useFrame((_, dt) => {
    const { chapter, mouse } = progressStore;
    const z = chapter === 2 ? 9 : chapter === 4 || chapter === 7 ? 8 : 7;
    const targetX = reduced ? 0 : mouse[0] * 0.6;
    const targetY = reduced ? 0 : mouse[1] * 0.4;
    camera.position.x = THREE.MathUtils.damp(camera.position.x, targetX, 3, dt);
    camera.position.y = THREE.MathUtils.damp(camera.position.y, targetY, 3, dt);
    camera.position.z = THREE.MathUtils.damp(camera.position.z, z, 2, dt);
    camera.lookAt(0, 0, 0);
  });
  return null;
};

/** Pauses rendering when the tab is hidden. */
const usePauseWhenHidden = () => {
  const [visible, setVisible] = useState(!document.hidden);
  useEffect(() => {
    const on = () => setVisible(!document.hidden);
    document.addEventListener('visibilitychange', on);
    return () => document.removeEventListener('visibilitychange', on);
  }, []);
  return visible;
};

const LandingScene = forwardRef<LandingSceneHandle, { reducedMotion: boolean; mobile: boolean; className?: string }>(
  ({ reducedMotion, mobile, className }, ref) => {
    useImperativeHandle(ref, () => ({
      setProgress(p) {
        const { chapter, t } = chapterProgress(p);
        progressStore.p = p; progressStore.chapter = chapter; progressStore.t = t;
      },
      setMouse(x, y) { progressStore.mouse = [x, y]; },
    }), []);
    const visible = usePauseWhenHidden();
    const count = mobile ? 6000 : 20000;
    const cameraRef = useRef<THREE.PerspectiveCamera | null>(null);
    return (
      <div className={className} aria-hidden>
        <Canvas dpr={[1, 2]} frameloop={visible ? 'always' : 'never'} camera={{ position: [0, 0, 7], fov: 45 }} gl={{ antialias: true, powerPreference: 'high-performance', alpha: true }} onCreated={({ camera }) => { cameraRef.current = camera as THREE.PerspectiveCamera; }}>
          <color attach="background" args={['#07080d']} />
          <ambientLight intensity={0.4} />
          <pointLight position={[4, 4, 6]} intensity={30} color="#33e0ff" />
          <pointLight position={[-4, -2, 4]} intensity={20} color="#b56bff" />
          <CameraRig reduced={reducedMotion} />
          <NoiseField count={count} reduced={reducedMotion} />
          <Accents />
          {!mobile && (
            <EffectComposer disableNormalPass>
              <Bloom intensity={0.9} luminanceThreshold={0.2} luminanceSmoothing={0.6} mipmapBlur />
              <Vignette eskil={false} offset={0.2} darkness={0.8} />
            </EffectComposer>
          )}
        </Canvas>
      </div>
    );
  },
);
LandingScene.displayName = 'LandingScene';
export default LandingScene;
```

- [ ] **Step 4: Manual verify** — temporarily mount in the Landing stub inside your worktree (do not commit): a `LandingScene` with a slider `input[type=range]` calling `ref.current.setProgress(v)`. Expected: slab of cyan particles at 0; grid at ~0.17; violet swirl at ~0.28; six labelled prisms at ~0.4; S3/KMS/DynamoDB nodes with an orange shard bouncing at ~0.5; key card at ~0.6; green ring at ~0.72; three discs at ~0.83; slab again at 1. 60 fps on an M-series laptop; no console warnings.
- [ ] **Step 5: `npx tsc -p tsconfig.app.json --noEmit && npm run lint`** → clean.
- [ ] **Step 6: Commit** `git add apps/web/src/three/landing && git commit -m "feat(web/three): landing particle morph scene with chapter accents"`

### Task 9: `AuthScene`

**Files:**
- Create: `apps/web/src/three/auth/AuthScene.tsx`

**Interfaces:**
- Produces: `export default ({ className }: { className?: string }) => JSX.Element`.

- [ ] **Step 1: Implement**

```tsx
import { useMemo, useRef } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { Instances, Instance } from '@react-three/drei';
import { EffectComposer, Bloom } from '@react-three/postprocessing';
import * as THREE from 'three';
import { useReducedMotion } from '@/hooks/useReducedMotion';

const COUNT = 180;

const Cloud = ({ reduced }: { reduced: boolean }) => {
  const group = useRef<THREE.Group>(null);
  const items = useMemo(() => Array.from({ length: COUNT }, (_, i) => ({
    pos: new THREE.Vector3((Math.random() - 0.5) * 10, (Math.random() - 0.5) * 8, (Math.random() - 0.5) * 6),
    rot: new THREE.Euler(Math.random() * Math.PI, Math.random() * Math.PI, 0),
    scale: 0.08 + Math.random() * 0.22,
    speed: 0.2 + Math.random() * 0.6,
    color: i % 5 === 0 ? '#b56bff' : '#33e0ff',
  })), []);
  const mouse = useRef([0, 0]);
  useFrame(({ pointer, clock }, dt) => {
    if (!group.current) return;
    mouse.current = [pointer.x, pointer.y];
    if (!reduced) {
      group.current.rotation.y = THREE.MathUtils.damp(group.current.rotation.y, pointer.x * 0.25, 2, dt);
      group.current.rotation.x = THREE.MathUtils.damp(group.current.rotation.x, -pointer.y * 0.2, 2, dt);
      group.current.children.forEach((c, i) => { c.position.y = items[i].pos.y + Math.sin(clock.elapsedTime * items[i].speed + i) * 0.25; c.rotation.z += dt * 0.2 * items[i].speed; });
    }
  });
  return (
    <group ref={group}>
      <Instances limit={COUNT}>
        <octahedronGeometry args={[1, 0]} />
        <meshStandardMaterial emissiveIntensity={0.8} metalness={0.3} roughness={0.4} />
        {items.map((it, i) => <Instance key={i} position={it.pos} rotation={it.rot} scale={it.scale} color={it.color} />)}
      </Instances>
    </group>
  );
};

const AuthScene = ({ className }: { className?: string }) => {
  const reduced = useReducedMotion();
  return (
    <div className={className} aria-hidden>
      <Canvas dpr={[1, 2]} camera={{ position: [0, 0, 8], fov: 50 }} gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}>
        <ambientLight intensity={0.5} />
        <pointLight position={[3, 3, 5]} intensity={25} color="#33e0ff" />
        <pointLight position={[-3, -2, 3]} intensity={15} color="#b56bff" />
        <Cloud reduced={reduced} />
        <EffectComposer disableNormalPass><Bloom intensity={0.7} luminanceThreshold={0.3} mipmapBlur /></EffectComposer>
      </Canvas>
    </div>
  );
};
export default AuthScene;
```

Note: `<Instance color>` needs the material to accept vertex colours — drei's `Instances` sets `vertexColors` automatically; if colours don't show, add `vertexColors` to the material.

- [ ] **Step 2: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint`; `npm run dev` → `/login` (Lane D's `AuthLayout` imports this path) shows a drifting shard cloud that tilts with the cursor.
- [ ] **Step 3: Commit** `git add apps/web/src/three/auth && git commit -m "feat(web/three): auth shard cloud scene"`

### Task 10: `ShardConstellation`

**Files:**
- Create: `apps/web/src/three/ShardConstellation.tsx`

**Interfaces:**
- Consumes: `ShardRecord` from `@/lib/api`; `groupShardsByTarget`, `formatBytes` from `@/lib/fileDetail`.
- Produces: `export default ({ shards, driveNames }: { shards: ShardRecord[]; driveNames: Record<string, string> }) => JSX.Element`.

- [ ] **Step 1: Implement**

```tsx
import { useMemo, useState } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { Html, OrbitControls } from '@react-three/drei';
import * as THREE from 'three';
import type { ShardRecord } from '@/lib/api';
import { groupShardsByTarget, formatBytes } from '@/lib/fileDetail';
import { useReducedMotion } from '@/hooks/useReducedMotion';

const STATUS_COLOR: Record<string, string> = { verified: '#3dffa0', pending: '#ffb02e', corrupted: '#ff4d4d', missing: '#7a2a2a' };

const Shard = ({ s, position, onHover }: { s: ShardRecord; position: [number, number, number]; onHover: (s: ShardRecord | null) => void }) => {
  const [hover, setHover] = useState(false);
  const color = STATUS_COLOR[s.status] ?? '#888';
  return (
    <mesh position={position} scale={hover ? 1.35 : 1}
      onPointerOver={(e) => { e.stopPropagation(); setHover(true); onHover(s); }}
      onPointerOut={() => { setHover(false); onHover(null); }}>
      <octahedronGeometry args={[0.22, 0]} />
      <meshStandardMaterial color={color} emissive={color} emissiveIntensity={s.status === 'pending' ? 1.2 : 0.6} />
      {hover && (
        <Html center distanceFactor={6} className="pointer-events-none whitespace-nowrap rounded-md border border-border bg-background/90 px-2 py-1 font-mono text-[11px]">
          <div>shard #{s.shard_id} · {s.status}</div>
          <div>{formatBytes(s.size)}</div>
          <div className="text-muted-foreground">sha256 {s.sha256.slice(0, 16)}…</div>
        </Html>
      )}
    </mesh>
  );
};

const Rings = ({ shards, driveNames, reduced }: { shards: ShardRecord[]; driveNames: Record<string, string>; reduced: boolean }) => {
  const targets = useMemo(() => groupShardsByTarget(shards, driveNames), [shards, driveNames]);
  const [, setHovered] = useState<ShardRecord | null>(null);
  const groups = useMemo(() => targets.map((t, ti) => {
    const R = 1.2 + ti * 1.1;
    return { label: t.label, R, items: t.shards.map((s, i) => {
      const a = (i / t.shards.length) * Math.PI * 2;
      return { s, position: [Math.cos(a) * R, (ti % 2 ? 0.3 : -0.3), Math.sin(a) * R] as [number, number, number] };
    }) };
  }), [targets]);
  const ref = useState(() => new THREE.Group())[0];
  useFrame((_, dt) => { if (!reduced) ref.rotation.y += dt * 0.08; });
  return (
    <primitive object={ref}>
      {groups.map((g, gi) => (
        <group key={gi}>
          <mesh rotation={[Math.PI / 2, 0, 0]}><torusGeometry args={[g.R, 0.006, 8, 128]} /><meshBasicMaterial color="#1e2433" /></mesh>
          <Html position={[g.R + 0.35, 0, 0]} center className="pointer-events-none whitespace-nowrap font-mono text-[10px] text-muted-foreground">{g.label}</Html>
          {g.items.map(({ s, position }) => <Shard key={s.shard_id} s={s} position={position} onHover={setHovered} />)}
        </group>
      ))}
    </primitive>
  );
};

const ShardConstellation = ({ shards, driveNames }: { shards: ShardRecord[]; driveNames: Record<string, string> }) => {
  const reduced = useReducedMotion();
  return (
    <div className="h-[420px] w-full rounded-2xl glass overflow-hidden" aria-hidden>
      <Canvas dpr={[1, 2]} frameloop={reduced ? 'demand' : 'always'} camera={{ position: [0, 3.5, 6], fov: 45 }} gl={{ antialias: true, alpha: true }}>
        <ambientLight intensity={0.6} />
        <pointLight position={[4, 5, 4]} intensity={30} color="#33e0ff" />
        <Rings shards={shards} driveNames={driveNames} reduced={reduced} />
        <OrbitControls enablePan={false} enableZoom={false} maxPolarAngle={Math.PI / 2.2} minPolarAngle={Math.PI / 4} />
      </Canvas>
    </div>
  );
};
export default ShardConstellation;
```

- [ ] **Step 2: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint` → clean. Mount temporarily with fake shards (3 verified, 1 pending, 1 corrupted across two buckets) in your worktree: two rings, colours correct, hover tooltip, drag to orbit. Don't commit the harness.
- [ ] **Step 3: Commit** `git add apps/web/src/three/ShardConstellation.tsx && git commit -m "feat(web/three): 3D shard constellation for file detail"`

---

## Lane B — Landing page and Guide

### Task 11: `useLenis` + landing chrome (Nav, ChapterRail, Cursor, Footer)

**Files:**
- Create: `apps/web/src/hooks/useLenis.ts`
- Create: `apps/web/src/components/landing/Nav.tsx`, `ChapterRail.tsx`, `Cursor.tsx`, `Footer.tsx`

**Interfaces:**
- Consumes: `CHAPTERS` (`@/lib/chapters`), `useAuth`, `useReducedMotion`, `MagneticButton`.
- Produces: `useLenis(enabled: boolean): void`; `Nav`, `ChapterRail({ current: number })`, `Cursor`, `Footer` (named exports).

- [ ] **Step 1: `useLenis.ts`**

```ts
import { useEffect } from 'react';
import Lenis from 'lenis';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

/** Smooth scroll synced with ScrollTrigger. No-op when disabled (reduced motion). */
export const useLenis = (enabled: boolean) => {
  useEffect(() => {
    if (!enabled) return;
    const lenis = new Lenis({ lerp: 0.1, smoothWheel: true });
    lenis.on('scroll', ScrollTrigger.update);
    const tick = (time: number) => lenis.raf(time * 1000);
    gsap.ticker.add(tick);
    gsap.ticker.lagSmoothing(0);
    return () => { gsap.ticker.remove(tick); lenis.destroy(); };
  }, [enabled]);
};
```

- [ ] **Step 2: `Nav.tsx`**

```tsx
import { Link } from 'react-router-dom';
import { useAuth } from '@/hooks/useAuth';
import { Button } from '@/components/ui/button';
import { MagneticButton } from '@/components/ds';

export const Nav = () => {
  const { isAuthenticated } = useAuth();
  return (
    <header className="fixed inset-x-0 top-0 z-50">
      <div className="mx-auto mt-4 flex max-w-6xl items-center justify-between rounded-full glass px-5 py-2.5">
        <Link to="/" className="flex items-center gap-2 font-semibold">
          <span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent shadow-glow-primary" />
          ShardX
        </Link>
        <nav className="flex items-center gap-1">
          <Button variant="ghost" size="sm" asChild><Link to="/guide">Guide</Link></Button>
          <Button variant="ghost" size="sm" asChild><a href="https://github.com/DEEPESH-845/shardX" target="_blank" rel="noreferrer">GitHub</a></Button>
          {isAuthenticated
            ? <MagneticButton size="sm" asChild><Link to="/files">Open vault</Link></MagneticButton>
            : <MagneticButton size="sm" asChild><Link to="/login">Sign in</Link></MagneticButton>}
        </nav>
      </div>
    </header>
  );
};
```

- [ ] **Step 3: `ChapterRail.tsx`**

```tsx
import { CHAPTERS } from '@/lib/chapters';
import { cn } from '@/lib/utils';

export const ChapterRail = ({ current }: { current: number }) => (
  <nav aria-label="Chapters" className="fixed right-6 top-1/2 z-40 hidden -translate-y-1/2 flex-col gap-3 lg:flex">
    {CHAPTERS.map((c, i) => (
      <a key={c.id} href={`#${c.id}`} className="group flex items-center justify-end gap-3" aria-current={i === current ? 'step' : undefined}>
        <span className={cn('font-mono text-[10px] uppercase tracking-widest transition-opacity', i === current ? 'opacity-100 text-primary' : 'opacity-0 group-hover:opacity-60')}>{c.label}</span>
        <span className={cn('h-1.5 w-1.5 rounded-full transition-all', i === current ? 'bg-primary shadow-glow-primary scale-150' : 'bg-muted-foreground/40')} />
      </a>
    ))}
  </nav>
);
```

- [ ] **Step 4: `Cursor.tsx`** — pointer devices only, no-op with reduced motion.

```tsx
import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { useReducedMotion } from '@/hooks/useReducedMotion';

export const Cursor = () => {
  const dot = useRef<HTMLDivElement>(null);
  const ring = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  useEffect(() => {
    if (reduced || !window.matchMedia('(pointer: fine)').matches || !dot.current || !ring.current) return;
    const dx = gsap.quickTo(dot.current, 'x', { duration: 0.1 }), dy = gsap.quickTo(dot.current, 'y', { duration: 0.1 });
    const rx = gsap.quickTo(ring.current, 'x', { duration: 0.4, ease: 'power3' }), ry = gsap.quickTo(ring.current, 'y', { duration: 0.4, ease: 'power3' });
    const move = (e: MouseEvent) => { dx(e.clientX); dy(e.clientY); rx(e.clientX); ry(e.clientY); };
    const over = (e: MouseEvent) => { const hot = (e.target as HTMLElement).closest('a,button,[role=button]'); gsap.to(ring.current, { scale: hot ? 2 : 1, duration: 0.3 }); };
    window.addEventListener('mousemove', move); window.addEventListener('mouseover', over);
    document.documentElement.classList.add('cursor-none');
    return () => { window.removeEventListener('mousemove', move); window.removeEventListener('mouseover', over); document.documentElement.classList.remove('cursor-none'); };
  }, [reduced]);
  return (
    <>
      <div ref={dot} aria-hidden className="pointer-events-none fixed left-0 top-0 z-[95] h-1.5 w-1.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary mix-blend-difference" />
      <div ref={ring} aria-hidden className="pointer-events-none fixed left-0 top-0 z-[95] h-8 w-8 -translate-x-1/2 -translate-y-1/2 rounded-full border border-primary/70 mix-blend-difference" />
    </>
  );
};
```

Add to `index.css` `@layer utilities`: `.cursor-none, .cursor-none * { cursor: none !important; }` (Lane B may append this one utility to `index.css`; it is the only exception to ownership).

- [ ] **Step 5: `Footer.tsx`**

```tsx
import { Link } from 'react-router-dom';
export const Footer = () => (
  <footer className="relative z-10 border-t border-border/60 bg-background/80 py-10 backdrop-blur">
    <div className="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 px-6 text-sm text-muted-foreground md:flex-row">
      <span>© {new Date().getFullYear()} ShardX · MIT</span>
      <div className="flex gap-6">
        <Link to="/guide" className="hover:text-foreground">Guide</Link>
        <a href="https://github.com/DEEPESH-845/shardX" className="hover:text-foreground" target="_blank" rel="noreferrer">GitHub</a>
        <Link to="/login" className="hover:text-foreground">Sign in</Link>
      </div>
    </div>
  </footer>
);
```

- [ ] **Step 6: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint` → clean.
- [ ] **Step 7: Commit** `git add apps/web/src/hooks/useLenis.ts apps/web/src/components/landing apps/web/src/index.css && git commit -m "feat(web/landing): lenis hook, nav, chapter rail, cursor, footer"`

### Task 12: Chapter content + CSS/SVG fallbacks

**Files:**
- Create: `apps/web/src/components/landing/content.tsx`
- Create: `apps/web/src/components/landing/Fallback.tsx`

**Interfaces:**
- Produces: `CHAPTER_CONTENT: { id: string; eyebrow: string; headline: string; body: string; pills: string[] }[]` (9, same order/ids as `CHAPTERS`); `ChapterFallback({ id }: { id: string })` — inline SVG illustration per chapter.

- [ ] **Step 1: `content.tsx`**

```tsx
export const CHAPTER_CONTENT = [
  { id: 'hero', eyebrow: 'Zero-knowledge object storage', headline: 'Your files become noise.', body: 'ShardX obfuscates every upload with a ChaCha20 noise stream, splits it into shards and stores them under your own KMS key. The only thing that can put them back together is a key file that never leaves your device.', pills: ['AWS S3 · KMS · DynamoDB', 'Go + React', 'MIT'] },
  { id: 'upload', eyebrow: '01 / Upload', headline: 'Streamed in 5 MB chunks.', body: 'Your browser opens an upload session and streams the file chunk by chunk. Pause, resume or cancel at any time — a failed chunk retries with exponential backoff, never from zero.', pills: ['5 MB chunks', 'pause / resume', 'retry with backoff'] },
  { id: 'obfuscate', eyebrow: '02 / Obfuscate', headline: 'ChaCha20 noise, everywhere.', body: 'A 32-byte seed initialises a ChaCha20-DRBG. Deterministic noise is injected at calculated offsets throughout the file. What reaches the bucket is statistically indistinguishable from random bytes.', pills: ['32-byte seed', 'ChaCha20-DRBG', 'deterministic offsets'] },
  { id: 'shard', eyebrow: '03 / Shard', headline: 'Split by strategy.', body: 'The obfuscated stream is cut into shards using the strategy you pick — balanced, greedy, proportional or manual. Preview the plan before you commit.', pills: ['balanced', 'greedy', 'proportional', 'manual'] },
  { id: 'store', eyebrow: '04 / Store', headline: 'S3, under your KMS key.', body: 'Each shard is written to S3 with SSE-KMS using a customer-managed key. Its SHA-256, size and location are recorded in DynamoDB. A shard that fails is queued to SQS and retried by a worker — you never have to re-upload.', pills: ['SSE-KMS CMK', 'DynamoDB shard records', 'SQS retry + DLQ'] },
  { id: 'keyfile', eyebrow: '05 / Key file', headline: 'The server never sees the seed.', body: 'The seed and the chunk map exist only in your .2xpfm.key file. A full compromise of the bucket and the database yields nothing readable. Keep it safe; it is the only way back.', pills: ['.2xpfm.key', 'client-only', 'seed + chunk map'] },
  { id: 'integrity', eyebrow: '06 / Integrity', headline: 'Every shard, verified.', body: 'The Integrity Engine scores each file, maps where every shard lives and records an audit timeline of every lifecycle event — straight from DynamoDB and EventBridge.', pills: ['health score', 'shard map', 'audit timeline'] },
  { id: 'drive', eyebrow: '07 / Drive mode', headline: 'No AWS? Use Drive.', body: 'The same pipeline can shard a file across several linked Google Drive accounts. Three free accounts give you 45 GB of zero-knowledge storage without a cloud bill.', pills: ['3 × 15 GB', 'OAuth linking', 'same pipeline'] },
  { id: 'cta', eyebrow: 'Begin', headline: 'Create your vault.', body: 'Sign up, drop a file, download your key. Read the guide if you want the whole story first.', pills: [] },
] as const;
```

- [ ] **Step 2: `Fallback.tsx`** — one `<svg>` per chapter, 320×200, using `currentColor` strokes with `text-primary`. Keep each drawing to ≤ 8 primitives: hero = a rounded rect; upload = 5×4 grid of squares; obfuscate = 60 scattered `<circle r=1.5>` generated in a loop; shard = 6 diamonds; store = a cylinder + two boxes with arrows; keyfile = a card with a key glyph; integrity = a ring + tick; drive = three circles; cta = the rounded rect again. Export:

```tsx
export const ChapterFallback = ({ id }: { id: string }) => (
  <svg viewBox="0 0 320 200" className="mx-auto h-auto w-full max-w-sm text-primary" role="img" aria-label={`${id} illustration`}>
    {/* ...per-id switch here... */}
  </svg>
);
```

- [ ] **Step 3: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint` → clean.
- [ ] **Step 4: Commit** `git add apps/web/src/components/landing && git commit -m "feat(web/landing): chapter copy and svg fallbacks"`

### Task 13: `Landing.tsx`

**Files:**
- Modify: `apps/web/src/pages/Landing.tsx` (replace stub)

**Interfaces:**
- Consumes: `LandingScene` default export + `LandingSceneHandle` from `@/three/landing/LandingScene` (Lane A). Until it lands, create an **uncommitted** stub in your worktree: `forwardRef(() => <div/>)` with matching types.
- Consumes: `CHAPTERS`, `chapterProgress`, `CHAPTER_CONTENT`, `ChapterFallback`, `Nav`, `ChapterRail`, `Cursor`, `Footer`, `useLenis`, `useReducedMotion`, `useWebGL`, `SplitText`, `Reveal`, `MagneticButton`, `Loader`.

- [ ] **Step 1: Implement**

```tsx
import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import { useGSAP } from '@gsap/react';
import { ArrowDown } from 'lucide-react';
import { CHAPTERS, chapterProgress } from '@/lib/chapters';
import { useLenis } from '@/hooks/useLenis';
import { useReducedMotion } from '@/hooks/useReducedMotion';
import { useWebGL } from '@/hooks/useWebGL';
import { MagneticButton, Reveal, SplitText } from '@/components/ds';
import { Nav } from '@/components/landing/Nav';
import { ChapterRail } from '@/components/landing/ChapterRail';
import { Cursor } from '@/components/landing/Cursor';
import { Footer } from '@/components/landing/Footer';
import { CHAPTER_CONTENT } from '@/components/landing/content';
import { ChapterFallback } from '@/components/landing/Fallback';
import type { LandingSceneHandle } from '@/three/landing/LandingScene';

gsap.registerPlugin(ScrollTrigger, useGSAP);

const LandingScene = lazy(() => import('@/three/landing/LandingScene'));

const useIsMobile = () => {
  const [m, setM] = useState(() => window.innerWidth < 768);
  useEffect(() => {
    const on = () => setM(window.innerWidth < 768);
    window.addEventListener('resize', on);
    return () => window.removeEventListener('resize', on);
  }, []);
  return m;
};

const Landing = () => {
  const reduced = useReducedMotion();
  const webgl = useWebGL();
  const mobile = useIsMobile();
  const scene = useRef<LandingSceneHandle>(null);
  const main = useRef<HTMLElement>(null);
  const [current, setCurrent] = useState(0);
  useLenis(!reduced);

  // One master ScrollTrigger → scene progress + current chapter (state only on change).
  useGSAP(() => {
    if (!main.current) return;
    const st = ScrollTrigger.create({
      trigger: main.current, start: 'top top', end: 'bottom bottom', scrub: reduced ? false : 0.8,
      onUpdate: (self) => {
        scene.current?.setProgress(self.progress);
        const c = chapterProgress(self.progress).chapter;
        setCurrent((prev) => (prev === c ? prev : c));
      },
    });
    return () => st.kill();
  }, { dependencies: [reduced] });

  useEffect(() => {
    if (reduced) return;
    const on = (e: MouseEvent) => scene.current?.setMouse((e.clientX / window.innerWidth) * 2 - 1, -((e.clientY / window.innerHeight) * 2 - 1));
    window.addEventListener('mousemove', on);
    return () => window.removeEventListener('mousemove', on);
  }, [reduced]);

  return (
    <div className="relative">
      <Nav />
      <Cursor />
      {webgl ? (
        <Suspense fallback={null}>
          <LandingScene ref={scene} reducedMotion={reduced} mobile={mobile} className="fixed inset-0 z-0" />
        </Suspense>
      ) : (
        <div className="fixed inset-0 z-0 bg-[radial-gradient(ellipse_at_center,hsl(228_20%_8%),hsl(228_20%_4%))]" />
      )}
      <ChapterRail current={current} />

      <main ref={main} className="relative z-10">
        {CHAPTER_CONTENT.map((c, i) => (
          <section key={c.id} id={c.id} aria-labelledby={`${c.id}-h`} className={i === 0 ? 'flex min-h-screen items-center' : 'min-h-[120vh]'}>
            <div className={`sticky top-[28vh] mx-auto w-full max-w-6xl px-6 ${i % 2 ? 'lg:pl-[52%]' : 'lg:pr-[52%]'}`}>
              <p className="eyebrow mb-4">{c.eyebrow}</p>
              <h2 id={`${c.id}-h`} className={`font-semibold leading-[1.02] tracking-tight ${i === 0 ? 'text-5xl md:text-7xl' : 'text-4xl md:text-6xl'}`}>
                <SplitText text={c.headline} by="word" stagger={0.06} trigger={i !== 0} />
              </h2>
              <Reveal delay={0.2} className="mt-6 max-w-lg text-base leading-relaxed text-muted-foreground md:text-lg"><p>{c.body}</p></Reveal>
              {c.pills.length > 0 && (
                <Reveal delay={0.35} className="mt-6 flex flex-wrap gap-2">
                  {c.pills.map((p) => <span key={p} className="rounded-full border border-border bg-card/60 px-3 py-1 font-mono text-xs text-foreground/80">{p}</span>)}
                </Reveal>
              )}
              {!webgl && <div className="mt-8"><ChapterFallback id={c.id} /></div>}
              {i === 0 && (
                <Reveal delay={0.5} className="mt-10 flex flex-wrap items-center gap-3">
                  <MagneticButton size="lg" asChild><Link to="/signup">Create your vault</Link></MagneticButton>
                  <MagneticButton size="lg" variant="outline" asChild><a href="#upload">See how it works <ArrowDown className="ml-2 h-4 w-4" /></a></MagneticButton>
                </Reveal>
              )}
              {i === CHAPTERS.length - 1 && (
                <Reveal delay={0.4} className="mt-10 flex flex-wrap items-center gap-3">
                  <MagneticButton size="lg" asChild><Link to="/signup">Create your vault</Link></MagneticButton>
                  <MagneticButton size="lg" variant="outline" asChild><Link to="/guide">Read the guide</Link></MagneticButton>
                </Reveal>
              )}
            </div>
          </section>
        ))}
      </main>
      <Footer />
    </div>
  );
};
export default Landing;
```

- [ ] **Step 2: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm run build`; `dist/assets` must show `Landing-*.js` and a separate `LandingScene-*.js`/three chunk, and the main `index-*.js` must not contain "three" (`grep -l "THREE.REVISION\|WebGLRenderer" dist/assets/index-*.js` returns nothing). `npm run dev` → scroll through all 9 chapters: rail updates, text reveals, scene morphs (or fallback SVGs when WebGL is off — test with `about:config webgl.disabled=true` in Firefox or `--disable-webgl` in Chrome). Enable "reduce motion" in OS: no Lenis, no cursor, headlines visible immediately.
- [ ] **Step 3: Commit** (exclude the stub) `git add apps/web/src/pages/Landing.tsx && git commit -m "feat(web): scroll-driven landing page"`

### Task 14: `Guide.tsx`

**Files:**
- Modify: `apps/web/src/pages/Guide.tsx` (replace stub)
- Create: `apps/web/src/components/guide/sections.tsx`

**Interfaces:**
- Produces: `GUIDE_SECTIONS: { id: string; title: string; body: JSX.Element }[]` (9 sections listed in the spec). Ids, in order: `what`, `account`, `upload`, `finalize`, `detail`, `keyfile`, `drive`, `reliability`, `faq` (Playwright relies on `keyfile`).

- [ ] **Step 1: `sections.tsx`** — write the manual. Each `body` is JSX using `<p>`, `<ul>`, `<table>`, `<code>`. Cover exactly: (1) What ShardX is; (2) Creating an account — email + password ≥ 6 chars, Cognito-backed, sign in afterwards; (3) Uploading — drop/select, 5 MB chunks, pause/resume/cancel, then a table of strategies:

| Strategy | What it does | Use when |
|---|---|---|
| balanced | Equal-size shards across all targets | default; unknown target sizes |
| greedy | Fills the target with the most free space first | one target is much larger |
| proportional | Shard sizes proportional to each target's free space | mixed capacities |
| manual | You pass explicit shard sizes | reproducible layouts / testing |

(4) After Finalize — returns immediately; status polls every 3 s: `uploading → processing → complete | failed`; (5) File Detail — health % = verified/total shards; shard colours green verified / amber pending / red corrupted / dark-red missing; timeline event names `FILE_CREATED, SHARD_VERIFIED, SHARD_QUEUED_FOR_RETRY, SHARD_CORRUPTED, FILE_HEALTHY, UPLOAD_FAILED`; (6) The Key File — `<filename>.2xpfm.key`, contains seed + chunk map, download only once status is complete, lose it = file unrecoverable, server cannot regenerate; (7) Drive mode — link Google accounts from Profile via popup; pooled free space; (8) Reliability — failed shard → SQS → worker retries; DLQ after 5 receives with a CloudWatch alarm; "pending" means retry in flight; (9) FAQ — 6 Q/As: "Can ShardX read my files?", "What if I lose the key file?", "Is the file downloadable yet?" (answer: reconstruction is on the roadmap; today you download the key file), "Which strategy should I pick?", "Where do my shards live?", "Does it work without AWS?".

- [ ] **Step 2: `Guide.tsx`**

```tsx
import { useEffect, useState } from 'react';
import { Nav } from '@/components/landing/Nav';
import { Footer } from '@/components/landing/Footer';
import { Reveal } from '@/components/ds';
import { GUIDE_SECTIONS } from '@/components/guide/sections';
import { cn } from '@/lib/utils';

const Guide = () => {
  const [active, setActive] = useState(GUIDE_SECTIONS[0].id);
  useEffect(() => {
    const io = new IntersectionObserver((entries) => {
      const top = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)[0];
      if (top) setActive(top.target.id);
    }, { rootMargin: '-30% 0px -60% 0px' });
    GUIDE_SECTIONS.forEach((s) => { const el = document.getElementById(s.id); if (el) io.observe(el); });
    return () => io.disconnect();
  }, []);
  return (
    <div>
      <Nav />
      <div className="mx-auto max-w-6xl px-6 pb-24 pt-32 lg:grid lg:grid-cols-[220px_1fr] lg:gap-16">
        <aside className="mb-10 lg:sticky lg:top-32 lg:mb-0 lg:self-start">
          <p className="eyebrow mb-4">User guide</p>
          <ol className="space-y-2 text-sm">
            {GUIDE_SECTIONS.map((s, i) => (
              <li key={s.id}><a href={`#${s.id}`} className={cn('flex gap-3 transition-colors hover:text-foreground', active === s.id ? 'text-primary' : 'text-muted-foreground')}><span className="font-mono text-xs">{String(i + 1).padStart(2, '0')}</span>{s.title}</a></li>
            ))}
          </ol>
        </aside>
        <article className="prose-invert max-w-none space-y-20">
          {GUIDE_SECTIONS.map((s, i) => (
            <Reveal key={s.id} as="section" className="scroll-mt-32">
              <section id={s.id} aria-labelledby={`${s.id}-h`}>
                <p className="eyebrow mb-3">{String(i + 1).padStart(2, '0')}</p>
                <h2 id={`${s.id}-h`} className="mb-6 text-3xl font-semibold tracking-tight">{s.title}</h2>
                <div className="space-y-4 text-muted-foreground [&_code]:rounded [&_code]:bg-muted [&_code]:px-1.5 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-xs [&_code]:text-foreground [&_strong]:text-foreground [&_table]:w-full [&_table]:text-sm [&_td]:border-t [&_td]:border-border [&_td]:py-2 [&_td]:pr-4 [&_th]:pb-2 [&_th]:text-left [&_th]:text-foreground [&_ul]:list-disc [&_ul]:pl-5">{s.body}</div>
              </section>
            </Reveal>
          ))}
        </article>
      </div>
      <Footer />
    </div>
  );
};
export default Guide;
```

- [ ] **Step 3: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint`; `/guide` renders 9 sections, TOC highlights while scrolling, anchors work.
- [ ] **Step 4: Commit** `git add apps/web/src/pages/Guide.tsx apps/web/src/components/guide && git commit -m "feat(web): written user guide page"`

---

## Lane C — App shell, app pages, tour

### Task 15: `AppShell` (replaces `Layout`)

**Files:**
- Create: `apps/web/src/components/AppShell.tsx`
- Delete: `apps/web/src/components/Layout.tsx` (after all three app pages import `AppShell` in Tasks 17–19; delete in Task 19's commit)

**Interfaces:**
- Consumes: `useAuth`, `getAuthEmail`, `getInitialsFromName`, `MagneticButton`.
- Produces: `AppShell: React.FC<{ children: ReactNode; onHelp?: () => void }>`; sidebar "?" button has `id="tour-help"`.

- [ ] **Step 1: Implement**

```tsx
import { useRef, useState, type ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import gsap from 'gsap';
import { useGSAP } from '@gsap/react';
import { Files, User, BookOpen, LogOut, HelpCircle, Menu, X } from 'lucide-react';
import { useAuth } from '@/hooks/useAuth';
import { getAuthEmail } from '@/lib/api';
import { getInitialsFromName, cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback } from '@/components/ui/avatar';

const NAV = [
  { to: '/files', label: 'Files', icon: Files, id: 'nav-files' },
  { to: '/profile', label: 'Profile', icon: User, id: 'nav-profile' },
  { to: '/guide', label: 'Guide', icon: BookOpen, id: 'nav-guide' },
];

export const AppShell = ({ children, onHelp }: { children: ReactNode; onHelp?: () => void }) => {
  const { logout } = useAuth();
  const { pathname } = useLocation();
  const [open, setOpen] = useState(false);
  const email = getAuthEmail();
  const pill = useRef<HTMLSpanElement>(null);
  const list = useRef<HTMLElement>(null);

  // Sliding active indicator.
  useGSAP(() => {
    const active = list.current?.querySelector<HTMLElement>('[aria-current="page"]');
    if (!active || !pill.current) return;
    gsap.to(pill.current, { y: active.offsetTop, height: active.offsetHeight, duration: 0.4, ease: 'power3.out' });
  }, { dependencies: [pathname], scope: list });

  const nav = (
    <nav ref={list} className="relative flex flex-col gap-1" aria-label="Primary">
      <span ref={pill} aria-hidden className="absolute left-0 w-full rounded-lg bg-primary/10 ring-1 ring-primary/30" style={{ height: 0 }} />
      {NAV.map(({ to, label, icon: Icon, id }) => {
        const active = pathname === to || (to === '/files' && pathname.startsWith('/files/'));
        return (
          <Link key={to} id={id} to={to} aria-current={active ? 'page' : undefined} onClick={() => setOpen(false)}
            className={cn('relative z-10 flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm transition-colors', active ? 'text-primary' : 'text-muted-foreground hover:text-foreground')}>
            <Icon className="h-4 w-4" /> {label}
          </Link>
        );
      })}
    </nav>
  );

  const user = (
    <div className="mt-auto space-y-2">
      <Button id="tour-help" variant="ghost" size="sm" className="w-full justify-start gap-3 text-muted-foreground" onClick={onHelp}><HelpCircle className="h-4 w-4" /> Show me around</Button>
      <div className="flex items-center gap-3 rounded-lg border border-border/60 p-2">
        <Avatar className="h-8 w-8"><AvatarFallback className="bg-gradient-to-br from-primary to-accent text-xs text-background">{getInitialsFromName(email ?? 'Account')}</AvatarFallback></Avatar>
        <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">{email ?? 'Account'}</span>
        <Button variant="ghost" size="icon" aria-label="Log out" onClick={logout}><LogOut className="h-4 w-4" /></Button>
      </div>
    </div>
  );

  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[240px_1fr]">
      <aside className="hidden lg:flex lg:flex-col lg:gap-8 lg:border-r lg:border-border/60 lg:p-5 lg:sticky lg:top-0 lg:h-screen">
        <Link to="/" className="flex items-center gap-2 px-2 font-semibold"><span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent shadow-glow-primary" /> ShardX</Link>
        {nav}
        {user}
      </aside>
      <header className="sticky top-0 z-40 flex items-center justify-between border-b border-border/60 glass px-4 py-3 lg:hidden">
        <Link to="/" className="flex items-center gap-2 font-semibold"><span className="h-3 w-3 rotate-45 bg-gradient-to-br from-primary to-accent" /> ShardX</Link>
        <Button variant="ghost" size="icon" aria-label={open ? 'Close menu' : 'Open menu'} aria-expanded={open} onClick={() => setOpen((v) => !v)}>{open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}</Button>
      </header>
      {open && (
        <div className="fixed inset-0 z-30 flex flex-col gap-6 bg-background p-5 pt-20 lg:hidden">
          {nav}{user}
        </div>
      )}
      <main className="mx-auto w-full max-w-[1200px] px-5 py-8 lg:px-8 lg:py-10">{children}</main>
    </div>
  );
};
```

- [ ] **Step 2: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint` → clean.
- [ ] **Step 3: Commit** `git add apps/web/src/components/AppShell.tsx && git commit -m "feat(web): sidebar app shell"`

### Task 16: Tour reducer + `Tour` component

**Files:**
- Create: `apps/web/src/lib/tour.ts`
- Test: `apps/web/src/lib/__tests__/tour.test.ts`
- Create: `apps/web/src/components/Tour.tsx`

**Interfaces:**
- Produces: `TourStep = { selector: string; title: string; body: string; placement: 'top'|'bottom'|'left'|'right' }`; `TourState = { open: boolean; index: number }`; `TourAction = { type: 'start' } | { type: 'next'; available: boolean[] } | { type: 'prev'; available: boolean[] } | { type: 'skip' } | { type: 'done' }`; `tourReducer(state, action): TourState`; `TOUR_DONE_KEY = 'shardx_tour_done'`; `FILES_TOUR: TourStep[]`.
- Produces: `Tour: React.FC<{ steps: TourStep[]; state: TourState; dispatch: React.Dispatch<TourAction> }>`.

- [ ] **Step 1: Failing test**

```ts
import { describe, it, expect } from 'vitest';
import { tourReducer } from '../tour';

const all = [true, true, true, true];
describe('tourReducer', () => {
  it('start opens at 0', () => expect(tourReducer({ open: false, index: 3 }, { type: 'start' })).toEqual({ open: true, index: 0 }));
  it('next advances', () => expect(tourReducer({ open: true, index: 0 }, { type: 'next', available: all })).toEqual({ open: true, index: 1 }));
  it('next skips unavailable steps', () => expect(tourReducer({ open: true, index: 0 }, { type: 'next', available: [true, false, true, true] })).toEqual({ open: true, index: 2 }));
  it('next past the end closes', () => expect(tourReducer({ open: true, index: 3 }, { type: 'next', available: all })).toEqual({ open: false, index: 3 }));
  it('prev goes back, skipping unavailable', () => expect(tourReducer({ open: true, index: 2 }, { type: 'prev', available: [true, false, true, true] })).toEqual({ open: true, index: 0 }));
  it('prev at 0 stays', () => expect(tourReducer({ open: true, index: 0 }, { type: 'prev', available: all })).toEqual({ open: true, index: 0 }));
  it('skip/done close', () => {
    expect(tourReducer({ open: true, index: 1 }, { type: 'skip' }).open).toBe(false);
    expect(tourReducer({ open: true, index: 1 }, { type: 'done' }).open).toBe(false);
  });
});
```

- [ ] **Step 2: Run** `npx vitest run src/lib/__tests__/tour.test.ts` → FAIL.

- [ ] **Step 3: `tour.ts`**

```ts
export type TourStep = { selector: string; title: string; body: string; placement: 'top' | 'bottom' | 'left' | 'right' };
export type TourState = { open: boolean; index: number };
export type TourAction = { type: 'start' } | { type: 'next'; available: boolean[] } | { type: 'prev'; available: boolean[] } | { type: 'skip' } | { type: 'done' };

export const TOUR_DONE_KEY = 'shardx_tour_done';

export const FILES_TOUR: TourStep[] = [
  { selector: '#dropzone', title: 'Drop a file here', body: 'Anything up to 100 GB. It streams in 5 MB chunks and you can pause, resume or cancel.', placement: 'bottom' },
  { selector: '#strategy', title: 'Pick a distribution strategy', body: 'Balanced is a safe default. Preview the plan to see where each shard will land before you finalize.', placement: 'top' },
  { selector: '#nav-files', title: 'Health & shard map', body: 'Once processing completes, open a file to see its health score, a 3D map of every shard and the audit timeline.', placement: 'right' },
  { selector: '#downloadKey', title: 'Download your key file', body: 'The .2xpfm.key holds the seed and chunk map. The server never has it — without it nobody, including us, can rebuild the file.', placement: 'top' },
];

export const tourReducer = (s: TourState, a: TourAction): TourState => {
  switch (a.type) {
    case 'start': return { open: true, index: 0 };
    case 'skip': case 'done': return { ...s, open: false };
    case 'next': {
      let i = s.index + 1;
      while (i < a.available.length && !a.available[i]) i++;
      return i >= a.available.length ? { ...s, open: false } : { open: true, index: i };
    }
    case 'prev': {
      let i = s.index - 1;
      while (i >= 0 && !a.available[i]) i--;
      return i < 0 ? s : { open: true, index: i };
    }
  }
};
```

- [ ] **Step 4: Run** → PASS. Commit: `git add apps/web/src/lib/tour.ts apps/web/src/lib/__tests__/tour.test.ts && git commit -m "feat(web): tour reducer"`

- [ ] **Step 5: `Tour.tsx`**

```tsx
import { useEffect, useLayoutEffect, useRef, useState, type Dispatch } from 'react';
import gsap from 'gsap';
import { Button } from '@/components/ui/button';
import { useReducedMotion } from '@/hooks/useReducedMotion';
import type { TourAction, TourState, TourStep } from '@/lib/tour';

const PAD = 8;

export const Tour = ({ steps, state, dispatch }: { steps: TourStep[]; state: TourState; dispatch: Dispatch<TourAction> }) => {
  const spot = useRef<HTMLDivElement>(null);
  const card = useRef<HTMLDivElement>(null);
  const reduced = useReducedMotion();
  const [rect, setRect] = useState<DOMRect | null>(null);
  const available = steps.map((s) => !!document.querySelector(s.selector));
  const step = steps[state.index];

  // If the current step's target isn't on the page, advance.
  useEffect(() => {
    if (state.open && !available[state.index]) dispatch({ type: 'next', available });
  });

  useLayoutEffect(() => {
    if (!state.open) return;
    const el = document.querySelector(step.selector);
    if (!el) return;
    el.scrollIntoView({ block: 'center', behavior: reduced ? 'auto' : 'smooth' });
    const measure = () => setRect(el.getBoundingClientRect());
    measure();
    window.addEventListener('resize', measure);
    window.addEventListener('scroll', measure, true);
    return () => { window.removeEventListener('resize', measure); window.removeEventListener('scroll', measure, true); };
  }, [state.open, state.index, step, reduced]);

  useLayoutEffect(() => {
    if (!rect || !spot.current || !card.current) return;
    const target = { x: rect.left - PAD, y: rect.top - PAD, width: rect.width + PAD * 2, height: rect.height + PAD * 2 };
    gsap.to(spot.current, { ...target, duration: reduced ? 0 : 0.5, ease: 'power3.inOut' });
    const c = card.current.getBoundingClientRect();
    const pos = {
      top: { x: rect.left, y: rect.top - c.height - 16 },
      bottom: { x: rect.left, y: rect.bottom + 16 },
      left: { x: rect.left - c.width - 16, y: rect.top },
      right: { x: rect.right + 16, y: rect.top },
    }[step.placement];
    const x = Math.max(12, Math.min(window.innerWidth - c.width - 12, pos.x));
    const y = Math.max(12, Math.min(window.innerHeight - c.height - 12, pos.y));
    gsap.to(card.current, { x, y, autoAlpha: 1, duration: reduced ? 0 : 0.4, ease: 'power3.out' });
    card.current.querySelector<HTMLElement>('button[data-primary]')?.focus();
  }, [rect, step, reduced]);

  useEffect(() => {
    if (!state.open) return;
    const on = (e: KeyboardEvent) => {
      if (e.key === 'Escape') dispatch({ type: 'skip' });
      if (e.key === 'ArrowRight') dispatch({ type: 'next', available });
      if (e.key === 'ArrowLeft') dispatch({ type: 'prev', available });
      if (e.key === 'Tab' && card.current) {
        const f = card.current.querySelectorAll<HTMLElement>('button');
        const first = f[0], last = f[f.length - 1];
        if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
        else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
      }
    };
    window.addEventListener('keydown', on);
    return () => window.removeEventListener('keydown', on);
  });

  if (!state.open) return null;
  const isLast = !available.slice(state.index + 1).some(Boolean);
  return (
    <div className="fixed inset-0 z-[80]" role="dialog" aria-modal="true" aria-labelledby="tour-title">
      <div className="absolute inset-0" onClick={() => dispatch({ type: 'skip' })} />
      <div ref={spot} className="pointer-events-none absolute left-0 top-0 rounded-xl ring-2 ring-primary/70 shadow-[0_0_0_9999px_rgba(3,4,8,0.72)]" />
      <div ref={card} className="absolute left-0 top-0 w-80 rounded-2xl glass p-5 opacity-0">
        <p className="eyebrow mb-2">{state.index + 1} / {steps.length}</p>
        <h3 id="tour-title" className="text-base font-semibold">{step.title}</h3>
        <p className="mt-2 text-sm text-muted-foreground">{step.body}</p>
        <div className="mt-4 flex items-center justify-between">
          <Button variant="ghost" size="sm" onClick={() => dispatch({ type: 'skip' })}>Skip</Button>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => dispatch({ type: 'prev', available })} disabled={state.index === 0}>Back</Button>
            <Button data-primary size="sm" onClick={() => dispatch(isLast ? { type: 'done' } : { type: 'next', available })}>{isLast ? 'Done' : 'Next'}</Button>
          </div>
        </div>
      </div>
    </div>
  );
};
```

- [ ] **Step 6: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint`. Commit: `git add apps/web/src/components/Tour.tsx && git commit -m "feat(web): spotlight tour component"`

### Task 17: `Files.tsx` restyle + tour wiring

**Files:**
- Modify: `apps/web/src/pages/Files.tsx` — JSX only, plus tour state hooks at the top of the component.

**Interfaces:**
- Consumes: `AppShell`, `Tour`, `tourReducer`, `FILES_TOUR`, `TOUR_DONE_KEY`, `GlowCard`, `MagneticButton`, `Reveal`.
- Requires DOM ids: `#dropzone` on the drop card, `#strategy` on the strategy control, `#finalize`, `#downloadKey` (keep — the Playwright tests use them).

- [ ] **Step 1: Tour state** (add after existing `useState` lines; keep every existing handler as-is)

```tsx
const [tour, dispatchTour] = useReducer(tourReducer, { open: false, index: 0 });
useEffect(() => {
  if (localStorage.getItem(TOUR_DONE_KEY) !== '1') dispatchTour({ type: 'start' });
}, []);
useEffect(() => {
  if (!tour.open && tour.index > 0) localStorage.setItem(TOUR_DONE_KEY, '1');
}, [tour]);
```

(`import { useReducer } ...`; `import { tourReducer, FILES_TOUR, TOUR_DONE_KEY } from '@/lib/tour'`.) The "done on any close" rule is intentional: skipping counts as done; the sidebar "?" replays.

- [ ] **Step 2: JSX** — replace `<Layout>` with `<AppShell onHelp={() => dispatchTour({ type: 'start' })}>` and append `<Tour steps={FILES_TOUR} state={tour} dispatch={dispatchTour} />` inside it. Then restyle:

Header:
```tsx
<div className="mb-8">
  <p className="eyebrow mb-2">Vault</p>
  <h1 className="text-3xl font-semibold tracking-tight">Your uploads</h1>
  <p className="mt-1 text-muted-foreground">Drop a file. It becomes noise, gets sharded and stored — and only your key file can bring it back.</p>
</div>
```

Drop zone (same handlers):
```tsx
<div id="dropzone" onDragOver={...} onDragLeave={...} onDrop={handleDrop}
  className={cn('relative overflow-hidden rounded-2xl border-2 border-dashed p-12 text-center transition-all duration-300', isDragging ? 'border-primary bg-primary/5 shadow-glow-primary scale-[1.01]' : 'border-border hover:border-primary/50')}>
  <div aria-hidden className="pointer-events-none absolute inset-0 bg-[linear-gradient(hsl(var(--border))_1px,transparent_1px),linear-gradient(90deg,hsl(var(--border))_1px,transparent_1px)] bg-[size:32px_32px] opacity-30 [mask-image:radial-gradient(ellipse_at_center,black,transparent_70%)]" />
  <div className="relative">
    <div className="mx-auto mb-4 grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary"><Upload className="h-6 w-6" /></div>
    <h2 className="text-lg font-semibold">Drop a file to shard it</h2>
    <p className="mt-1 text-sm text-muted-foreground">or</p>
    <input type="file" id="fileInput" className="sr-only" onChange={handleFileInput} />
    <MagneticButton asChild className="mt-4"><label htmlFor="fileInput" className="cursor-pointer">Choose a file</label></MagneticButton>
  </div>
</div>
```

Upload cards: each becomes `<GlowCard key={upload.sessionId}>` with: filename (`font-medium truncate`) + size (`font-mono text-xs text-muted-foreground`), a status pill (`rounded-full border px-2 py-0.5 text-xs` coloured by status: uploading/processing `text-primary border-primary/40`, complete `text-success border-success/40`, failed `text-destructive border-destructive/40`, awaiting_strategy `text-warning border-warning/40`), a **segmented progress bar**:

```tsx
const SEGMENTS = 24;
const pct = upload.status === 'processing' ? upload.processingProgress || 0 : upload.progress;
<div className="mt-4 flex gap-1" role="progressbar" aria-valuenow={Math.round(pct)} aria-valuemin={0} aria-valuemax={100}>
  {Array.from({ length: SEGMENTS }, (_, i) => (
    <span key={i} className={cn('h-1.5 flex-1 rounded-full transition-colors duration-300', i < (pct / 100) * SEGMENTS ? 'bg-primary shadow-glow-primary' : 'bg-muted')} />
  ))}
</div>
```

Strategy picker → segmented control with `id="strategy"`:
```tsx
<div id="strategy" role="radiogroup" aria-label="Distribution strategy" className="inline-flex rounded-lg border border-border p-1">
  {(['balanced', 'greedy', 'proportional'] as ChunkingStrategy[]).map((s) => (
    <button key={s} type="button" role="radio" aria-checked={upload.strategy === s} onClick={() => updateUpload(upload.sessionId, { strategy: s })}
      className={cn('rounded-md px-3 py-1.5 font-mono text-xs transition-colors', upload.strategy === s ? 'bg-primary/15 text-primary' : 'text-muted-foreground hover:text-foreground')}>{s}</button>
  ))}
</div>
```

Plan preview → stacked bar:
```tsx
{upload.planPreview && (() => {
  const total = upload.planPreview.plan.reduce((n, p) => n + p.size, 0);
  const ids = [...new Set(upload.planPreview.plan.map((p) => p.drive_account_id))];
  const hue = (id: string) => 190 + (ids.indexOf(id) * 47) % 120;
  return (
    <div className="mt-3 space-y-2 text-xs">
      <div className="text-muted-foreground">{upload.planPreview.num_chunks} shards</div>
      <div className="flex h-3 w-full overflow-hidden rounded-full">
        {upload.planPreview.plan.map((p) => <span key={p.chunk_id} title={`chunk ${p.chunk_id} · ${formatBytes(p.size)}`} style={{ width: `${(p.size / total) * 100}%`, background: `hsl(${hue(p.drive_account_id)} 90% 55%)` }} />)}
      </div>
      <div className="flex flex-wrap gap-3 font-mono text-[11px] text-muted-foreground">{ids.map((id) => <span key={id} className="flex items-center gap-1"><span className="h-2 w-2 rounded-full" style={{ background: `hsl(${hue(id)} 90% 55%)` }} />{id.slice(-8)}</span>)}</div>
    </div>
  );
})()}
```
(`import { formatBytes } from '@/lib/fileDetail'`; `import { cn } from '@/lib/utils'`.)

Controls keep their ids: `<Button id="finalize" …>Finalize & process</Button>`, `<Button id="downloadKey" …>Download key file</Button>`, "View health & shard map" link to `/files/:id`. Pause/Resume/Cancel become icon+label buttons (`Pause`, `Play`, `X` from lucide).

Empty state (when `uploads.length === 0`) under the drop zone:
```tsx
<Reveal className="mt-10 text-center text-sm text-muted-foreground">
  Nothing uploaded yet. <button className="text-primary hover:underline" onClick={() => dispatchTour({ type: 'start' })}>Take the 30-second tour</button> or <Link to="/guide" className="text-primary hover:underline">read the guide</Link>.
</Reveal>
```

- [ ] **Step 3: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm test`; `npm run dev` with the API running: upload a small file end to end (chunks → strategy → preview → finalize → processing → complete → download key). Tour appears on first visit, not on reload; "?" replays it; Esc skips.
- [ ] **Step 4: Commit** `git add apps/web/src/pages/Files.tsx && git commit -m "feat(web): files page restyle with segmented progress, plan bar, tour"`

### Task 18: `FileDetail.tsx` + health ring + restyled ShardMap/AuditTimeline

**Files:**
- Modify: `apps/web/src/pages/FileDetail.tsx`, `apps/web/src/components/FileHealthCard.tsx`, `apps/web/src/components/ShardMap.tsx`, `apps/web/src/components/AuditTimeline.tsx`

**Interfaces:**
- Consumes: `ShardConstellation` default export from `@/three/ShardConstellation` (Lane A) — stub uncommitted locally if needed; `useWebGL`.

- [ ] **Step 1: `FileHealthCard`** — replace the number+bar with an SVG ring that counts up:

```tsx
const Ring = ({ value }: { value: number }) => {
  const ref = useRef<SVGCircleElement>(null);
  const num = useRef<HTMLSpanElement>(null);
  const reduced = useReducedMotion();
  const C = 2 * Math.PI * 54;
  useEffect(() => {
    if (!ref.current || !num.current) return;
    const o = { v: 0 };
    gsap.to(o, { v: value, duration: reduced ? 0 : 1.4, ease: 'power3.out', onUpdate: () => { ref.current!.style.strokeDashoffset = String(C - (C * o.v) / 100); num.current!.textContent = `${Math.round(o.v)}%`; } });
  }, [value, reduced, C]);
  return (
    <div className="relative h-36 w-36">
      <svg viewBox="0 0 120 120" className="h-full w-full -rotate-90">
        <circle cx="60" cy="60" r="54" className="fill-none stroke-muted" strokeWidth="8" />
        <circle ref={ref} cx="60" cy="60" r="54" className="fill-none stroke-primary drop-shadow-[0_0_8px_hsl(var(--primary)/0.7)]" strokeWidth="8" strokeLinecap="round" strokeDasharray={C} strokeDashoffset={C} />
      </svg>
      <span ref={num} className="absolute inset-0 grid place-items-center text-2xl font-semibold">0%</span>
    </div>
  );
};
```
Card body: ring left, level badge + `healthSummary` + integrity counts right; wrap in `GlowCard`.

- [ ] **Step 2: `ShardMap`** — wrap in `GlowCard`, shard tiles become `h-9 min-w-9 rounded-md font-mono text-xs` with `STATUS_CLASS` = `verified: 'bg-success/15 text-success border-success/40'`, `pending: 'bg-warning/15 text-warning border-warning/40 animate-pulse'`, `corrupted: 'bg-destructive/15 text-destructive border-destructive/40'`, `missing: 'bg-destructive/5 text-destructive/60 border-destructive/20'`. Logic unchanged.

- [ ] **Step 3: `AuditTimeline`** — vertical line `before:absolute before:left-[5px] before:top-2 before:bottom-2 before:w-px before:bg-gradient-to-b before:from-primary/60 before:to-transparent`, event dots `h-2.5 w-2.5 rounded-full shadow-[0_0_10px_currentColor]` with tone colours, timestamps `font-mono text-[11px]`. Wrap in `GlowCard`. Logic unchanged.

- [ ] **Step 4: `FileDetail.tsx`** — `AppShell` instead of `Layout`; header with back link, `eyebrow` "File", `h1` = sessionId in mono; grid: `FileHealthCard` + lazy `ShardConstellation` (only when `useWebGL()`; `Suspense` fallback = `Skeleton h-[420px]`), then `ShardMap`, then `AuditTimeline`. Queries unchanged.

```tsx
const ShardConstellation = lazy(() => import('@/three/ShardConstellation'));
// ...
{webgl && shards.data ? (
  <Suspense fallback={<Skeleton className="h-[420px] rounded-2xl" />}><ShardConstellation shards={shards.data.shards} driveNames={driveNames} /></Suspense>
) : null}
```

- [ ] **Step 5: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm test`; open a completed file: ring counts up, constellation renders, 2D map and timeline below.
- [ ] **Step 6: Commit** `git add apps/web/src/pages/FileDetail.tsx apps/web/src/components/FileHealthCard.tsx apps/web/src/components/ShardMap.tsx apps/web/src/components/AuditTimeline.tsx && git commit -m "feat(web): file detail with health ring and 3D constellation"`

### Task 19: `Profile.tsx`, `OAuthFinished.tsx`, `NotFound.tsx`; delete `Layout.tsx`

**Files:**
- Modify: `apps/web/src/pages/Profile.tsx` (JSX only), `apps/web/src/pages/OAuthFinished.tsx` (JSX only), `apps/web/src/pages/NotFound.tsx`
- Delete: `apps/web/src/components/Layout.tsx`

- [ ] **Step 1: Profile** — `AppShell`; header `eyebrow` "Storage", `h1` "Storage profile"; Refresh = `Button variant="outline"`, Link Drive = `MagneticButton`; totals card = `GlowCard` with three stats in `font-mono text-2xl`; each account = `GlowCard` with a small SVG gauge (reuse the `Ring` from Task 18 — move it to `src/components/Ring.tsx` and import in both places) at `h-20 w-20` showing used %, `Unlimited` shown as a pill when `formatBytes` returns "Unlimited". All handlers unchanged.
- [ ] **Step 2: OAuthFinished** — centered `GlowCard`, success icon in a `bg-success/10` circle, same effect hook, "Back to profile" `MagneticButton`.
- [ ] **Step 3: NotFound**

```tsx
import { Link } from 'react-router-dom';
import { MagneticButton } from '@/components/ds';
const NotFound = () => (
  <main className="grid min-h-screen place-items-center p-6 text-center">
    <div>
      <p className="eyebrow mb-4">404</p>
      <h1 className="text-4xl font-semibold tracking-tight md:text-6xl">This shard doesn't exist.</h1>
      <p className="mt-4 text-muted-foreground">The page you asked for was never written — or its key file is missing.</p>
      <div className="mt-8 flex justify-center gap-3">
        <MagneticButton asChild><Link to="/">Home</Link></MagneticButton>
        <MagneticButton variant="outline" asChild><Link to="/files">Your vault</Link></MagneticButton>
      </div>
    </div>
  </main>
);
export default NotFound;
```

- [ ] **Step 4:** `rm apps/web/src/components/Layout.tsx`; `grep -r "components/Layout" apps/web/src` → nothing.
- [ ] **Step 5: Verify** `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm test && npm run build`.
- [ ] **Step 6: Commit** `git add -A apps/web/src && git commit -m "feat(web): profile, oauth, 404 restyle; remove old layout"`

### Task 20: Playwright smoke tests

**Files:**
- Create: `apps/web/playwright.config.ts`, `apps/web/e2e/smoke.spec.ts`
- Modify: `apps/web/package.json` scripts: add `"e2e": "playwright test"`.

- [ ] **Step 1: Config**

```ts
import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  use: { baseURL: 'http://localhost:5173', headless: true },
  webServer: { command: 'npm run dev', port: 5173, reuseExistingServer: true },
});
```

- [ ] **Step 2: Tests** (no API needed — auth-free routes plus a fake token for the tour)

```ts
import { test, expect } from '@playwright/test';

test('landing reveals all chapters and CTA', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 2, name: /Your files become noise/ })).toBeVisible();
  await page.mouse.wheel(0, 20000);
  await page.waitForTimeout(1500);
  await expect(page.getByRole('link', { name: 'Read the guide' })).toBeVisible();
  for (const id of ['upload', 'obfuscate', 'shard', 'store', 'keyfile', 'integrity', 'drive', 'cta']) await expect(page.locator(`#${id}`)).toHaveCount(1);
});

test('reduced motion still shows headlines immediately', async ({ browser }) => {
  const ctx = await browser.newContext({ reducedMotion: 'reduce' });
  const page = await ctx.newPage();
  await page.goto('/');
  await expect(page.locator('#store h2')).toHaveText(/S3, under your KMS key/);
});

test('guide TOC navigates', async ({ page }) => {
  await page.goto('/guide');
  await page.getByRole('link', { name: /Key File/ }).first().click();
  await expect(page).toHaveURL(/#/);
  await expect(page.locator('section#keyfile h2')).toBeVisible();
});

test('login page renders split layout', async ({ page }) => {
  await page.goto('/login');
  await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible();
  await expect(page.getByLabel('Email')).toBeVisible();
});

test('tour shows once on /files', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('auth_token', 'x'));
  await page.goto('/files');
  await expect(page.getByRole('dialog')).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole('dialog')).toHaveCount(0);
  await page.locator('#tour-help').click();
  await expect(page.getByRole('dialog')).toBeVisible();
});
```

The guide test needs the keyfile section's `id` to be `keyfile` — set `GUIDE_SECTIONS[5].id = 'keyfile'` in Task 14.

- [ ] **Step 3: Run** `npx playwright install chromium && npm run e2e` → 5 passed.
- [ ] **Step 4: Commit** `git add apps/web/playwright.config.ts apps/web/e2e apps/web/package.json && git commit -m "test(web): playwright smoke for landing, guide, auth, tour"`

---

## Integration (orchestrator)

### Task 21: Merge lanes and verify

- [ ] Merge order: D (already on main) → A → B → C. Resolve conflicts only in `package-lock.json` and `index.css` (Lane B's `.cursor-none` utility).
- [ ] Delete any leftover uncommitted stubs; ensure `src/three/auth/AuthScene.tsx`, `src/three/landing/LandingScene.tsx`, `src/three/ShardConstellation.tsx` are the Lane A versions.
- [ ] Run: `npx tsc -p tsconfig.app.json --noEmit && npm run lint && npm test && npm run build && npm run e2e`.
- [ ] Bundle check: `ls -la dist/assets | sort -k5 -n | tail`; `index-*.js` gz ≤ 250 KB (`gzip -c dist/assets/index-*.js | wc -c`); landing + three chunks gz ≤ 700 KB combined.
- [ ] `grep -rn "three" dist/assets/index-*.js | head -1` → empty.
- [ ] Manual: Lighthouse on `/` and `/files` — perf ≥ 80 desktop, a11y ≥ 95. Fix any a11y flags before continuing.

### Task 22: Docs + review

- [ ] Update `apps/web/README.md` pages table: add `/` (landing, no API), `/guide` (no API); note WebGL/reduced-motion fallbacks and `npm run e2e`.
- [ ] Update root `README.md` architecture ASCII box: `React (Vite)` → add `/  /guide` rows.
- [ ] Run `superpowers:requesting-code-review` on the full diff `main..HEAD`; fix findings.
- [ ] Commit: `git commit -m "docs(web): document landing, guide, e2e"`.
