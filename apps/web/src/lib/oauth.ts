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
  const data: unknown = event.data;
  return (
    typeof data === 'object' &&
    data !== null &&
    (data as Partial<OAuthFinishMessage>).type === 'oauth_finished' &&
    typeof (data as Partial<OAuthFinishMessage>).success === 'boolean'
  );
}
