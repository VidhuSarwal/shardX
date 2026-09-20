import { describe, it, expect } from 'vitest';
import { tourReducer } from '../tour';

const all = [true, true, true, true];
describe('tourReducer', () => {
  it('start opens at 0', () => expect(tourReducer({ open: false, index: 3 }, { type: 'start' })).toEqual({ open: true, index: 0 }));
  it('next advances', () => expect(tourReducer({ open: true, index: 0 }, { type: 'next', available: all })).toEqual({ open: true, index: 1 }));
  it('next skips unavailable steps', () => expect(tourReducer({ open: true, index: 0 }, { type: 'next', available: [true, false, true, true] })).toEqual({ open: true, index: 2 }));
  it('next past the end closes', () => expect(tourReducer({ open: true, index: 3 }, { type: 'next', available: all })).toEqual({ open: false, index: 3 }));
  it('prev goes back, skipping unavailable', () => expect(tourReducer({ open: true, index: 2 }, { type: 'prev', available: [true, false, true, true] })).toEqual({ open: true, index: 0 }));
  it('prev at 0 stays', () => expect(tourReducer({ open: true, index: 0 }, { type: 'prev', available: all })).toEqual({ open: true, index: 0 }));
  it('skip/done close', () => {
    expect(tourReducer({ open: true, index: 1 }, { type: 'skip' }).open).toBe(false);
    expect(tourReducer({ open: true, index: 1 }, { type: 'done' }).open).toBe(false);
  });
});
