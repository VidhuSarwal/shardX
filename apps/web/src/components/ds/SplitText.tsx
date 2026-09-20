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
    <span ref={ref} className={className}>
      <span className="sr-only">{text}</span>
      {parts.map((p, i) => (
        <span key={i} className="inline-block overflow-hidden align-bottom" aria-hidden>
          <span data-part className="inline-block will-change-transform">{p === ' ' ? ' ' : p}{by === 'word' && i < parts.length - 1 ? ' ' : ''}</span>
        </span>
      ))}
    </span>
  );
};
