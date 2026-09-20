import { useEffect, useReducer, useRef, useState } from 'react';
import { AppShell } from '@/components/AppShell';
import { Tour } from '@/components/Tour';
import { tourReducer, FILES_TOUR, TOUR_DONE_KEY } from '@/lib/tour';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Button } from '@/components/ui/button';
import { GlowCard, MagneticButton, Reveal } from '@/components/ds';
import { Upload, Pause, Play, X } from 'lucide-react';
import { api, ApiError, type ChunkPlan, type ChunkingStrategy } from '@/lib/api';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';
import { formatBytes } from '@/lib/fileDetail';

const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB recommended 5–10MB
const MAX_RETRIES = 5;
const SEGMENTS = 24;

const backoff = (attempt: number) => Math.min(16000, 1000 * Math.pow(2, attempt));

interface UploadSession {
  sessionId: string;
  file: File;
  uploadedBytes: number;
  status: 'uploading' | 'awaiting_strategy' | 'finalizing' | 'processing' | 'complete' | 'failed';
  progress: number;
  processingProgress?: number;
  paused?: boolean;
  strategy: ChunkingStrategy;
  planPreview?: { plan: ChunkPlan[]; num_chunks: number } | null;
}

const STATUS_PILL: Record<UploadSession['status'], string> = {
  uploading: 'text-primary border-primary/40',
  finalizing: 'text-primary border-primary/40',
  processing: 'text-primary border-primary/40',
  complete: 'text-success border-success/40',
  failed: 'text-destructive border-destructive/40',
  awaiting_strategy: 'text-warning border-warning/40',
};

const sanitizeFilename = (name: string) => name.replace(/\s+/g, '_');

