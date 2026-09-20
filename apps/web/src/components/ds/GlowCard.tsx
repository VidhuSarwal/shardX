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
