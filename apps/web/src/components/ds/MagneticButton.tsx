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
