import { describe, it, expect } from 'vitest';
import { buildTargets, CHAPTER_TARGETS } from '../targets';

describe('buildTargets', () => {
  const t = buildTargets(1000, 7);
  it('every target has n*3 floats', () => {
    for (const arr of Object.values(t)) expect(arr.length).toBe(3000);
  });
  it('is deterministic for a seed', () => {
    expect(buildTargets(50, 3).noise).toEqual(buildTargets(50, 3).noise);
  });
  it('slab points lie inside the slab box', () => {
    for (let i = 0; i < 1000; i++) {
      expect(Math.abs(t.slab[i * 3])).toBeLessThanOrEqual(1.5);
      expect(Math.abs(t.slab[i * 3 + 1])).toBeLessThanOrEqual(1.0);
      expect(Math.abs(t.slab[i * 3 + 2])).toBeLessThanOrEqual(0.08);
    }
  });
  it('chapter map covers 9 chapters and chains', () => {
    expect(CHAPTER_TARGETS).toHaveLength(9);
    for (let i = 1; i < 9; i++) expect(CHAPTER_TARGETS[i][0]).toBe(CHAPTER_TARGETS[i - 1][1]);
  });
});
