export type OAuthFinishMessage = {
  type: 'oauth_finished';
  success: boolean;
  provider?: string;
};

export function isTrustedOAuthMessage(
  event: Pick<MessageEvent, 'origin' | 'data'>,
  expectedOrigin: string,
): event is MessageEvent & { data: OAuthFinishMessage } {
  if (event.origin !== expectedOrigin) return false;
  const data = (event as any).data;
  return (
    data &&
    typeof data === 'object' &&
    (data as any).type === 'oauth_finished' &&
    typeof (data as any).success === 'boolean'
  );
}
