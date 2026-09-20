import { describe, it, expect } from 'vitest';
import { isTrustedOAuthMessage } from '@/lib/oauth';

describe('isTrustedOAuthMessage', () => {
  const origin = 'http://localhost:5173';

  it('accepts valid message with correct origin', () => {
    const event = { origin, data: { type: 'oauth_finished', success: true, provider: 'google' } } as any;
    expect(isTrustedOAuthMessage(event, origin)).toBe(true);
  });

  it('rejects message with wrong origin', () => {
    const event = { origin: 'https://evil.example', data: { type: 'oauth_finished', success: true } } as any;
    expect(isTrustedOAuthMessage(event, origin)).toBe(false);
  });

  it('rejects wrong shape', () => {
    const event = { origin, data: { type: 'something_else' } } as any;
    expect(isTrustedOAuthMessage(event, origin)).toBe(false);
  });
});
