import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { toast } from 'sonner';
import { api, DownloadStatusResponse } from '@/lib/api';
import {
  Activity,
  AlertCircle,
  CalendarCheck,
  Download as DownloadIcon,
  FileKey,
  HardDrive,
  Loader2,
  ShieldCheck,
  TimerReset,
} from 'lucide-react';

const POLL_INTERVAL_MS = 3000;
const MAX_POLL_DURATION_MS = 10 * 60 * 1000; // 10 minutes
const CHECKSUM_MAX_SIZE_BYTES = 600 * 1024 * 1024; // 600MB sanity limit for in-browser hashing

type DownloadState =
  | 'idle'
  | 'initiating'
  | 'downloading'
  | 'decrypting'
  | 'downloading_file'
  | 'complete'
  | 'failed';

type KeyFileMetadata = {
  originalFilename?: string | null;
  originalSize?: number | null;
  fileId?: string | null;
  version?: string | null;
};

const backendStatusText: Record<DownloadStatusResponse['status'], string> = {
  downloading: 'Downloading encrypted chunks from linked drives…',
  decrypting: 'Decrypting and rebuilding your file…',
  complete: 'File reconstructed — preparing download…',
  failed: 'Download failed',
};

const statusColors: Record<DownloadState, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  idle: { label: 'Idle', variant: 'outline' },
  initiating: { label: 'Initiating', variant: 'secondary' },
  downloading: { label: 'Downloading', variant: 'secondary' },
  decrypting: { label: 'Decrypting', variant: 'secondary' },
  downloading_file: { label: 'Transferring', variant: 'secondary' },
  complete: { label: 'Complete', variant: 'default' },
  failed: { label: 'Failed', variant: 'destructive' },
};

