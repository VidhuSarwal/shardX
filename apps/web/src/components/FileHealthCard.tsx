import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { ShieldCheck, ShieldAlert, ShieldX, ShieldQuestion } from 'lucide-react';
import type { FileHealth } from '@/lib/api';
import { healthLevel, healthSummary, type HealthLevel } from '@/lib/fileDetail';

const LEVEL_UI: Record<HealthLevel, { icon: typeof ShieldCheck; className: string; label: string }> = {
  healthy: { icon: ShieldCheck, className: 'text-success', label: 'Healthy' },
  degraded: { icon: ShieldAlert, className: 'text-warning', label: 'Degraded' },
  critical: { icon: ShieldX, className: 'text-destructive', label: 'Critical' },
  unknown: { icon: ShieldQuestion, className: 'text-muted-foreground', label: 'Unknown' },
};

export const FileHealthCard = ({ health }: { health: FileHealth }) => {
  const level = healthLevel(health);
  const { icon: Icon, className, label } = LEVEL_UI[level];
  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between">
          <div>
            <CardTitle className="text-base">File Health</CardTitle>
            <CardDescription>{healthSummary(health)}</CardDescription>
          </div>
          <Badge variant="outline" className={className}>
            <Icon className="w-3.5 h-3.5 mr-1" />
            {label}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-end justify-between">
          <span className="text-3xl font-bold">{health.health_percentage.toFixed(0)}%</span>
          <span className="text-xs text-muted-foreground">
            {health.integrity_checks_passed}/{health.integrity_checks_total} integrity checks passed
          </span>
        </div>
        <Progress value={health.health_percentage} className="h-2" />
        {health.corruption_events > 0 && (
          <p className="text-xs text-destructive">
            {health.corruption_events} shard{health.corruption_events === 1 ? '' : 's'} not verified
          </p>
        )}
      </CardContent>
    </Card>
  );
};
