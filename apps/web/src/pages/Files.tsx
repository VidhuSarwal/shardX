import { useEffect, useMemo, useRef, useState } from 'react';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Upload, FileText, Loader2, CheckCircle, AlertCircle } from 'lucide-react';
import { api } from '@/lib/api';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';

const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB recommended 5–10MB
const MAX_RETRIES = 5;

const backoff = (attempt: number) => Math.min(16000, 1000 * Math.pow(2, attempt));

interface UploadSession {
  sessionId: string;
  uploadUrl: string;
  statusUrl?: string;
  file: File;
  uploadedBytes: number;
  status: 'uploading' | 'awaiting_strategy' | 'finalizing' | 'processing' | 'complete' | 'failed';
  progress: number;
  processingProgress?: number;
  paused?: boolean;
  strategy?: 'greedy' | 'balanced' | 'proportional';
  planPreview?: {
    plan: Array<{ chunk_id: number; drive_account_id: string; size: number; start_offset: number; end_offset: number }>;
    num_chunks: number;
  } | null;
}

const sanitizeFilename = (name: string) => name.replace(/\s+/g, '_');

const Files = () => {
  const [uploads, setUploads] = useState<UploadSession[]>([]);
  const [isDragging, setIsDragging] = useState(false);

  const updateUpload = (sessionId: string, updates: Partial<UploadSession>) => {
    setUploads(prev =>
      prev.map(u => (u.sessionId === sessionId ? { ...u, ...updates } : u))
    );
  };

  const pausedRef = useRef<Record<string, boolean>>({});
  const canceledRef = useRef<Record<string, boolean>>({});

  const persistOffset = (sessionId: string, offset: number) => {
    try {
      localStorage.setItem(`upload_offset_${sessionId}`, String(offset));
    } catch {}
  };

  const getPersistedOffset = (sessionId: string) => {
    try {
      const v = localStorage.getItem(`upload_offset_${sessionId}`);
      return v ? Number(v) : 0;
    } catch {
      return 0;
    }
  };

  const clearPersistedOffset = (sessionId: string) => {
    try {
      localStorage.removeItem(`upload_offset_${sessionId}`);
    } catch {}
  };

  const uploadChunks = async (uploadUrl: string, sessionId: string, file: File) => {
    let offset = getPersistedOffset(sessionId) || 0;
    let lastConfirmed = offset;

    while (offset < file.size) {
      // If canceled, stop immediately
      if (canceledRef.current[sessionId]) {
        toast.message('Upload canceled');
        return false;
      }
      // Pause support
      if (pausedRef.current[sessionId]) {
        await new Promise<void>((resolve) => {
          const id = setInterval(() => {
            if (canceledRef.current[sessionId]) {
              clearInterval(id);
              resolve();
            } else if (!pausedRef.current[sessionId]) {
              clearInterval(id);
              resolve();
            }
          }, 300);
        });
        if (canceledRef.current[sessionId]) {
          toast.message('Upload canceled');
          return false;
        }
      }

      const chunk = file.slice(offset, Math.min(file.size, offset + CHUNK_SIZE));

      let attempt = 0;
      // retry loop for this chunk
      // eslint-disable-next-line no-constant-condition
      while (true) {
        try {
          const result = await api.uploadChunkTo(uploadUrl, chunk, offset);
          lastConfirmed = result.uploaded;
          offset += chunk.size; // advance only on success of this chunk

          updateUpload(sessionId, {
            uploadedBytes: result.uploaded,
            progress: result.progress,
          });
          persistOffset(sessionId, lastConfirmed);
          break; // move to next chunk
        } catch (e) {
          const err = e as Error;
          // Unauthorized: surface error and stop (no refresh endpoint available here)
          if (err.message.toLowerCase().includes('unauthorized') || err.message.includes('401')) {
            updateUpload(sessionId, { status: 'failed' });
            toast.error('Session expired. Please log in again.');
            return false;
          }

          // Bad Request (offset/form). Try correcting offset once.
          if (err.message.startsWith('HTTP 400')) {
            // Reset offset to last confirmed and retry once immediately
            offset = lastConfirmed;
            attempt++;
            if (attempt > 1) {
              updateUpload(sessionId, { status: 'failed' });
              toast.error('Upload failed due to chunk offset mismatch.');
              return false;
            }
            continue;
          }

          // Retry on transient/network/server errors
          if (attempt < MAX_RETRIES) {
            const wait = backoff(attempt);
            attempt++;
            await new Promise((r) => setTimeout(r, wait));
            continue;
          }

          updateUpload(sessionId, { status: 'failed' });
          toast.error('Upload failed: ' + (err?.message || 'Unknown error'));
          return false;
        }
      }
    }

    return true;
  };

  const cancelUpload = (sessionId: string) => {
    // Mark canceled for any ongoing chunk loop
    canceledRef.current[sessionId] = true;
    // Ensure we are not paused-waiting forever
    pausedRef.current[sessionId] = false;
    // Clear persisted offset
    clearPersistedOffset(sessionId);
    // Remove from UI list
    setUploads(prev => prev.filter(u => u.sessionId !== sessionId));
    toast.info('Upload canceled. You can re-upload this file anytime.');
  };

  const pollStatus = async (sessionId: string, statusUrl: string) => {
    const interval = setInterval(async () => {
      try {
        const status = await api.getUploadStatusByUrl(statusUrl);

        updateUpload(sessionId, {
          status: status.status,
          processingProgress: status.processing_progress,
        });

        if (status.status === 'complete') {
          clearInterval(interval);
          toast.success('Upload complete!');
        } else if (status.status === 'failed') {
          clearInterval(interval);
          toast.error('Processing failed: ' + status.error_message);
        }
      } catch (error) {
        clearInterval(interval);
        toast.error('Failed to check status');
      }
    }, 3000);
  };

  const handleFileSelect = async (file: File) => {
    try {
      const sanitizedFilename = sanitizeFilename(file.name);
      const initResult = await api.initiateUpload(sanitizedFilename, file.size);

      const newUpload: UploadSession = {
        sessionId: initResult.session_id,
        uploadUrl: initResult.upload_url,
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
      const success = await uploadChunks(initResult.upload_url, initResult.session_id, file);

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

  const getStatusIcon = (status: UploadSession['status']) => {
    switch (status) {
      case 'uploading':
      case 'finalizing':
      case 'processing':
        return <Loader2 className="w-4 h-4 animate-spin" />;
      case 'complete':
        return <CheckCircle className="w-4 h-4 text-success" />;
      case 'failed':
        return <AlertCircle className="w-4 h-4 text-destructive" />;
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
      const preview = await api.calculateChunking(u.file.size, u.strategy || 'balanced');
      updateUpload(u.sessionId, { planPreview: preview });
    } catch (e) {
      toast.error('Failed to calculate distribution plan');
    }
  };

  const onFinalize = async (u: UploadSession) => {
    try {
      updateUpload(u.sessionId, { status: 'finalizing' });

      const finalize = await api.finalizeUpload(u.sessionId, u.strategy || 'balanced');
      updateUpload(u.sessionId, { statusUrl: finalize.status_url });
      updateUpload(u.sessionId, { status: 'processing' });
      pollStatus(u.sessionId, finalize.status_url);
    } catch (e: any) {
      // If finalize fails due to timing (400), wait a moment and retry once
      const msg = e?.message || '';
      if (msg.startsWith('HTTP 400')) {
        setTimeout(async () => {
          try {
            const finalize = await api.finalizeUpload(u.sessionId, u.strategy || 'balanced');
            updateUpload(u.sessionId, { statusUrl: finalize.status_url, status: 'processing' });
            pollStatus(u.sessionId, finalize.status_url);
          } catch (ee) {
            updateUpload(u.sessionId, { status: 'failed' });
            toast.error('Finalize failed. ' + (ee instanceof Error ? ee.message : 'Please retry.'));
          }
        }, 1500);
      } else {
        updateUpload(u.sessionId, { status: 'failed' });
        toast.error('Finalize failed: ' + msg);
      }
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
    } catch (e: any) {
      // Surface backend-provided error message when available
      const msg = e?.message || 'Could not download key file. Please try again.';
      toast.error(msg);
    }
  };

  return (
    <ProtectedRoute>
      <Layout>
        <div className="space-y-6">
          <div>
            <h1 className="text-3xl font-bold mb-2">Your Uploads</h1>
            <p className="text-muted-foreground">
              Upload large files with automatic distribution across your Google Drive accounts
            </p>
          </div>

          {/* Upload Area */}
          <Card
            className={`border-2 border-dashed transition-all ${
              isDragging
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-primary/50'
            }`}
            onDragOver={(e) => {
              e.preventDefault();
              setIsDragging(true);
            }}
            onDragLeave={() => setIsDragging(false)}
            onDrop={handleDrop}
          >
            <CardContent className="flex flex-col items-center justify-center py-12">
              <div className="p-4 rounded-full bg-primary/10 mb-4">
                <Upload className="w-8 h-8 text-primary" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Upload a file</h3>
              <p className="text-sm text-muted-foreground mb-4">
                Drag and drop or click to select
              </p>
              <input
                type="file"
                id="fileInput"
                className="hidden"
                onChange={handleFileInput}
              />
              <Button asChild>
                <label htmlFor="fileInput" className="cursor-pointer">
                  Select File
                </label>
              </Button>
              <p className="text-xs text-muted-foreground mt-4">
                Maximum file size: 100 GB
              </p>
            </CardContent>
          </Card>

          {/* Active Uploads */}
          {uploads.length > 0 && (
            <div className="space-y-4">
              <h2 className="text-xl font-semibold">Active Uploads</h2>
              {uploads.map((upload) => (
                <Card key={upload.sessionId}>
                  <CardHeader>
                    <div className="flex items-start justify-between">
                      <div className="flex items-start gap-3">
                        <FileText className="w-5 h-5 text-muted-foreground mt-0.5" />
                        <div>
                          <CardTitle className="text-base">
                            {upload.file.name}
                          </CardTitle>
                          <CardDescription>
                            {(upload.file.size / (1024 * 1024)).toFixed(2)} MB
                          </CardDescription>
                        </div>
                      </div>
                      <Badge variant={upload.status === 'complete' ? 'default' : 'secondary'}>
                        <span className="flex items-center gap-1">
                          {getStatusIcon(upload.status)}
                          {getStatusText(upload)}
                        </span>
                      </Badge>
                    </div>
                  </CardHeader>
                  <CardContent>
                    <Progress
                      value={
                        upload.status === 'processing'
                          ? upload.processingProgress || 0
                          : upload.progress
                      }
                      className="h-2"
                    />
                    {/* Controls */}
                    {(upload.status === 'uploading') && (
                      <div className="mt-3 flex items-center gap-2">
                        <Button variant="outline" size="sm" onClick={() => togglePause(upload.sessionId)}>
                          {upload.paused ? 'Resume' : 'Pause'}
                        </Button>
                        <Button variant="destructive" size="sm" onClick={() => cancelUpload(upload.sessionId)}>
                          Cancel
                        </Button>
                      </div>
                    )}

                    {/* Strategy selection */}
                    {upload.status === 'awaiting_strategy' && (
                      <div className="mt-4 space-y-3">
                        <div className="flex items-center gap-2">
                          <label className="text-sm">Strategy:</label>
                          <select
                            className="border rounded px-2 py-1 text-sm bg-background"
                            value={upload.strategy}
                            onChange={(e) => {
                              const nextStrategy = e.target.value as UploadSession['strategy'];
                              updateUpload(upload.sessionId, { strategy: nextStrategy });
                            }}
                          >
                            <option value="balanced">balanced</option>
                            <option value="greedy">greedy</option>
                            <option value="proportional">proportional</option>
                          </select>
                          <Button size="sm" variant="secondary" onClick={() => onPreviewPlan(upload)}>Preview Distribution</Button>
                        </div>
                        {upload.planPreview && (
                          <div className="text-xs p-2 rounded bg-muted/50">
                            <div className="font-medium mb-1">Plan preview ({upload.planPreview.num_chunks} chunks):</div>
                            <ul className="list-disc pl-5 space-y-0.5">
                              {upload.planPreview.plan.map(p => (
                                <li key={p.chunk_id}>chunk {p.chunk_id}: drive {p.drive_account_id} size {p.size} bytes [{p.start_offset}–{p.end_offset}]</li>
                              ))}
                            </ul>
                          </div>
                        )}
                        <div className="flex items-center gap-2">
                          <Button id='finalize' size="sm" onClick={() => onFinalize(upload)}>Finalize & Start Processing</Button>
                          <Button size="sm" variant="destructive" onClick={() => cancelUpload(upload.sessionId)}>
                            Cancel Upload
                          </Button>
                        </div>
                      </div>
                    )}
                    {upload.status === 'complete' && (
                      <div className="mt-4 flex gap-2">
                        <Button id='downloadKey' size="sm" onClick={() => onDownloadKey(upload)}>
                          Download Key File
                        </Button>
                        <Button size="sm" variant="outline" asChild>
                          <Link to={`/files/${upload.sessionId}`}>View Health & Shard Map</Link>
                        </Button>
                      </div>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}

        </div>
      </Layout>
    </ProtectedRoute>
  );
};

export default Files;
