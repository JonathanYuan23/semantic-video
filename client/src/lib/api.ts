export const API_BASE = import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

type RequestInitWithoutBody = Omit<RequestInit, "body">;

async function apiFetch<T>(path: string, options: RequestInit = {}): Promise<T> {
  const method = (options.method || "GET").toUpperCase();
  const url = `${API_BASE}${path}`;
  const headers = new Headers(options.headers || {});
  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }

  let res: Response;
  try {
    res = await fetch(url, { ...options, headers });
  } catch (err) {
    console.error(`[api] ${method} ${url} network error`, err);
    throw err;
  }

  if (!res.ok) {
    const text = await res.text();
    console.error(`[api] ${method} ${url} failed`, {
      status: res.status,
      statusText: res.statusText,
      body: text,
    });
    throw new Error(text || res.statusText);
  }

  if (res.status === 204) {
    console.debug(`[api] ${method} ${url} ok`, { status: res.status });
    return undefined as T;
  }

  const json = (await res.json()) as T;
  console.debug(`[api] ${method} ${url} ok`, { status: res.status });
  return json;
}

export interface CloudStatus {
  userId: string;
  connected: boolean;
  lastSuccessfulUpload?: string;
  pendingBatches: number;
}

export interface StatusResponse {
  status: string;
}

export interface VideoRecord {
  id: string;
  path: string;
  durationSeconds?: number;
  indexStatus: string;
  framesExtracted: number;
  framesUploaded: number;
  totalFramesExpected: number;
  lastIndexedAt?: string;
  lastError?: string;
}

export interface AddVideoResponse {
  videoId: string;
  status: string;
}

export interface AddFolderResponse {
  folderId: string;
  status: string;
}

export interface JobRecord {
  id: string;
  videoId: string;
  type: string;
  status: string;
  progress: number;
  createdAt: string;
  updatedAt: string;
}

export interface SearchTimestamp {
  start: number;
  end: number;
  relevanceScore: number;
}

export interface SearchMatch {
  videoId: string;
  videoPath: string;
  timestamp: number;
  relevanceScore: number;
}

const mapVideo = (v: any): VideoRecord => ({
  id: v.video_id,
  path: v.path,
  durationSeconds: v.duration_seconds,
  indexStatus: v.index_status,
  framesExtracted: v.frames_extracted,
  framesUploaded: v.frames_uploaded,
  totalFramesExpected: v.total_frames_expected,
  lastIndexedAt: v.last_indexed_at,
  lastError: v.last_error,
});

const mapJob = (j: any): JobRecord => ({
  id: j.job_id,
  videoId: j.video_id,
  type: j.type,
  status: j.status,
  progress: j.progress,
  createdAt: j.created_at,
  updatedAt: j.updated_at,
});

export async function getCloudStatus(): Promise<CloudStatus> {
  const data = await apiFetch<any>("/cloud/status");
  return {
    userId: data.user_id,
    connected: data.connected,
    lastSuccessfulUpload: data.last_successful_upload,
    pendingBatches: data.pending_batches,
  };
}

export async function authenticateCloud(accessToken: string): Promise<StatusResponse> {
  return apiFetch<StatusResponse>("/cloud/auth", {
    method: "POST",
    body: JSON.stringify({ access_token: accessToken }),
  });
}

export async function listVideos(): Promise<VideoRecord[]> {
  const data = await apiFetch<any[]>("/videos");
  return data.map(mapVideo);
}

export async function addVideo(path: string): Promise<AddVideoResponse> {
  const data = await apiFetch<any>("/videos", {
    method: "POST",
    body: JSON.stringify({ path }),
  });
  return {
    videoId: data.video_id,
    status: data.status,
  };
}

export async function addFolder(path: string, recursive = false): Promise<AddFolderResponse> {
  const data = await apiFetch<any>("/folders", {
    method: "POST",
    body: JSON.stringify({ path, recursive }),
  });
  return {
    folderId: data.folder_id,
    status: data.status,
  };
}

export async function getVideo(videoId: string): Promise<VideoRecord> {
  const data = await apiFetch<any>(`/videos/${videoId}`);
  return mapVideo(data);
}

export async function startExtract(videoId: string, reindex = false) {
  return apiFetch<StatusResponse & { job_id: string }>(`/videos/${videoId}/extract`, {
    method: "POST",
    body: JSON.stringify({ reindex }),
  });
}

export async function cancelJob(videoId: string) {
  return apiFetch<StatusResponse>(`/videos/${videoId}/cancel`, { method: "POST" });
}

export async function listJobs(): Promise<JobRecord[]> {
  const data = await apiFetch<any[]>("/jobs");
  return data.map(mapJob);
}

export async function searchVideos(query: string, topK = 5, clusterThreshold = 5): Promise<SearchMatch[]> {
  const data = await apiFetch<{ results: any[] }>("/search_video", {
    method: "POST",
    body: JSON.stringify({ query, top_k: topK, cluster_threshold: clusterThreshold }),
  });

  const matches: SearchMatch[] = [];
  (data.results || []).forEach((item) => {
    const videoId = item.video_id;
    const videoPath = item.video_path;
    (item.timestamps || []).forEach((ts: any) => {
      matches.push({
        videoId,
        videoPath,
        timestamp: typeof ts.start === "number" ? ts.start : 0,
        relevanceScore: typeof ts.relevance_score === "number" ? ts.relevance_score : item.max_relevance_score || 0,
      });
    });
  });

  matches.sort((a, b) => b.relevanceScore - a.relevanceScore);
  return matches;
}

export async function getConfig() {
  return apiFetch("/config");
}

export async function updateConfig(body: Record<string, unknown>) {
  return apiFetch("/config", {
    method: "PUT",
    body: JSON.stringify(body),
  });
}
