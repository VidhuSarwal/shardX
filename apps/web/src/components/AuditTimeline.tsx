import { GlowCard } from '@/components/ds';
import type { TimelineEvent } from '@/lib/api';
import { timelineLabel, timelineTone } from '@/lib/fileDetail';
import { cn } from '@/lib/utils';

const TONE_DOT: Record<ReturnType<typeof timelineTone>, string> = {
  ok: 'bg-success text-success',
  warn: 'bg-warning text-warning',
  error: 'bg-destructive text-destructive',
  info: 'bg-primary text-primary',
};

export const AuditTimeline = ({ events }: { events: TimelineEvent[] }) => (
  <GlowCard>
    <h3 className="text-base font-semibold">Audit timeline</h3>
    <p className="text-sm text-muted-foreground">{events.length} event{events.length === 1 ? '' : 's'}</p>
    <ol className="relative mt-4 space-y-4 before:absolute before:left-[5px] before:top-2 before:bottom-2 before:w-px before:bg-gradient-to-b before:from-primary/60 before:to-transparent">
      {events.map((e, i) => (
        <li key={i} className="relative pl-6">
          <span className={cn('absolute left-0 top-1.5 h-2.5 w-2.5 rounded-full shadow-[0_0_10px_currentColor]', TONE_DOT[timelineTone(e.type)])} />
          <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
            <span className="text-sm font-medium">{timelineLabel(e)}</span>
            <time className="font-mono text-[11px] text-muted-foreground whitespace-nowrap">
              {e.at ? new Date(e.at).toLocaleString() : '—'}
            </time>
          </div>
          <div className="text-xs text-muted-foreground font-mono">{e.type}</div>
          {e.detail && typeof e.detail['error'] === 'string' && (
            <p className="text-xs text-destructive mt-1">{String(e.detail['error'])}</p>
          )}
        </li>
      ))}
    </ol>
  </GlowCard>
);
