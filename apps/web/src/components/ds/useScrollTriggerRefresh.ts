import { useEffect } from 'react';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

/** Recomputes ScrollTrigger boundaries once fonts and the window load event have settled. */
export const useScrollTriggerRefresh = () =>
  useEffect(() => {
    const r = () => ScrollTrigger.refresh();
    document.fonts?.ready.then(r);
    window.addEventListener('load', r);
    return () => window.removeEventListener('load', r);
  }, []);
