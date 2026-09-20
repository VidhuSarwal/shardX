import { useEffect, useRef, useState } from 'react';
import { Layout } from '@/components/Layout';
import { ProtectedRoute } from '@/components/ProtectedRoute';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Progress } from '@/components/ui/progress';
import { Badge } from '@/components/ui/badge';
import { Upload, FileText, Loader2, CheckCircle, AlertCircle } from 'lucide-react';
import { api, ApiError, type ChunkPlan, type ChunkingStrategy } from '@/lib/api';
import { Link } from 'react-router-dom';
import { toast } from 'sonner';

const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB recommended 5–10MB
const MAX_RETRIES = 5;

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
                            onChange={(e) => updateUpload(upload.sessionId, { strategy: e.target.value as ChunkingStrategy })}
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
