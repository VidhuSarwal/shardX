import { CHAPTERS } from '@/lib/chapters';
import { cn } from '@/lib/utils';

export const ChapterRail = ({ current }: { current: number }) => (
  <nav aria-label="Chapters" className="fixed right-6 top-1/2 z-40 hidden -translate-y-1/2 flex-col gap-3 lg:flex">
    {CHAPTERS.map((c, i) => (
      <a key={c.id} href={`#${c.id}`} className="group flex items-center justify-end gap-3" aria-current={i === current ? 'step' : undefined}>
        <span className={cn('font-mono text-[10px] uppercase tracking-widest transition-opacity', i === current ? 'opacity-100 text-primary' : 'opacity-0 group-hover:opacity-60 group-focus-visible:opacity-60')}>{c.label}</span>
        <span className={cn('h-1.5 w-1.5 rounded-full transition-all', i === current ? 'bg-primary shadow-glow-primary scale-150' : 'bg-muted-foreground/40')} />
      </a>
    ))}
  </nav>
);