const Files = () => {
  const [uploads, setUploads] = useState<UploadSession[]>([]);
  const [isDragging, setIsDragging] = useState(false);

  const [tour, dispatchTour] = useReducer(tourReducer, { open: false, index: 0 });
  // Tracks whether the tour has actually opened, so a close at step 0 (e.g. an
  // immediate Escape) still counts as "done" — `tour.index` alone can't tell
  // "never opened" apart from "closed while still on the first step".
  const tourOpened = useRef(false);
  useEffect(() => {
    if (localStorage.getItem(TOUR_DONE_KEY) !== '1') dispatchTour({ type: 'start' });
  }, []);
  useEffect(() => {
    if (tour.open) tourOpened.current = true;
    else if (tourOpened.current) localStorage.setItem(TOUR_DONE_KEY, '1');
  }, [tour]);

  const updateUpload = (sessionId: string, updates: Partial<UploadSession>) => {
    setUploads(prev =>
      prev.map(u => (u.sessionId === sessionId ? { ...u, ...updates } : u))
    );
  };

  const pausedRef = useRef<Record<string, boolean>>({});
  const canceledRef = useRef<Record<string, boolean>>({});

  const pollTimers = useRef<Record<string, ReturnType<typeof setInterval>>>({});
  useEffect(() => () => Object.values(pollTimers.current).forEach(clearInterval), []);

  const uploadChunks = async (sessionId: string, file: File) => {
    let offset = 0;

    while (offset < file.size) {
      // Pause: wait until resumed or canceled.
      while (pausedRef.current[sessionId] && !canceledRef.current[sessionId]) {
        await new Promise((r) => setTimeout(r, 300));
      }
      if (canceledRef.current[sessionId]) return false;

      const chunk = file.slice(offset, Math.min(file.size, offset + CHUNK_SIZE));

      let attempt = 0;
      for (;;) {
        try {
          const result = await api.uploadChunk(sessionId, chunk, offset);
          offset += chunk.size;
          updateUpload(sessionId, { uploadedBytes: result.uploaded, progress: result.progress });
          break;
        } catch (e) {
          // 4xx (expired/invalid session, bad form) won't succeed on retry; only
          // retry network failures and 5xx.
          const status = e instanceof ApiError ? e.status : 0;
          if ((status === 0 || status >= 500) && attempt < MAX_RETRIES) {
            await new Promise((r) => setTimeout(r, backoff(attempt++)));
            continue;
          }
          updateUpload(sessionId, { status: 'failed' });
          toast.error('Upload failed: ' + (e instanceof Error ? e.message : 'Unknown error'));
          return false;
        }
      }
    }

    return true;
  };

  const cancelUpload = (sessionId: string) => {
    canceledRef.current[sessionId] = true;
    pausedRef.current[sessionId] = false;
    setUploads(prev => prev.filter(u => u.sessionId !== sessionId));
    toast.info('Upload canceled. You can re-upload this file anytime.');
  };

  const pollStatus = (sessionId: string) => {
    const stop = () => {
      clearInterval(pollTimers.current[sessionId]);
      delete pollTimers.current[sessionId];
    };
    pollTimers.current[sessionId] = setInterval(async () => {
      try {
        const status = await api.getUploadStatus(sessionId);
        updateUpload(sessionId, { status: status.status, processingProgress: status.processing_progress });

        if (status.status === 'complete') {
          stop();
          toast.success('Upload complete!');
        } else if (status.status === 'failed') {
          stop();
          toast.error('Processing failed: ' + status.error_message);
        }
      } catch (e) {
        stop();
        updateUpload(sessionId, { status: 'failed' });
        toast.error('Failed to check status: ' + (e instanceof Error ? e.message : 'Unknown error'));
      }
    }, 3000);
  };

  const handleFileSelect = async (file: File) => {
    try {
      const sanitizedFilename = sanitizeFilename(file.name);
      const initResult = await api.initiateUpload(sanitizedFilename, file.size);

      const newUpload: UploadSession = {
        sessionId: initResult.session_id,
        file,
        uploadedBytes: 0,
        status: 'uploading',
        progress: 0,
        paused: false,
        strategy: 'balanced',
        planPreview: null,
      };

      setUploads(prev => [...prev, newUpload]);

      // Upload chunks
      const success = await uploadChunks(initResult.session_id, file);

      if (success) {
        // All chunks uploaded. Let user preview/select strategy before finalize.
        updateUpload(initResult.session_id, { status: 'awaiting_strategy' });
      }
    } catch (error) {
      toast.error('Failed to start upload: ' + (error instanceof Error ? error.message : 'Unknown error'));
    }
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    const file = e.dataTransfer.files[0];
    if (file) {
      handleFileSelect(file);
    }
  };

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      handleFileSelect(file);
    }
  };

  const getStatusText = (upload: UploadSession) => {
    switch (upload.status) {
      case 'uploading':
        return `Uploading: ${upload.progress.toFixed(1)}%`;
      case 'awaiting_strategy':
        return 'Ready to finalize: choose distribution strategy';
      case 'finalizing':
        return 'Finalizing...';
      case 'processing':
        return `Processing: ${upload.processingProgress?.toFixed(1) || 0}%`;
      case 'complete':
        return 'Complete';
      case 'failed':
        return 'Failed';
    }
  };

  const togglePause = (sessionId: string) => {
    setUploads(prev => prev.map(u => {
      if (u.sessionId !== sessionId) return u;
      const nextPaused = !u.paused;
      pausedRef.current[sessionId] = nextPaused;
      return { ...u, paused: nextPaused };
    }));
  };

  const onPreviewPlan = async (u: UploadSession) => {
    try {
      const preview = await api.calculateChunking(u.file.size, u.strategy);
      updateUpload(u.sessionId, { planPreview: preview });
    } catch (e) {
      toast.error('Failed to calculate distribution plan: ' + (e instanceof Error ? e.message : 'Unknown error'));
    }
  };

  const onFinalize = async (u: UploadSession) => {
    updateUpload(u.sessionId, { status: 'finalizing' });
    try {
      await api.finalizeUpload(u.sessionId, u.strategy);
      updateUpload(u.sessionId, { status: 'processing' });
      pollStatus(u.sessionId);
    } catch (e) {
      // Backend rejects finalize with 400 while the session is still incomplete
      // (or expired); let the user retry rather than failing the upload.
      updateUpload(u.sessionId, { status: 'awaiting_strategy' });
      toast.error('Finalize failed: ' + (e instanceof Error ? e.message : 'Please retry.'));
    }
  };

  const onDownloadKey = async (u: UploadSession) => {
    try {
      const blob = await api.downloadKey(u.sessionId);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${u.file.name}.2xpfm.key`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      toast.success('Key file downloaded. Store it securely.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Could not download key file. Please try again.');
    }
  };

  return (
    <ProtectedRoute>
      <AppShell onHelp={() => dispatchTour({ type: 'start' })}>
        <div className="mb-8">
          <p className="eyebrow mb-2">Vault</p>
          <h1 className="text-3xl font-semibold tracking-tight">Your uploads</h1>
          <p className="mt-1 text-muted-foreground">Drop a file. It becomes noise, gets sharded and stored — and only your key file can bring it back.</p>
        </div>

        {/* Upload Area */}
        <div
          id="dropzone"
          onDragOver={(e) => {
            e.preventDefault();
            setIsDragging(true);
          }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={handleDrop}
          className={cn('relative overflow-hidden rounded-2xl border-2 border-dashed p-12 text-center transition-all duration-300', isDragging ? 'border-primary bg-primary/5 shadow-glow-primary scale-[1.01]' : 'border-border hover:border-primary/50')}
        >
          <div aria-hidden className="pointer-events-none absolute inset-0 bg-[linear-gradient(hsl(var(--border))_1px,transparent_1px),linear-gradient(90deg,hsl(var(--border))_1px,transparent_1px)] bg-[size:32px_32px] opacity-30 [mask-image:radial-gradient(ellipse_at_center,black,transparent_70%)]" />
          <div className="relative">
            <div className="mx-auto mb-4 grid h-14 w-14 place-items-center rounded-2xl bg-primary/10 text-primary"><Upload className="h-6 w-6" /></div>
            <h2 className="text-lg font-semibold">Drop a file to shard it</h2>
            <p className="mt-1 text-sm text-muted-foreground">or</p>
            <input type="file" id="fileInput" className="sr-only" onChange={handleFileInput} />
            <MagneticButton asChild className="mt-4"><label htmlFor="fileInput" className="cursor-pointer">Choose a file</label></MagneticButton>
          </div>
        </div>

        {/* Empty state */}
        {uploads.length === 0 && (
          <Reveal className="mt-10 text-center text-sm text-muted-foreground">
            Nothing uploaded yet. <button className="text-primary hover:underline" onClick={() => dispatchTour({ type: 'start' })}>Take the 30-second tour</button> or <Link to="/guide" className="text-primary hover:underline">read the guide</Link>.
          </Reveal>
        )}

        {/* Active Uploads */}
        {uploads.length > 0 && (
          <div className="mt-10 space-y-4">
            <h2 className="text-xl font-semibold">Active uploads</h2>
            {uploads.map((upload) => {
              const pct = upload.status === 'processing' ? upload.processingProgress || 0 : upload.progress;
              return (
                <GlowCard key={upload.sessionId}>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-medium truncate">{upload.file.name}</p>
                      <p className="font-mono text-xs text-muted-foreground">{(upload.file.size / (1024 * 1024)).toFixed(2)} MB</p>
                    </div>
                    <span className={cn('shrink-0 rounded-full border px-2 py-0.5 text-xs', STATUS_PILL[upload.status])}>{getStatusText(upload)}</span>
                  </div>

                  <div className="mt-4 flex gap-1" role="progressbar" aria-valuenow={Math.round(pct)} aria-valuemin={0} aria-valuemax={100}>
                    {Array.from({ length: SEGMENTS }, (_, i) => (
                      <span key={i} className={cn('h-1.5 flex-1 rounded-full transition-colors duration-300', i < (pct / 100) * SEGMENTS ? 'bg-primary shadow-glow-primary' : 'bg-muted')} />
                    ))}
                  </div>

                  {/* Controls */}
                  {(upload.status === 'uploading') && (
                    <div className="mt-4 flex items-center gap-2">
                      <Button variant="outline" size="sm" onClick={() => togglePause(upload.sessionId)}>
                        {upload.paused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
                        {upload.paused ? 'Resume' : 'Pause'}
                      </Button>
                      <Button variant="destructive" size="sm" onClick={() => cancelUpload(upload.sessionId)}>
                        <X className="h-4 w-4" /> Cancel
                      </Button>
                    </div>
                  )}

                  {/* Strategy selection */}
                  {upload.status === 'awaiting_strategy' && (
                    <div className="mt-4 space-y-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <div id="strategy" role="radiogroup" aria-label="Distribution strategy" className="inline-flex rounded-lg border border-border p-1">
                          {(['balanced', 'greedy', 'proportional'] as ChunkingStrategy[]).map((s) => (
                            <button key={s} type="button" role="radio" aria-checked={upload.strategy === s} onClick={() => updateUpload(upload.sessionId, { strategy: s })}
                              className={cn('rounded-md px-3 py-1.5 font-mono text-xs transition-colors', upload.strategy === s ? 'bg-primary/15 text-primary' : 'text-muted-foreground hover:text-foreground')}>{s}</button>
                          ))}
                        </div>
                        <Button size="sm" variant="secondary" onClick={() => onPreviewPlan(upload)}>Preview distribution</Button>
                      </div>
                      {upload.planPreview && (() => {
                        const total = upload.planPreview.plan.reduce((n, p) => n + p.size, 0);
                        const ids = [...new Set(upload.planPreview.plan.map((p) => p.drive_account_id))];
                        const hue = (id: string) => 190 + (ids.indexOf(id) * 47) % 120;
                        return (
                          <div className="mt-3 space-y-2 text-xs">
                            <div className="text-muted-foreground">{upload.planPreview.num_chunks} shards</div>
                            <div className="flex h-3 w-full overflow-hidden rounded-full">
                              {upload.planPreview.plan.map((p) => <span key={p.chunk_id} title={`chunk ${p.chunk_id} · ${formatBytes(p.size)}`} style={{ width: `${(p.size / total) * 100}%`, background: `hsl(${hue(p.drive_account_id)} 90% 55%)` }} />)}
                            </div>
                            <div className="flex flex-wrap gap-3 font-mono text-[11px] text-muted-foreground">{ids.map((id) => <span key={id} className="flex items-center gap-1"><span className="h-2 w-2 rounded-full" style={{ background: `hsl(${hue(id)} 90% 55%)` }} />{id.slice(-8)}</span>)}</div>
                          </div>
                        );
                      })()}
                      <div className="flex items-center gap-2">
                        <Button id="finalize" size="sm" onClick={() => onFinalize(upload)}>Finalize & process</Button>
                        <Button size="sm" variant="destructive" onClick={() => cancelUpload(upload.sessionId)}>
                          Cancel upload
                        </Button>
                      </div>
                    </div>
                  )}
                  {upload.status === 'complete' && (
                    <div className="mt-4 flex gap-2">
                      <Button id="downloadKey" size="sm" onClick={() => onDownloadKey(upload)}>
                        Download key file
                      </Button>
                      <Button size="sm" variant="outline" asChild>
                        <Link to={`/files/${upload.sessionId}`}>View health & shard map</Link>
                      </Button>
                    </div>
                  )}
                </GlowCard>
              );
            })}
          </div>
        )}

        <Tour steps={FILES_TOUR} state={tour} dispatch={dispatchTour} />
      </AppShell>
    </ProtectedRoute>
  );
};

export default Files;
