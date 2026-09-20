import { Link, useParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { ArrowLeft } from 'lucide-react';
import { api } from '@/lib/api';
import { FileHealthCard } from '@/components/FileHealthCard';
import { ShardMap } from '@/components/ShardMap';
import { AuditTimeline } from '@/components/AuditTimeline';

const FileDetail = () => {
  const { sessionId = '' } = useParams();

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
      <Layout>
        <div className="space-y-6">
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" asChild>
              <Link to="/files">
                <ArrowLeft className="w-4 h-4 mr-1" /> Uploads
              </Link>
            </Button>
          </div>
          <div>
            <h1 className="text-3xl font-bold mb-1">File Detail</h1>
            <p className="text-muted-foreground font-mono text-sm">{sessionId}</p>
          </div>

          {error && (
            <p className="text-sm text-destructive">{error instanceof Error ? error.message : 'Failed to load'}</p>
          )}

          <div className="grid gap-6 md:grid-cols-2">
            {health.data ? <FileHealthCard health={health.data} /> : <Skeleton className="h-40" />}
            {shards.data ? <ShardMap shards={shards.data.shards} driveNames={driveNames} /> : <Skeleton className="h-40" />}
          </div>
          {timeline.data ? <AuditTimeline events={timeline.data.events} /> : <Skeleton className="h-40" />}
        </div>
      </Layout>
    </ProtectedRoute>
  );
};

export default FileDetail;