const formatBytes = (bytes: number | null | undefined) => {
  if (!bytes && bytes !== 0) return '—';
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, exponent);
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[exponent]}`;
};

const formatDateTime = (iso?: string | null) => {
  if (!iso) return '—';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString();
};

const deriveFilenameFromKey = (file?: File | null) => {
  if (!file) return 'reconstructed.bin';
  if (file.name.endsWith('.2xpfm.key')) {
    return file.name.replace(/\.2xpfm\.key$/, '');
  }
  if (file.name.endsWith('.key')) {
    return file.name.replace(/\.key$/, '');
  }
  return `${file.name.replace(/\.[^/.]+$/, '') || 'reconstructed'}.bin`;
};

const parseKeyFileMetadata = async (file: File): Promise<KeyFileMetadata | null> => {
  try {
    const text = await file.text();
    const data = JSON.parse(text);
    const originalSizeRaw = data.original_size ?? data.originalSize;
    let originalSize: number | null = null;
    if (typeof originalSizeRaw === 'number') {
      originalSize = originalSizeRaw;
    } else if (typeof originalSizeRaw === 'string') {
      const parsed = Number(originalSizeRaw);
      originalSize = Number.isFinite(parsed) ? parsed : null;
    }
    return {
      originalFilename: data.original_filename ?? data.originalFilename ?? null,
      originalSize,
      fileId: data.file_id ?? data.fileId ?? null,
      version: data.version ?? null,
    };
  } catch {
    return null;
  }
};

const Download = () => {
  const [keyFile, setKeyFile] = useState<File | null>(null);
  const [keyMetadata, setKeyMetadata] = useState<KeyFileMetadata | null>(null);
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [downloadState, setDownloadState] = useState<DownloadState>('idle');
  const [statusMessage, setStatusMessage] = useState('Select your key file to begin.');
  const [progress, setProgress] = useState(0);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [downloadedFileName, setDownloadedFileName] = useState<string>(() => deriveFilenameFromKey(null));
  const [completedAt, setCompletedAt] = useState<string | null>(null);
  const [backendStatus, setBackendStatus] = useState<string | null>(null);

  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const pollStartedAtRef = useRef<number>(0);

  const isBusy = useMemo(
    () => ['initiating', 'downloading', 'decrypting', 'downloading_file'].includes(downloadState),
    [downloadState]
  );

  const cleanupPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      cleanupPolling();
    };
  }, [cleanupPolling]);

  const resetState = useCallback(() => {
    cleanupPolling();
    setSessionId(null);
    setDownloadState('idle');
    setStatusMessage('Select your key file to begin.');
    setProgress(0);
    setErrorMessage(null);
    setKeyFile(null);
    setKeyMetadata(null);
    setDownloadedFileName(deriveFilenameFromKey(null));
    setCompletedAt(null);
    setBackendStatus(null);
  }, [cleanupPolling]);

  const verifyChecksumIfPossible = async (blob: Blob, expected: string | null) => {
    if (!expected || !window?.crypto?.subtle) {
      return true;
    }
    if (blob.size > CHECKSUM_MAX_SIZE_BYTES) {
      toast.message('Checksum skipped (file too large for browser verification).');
      return true;
    }
    const buffer = await blob.arrayBuffer();
    const digest = await window.crypto.subtle.digest('SHA-256', buffer);
    const hashArray = Array.from(new Uint8Array(digest));
    const hex = hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
    return hex === expected.toLowerCase();
  };

  const fetchFile = useCallback(
    async (activeSessionId: string, expectedChecksum: string | null) => {
      setDownloadState('downloading_file');
      setStatusMessage('Downloading file…');
      try {
        const blob = await api.downloadFile(activeSessionId);
        const passedChecksum = await verifyChecksumIfPossible(blob, expectedChecksum);
        const fileName = downloadedFileName || deriveFilenameFromKey(keyFile);
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = fileName;
        document.body.appendChild(link);
        link.click();
        link.remove();
    URL.revokeObjectURL(url);
    setDownloadState('complete');
        setStatusMessage(passedChecksum ? 'Download complete.' : 'Download complete (checksum not confirmed).');
        if (passedChecksum) {
          toast.success('File downloaded and checksum verified.');
        } else {
          toast.success('File downloaded.');
        }
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Could not download file.';
        setDownloadState('failed');
        setErrorMessage(message);
        setStatusMessage(message);
        toast.error(message);
        throw error;
      }
    },
    [downloadedFileName, keyFile]
  );

  const pollStatus = useCallback(
    (activeSessionId: string) => {
      pollStartedAtRef.current = Date.now();
      cleanupPolling();
      pollTimerRef.current = setInterval(async () => {
        const elapsed = Date.now() - pollStartedAtRef.current;
        if (elapsed > MAX_POLL_DURATION_MS) {
          cleanupPolling();
          setDownloadState('failed');
          setErrorMessage('Download timed out. Please try again.');
          setStatusMessage('Timed out while waiting for the server to reconstruct the file.');
          toast.error('Download timed out.');
          return;
        }

        try {
          const status = await api.getDownloadStatus(activeSessionId);
          setProgress(Math.min(100, Number(status.progress ?? 0)));
          setBackendStatus(status.status);
          setCompletedAt(status.completed_at ?? null);

          if (status.status === 'complete') {
            cleanupPolling();
            setStatusMessage(backendStatusText.complete);
            try {
              await fetchFile(activeSessionId, null);
            } catch (err) {
              // fetchFile handles messaging
            }
          } else if (status.status === 'failed') {
            cleanupPolling();
            setDownloadState('failed');
            const backendError = status.error_message || backendStatusText.failed;
            setErrorMessage(backendError);
            setStatusMessage(backendError);
            toast.error(backendError);
          } else {
            const derivedState = status.status === 'decrypting' ? 'decrypting' : 'downloading';
            setDownloadState(derivedState);
            setStatusMessage(`${backendStatusText[status.status]} (${Math.round(status.progress ?? 0)}%)`);
          }
        } catch (error) {
          cleanupPolling();
          const message = error instanceof Error ? error.message : 'Failed to fetch status.';
          setDownloadState('failed');
          setErrorMessage(message);
          setStatusMessage(message);
          toast.error(message);
        }
      }, POLL_INTERVAL_MS);
    },
    [cleanupPolling, fetchFile]
  );

  const handleStartDownload = async () => {
    if (!keyFile) {
      toast.error('Please select a key file first.');
      return;
    }
    setDownloadedFileName(deriveFilenameFromKey(keyFile));
    setErrorMessage(null);
    setDownloadState('initiating');
    setStatusMessage('Uploading key file…');
    setProgress(0);
    try {
      const response = await api.initiateDownload(keyFile);
      setSessionId(response.session_id);
      toast.success('Download session created.');
      setBackendStatus('downloading');
      setCompletedAt(null);
      setDownloadState('downloading');
      setStatusMessage(backendStatusText.downloading);
      pollStatus(response.session_id);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to initiate download.';
      setDownloadState('failed');
      setErrorMessage(message);
      setStatusMessage(message);
      toast.error(message);
    }
  };

  const handleCancel = () => {
    cleanupPolling();
    setStatusMessage('Download canceled.');
    setDownloadState('failed');
    setErrorMessage('Download canceled by user.');
    setBackendStatus(null);
    setCompletedAt(null);
    setSessionId(null);
    toast.message('Download canceled.');
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    setKeyFile(file);
    if (file) {
      setDownloadedFileName(deriveFilenameFromKey(file));
      toast.message('Key file selected.');
      parseKeyFileMetadata(file)
        .then((metadata) => setKeyMetadata(metadata))
        .catch(() => setKeyMetadata(null));
    } else {
      setKeyMetadata(null);
      setDownloadedFileName(deriveFilenameFromKey(null));
    }
  };

  const activeStatus = statusColors[downloadState];

  return (
    <ProtectedRoute>
      <Layout>
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold mb-1">Download Files</h1>
            <p className="text-muted-foreground">
              Provide your secure key file to reconstruct and retrieve the original upload.
            </p>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card className="self-start">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileKey className="w-5 h-5 text-primary" /> Key file
                </CardTitle>
                <CardDescription>
                  Upload the `.2xpfm.key` file you downloaded after uploading your content.
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="keyFile">Key file</Label>
                  <div className="flex items-center gap-3">
                    <input
                      ref={fileInputRef}
                      id="keyFile"
                      type="file"
                      accept=".key,.json,.txt,.bin,.2xpfm.key"
                      className="hidden"
                      onChange={handleFileSelect}
                    />
                    <Button variant="outline" onClick={() => fileInputRef.current?.click()}>
                      Select file
                    </Button>
                    {keyFile ? (
                      <div className="text-sm text-muted-foreground">
                        <p className="font-medium">{keyFile.name}</p>
                        <p>{formatBytes(keyFile.size)}</p>
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No file selected</p>
                    )}
                  </div>
                </div>

                <Alert>
                  <ShieldCheck className="h-4 w-4" />
                  <AlertTitle>Security reminder</AlertTitle>
                  <AlertDescription>
                    Key files never leave your device except for this encrypted handshake. We do not store them on the
                    server.
                  </AlertDescription>
                </Alert>

                <div className="flex flex-wrap gap-3">
                  <Button onClick={handleStartDownload} disabled={!keyFile || isBusy}>
                    {downloadState === 'initiating' ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <DownloadIcon className="mr-2 h-4 w-4" />
                    )}
                    Start download
                  </Button>
                  <Button variant="secondary" onClick={handleCancel} disabled={!isBusy}>
                    <TimerReset className="mr-2 h-4 w-4" /> Cancel
                  </Button>
                  <Button variant="ghost" onClick={resetState} disabled={isBusy && downloadState !== 'failed'}>
                    Reset
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <DownloadIcon className="w-5 h-5 text-primary" /> Status
                </CardTitle>
                <CardDescription>Track progress, download readiness, and verification info.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex items-center gap-2">
                  <Badge variant={activeStatus.variant}>{activeStatus.label}</Badge>
                  <span className="text-sm text-muted-foreground">{statusMessage}</span>
                </div>
                <div>
                  <Progress value={progress} className="h-2" />
                  <div className="flex items-center justify-between text-xs text-muted-foreground mt-1">
                    <span>{progress.toFixed(1)}%</span>
                    <span>{backendStatus ?? '—'}</span>
                  </div>
                </div>

                <div className="grid gap-3 sm:grid-cols-2">
                  <InfoStat label="Session ID" value={sessionId ?? '—'} icon={<ShieldCheck className="w-4 h-4" />} />
                  <InfoStat label="Server status" value={backendStatus ?? '—'} icon={<Activity className="w-4 h-4" />} />
                  <InfoStat label="Completed at" value={formatDateTime(completedAt)} icon={<CalendarCheck className="w-4 h-4" />} />
                  <InfoStat label="Key file ID" value={keyMetadata?.fileId ?? '—'} icon={<FileKey className="w-4 h-4" />} />
                  <InfoStat
                    label="Expected file"
                    value={keyMetadata?.originalFilename ?? downloadedFileName ?? '—'}
                    icon={<DownloadIcon className="w-4 h-4" />}
                  />
                  <InfoStat
                    label="Original size"
                    value={keyMetadata?.originalSize ? formatBytes(keyMetadata.originalSize) : '—'}
                    icon={<HardDrive className="w-4 h-4" />}
                  />
                </div>

                {errorMessage && (
                  <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertTitle>Download problem</AlertTitle>
                    <AlertDescription>{errorMessage}</AlertDescription>
                  </Alert>
                )}

                <Separator />
                <div className="space-y-1 text-sm text-muted-foreground">
                  <p className="font-medium text-foreground">How it works</p>
                  <ol className="list-decimal list-inside space-y-1">
                    <li>Upload your key file to request a secure reconstruction session.</li>
                    <li>We poll the backend every 3 seconds for progress updates.</li>
                    <li>Once complete, the file downloads automatically (with optional checksum verification).</li>
                  </ol>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>
      </Layout>
    </ProtectedRoute>
  );
};

const InfoStat = ({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon: ReactNode;
}) => (
  <div className="rounded-lg border border-border/70 p-3">
    <div className="text-xs text-muted-foreground flex items-center gap-1">
      {icon}
      {label}
    </div>
    <div className="text-sm font-medium truncate" title={value}>
      {value || '—'}
    </div>
  </div>
);

export default Download;
