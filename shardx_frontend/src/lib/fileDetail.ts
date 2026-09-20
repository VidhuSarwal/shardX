import type { FileHealth, ShardRecord, TimelineEvent } from '@/lib/api';

// Pure helpers behind the File Health card, Shard Map and Audit Timeline.
// Kept render-free so they can be unit-tested like src/lib/oauth.ts.

export type HealthLevel = 'healthy' | 'degraded' | 'critical' | 'unknown';

export const healthLevel = (h: Pick<FileHealth, 'health_percentage' | 'shards_total'>): HealthLevel => {
  if (h.shards_total === 0) return 'unknown';
  if (h.health_percentage >= 100) return 'healthy';
  if (h.health_percentage >= 50) return 'degraded';
  return 'critical';
};

export const healthSummary = (h: FileHealth): string =>
  h.shards_total === 0
    ? 'No shard records yet'
    : `${h.shards_available}/${h.shards_total} shards healthy`;

export type StorageTarget = {
  /** bucket + region for S3, drive account ID for Drive. */
  key: string;
  label: string;
  shards: ShardRecord[];
  bytes: number;
};

/** Groups shards by where they live, in shard-id order within each target. */
export const groupShardsByTarget = (
  shards: ShardRecord[],
  driveNames: Record<string, string> = {},
): StorageTarget[] => {
  const byKey = new Map<string, StorageTarget>();
  for (const s of [...shards].sort((a, b) => a.shard_id - b.shard_id)) {
    const bucket = s.bucket || 'unknown';
    const key = s.region ? `${bucket}@${s.region}` : bucket;
    let t = byKey.get(key);
    if (!t) {
      const label = s.region ? `s3://${bucket} (${s.region})` : driveNames[bucket] || `Drive ${bucket.slice(-6)}`;
      t = { key, label, shards: [], bytes: 0 };
      byKey.set(key, t);
    }
    t.shards.push(s);
    t.bytes += s.size;
  }
  return [...byKey.values()];
};

export const formatBytes = (n: number): string => {
  if (n < 1024) return `${n} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let v = n / 1024;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`;
};

const TIMELINE_LABELS: Record<string, string> = {
  FILE_CREATED: 'Upload finalized',
  SHARD_VERIFIED: 'Shard verified',
  SHARD_QUEUED_FOR_RETRY: 'Shard queued for retry',
  SHARD_CORRUPTED: 'Shard corrupted',
  FILE_HEALTHY: 'File healthy',
  UPLOAD_FAILED: 'Upload failed',
};

export const timelineLabel = (e: TimelineEvent): string => {
  const base = TIMELINE_LABELS[e.type] ?? e.type;
  const shard = e.detail?.['shard_id'];
  return typeof shard === 'number' ? `${base} #${shard}` : base;
};

export const timelineTone = (type: string): 'ok' | 'warn' | 'error' | 'info' => {
  if (type === 'SHARD_VERIFIED' || type === 'FILE_HEALTHY') return 'ok';
  if (type === 'SHARD_QUEUED_FOR_RETRY') return 'warn';
  if (type === 'SHARD_CORRUPTED' || type === 'UPLOAD_FAILED') return 'error';
  return 'info';
};
