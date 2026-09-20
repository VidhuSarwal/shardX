// Thin client for apps/api (cmd/server/main.go). Every endpoint here has a
// registered route on the backend; keep it that way.
export const API_BASE_URL: string =
  import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:5555';

export const getAuthToken = (): string | null => localStorage.getItem('auth_token');
export const getAuthEmail = (): string | null => localStorage.getItem('auth_email');

export const setAuthToken = (token: string, email?: string): void => {
  localStorage.setItem('auth_token', token);
  if (email) localStorage.setItem('auth_email', email);
  window.dispatchEvent(new Event('auth:changed'));
};

export const clearAuthToken = (): void => {
  localStorage.removeItem('auth_token');
  localStorage.removeItem('auth_email');
  window.dispatchEvent(new Event('auth:changed'));
};

/** Non-2xx response. `status` lets callers branch without parsing messages. */
export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

type RequestOpts = RequestInit & {
  /** When true, a 401 is thrown to the caller instead of clearing the token and redirecting. */
  skipAuthRedirect?: boolean;
};

// Backend errors are http.Error() plain-text bodies; JSON only on success.
const request = async (endpoint: string, { skipAuthRedirect, ...init }: RequestOpts = {}): Promise<Response> => {
  const headers = new Headers(init.headers);
  const token = getAuthToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  if (init.body && !(init.body instanceof FormData)) headers.set('Content-Type', 'application/json');

  const response = await fetch(`${API_BASE_URL}${endpoint}`, { ...init, headers });

  if (response.status === 401 && !skipAuthRedirect) {
    clearAuthToken();
    window.location.href = '/login';
  }
  if (!response.ok) {
    const text = (await response.text()).trim();
    throw new ApiError(response.status, text || `HTTP ${response.status}`);
  }
  return response;
};

export const apiRequest = async <T>(endpoint: string, options?: RequestOpts): Promise<T> =>
  (await request(endpoint, options)).json();

export const apiRequestBlob = async (endpoint: string, options?: RequestOpts): Promise<Blob> =>
  (await request(endpoint, options)).blob();

// ---- Response shapes (mirror the Go handlers' JSON) ----

export type ChunkingStrategy = 'greedy' | 'balanced' | 'proportional' | 'manual';

/** models.DriveSpaceInfo */
export type DriveSpace = {
  account_id: string;
  display_name: string;
  owner_name?: string;
  owner_email?: string;
  total_space: number;
  used_space: number;
  free_space: number;
  available: boolean;
  error?: string;
};

/** handlers.ListDriveAccounts */
export type DriveAccount = {
  id: string;
  provider: string;
  display_name: string;
  created_at: string;
};

/** models.ChunkPlan */
export type ChunkPlan = {
  chunk_id: number;
  drive_account_id: string;
  size: number;
  start_offset: number;
  end_offset: number;
};

/** filehandlers.GetUploadStatusHandler */
export type UploadStatus = {
  status: 'uploading' | 'processing' | 'complete' | 'failed';
  uploaded_size: number;
  total_size: number;
  processing_progress: number;
  error_message: string;
  completed_at: string | null;
};

/** integrity.FileHealth */
export type FileHealth = {
  file_id: string;
  health_percentage: number;
  shards_available: number;
  shards_total: number;
  integrity_checks_passed: number;
  integrity_checks_total: number;
  corruption_events: number;
};

export type ShardStatus = 'verified' | 'pending' | 'corrupted' | 'missing';

/** models.ShardRecord */
export type ShardRecord = {
  file_id: string;
  shard_id: number;
  sha256: string;
  size: number;
  /** Drive account ID in Drive mode; S3 bucket name in S3 mode. */
  bucket?: string;
  region?: string;
  created_at: string;
  status: ShardStatus;
};

/** filehandlers.TimelineEvent */
export type TimelineEvent = {
  type: string;
  at: string | null;
  detail?: Record<string, unknown>;
};

export const api = {
  // Auth
  signup: (email: string, password: string) =>
    apiRequest<{ message: string }>('/api/signup', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),

  login: (email: string, password: string) =>
    apiRequest<{ token: string }>('/api/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),

  // Drive
  getDriveLinkUrl: () => apiRequest<{ auth_url: string }>('/api/drive/link'),
  getDriveAccounts: () => apiRequest<DriveAccount[]>('/api/drive/accounts'),
  getDriveSpace: () => apiRequest<DriveSpace[]>('/api/drive/space'),

  // Upload
  initiateUpload: (filename: string, file_size: number) =>
    apiRequest<{
      session_id: string;
      upload_url: string;
      drive_spaces: DriveSpace[];
      max_file_size: number;
    }>('/api/files/upload/initiate', {
      method: 'POST',
      body: JSON.stringify({ filename, file_size }),
    }),

  uploadChunk: (sessionId: string, chunk: Blob, offset: number) => {
    const formData = new FormData();
    formData.append('chunk', chunk);
    formData.append('offset', String(offset));
    return apiRequest<{ uploaded: number; total: number; progress: number }>(
      `/api/files/upload/chunk?session_id=${encodeURIComponent(sessionId)}`,
      { method: 'POST', body: formData },
    );
  },

  finalizeUpload: (session_id: string, strategy: ChunkingStrategy, manual_chunk_sizes?: number[]) =>
    apiRequest<{ message: string; session_id: string; status_url: string }>('/api/files/upload/finalize', {
      method: 'POST',
      body: JSON.stringify({ session_id, strategy, manual_chunk_sizes }),
    }),

  getUploadStatus: (sessionId: string) =>
    apiRequest<UploadStatus>(`/api/files/upload/status/${sessionId}`),

  calculateChunking: (file_size: number, strategy: ChunkingStrategy, manual_chunk_sizes?: number[]) =>
    apiRequest<{ plan: ChunkPlan[]; num_chunks: number }>('/api/files/chunking/calculate', {
      method: 'POST',
      body: JSON.stringify({ file_size, strategy, manual_chunk_sizes }),
    }),

  // Key file (only available once status is "complete")
  downloadKey: (sessionId: string) =>
    apiRequestBlob(`/api/files/download-key/${sessionId}`, { skipAuthRedirect: true }),

  // Integrity Engine
  getFileHealth: (sessionId: string) => apiRequest<FileHealth>(`/api/files/${sessionId}/health`),

  getFileShards: (sessionId: string) =>
    apiRequest<{ file_id: string; shards: ShardRecord[] }>(`/api/files/${sessionId}/shards`),

  getFileTimeline: (sessionId: string) =>
    apiRequest<{ file_id: string; events: TimelineEvent[] }>(`/api/files/${sessionId}/timeline`),
};
