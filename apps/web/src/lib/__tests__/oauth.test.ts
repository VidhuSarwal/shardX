import { describe, it, expect } from 'vitest';
import { isTrustedOAuthMessage } from '@/lib/oauth';

describe('isTrustedOAuthMessage', () => {
  const origin = 'http://localhost:5173';

  it('accepts valid message with correct origin', () => {
    const event = { origin, data: { type: 'oauth_finished', success: true, provider: 'google' } };
    expect(isTrustedOAuthMessage(event, origin)).toBe(true);
  });

  it('rejects message with wrong origin', () => {
    const event = { origin: 'https://evil.example', data: { type: 'oauth_finished', success: true } };
    expect(isTrustedOAuthMessage(event, origin)).toBe(false);
  });

  it('rejects wrong shape', () => {
    const event = { origin, data: { type: 'something_else' } };
    expect(isTrustedOAuthMessage(event, origin)).toBe(false);
  });
});
