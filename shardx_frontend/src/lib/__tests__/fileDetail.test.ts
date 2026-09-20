import { describe, it, expect } from 'vitest';
import {
  healthLevel,
  healthSummary,
  groupShardsByTarget,
  formatBytes,
  timelineLabel,
  timelineTone,
} from '@/lib/fileDetail';
import type { FileHealth, ShardRecord } from '@/lib/api';

const health = (over: Partial<FileHealth>): FileHealth => ({
  file_id: 'f',
  health_percentage: 100,
  shards_available: 3,
  shards_total: 3,
  integrity_checks_passed: 3,
  integrity_checks_total: 3,
  corruption_events: 0,
  ...over,
});

const shard = (over: Partial<ShardRecord>): ShardRecord => ({
  file_id: 'f',
  shard_id: 1,
  sha256: 'abc',
  size: 1024,
  created_at: '2026-01-01T00:00:00Z',
  status: 'verified',
  ...over,
});

describe('healthLevel / healthSummary', () => {
  it('buckets by percentage', () => {
    expect(healthLevel(health({}))).toBe('healthy');
    expect(healthLevel(health({ health_percentage: 66.6 }))).toBe('degraded');
    expect(healthLevel(health({ health_percentage: 33.3 }))).toBe('critical');
    expect(healthLevel(health({ shards_total: 0, health_percentage: 0 }))).toBe('unknown');
  });

  it('summarizes X/Y shards', () => {
    expect(healthSummary(health({ shards_available: 2 }))).toBe('2/3 shards healthy');
    expect(healthSummary(health({ shards_total: 0 }))).toBe('No shard records yet');
  });
});

describe('groupShardsByTarget', () => {
  it('groups S3 shards by bucket+region and sums bytes', () => {
    const groups = groupShardsByTarget([
      shard({ shard_id: 2, bucket: 'b', region: 'us-east-1', size: 10 }),
      shard({ shard_id: 1, bucket: 'b', region: 'us-east-1', size: 5 }),
    ]);
    expect(groups).toHaveLength(1);
    expect(groups[0].label).toBe('s3://b (us-east-1)');
    expect(groups[0].bytes).toBe(15);
    expect(groups[0].shards.map((s) => s.shard_id)).toEqual([1, 2]);
  });

  it('labels Drive targets by account name when known', () => {
    const groups = groupShardsByTarget(
      [shard({ bucket: 'acct1' }), shard({ shard_id: 2, bucket: 'acct2' })],
      { acct1: 'Alice Drive' },
    );
    expect(groups.map((g) => g.label)).toEqual(['Alice Drive', 'Drive acct2']);
  });
});

describe('formatBytes', () => {
  it('scales units', () => {
    expect(formatBytes(512)).toBe('512 B');
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB');
    expect(formatBytes(150 * 1024 * 1024)).toBe('150 MB');
  });
});

describe('timeline helpers', () => {
  it('labels events and appends shard id', () => {
    expect(timelineLabel({ type: 'FILE_CREATED', at: null })).toBe('Upload finalized');
    expect(timelineLabel({ type: 'SHARD_VERIFIED', at: null, detail: { shard_id: 3 } })).toBe('Shard verified #3');
    expect(timelineLabel({ type: 'SOMETHING_NEW', at: null })).toBe('SOMETHING_NEW');
  });

  it('maps tone', () => {
    expect(timelineTone('FILE_HEALTHY')).toBe('ok');
    expect(timelineTone('SHARD_QUEUED_FOR_RETRY')).toBe('warn');
    expect(timelineTone('UPLOAD_FAILED')).toBe('error');
    expect(timelineTone('FILE_CREATED')).toBe('info');
  });
});
