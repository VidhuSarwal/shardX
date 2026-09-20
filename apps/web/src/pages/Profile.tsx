import { useEffect, useRef, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Button } from '@/components/ui/button';
import { GlowCard, MagneticButton } from '@/components/ds';
import { Ring } from '@/components/Ring';
import { HardDrive, Plus, RefreshCw } from 'lucide-react';
import { api, type DriveAccount, type DriveSpace } from '@/lib/api';
import { FRONTEND_BASE_URL } from '@/lib/config';
import { isTrustedOAuthMessage } from '@/lib/oauth';
import { formatBytes as fmt } from '@/lib/fileDetail';
import { cn } from '@/lib/utils';
import { toast } from 'sonner';

// S3 mode reports free space as math.MaxInt64 (no usage accounting yet).
const UNLIMITED = 2 ** 62;
const formatBytes = (n: number) => (n >= UNLIMITED ? 'Unlimited' : fmt(n));

const Profile = () => {
  const [accounts, setAccounts] = useState<DriveAccount[]>([]);
  const [spaces, setSpaces] = useState<DriveSpace[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLinking, setIsLinking] = useState(false);
  const popupRef = useRef<Window | null>(null);
  const receivedMessageRef = useRef(false);
  const popupPollRef = useRef<number | null>(null);

  const loadData = async () => {
    setIsLoading(true);
    try {
      const [accountsData, spacesData] = await Promise.all([
        api.getDriveAccounts(),
        api.getDriveSpace(),
      ]);
      setAccounts(accountsData);
      setSpaces(spacesData);
    } catch (error) {
      toast.error('Failed to load drive accounts: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleLinkDrive = async () => {
    setIsLinking(true);
    receivedMessageRef.current = false;

    try {
      const { auth_url } = await api.getDriveLinkUrl();
      const popup = window.open(auth_url, 'oauth_popup', 'width=600,height=700');
      popupRef.current = popup;

      if (!popup) {
        toast.info('Please enable popups for this site to link your Drive account.');
        setIsLinking(false);
        return;
      }

      // Listen for postMessage from /oauth/finished
      const onMessage = (event: MessageEvent) => {
        if (!isTrustedOAuthMessage(event, FRONTEND_BASE_URL)) return;

        const { success } = event.data;
        receivedMessageRef.current = true;
        if (success) {
          popupRef.current?.close();
          loadData();
          setIsLinking(false);
          toast.success('Drive account linked.');
        } else {
          setIsLinking(false);
          toast.error('Drive linking failed.');
        }
        window.removeEventListener('message', onMessage);
        if (popupPollRef.current) {
          window.clearInterval(popupPollRef.current);
          popupPollRef.current = null;
        }
      };
      window.addEventListener('message', onMessage);

      // Detect popup closed without message
      popupPollRef.current = window.setInterval(() => {
        if (!popupRef.current || popupRef.current.closed) {
          if (!receivedMessageRef.current) {
            // Do not show an error; just refresh usage data as requested
            loadData();
            setIsLinking(false);
          }
          if (popupPollRef.current) {
            window.clearInterval(popupPollRef.current);
            popupPollRef.current = null;
          }
          window.removeEventListener('message', onMessage);
        }
      }, 500);
    } catch (error) {
      toast.error('Failed to start drive linking');
      setIsLinking(false);
    }
  };

  const totalSpace = spaces.reduce((sum, s) => sum + s.total_space, 0);
  const totalUsed = spaces.reduce((sum, s) => sum + s.used_space, 0);
  const totalFree = spaces.reduce((sum, s) => sum + s.free_space, 0);

  return (
    <ProtectedRoute>
      <AppShell>
        <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="eyebrow mb-2">Storage</p>
            <h1 className="text-3xl font-semibold tracking-tight">Storage profile</h1>
            <p className="mt-1 text-muted-foreground">Manage your connected Google Drive accounts</p>
          </div>
          <div className="flex gap-2">
            <Button onClick={loadData} variant="outline" disabled={isLoading}>
              <RefreshCw className="h-4 w-4" /> Refresh
            </Button>
            <MagneticButton onClick={handleLinkDrive} disabled={isLinking}>
              <Plus className="h-4 w-4" /> {isLinking ? 'Opening...' : 'Link Google Drive'}
            </MagneticButton>
          </div>
        </div>

        {/* Total Storage Overview */}
        {spaces.length > 0 && (
          <GlowCard className="bg-gradient-to-br from-primary/10 to-accent/10">
            <h2 className="text-base font-semibold">Total storage</h2>
            <p className="text-sm text-muted-foreground">Combined capacity across all accounts</p>
            <div className="mt-4 grid grid-cols-3 gap-4 text-center">
              <div>
                <div className="font-mono text-2xl">{formatBytes(totalSpace)}</div>
                <div className="text-sm text-muted-foreground">Total</div>
              </div>
              <div>
                <div className="font-mono text-2xl text-warning">{formatBytes(totalUsed)}</div>
                <div className="text-sm text-muted-foreground">Used</div>
              </div>
              <div>
                <div className="font-mono text-2xl text-success">{formatBytes(totalFree)}</div>
                <div className="text-sm text-muted-foreground">Free</div>
              </div>
            </div>
          </GlowCard>
        )}

        {/* Connected Accounts */}
        <div className="mt-10 space-y-4">
          <h2 className="text-xl font-semibold">Connected accounts</h2>

          {isLoading ? (
            <GlowCard className="flex items-center justify-center py-12">
              <div className="text-center">
                <div className="mx-auto mb-2 h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
                <p className="text-sm text-muted-foreground">Loading accounts...</p>
              </div>
            </GlowCard>
          ) : spaces.length === 0 ? (
            <GlowCard className="flex flex-col items-center justify-center py-12 text-center">
              <HardDrive className="mb-3 h-12 w-12 text-muted-foreground" />
              <h3 className="font-semibold">No drives connected</h3>
              <p className="mb-4 mt-1 text-sm text-muted-foreground">Link your first Google Drive to start uploading</p>
              <MagneticButton onClick={handleLinkDrive} disabled={isLinking}>
                <Plus className="h-4 w-4" /> Link Google Drive
              </MagneticButton>
            </GlowCard>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {spaces.map((space) => {
                const account = accounts.find((a) => a.id === space.account_id);
                const usedPct = space.total_space > 0 ? (space.used_space / space.total_space) * 100 : 0;
                const unlimited = formatBytes(space.total_space) === 'Unlimited';

                return (
                  <GlowCard key={space.account_id} className="flex items-start gap-4">
                    <Ring value={Math.min(100, usedPct)} className="h-20 w-20 shrink-0" />
                    <div className="min-w-0 flex-1 space-y-3">
                      <div className="flex items-start justify-between gap-2">
                        <div className="min-w-0">
                          <p className="truncate font-medium">{space.display_name}</p>
                          {account && (
                            <p className="text-xs text-muted-foreground">
                              Connected {new Date(account.created_at).toLocaleDateString()}
                            </p>
                          )}
                        </div>
                        {unlimited && (
                          <span className="shrink-0 rounded-full border border-primary/40 px-2 py-0.5 text-[11px] text-primary">Unlimited</span>
                        )}
                      </div>

                      {/* Owner details */}
                      {(space.owner_name || space.owner_email) && (
                        <div className="text-xs text-muted-foreground">
                          {space.owner_name && (
                            <div>
                              Owner: <span className="font-medium text-foreground">{space.owner_name}</span>
                            </div>
                          )}
                          {space.owner_email && (
                            <div className="truncate">
                              Email: <span className="font-mono text-foreground/80">{space.owner_email}</span>
                            </div>
                          )}
                        </div>
                      )}
                      <div className="flex justify-between text-sm">
                        <span className="text-muted-foreground">Used</span>
                        <span className="font-mono font-medium">
                          {formatBytes(space.used_space)} / {formatBytes(space.total_space)}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm font-medium text-success">
                          {formatBytes(space.free_space)} free
                        </span>
                        <span className={cn('rounded-full border px-2 py-0.5 text-xs', space.available ? 'text-success border-success/40' : 'text-destructive border-destructive/40')}>
                          {space.available ? 'Available' : 'Unavailable'}
                        </span>
                      </div>
                      {space.error && (
                        <div className="text-xs text-destructive/80">{space.error}</div>
                      )}
                    </div>
                  </GlowCard>
                );
              })}
            </div>
          )}
        </div>
      </AppShell>
    </ProtectedRoute>
  );
};

export default Profile;
