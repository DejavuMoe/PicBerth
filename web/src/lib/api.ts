import { toast } from './toast.svelte';
import type {
  AuditPage,
  BatchAction,
  BatchResult,
  CheckReport,
  Facets,
  Folder,
  FolderCreate,
  FolderPatch,
  Image,
  ImagePage,
  ImagePatch,
  ImageQuery,
  ImageState,
  SessionInfo,
  Settings,
  Snapshot,
  Status,
  Token,
  TokenCreate,
  WritableSettings,
} from './types';

const BASE = '/api/v1';

export class ApiError extends Error {
  status: number;
  code: string;
  title: string;
  detail: string;
  retryAfter: number | null;

  constructor(init: {
    status: number;
    code: string;
    title?: string;
    detail?: string;
    retryAfter?: number | null;
  }) {
    super(init.detail || init.title || init.code);
    this.name = 'ApiError';
    this.status = init.status;
    this.code = init.code;
    this.title = init.title ?? '';
    this.detail = init.detail ?? '';
    this.retryAfter = init.retryAfter ?? null;
  }
}

export function isAbort(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError';
}

/** Build an ApiError from a non-2xx response (shared by fetch and XHR callers). */
export function problemFrom(
  status: number,
  text: string,
  contentType: string | null,
  retryAfter: string | null,
): ApiError {
  const retry = retryAfter && /^\d+$/.test(retryAfter) ? Number(retryAfter) : null;
  if (contentType && contentType.includes('json')) {
    try {
      const p = JSON.parse(text) as Record<string, unknown>;
      return new ApiError({
        status,
        code: typeof p.code === 'string' ? p.code : 'unknown',
        title: typeof p.title === 'string' ? p.title : '',
        detail: typeof p.detail === 'string' ? p.detail : '',
        retryAfter: retry,
      });
    } catch {
      // fall through to the generic mapping
    }
  }
  // Not problem+json: most likely a reverse proxy or gateway answered.
  if (status === 413) {
    return new ApiError({ status, code: 'proxy-too-large', detail: '文件超过服务器或反向代理允许的大小' });
  }
  if (status === 502 || status === 503 || status === 504) {
    return new ApiError({ status, code: 'unavailable', detail: '服务暂时不可用，请稍后重试' });
  }
  return new ApiError({ status, code: 'unknown', detail: `请求失败（HTTP ${status}）`, retryAfter: retry });
}

export function networkError(): ApiError {
  return new ApiError({ status: 0, code: 'network', detail: '无法连接到服务器' });
}

const SUMMARY: Record<string, string> = {
  validation: '请求内容无效',
  unauthorized: '登录已失效',
  forbidden: '没有权限',
  'not-found': '内容不存在，或当前不可见',
  conflict: '当前状态不允许此操作',
  'folder-exists': '已有同名文件夹',
  'folder-not-empty': '文件夹不为空',
  'folder-depth': '最多 3 层',
  'invalid-image': '图片已损坏或结构异常',
  'too-many-pixels': '图片像素过多（超过上限）',
  'insufficient-storage': '服务器磁盘空间不足',
  'proxy-too-large': '文件超过服务器或反向代理允许的大小',
  'bad-response': '服务器返回了无法解析的响应',
  internal: '服务器内部错误',
};

/** Limits the message may quote (from the session); optional so plain callers stay simple. */
export interface MessageLimits {
  maxUploadBytes?: number;
  allowedFormats?: readonly string[];
}

const FORMAT_LABEL: Record<string, string> = { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' };

/** Concise Chinese message for a known problem code. The server's English detail is never shown inline. */
export function errorMessage(e: unknown, limits?: MessageLimits): string {
  if (!(e instanceof ApiError)) return e instanceof Error && e.message ? e.message : '发生未知错误';
  if (e.code === 'network' || e.code === 'unavailable') return e.detail;
  if (e.code === 'rate-limited') {
    return e.retryAfter ? `请求过于频繁，请 ${e.retryAfter} 秒后重试` : '上传过于频繁，请稍后重试';
  }
  if (e.code === 'unsupported-format') {
    const names = limits?.allowedFormats?.map((f) => FORMAT_LABEL[f] ?? f.toUpperCase());
    return names?.length ? `不支持的格式，仅允许 ${names.join('、')}` : '不支持的图片格式';
  }
  if (e.code === 'too-large') {
    const mb = limits?.maxUploadBytes ? Math.round((limits.maxUploadBytes / 1048576) * 10) / 10 : 0;
    return mb ? `超过 ${mb} MB 上限` : '超过上限';
  }
  return SUMMARY[e.code] ?? (e.status > 0 ? `请求失败（HTTP ${e.status}）` : '发生未知错误');
}

/** Raw server text for a `title` attribute (debugging only). */
export function errorDetail(e: unknown): string {
  return e instanceof ApiError ? e.detail || e.title : '';
}

/** Retrying can only help for transient failures. */
export function isRetryable(e: unknown): boolean {
  if (!(e instanceof ApiError)) return false;
  return e.status === 0 || e.status >= 500 || e.status === 429 || e.code === 'proxy-too-large';
}

// ---------------------------------------------------------------------------
// Cross-cutting hooks

let knownVersion: string | null = null;
let versionToastShown = false;

/** One toast with 刷新 when the server version header changes after the first response. */
export function noteVersion(v: string | null) {
  if (!v) return;
  if (knownVersion === null) {
    knownVersion = v;
    return;
  }
  if (v !== knownVersion && !versionToastShown) {
    versionToastShown = true;
    toast('服务已更新，刷新页面以继续', {
      duration: 0,
      action: { label: '刷新', run: () => location.reload() },
    });
  }
}

let onUnauthorized: () => void = () => {};
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}
export function handleUnauthorized() {
  onUnauthorized();
}

