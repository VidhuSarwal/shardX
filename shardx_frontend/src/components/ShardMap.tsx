import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { Database } from 'lucide-react';
import type { ShardRecord } from '@/lib/api';
import { formatBytes, groupShardsByTarget } from '@/lib/fileDetail';

const STATUS_CLASS: Record<string, string> = {
  verified: 'bg-success/80 border-success',
  pending: 'bg-warning/80 border-warning animate-pulse',
  corrupted: 'bg-destructive/80 border-destructive',
  missing: 'bg-destructive/40 border-destructive',
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
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Shard Map</CardTitle>
        <CardDescription>
          {shards.length} shard{shards.length === 1 ? '' : 's'} across {targets.length} storage target
          {targets.length === 1 ? '' : 's'}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
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
                      className={`h-8 min-w-8 px-2 rounded border flex items-center justify-center text-xs font-mono text-white ${STATUS_CLASS[s.status] ?? 'bg-muted'}`}
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
      </CardContent>
    </Card>
  );
};
