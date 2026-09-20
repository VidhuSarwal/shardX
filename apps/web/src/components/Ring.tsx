import { useEffect, useRef } from 'react';
import gsap from 'gsap';
import { useReducedMotion } from '@/hooks/useReducedMotion';
import { cn } from '@/lib/utils';

/** Count-up SVG progress ring. Default size h-36 w-36; override via className. */
export const Ring = ({ value, className }: { value: number; className?: string }) => {
  const ref = useRef<SVGCircleElement>(null);
  const num = useRef<HTMLSpanElement>(null);
  const reduced = useReducedMotion();
  const C = 2 * Math.PI * 54;
  useEffect(() => {
    if (!ref.current || !num.current) return;
    const o = { v: 0 };
    const tween = gsap.to(o, {
      v: value,
      duration: reduced ? 0 : 1.4,
      ease: 'power3.out',
      onUpdate: () => {
        if (!ref.current || !num.current) return;
        ref.current.style.strokeDashoffset = String(C - (C * o.v) / 100);
        num.current.textContent = `${Math.round(o.v)}%`;
      },
    });
    return () => { tween.kill(); };
  }, [value, reduced, C]);
  return (
    <div className={cn('relative h-36 w-36', className)}>
      <svg viewBox="0 0 120 120" className="h-full w-full -rotate-90">
        <circle cx="60" cy="60" r="54" className="fill-none stroke-muted" strokeWidth="8" />
        <circle ref={ref} cx="60" cy="60" r="54" className="fill-none stroke-primary drop-shadow-[0_0_8px_hsl(var(--primary)/0.7)]" strokeWidth="8" strokeLinecap="round" strokeDasharray={C} strokeDashoffset={C} />
      </svg>
      <span ref={num} className="absolute inset-0 grid place-items-center text-2xl font-semibold">0%</span>
    </div>
  );
};
