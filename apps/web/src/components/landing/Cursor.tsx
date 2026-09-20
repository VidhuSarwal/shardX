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
    let shown = false;
    const move = (e: MouseEvent) => { if (!shown) { shown = true; gsap.set([dot.current, ring.current], { autoAlpha: 1 }); } dx(e.clientX); dy(e.clientY); rx(e.clientX); ry(e.clientY); };
    const over = (e: MouseEvent) => { const hot = (e.target as HTMLElement).closest('a,button,[role=button]'); gsap.to(ring.current, { scale: hot ? 2 : 1, duration: 0.3 }); };
    window.addEventListener('mousemove', move); window.addEventListener('mouseover', over);
    document.documentElement.classList.add('cursor-none');
    return () => { window.removeEventListener('mousemove', move); window.removeEventListener('mouseover', over); document.documentElement.classList.remove('cursor-none'); };
  }, [reduced]);
  const eligible = !reduced && typeof window !== 'undefined' && window.matchMedia('(pointer: fine)').matches;
  if (!eligible) return null;
  return (
    <>
      <div ref={dot} aria-hidden className="pointer-events-none fixed left-0 top-0 z-[95] h-1.5 w-1.5 opacity-0 -translate-x-1/2 -translate-y-1/2 rounded-full bg-primary mix-blend-difference" />
      <div ref={ring} aria-hidden className="pointer-events-none fixed left-0 top-0 z-[95] h-8 w-8 opacity-0 -translate-x-1/2 -translate-y-1/2 rounded-full border border-primary/70 mix-blend-difference" />
    </>
  );
};
