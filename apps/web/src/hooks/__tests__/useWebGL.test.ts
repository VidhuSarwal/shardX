import { describe, it, expect, vi } from 'vitest';
import { hasWebGL } from '../useWebGL';

describe('hasWebGL', () => {
  it('is false when canvas has no webgl context', () => {
    const doc = { createElement: () => ({ getContext: () => null }) } as unknown as Document;
    expect(hasWebGL(doc)).toBe(false);
  });
  it('is true when webgl context exists', () => {
    const doc = { createElement: () => ({ getContext: () => ({}) }) } as unknown as Document;
    expect(hasWebGL(doc)).toBe(true);
  });
  it('is false when getContext throws', () => {
    const doc = { createElement: () => ({ getContext: () => { throw new Error('x'); } }) } as unknown as Document;
    expect(hasWebGL(doc)).toBe(false);
  });
});
