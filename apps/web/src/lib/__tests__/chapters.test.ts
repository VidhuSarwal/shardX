import { describe, it, expect } from 'vitest';
import { chapterProgress, CHAPTERS } from '../chapters';

describe('chapterProgress', () => {
  it('has 9 chapters', () => expect(CHAPTERS).toHaveLength(9));
  it('starts at chapter 0, t 0', () => expect(chapterProgress(0)).toEqual({ chapter: 0, t: 0 }));
  it('ends at last chapter, t 1', () => expect(chapterProgress(1)).toEqual({ chapter: 8, t: 1 }));
  it('maps midpoint of chapter 4', () => {
    const r = chapterProgress(4.5 / 9);
    expect(r.chapter).toBe(4);
    expect(r.t).toBeCloseTo(0.5, 5);
  });
  it('clamps out-of-range', () => {
    expect(chapterProgress(-1)).toEqual({ chapter: 0, t: 0 });
    expect(chapterProgress(2)).toEqual({ chapter: 8, t: 1 });
  });
  it('supports custom n', () => expect(chapterProgress(0.5, 2)).toEqual({ chapter: 1, t: 0 }));
});
