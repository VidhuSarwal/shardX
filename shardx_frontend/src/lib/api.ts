const API_BASE_URL = 'http://localhost:5555';


export const getAuthToken = (): string | null => {
  return localStorage.getItem('auth_token');
};

export const setAuthToken = (token: string): void => {
  localStorage.setItem('auth_token', token);
  try { window.dispatchEvent(new Event('auth:changed')); } catch {}
};

export const clearAuthToken = (): void => {
  localStorage.removeItem('auth_token');
  try { window.dispatchEvent(new Event('auth:changed')); } catch {}
};

// Extended RequestInit with client-side controls
type RequestInitEx = RequestInit & {
  // When true, do not auto-clear token or redirect on 401. Caller will handle.
  skipAuthRedirect?: boolean;
};

export const apiRequest = async <T>(
  endpoint: string,
  options: RequestInitEx = {}
): Promise<T> => {
  const token = getAuthToken();
  const headers: HeadersInit = {
    ...options.headers,
  };

  if (token && !endpoint.includes('/oauth2/callback')) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  if (options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }

  const url = /^https?:\/\//i.test(endpoint) ? endpoint : `${API_BASE_URL}${endpoint}`;
  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    if (!(options as RequestInitEx).skipAuthRedirect) {
      clearAuthToken();
      window.location.href = '/login';
    }
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    // Prefer server-provided error message; fall back to plain text; else status code
    let errMsg = `HTTP ${response.status}`;
    try {
      const clone = response.clone();
      // Try JSON first
      const errorData = await clone.json();
      if (errorData) {
        // Common fields: message | error | detail
        errMsg = (errorData.message || errorData.error || errorData.detail || errMsg) as string;
      }
    } catch {
      try {
        const txt = await response.clone().text();
        if (txt && txt.trim().length > 0) {
          errMsg = txt.trim();
        }
      } catch {
        // ignore
      }
    }
    throw new Error(errMsg);
  }

  return response.json();
};

// Raw fetch helper that returns Blob/Text without JSON parsing and optional 401 skip
export const apiRequestBlob = async (
  endpoint: string,
  options: RequestInitEx = {}
): Promise<Blob> => {
  const token = getAuthToken();
  const headers: HeadersInit = {
    ...options.headers,
  };

  if (token && !endpoint.includes('/oauth2/callback')) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const url = /^https?:\/\//i.test(endpoint) ? endpoint : `${API_BASE_URL}${endpoint}`;
  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    if (!options.skipAuthRedirect) {
      clearAuthToken();
      window.location.href = '/login';
    }
    throw new Error('Unauthorized');
  }

  if (!response.ok) {
    // Try parse JSON error, then text, else status
    let errMsg = `HTTP ${response.status}`;
    try {
      const data = await response.clone().json();
      errMsg = (data && (data.message || data.error || data.detail)) || errMsg;
    } catch {
      try {
        const txt = await response.clone().text();
        if (txt && txt.trim().length > 0) {
          errMsg = txt.trim();
        }
      } catch {
        // ignore
      }
    }
    throw new Error(errMsg);
  }

  return response.blob();
};

export type DownloadStatusResponse = {
  status: 'downloading' | 'decrypting' | 'complete' | 'failed';
  progress: number;
  error_message?: string | null;
  completed_at?: string | null;
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
  getDriveLinkUrl: () =>
    apiRequest<{ auth_url: string }>('/api/drive/link'),

  getDriveAccounts: () =>
    apiRequest<Array<{
      id: string;
      provider: string;
      display_name: string;
      created_at: string;
    }>>('/api/drive/accounts'),

  getDriveSpace: () =>
    apiRequest<Array<{
      account_id: string;
      display_name: string;
      owner_name?: string;
      owner_email?: string;
      total_space: number;
      used_space: number;
      free_space: number;
      available: boolean;
      error?: string;
    }>>('/api/drive/space'),

  // Upload
  initiateUpload: (filename: string, file_size: number) =>
    apiRequest<{
      session_id: string;
      upload_url: string;
      drive_spaces: Array<{
        account_id: string;
        display_name: string;
        total_space: number;
        used_space: number;
        free_space: number;
        available: boolean;
      }>;
      max_file_size: number;
    }>('/api/files/upload/initiate', {
      method: 'POST',
      body: JSON.stringify({ filename, file_size }),
    }),

  uploadChunk: async (sessionId: string, chunk: Blob, offset: number) => {
    const formData = new FormData();
    formData.append('chunk', chunk);
    formData.append('offset', offset.toString());

    return apiRequest<{
      uploaded: number;
      total: number;
      progress: number;
    }>(`/api/files/upload/chunk?session_id=${sessionId}`, {
      method: 'POST',
      body: formData,
    });
  },

  uploadChunkTo: async (uploadUrl: string, chunk: Blob, offset: number) => {
    const formData = new FormData();
    formData.append('chunk', chunk);
    formData.append('offset', offset.toString());

    return apiRequest<{
      uploaded: number;
      total: number;
      progress: number;
    }>(uploadUrl, {
      method: 'POST',
      body: formData,
    });
  },

  finalizeUpload: (
    session_id: string,
    strategy: 'greedy' | 'balanced' | 'proportional' | 'manual',
    manual_chunk_sizes?: number[]
  ) =>
    apiRequest<{
      message: string;
      session_id: string;
      status_url: string;
    }>('/api/files/upload/finalize', {
      method: 'POST',
      body: JSON.stringify({ session_id, strategy, manual_chunk_sizes }),
    }),

  getUploadStatus: (sessionId: string) =>
    apiRequest<{
      status: 'uploading' | 'processing' | 'complete' | 'failed';
      uploaded_size: number;
      total_size: number;
      processing_progress: number;
      error_message: string;
      completed_at: string | null;
    }>(`/api/files/upload/status/${sessionId}`),

  getUploadStatusByUrl: (statusUrl: string) =>
    apiRequest<{
      status: 'uploading' | 'processing' | 'complete' | 'failed';
      uploaded_size: number;
      total_size: number;
      processing_progress: number;
      error_message: string;
      completed_at: string | null;
    }>(statusUrl),

  calculateChunking: (
    file_size: number,
    strategy: 'greedy' | 'balanced' | 'proportional' | 'manual',
    manual_chunk_sizes?: number[]
  ) =>
    apiRequest<{
      plan: Array<{
        chunk_id: number;
        drive_account_id: string;
        size: number;
        start_offset: number;
        end_offset: number;
      }>;
      num_chunks: number;
    }>('/api/files/chunking/calculate', {
      method: 'POST',
      body: JSON.stringify({ file_size, strategy, manual_chunk_sizes }),
    }),

  // Key file download (backend: GET /api/files/download-key/:session_id). Returns Blob for client-side save.
  downloadKey: (sessionId: string) =>
    apiRequestBlob(`/api/files/download-key/${sessionId}`, {
      method: 'GET',
      // Let caller handle any 401 (e.g., refresh once then retry)
      skipAuthRedirect: true,
    }),

  initiateDownload: (keyFile: File) => {
    const formData = new FormData();
    formData.append('key_file', keyFile);
    return apiRequest<{ session_id: string; expires_in_sec?: number }>(
      '/api/files/download/initiate',
      {
        method: 'POST',
        body: formData,
      }
    );
  },

  getDownloadStatus: (sessionId: string) =>
    apiRequest<DownloadStatusResponse>(`/api/files/download/status/${sessionId}`),

  downloadFile: (sessionId: string) =>
    apiRequestBlob(`/api/files/download/file/${sessionId}`, {
      method: 'GET',
      skipAuthRedirect: true,
    }),
};
