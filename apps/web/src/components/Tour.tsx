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