// ---------------------------------------------------------------------------
// Transport

type Primitive = string | number | boolean | null | undefined;
type QueryValue = Primitive | readonly string[];

function qs(query?: Record<string, QueryValue>): string {
  if (!query) return '';
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (Array.isArray(v)) {
      for (const x of v) p.append(k, String(x));
      continue;
    }
    if (v === undefined || v === null || v === '' || v === false) continue;
    p.set(k, String(v));
  }
  const s = p.toString();
  return s ? `?${s}` : '';
}

interface RequestOptions {
  query?: Record<string, QueryValue>;
  json?: unknown;
  signal?: AbortSignal;
  /** Do not treat a 401 as an expired session (login attempts). */
  keepOn401?: boolean;
}

async function request<T>(method: string, path: string, o: RequestOptions = {}): Promise<T> {
  const headers = new Headers({ 'X-PicBerth-Client': 'web', Accept: 'application/json' });
  let body: string | undefined;
  if (o.json !== undefined) {
    headers.set('Content-Type', 'application/json');
    body = JSON.stringify(o.json);
  }
  let res: Response;
  try {
    res = await fetch(BASE + path + qs(o.query), {
      method,
      headers,
      body,
      signal: o.signal,
      credentials: 'same-origin',
    });
  } catch (e) {
    if (isAbort(e)) throw e;
    throw networkError();
  }
  noteVersion(res.headers.get('X-PicBerth-Version'));
  if (res.status === 204) return undefined as T;
  if (!res.ok) {
    const text = await res.text().catch(() => '');
    const err = problemFrom(res.status, text, res.headers.get('Content-Type'), res.headers.get('Retry-After'));
    if (res.status === 401 && !o.keepOn401) onUnauthorized();
    throw err;
  }
  try {
    return (await res.json()) as T;
  } catch (e) {
    if (isAbort(e)) throw e;
    throw new ApiError({ status: res.status, code: 'unknown', detail: '服务器返回了无法解析的响应' });
  }
}

// ---------------------------------------------------------------------------
// Endpoints (docs/api.md only)

export const rawUrl = (id: string) => `${BASE}/images/${encodeURIComponent(id)}/raw`;
export const thumbUrl = (id: string) => `${BASE}/images/${encodeURIComponent(id)}/thumb`;

export const api = {
  session: () => request<SessionInfo>('GET', '/session'),
  login: (username: string, password: string) =>
    request<void>('POST', '/session', { json: { username, password }, keepOn401: true }),
  logout: () => request<void>('DELETE', '/session', { keepOn401: true }),
  setup: (setupCode: string, username: string, password: string) =>
    request<void>('POST', '/setup', { json: { setupCode, username, password }, keepOn401: true }),

  listImages: (q: ImageQuery, signal?: AbortSignal) =>
    request<ImagePage>('GET', '/images', { query: { ...q }, signal }),
  getImage: (id: string) => request<Image>('GET', `/images/${encodeURIComponent(id)}`),
  patchImage: (id: string, patch: ImagePatch) =>
    request<Image>('PATCH', `/images/${encodeURIComponent(id)}`, { json: patch }),
  facets: (state: ImageState) => request<Facets>('GET', '/images/facets', { query: { state } }),
  batch: async (action: BatchAction, ids: string[], folderId?: string | null): Promise<BatchResult[]> => {
    const out: BatchResult[] = [];
    // The server accepts at most 500 ids per call.
    for (let i = 0; i < ids.length; i += 500) {
      const chunk = ids.slice(i, i + 500);
      const res = await request<{ results: BatchResult[] }>('POST', '/images/batch', {
        json: action === 'move' ? { action, ids: chunk, folderId: folderId ?? null } : { action, ids: chunk },
      });
      out.push(...res.results);
    }
    return out;
  },

  listFolders: () => request<{ items: Folder[] }>('GET', '/folders'),
  createFolder: (folder: FolderCreate) => request<Folder>('POST', '/folders', { json: folder }),
  patchFolder: (id: string, patch: FolderPatch) =>
    request<Folder>('PATCH', `/folders/${encodeURIComponent(id)}`, { json: patch }),
  deleteFolder: (id: string) => request<void>('DELETE', `/folders/${encodeURIComponent(id)}`),

  listTokens: () => request<{ items: Token[] }>('GET', '/tokens'),
  createToken: (t: TokenCreate) => request<{ token: Token; secret: string }>('POST', '/tokens', { json: t }),
  revokeToken: (id: number) => request<void>('DELETE', `/tokens/${id}`),

  getSettings: () => request<Settings>('GET', '/settings'),
  patchSettings: (patch: Partial<WritableSettings>) => request<Settings>('PATCH', '/settings', { json: patch }),
  status: () => request<Status>('GET', '/status'),
  runCheck: (mode: 'quick' | 'full') => request<CheckReport>('POST', '/checks', { json: { mode } }),
  createSnapshot: () => request<Snapshot>('POST', '/snapshots'),
  audit: (limit: number, cursor?: string | null, kind?: string) =>
    request<AuditPage>('GET', '/audit', { query: { limit, cursor: cursor ?? undefined, kind: kind || undefined } }),
};
