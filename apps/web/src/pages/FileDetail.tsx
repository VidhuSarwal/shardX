import { lazy, Suspense } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { AppShell } from '@/components/AppShell';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft } from 'lucide-react';
import { api } from '@/lib/api';
import { FileHealthCard } from '@/components/FileHealthCard';
import { ShardMap } from '@/components/ShardMap';
import { AuditTimeline } from '@/components/AuditTimeline';
import { useWebGL } from '@/hooks/useWebGL';

const ShardConstellation = lazy(() => import('@/three/ShardConstellation'));

const FileDetail = () => {
  const { sessionId = '' } = useParams();
  const webgl = useWebGL();

  // Poll while any shard is still pending (S3 retry worker in flight).
  const shards = useQuery({
    queryKey: ['file', sessionId, 'shards'],
    queryFn: () => api.getFileShards(sessionId),
    enabled: !!sessionId,
    refetchInterval: (q) => (q.state.data?.shards.some((s) => s.status === 'pending') ? 5000 : false),
  });
  const health = useQuery({
    queryKey: ['file', sessionId, 'health'],
    queryFn: () => api.getFileHealth(sessionId),
    enabled: !!sessionId,
    refetchInterval: shards.data?.shards.some((s) => s.status === 'pending') ? 5000 : false,
  });
  const timeline = useQuery({
    queryKey: ['file', sessionId, 'timeline'],
    queryFn: () => api.getFileTimeline(sessionId),
    enabled: !!sessionId,
    refetchInterval: shards.data?.shards.some((s) => s.status === 'pending') ? 5000 : false,
  });
  const accounts = useQuery({ queryKey: ['driveAccounts'], queryFn: api.getDriveAccounts, retry: false });
  const driveNames = Object.fromEntries((accounts.data ?? []).map((a) => [a.id, a.display_name]));

  const error = health.error ?? shards.error ?? timeline.error;

  return (
    <ProtectedRoute>
      <AppShell>
        <div className="mb-8">
          <Button variant="ghost" size="sm" asChild className="-ml-3 mb-4">
            <Link to="/files">
              <ArrowLeft className="w-4 h-4" /> Uploads
            </Link>
          </Button>
          <p className="eyebrow mb-2">File</p>
          <h1 className="font-mono text-2xl font-semibold tracking-tight">{sessionId}</h1>
        </div>

        {error && (
          <p className="mb-6 text-sm text-destructive">{error instanceof Error ? error.message : 'Failed to load'}</p>
        )}

        <div className="grid gap-6 md:grid-cols-2">
          {health.data ? <FileHealthCard health={health.data} /> : <Skeleton className="h-40 rounded-2xl" />}
          {webgl && shards.data ? (
            <Suspense fallback={<Skeleton className="h-[420px] rounded-2xl" />}>
              <ShardConstellation shards={shards.data.shards} driveNames={driveNames} />
            </Suspense>
          ) : null}
        </div>

        <div className="mt-6 space-y-6">
          {shards.data ? <ShardMap shards={shards.data.shards} driveNames={driveNames} /> : <Skeleton className="h-40 rounded-2xl" />}
          {timeline.data ? <AuditTimeline events={timeline.data.events} /> : <Skeleton className="h-40 rounded-2xl" />}
        </div>
      </AppShell>
    </ProtectedRoute>
  );
};

export default FileDetail;
