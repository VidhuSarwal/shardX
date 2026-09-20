import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import type { TimelineEvent } from '@/lib/api';
import { timelineLabel, timelineTone } from '@/lib/fileDetail';

const TONE_DOT: Record<ReturnType<typeof timelineTone>, string> = {
  ok: 'bg-success',
  warn: 'bg-warning',
  error: 'bg-destructive',
  info: 'bg-primary',
};

export const AuditTimeline = ({ events }: { events: TimelineEvent[] }) => (
  <Card>
    <CardHeader>
      <CardTitle className="text-base">Audit Timeline</CardTitle>
      <CardDescription>{events.length} event{events.length === 1 ? '' : 's'}</CardDescription>
    </CardHeader>
    <CardContent>
      <ol className="relative border-l border-border ml-2 space-y-4">
        {events.map((e, i) => (
          <li key={i} className="ml-4">
            <span className={`absolute -left-1.5 mt-1.5 w-3 h-3 rounded-full ${TONE_DOT[timelineTone(e.type)]}`} />
            <div className="flex items-baseline justify-between gap-4">
              <span className="text-sm font-medium">{timelineLabel(e)}</span>
              <time className="text-xs text-muted-foreground whitespace-nowrap">
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
    </CardContent>
  </Card>
);
