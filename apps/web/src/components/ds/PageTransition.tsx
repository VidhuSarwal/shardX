import { useLayoutEffect, useRef, type ReactNode } from 'react';
import { useLocation } from 'react-router-dom';
import gsap from 'gsap';
import { useReducedMotion } from '@/hooks/useReducedMotion';
import { useScrollTriggerRefresh } from './useScrollTriggerRefresh';

export const PageTransition = ({ children }: { children: ReactNode }) => {
  const ref = useRef<HTMLDivElement>(null);
  const { pathname } = useLocation();
  const reduced = useReducedMotion();
  useScrollTriggerRefresh();
  useLayoutEffect(() => {
    window.scrollTo(0, 0);
    if (reduced || !ref.current) return;
    gsap.fromTo(ref.current, { opacity: 0, filter: 'blur(8px)' }, { opacity: 1, filter: 'blur(0px)', duration: 0.5, ease: 'power2.out', clearProps: 'opacity,filter' });
  }, [pathname, reduced]);
  return <div ref={ref}>{children}</div>;
};
