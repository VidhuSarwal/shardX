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
    gsap.set(ref.current, { opacity: 0, y: 24 });
    gsap.to(ref.current, { opacity: 1, y: 0, duration: 0.9, delay, ease: 'power3.out', scrollTrigger: { trigger: ref.current, start: 'top 85%', once: true } });
  }, { dependencies: [reduced] });
  return createElement(as as string, { ref, className }, children);
};
