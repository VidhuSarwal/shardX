import { GlowCard } from '@/components/ds';
import { Ring } from '@/components/Ring';
import { ShieldCheck, ShieldAlert, ShieldX, ShieldQuestion } from 'lucide-react';
import type { FileHealth } from '@/lib/api';
import { healthLevel, healthSummary, type HealthLevel } from '@/lib/fileDetail';
import { cn } from '@/lib/utils';

const LEVEL_UI: Record<HealthLevel, { icon: typeof ShieldCheck; className: string; label: string }> = {
  healthy: { icon: ShieldCheck, className: 'text-success border-success/40 bg-success/10', label: 'Healthy' },
  degraded: { icon: ShieldAlert, className: 'text-warning border-warning/40 bg-warning/10', label: 'Degraded' },
  critical: { icon: ShieldX, className: 'text-destructive border-destructive/40 bg-destructive/10', label: 'Critical' },
  unknown: { icon: ShieldQuestion, className: 'text-muted-foreground border-border bg-muted/10', label: 'Unknown' },
};

export const FileHealthCard = ({ health }: { health: FileHealth }) => {
  const level = healthLevel(health);
  const { icon: Icon, className, label } = LEVEL_UI[level];
  return (
    <GlowCard className="flex flex-col items-center gap-4 text-center sm:flex-row sm:items-center sm:gap-6 sm:text-left">
      <Ring value={health.health_percentage} className="shrink-0" />
      <div className="min-w-0 space-y-2">
        <span className={cn('inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs', className)}>
          <Icon className="h-3.5 w-3.5" /> {label}
        </span>
        <p className="text-sm text-muted-foreground">{healthSummary(health)}</p>
        <p className="text-xs text-muted-foreground">
          {health.integrity_checks_passed}/{health.integrity_checks_total} integrity checks passed
        </p>
        {health.corruption_events > 0 && (
          <p className="text-xs text-destructive">
            {health.corruption_events} shard{health.corruption_events === 1 ? '' : 's'} not verified
          </p>
        )}
      </div>
    </GlowCard>
  );
};
