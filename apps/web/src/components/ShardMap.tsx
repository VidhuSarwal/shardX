import { GlowCard } from '@/components/ds';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Database } from 'lucide-react';
import type { ShardRecord } from '@/lib/api';
import { formatBytes, groupShardsByTarget } from '@/lib/fileDetail';
import { cn } from '@/lib/utils';

const STATUS_CLASS: Record<string, string> = {
  verified: 'bg-success/15 text-success border-success/40',
  pending: 'bg-warning/15 text-warning border-warning/40 animate-pulse',
  corrupted: 'bg-destructive/15 text-destructive border-destructive/40',
  missing: 'bg-destructive/5 text-destructive/60 border-destructive/20',
};

export const ShardMap = ({
  shards,
  driveNames = {},
}: {
  shards: ShardRecord[];
  driveNames?: Record<string, string>;
}) => {
  const targets = groupShardsByTarget(shards, driveNames);
  return (
    <GlowCard>
      <h3 className="text-base font-semibold">Shard map</h3>
      <p className="text-sm text-muted-foreground">
        {shards.length} shard{shards.length === 1 ? '' : 's'} across {targets.length} storage target
        {targets.length === 1 ? '' : 's'}
      </p>
      <div className="mt-4 space-y-4">
        {targets.length === 0 && (
          <p className="text-sm text-muted-foreground">No shard records for this file yet.</p>
        )}
        {targets.map((t) => (
          <div key={t.key} className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="flex items-center gap-2 font-medium">
                <Database className="w-4 h-4 text-muted-foreground" />
                {t.label}
              </span>
              <span className="text-xs text-muted-foreground">
                {t.shards.length} shard{t.shards.length === 1 ? '' : 's'} · {formatBytes(t.bytes)}
              </span>
            </div>
            <div className="flex flex-wrap gap-1.5">
              {t.shards.map((s) => (
                <Tooltip key={s.shard_id}>
                  <TooltipTrigger asChild>
                    <div
                      className={cn('h-9 min-w-9 px-2 rounded-md border flex items-center justify-center font-mono text-xs', STATUS_CLASS[s.status] ?? 'bg-muted')}
                      aria-label={`shard ${s.shard_id} ${s.status}`}
                    >
                      {s.shard_id}
                    </div>
                  </TooltipTrigger>
                  <TooltipContent className="font-mono text-xs">
                    <div>shard #{s.shard_id} · {s.status}</div>
                    <div>{formatBytes(s.size)}</div>
                    <div className="text-muted-foreground">sha256 {s.sha256.slice(0, 16)}…</div>
                  </TooltipContent>
                </Tooltip>
              ))}
            </div>
          </div>
        ))}
      </div>
    </GlowCard>
  );
};
