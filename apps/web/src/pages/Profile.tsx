import { useEffect, useRef, useState } from 'react';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { HardDrive, Plus, RefreshCw } from 'lucide-react';
import { api, type DriveAccount, type DriveSpace } from '@/lib/api';
import { FRONTEND_BASE_URL } from '@/lib/config';
import { isTrustedOAuthMessage } from '@/lib/oauth';
import { formatBytes as fmt } from '@/lib/fileDetail';
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
      <Layout>
        <div className="space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold mb-2">Storage Profile</h1>
              <p className="text-muted-foreground">
              Manage your connected Google Drive accounts
              </p>
            </div>
            <div className="flex gap-2">
              <Button onClick={loadData} variant="outline" disabled={isLoading}>
                <RefreshCw className="w-4 h-4 mr-2" />
                Refresh
              </Button>
              <Button onClick={handleLinkDrive} disabled={isLinking}>
              <Plus className="w-4 h-4 mr-2" />
              {isLinking ? 'Opening...' : 'Link Google Drive'}
              </Button>
            </div>
          </div>

          {/* Total Storage Overview */}
          {spaces.length > 0 && (
            <Card className="bg-gradient-to-br from-primary/10 to-accent/10 border-primary/20">
              <CardHeader>
                <CardTitle>Total Storage</CardTitle>
                <CardDescription>
                  Combined capacity across all accounts
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <div className="text-2xl font-bold">{formatBytes(totalSpace)}</div>
                    <div className="text-sm text-muted-foreground">Total</div>
                  </div>
                  <div>
                    <div className="text-2xl font-bold text-warning">{formatBytes(totalUsed)}</div>
                    <div className="text-sm text-muted-foreground">Used</div>
                  </div>
                  <div>
                    <div className="text-2xl font-bold text-success">{formatBytes(totalFree)}</div>
                    <div className="text-sm text-muted-foreground">Free</div>
                  </div>
                </div>
                <Progress
                  value={totalSpace > 0 ? (totalUsed / totalSpace) * 100 : 0}
                  className="h-3"
                />
              </CardContent>
            </Card>
          )}

          {/* Connected Accounts */}
          <div className="space-y-4">
            <h2 className="text-xl font-semibold">Connected Accounts</h2>

            {isLoading ? (
              <Card>
                <CardContent className="flex items-center justify-center py-12">
                  <div className="text-center">
                    <div className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-2" />
                    <p className="text-sm text-muted-foreground">Loading accounts...</p>
                  </div>
                </CardContent>
              </Card>
            ) : spaces.length === 0 ? (
              <Card className="border-dashed">
                <CardContent className="flex flex-col items-center justify-center py-12">
                  <HardDrive className="w-12 h-12 text-muted-foreground mb-3" />
                  <h3 className="font-semibold mb-2">No drives connected</h3>
                  <p className="text-sm text-muted-foreground mb-4">
                    Link your first Google Drive to start uploading
                  </p>
                  <Button onClick={handleLinkDrive} disabled={isLinking}>
                    <Plus className="w-4 h-4 mr-2" />
                    Link Google Drive
                  </Button>
                </CardContent>
              </Card>
            ) : (
              <div className="grid gap-4 md:grid-cols-2">
                {spaces.map((space) => {
                  const account = accounts.find((a) => a.id === space.account_id);

                  return (
                    <Card key={space.account_id}>
                      <CardHeader>
                        <div className="flex items-start justify-between">
                          <div className="flex items-start gap-3">
                            <div className="p-2 rounded-lg bg-primary/10">
                              <HardDrive className="w-5 h-5 text-primary" />
                            </div>
                            <div>
                              <CardTitle className="text-base">
                                {space.display_name}
                              </CardTitle>
                              {account && (
                                <CardDescription className="text-xs">
                                  Connected {new Date(account.created_at).toLocaleDateString()}
                                </CardDescription>
                              )}
                            </div>
                          </div>
                        </div>
                      </CardHeader>
                      <CardContent className="space-y-3">
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
                            <span className="font-medium">
                              {formatBytes(space.used_space)} / {formatBytes(space.total_space)}
                            </span>
                          </div>
                          <Progress
                            value={space.total_space > 0 ? (space.used_space / space.total_space) * 100 : 0}
                            className="h-2"
                          />
                          <div className="flex justify-between items-center">
                            <span className="text-sm text-success font-medium">
                              {formatBytes(space.free_space)} free
                            </span>
                            <Badge variant={space.available ? 'default' : 'destructive'}>
                              {space.available ? 'Available' : 'Unavailable'}
                            </Badge>
                          </div>
                          {space.error && (
                            <div className="text-xs text-destructive/80">{space.error}</div>
                          )}
                      </CardContent>
                    </Card>
                  );
                })}
              </div>
            )}
          </div>

        </div>
      </Layout>
    </ProtectedRoute>
  );
};

export default Profile;
