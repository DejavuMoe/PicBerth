// Upload queue: concurrency 3, one XHR per file (for real progress), client pre-checks,
// window-wide drop / paste, link format preference.

import { ApiError, errorDetail, errorMessage, handleUnauthorized, isRetryable, networkError, noteVersion, problemFrom } from './api';
import { formatBytes, formatLimit, formatNames } from './format';
import { dropBlocked, isTypingTarget } from './modal';
import { isLibraryPath, navigate, router } from './router.svelte';
import { session } from './session.svelte';
import type { Image, ImageFormat, LinkFormat } from './types';

export type UploadStatus = 'queued' | 'uploading' | 'processing' | 'done' | 'error';

/** The folder new uploads go into. `id` is the `folder` part sent with each file (absent at the root). */
export interface UploadTarget {
  id: string | null;
  label: string;
}

export interface UploadItem {
  id: number;
  file: File;
  name: string;
  previewUrl: string | null;
  status: UploadStatus;
  /** 0..1 of request body bytes sent. */
  progress: number;
  image: Image | null;
  error: string | null;
  /** Raw server text, for a tooltip only. */
  errorDetail: string;
  retryable: boolean;
  /** Do not start before this timestamp (automatic retry after a short 429). */
  notBefore: number;
  autoRetries: number;
  folderId: string | null;
  folderLabel: string;
}

export const FORMAT_MIME: Record<ImageFormat, string[]> = {
  jpeg: ['image/jpeg', 'image/pjpeg'],
  png: ['image/png'],
  gif: ['image/gif'],
  webp: ['image/webp'],
  avif: ['image/avif'],
};
const FORMAT_EXT: Record<ImageFormat, string[]> = {
  jpeg: ['jpg', 'jpeg', 'jpe', 'jfif'],
  png: ['png'],
  gif: ['gif'],
  webp: ['webp'],
  avif: ['avif'],
};
const ALL_FORMATS = Object.keys(FORMAT_MIME) as ImageFormat[];

export const LINK_FORMATS: { value: LinkFormat; label: string }[] = [
  { value: 'url', label: 'URL' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'html', label: 'HTML' },
  { value: 'bbcode', label: 'BBCode' },
];

const STORAGE_KEY = 'picberth.linkFormat';

function loadFormat(): LinkFormat {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v === 'url' || v === 'markdown' || v === 'html' || v === 'bbcode') return v;
  } catch {
    // storage unavailable
  }
  return 'url';
}

type UploadedListener = (image: Image) => void;
const uploadedListeners = new Set<UploadedListener>();

/** Notified once per finished upload (including duplicates); returns the unsubscribe function. */
export function onUploaded(fn: UploadedListener): () => void {
  uploadedListeners.add(fn);
  return () => uploadedListeners.delete(fn);
}

export const uploads = $state({
  items: [] as UploadItem[],
  format: loadFormat(),
  drag: { active: false, count: 0 },
  announcement: '',
});

/** Where the library screen is; drops and pastes on other screens go to the root. */
export const uploadTarget = $state<{ current: UploadTarget }>({
  current: { id: null, label: '图片' },
});

export function setUploadTarget(target: UploadTarget | null) {
  uploadTarget.current = target ?? { id: null, label: '图片' };
}

export function setLinkFormat(f: LinkFormat) {
  uploads.format = f;
  try {
    localStorage.setItem(STORAGE_KEY, f);
  } catch {
    // ignore
  }
}

export function allowedFormats(): ImageFormat[] {
  return session.limits?.allowedFormats?.length ? session.limits.allowedFormats : ALL_FORMATS;
}

export function maxUploadBytes(): number {
  return session.limits?.maxUploadBytes ?? 20 * 1024 * 1024;
}

/** Value for <input type="file" accept>. */
export function acceptAttr(): string {
  return allowedFormats()
    .flatMap((f) => [...FORMAT_MIME[f], ...FORMAT_EXT[f].map((e) => `.${e}`)])
    .join(',');
}

function extOf(name: string): string {
  const i = name.lastIndexOf('.');
  return i >= 0 ? name.slice(i + 1).toLowerCase() : '';
}

function detectFormat(file: File): ImageFormat | null {
  const type = file.type.toLowerCase();
  for (const f of ALL_FORMATS) if (FORMAT_MIME[f].includes(type)) return f;
  // Unknown or generic types (some systems report none for .avif): trust the extension.
  if (type === '' || type === 'application/octet-stream') {
    const ext = extOf(file.name);
    for (const f of ALL_FORMATS) if (FORMAT_EXT[f].includes(ext)) return f;
  }
  return null;
}

function precheck(file: File): string | null {
  if (file.size === 0) return '文件为空';
  const fmt = detectFormat(file);
  const allowed = allowedFormats();
  if (!fmt) {
    return `不支持的格式，仅允许 ${formatNames(allowed).join('、')}`;
  }
  if (!allowed.includes(fmt)) return `服务器不允许 ${formatNames([fmt])[0]} 格式`;
  const max = maxUploadBytes();
  if (file.size > max) return `文件 ${formatBytes(file.size)}，超过 ${formatLimit(max)} 上限`;
  return null;
}

let seq = 1;
const MAX_CONCURRENT = 3;
let running = 0;
const inflight = new Map<number, XMLHttpRequest>();

function announce(message: string) {
  uploads.announcement = message;
}

export function addFiles(files: File[], target: UploadTarget | null = uploadTarget.current) {
  if (files.length === 0) return;
  const folder = target ?? { id: null, label: '图片' };
  const batch: UploadItem[] = files.map((file) => {
    const problem = precheck(file);
    return {
      id: seq++,
      file,
      name: file.name || '未命名',
      previewUrl: detectFormat(file) ? URL.createObjectURL(file) : null,
      status: problem ? 'error' : 'queued',
      progress: 0,
      image: null,
      error: problem,
      errorDetail: '',
      retryable: false,
      notBefore: 0,
      autoRetries: 0,
      folderId: folder.id,
      folderLabel: folder.label,
    };
  });
  const rejected = batch.filter((i) => i.status === 'error').length;
  if (rejected > 0) announce(`${rejected} 个文件未通过检查，未上传`);
  uploads.items.unshift(...batch);
  pump();
}

function pickQueued(): UploadItem | undefined {
  const now = Date.now();
  let best: UploadItem | undefined;
  for (const it of uploads.items) {
    if (it.status !== 'queued' || it.notBefore > now) continue;
    if (!best || it.id < best.id) best = it;
  }
  return best;
}

function pump() {
  while (running < MAX_CONCURRENT) {
    const next = pickQueued();
    if (!next) return;
    start(next);
  }
}

function fail(item: UploadItem, err: ApiError) {
  item.status = 'error';
  item.error = errorMessage(err, { maxUploadBytes: maxUploadBytes(), allowedFormats: allowedFormats() });
  item.errorDetail = errorDetail(err);
  item.retryable = isRetryable(err);
  announce(`${item.name} 上传失败：${item.error}`);
}

function start(item: UploadItem) {
  running++;
  item.status = 'uploading';
  item.progress = 0;
  item.error = null;

  const body = new FormData();
  // The text part comes first so a streaming server knows the destination before the file.
  if (item.folderId) body.append('folder', item.folderId);
  body.append('file', item.file, item.name);

  const xhr = new XMLHttpRequest();
  inflight.set(item.id, xhr);
  xhr.open('POST', '/api/v1/images');
  xhr.setRequestHeader('X-PicBerth-Client', 'web');
  xhr.setRequestHeader('Accept', 'application/json');
  xhr.responseType = 'text';

  xhr.upload.addEventListener('progress', (e) => {
    if (e.lengthComputable && e.total > 0) item.progress = Math.min(1, e.loaded / e.total);
  });
  xhr.upload.addEventListener('load', () => {
    if (item.status === 'uploading') {
      item.progress = 1;
      item.status = 'processing';
    }
  });

  const settle = () => {
    inflight.delete(item.id);
    running--;
    pump();
  };

  xhr.addEventListener('load', () => {
    noteVersion(xhr.getResponseHeader('X-PicBerth-Version'));
    if (xhr.status === 200 || xhr.status === 201) {
      try {
        const image = JSON.parse(xhr.responseText) as Image;
        item.image = image;
        item.status = 'done';
        announce(`已上传 ${item.name}`);
        // Outside this try: a listener's failure is not an upload failure.
        queueMicrotask(() => {
          for (const l of uploadedListeners) l(image);
        });
      } catch {
        fail(item, new ApiError({ status: xhr.status, code: 'bad-response' }));
      }
    } else {
      const err = problemFrom(
        xhr.status,
        xhr.responseText,
        xhr.getResponseHeader('Content-Type'),
        xhr.getResponseHeader('Retry-After'),
      );
      if (xhr.status === 401) handleUnauthorized();
      // The server answers 429 with a 2 s Retry-After when its upload slots are busy: wait quietly.
      if (err.code === 'rate-limited' && err.retryAfter !== null && err.retryAfter <= 10 && item.autoRetries < 4) {
        item.autoRetries++;
        item.status = 'queued';
        item.progress = 0;
        item.notBefore = Date.now() + err.retryAfter * 1000;
        setTimeout(pump, err.retryAfter * 1000 + 50);
      } else {
        fail(item, err);
      }
    }
    settle();
  });
  xhr.addEventListener('error', () => {
    fail(item, networkError());
    settle();
  });
  xhr.addEventListener('abort', settle);
  xhr.send(body);
}

export function retry(id: number) {
  const item = uploads.items.find((i) => i.id === id);
  if (!item || item.status !== 'error' || !item.retryable) return;
  item.status = 'queued';
  item.error = null;
  item.errorDetail = '';
  item.progress = 0;
  item.notBefore = 0;
  item.autoRetries = 0;
  pump();
}

function release(item: UploadItem) {
  if (item.previewUrl) URL.revokeObjectURL(item.previewUrl);
  inflight.get(item.id)?.abort();
}

export function removeItem(id: number) {
  const i = uploads.items.findIndex((it) => it.id === id);
  const item = uploads.items[i];
  if (!item) return;
  release(item);
  uploads.items.splice(i, 1);
  pump();
}

/** Remove settled rows (finished uploads and failures); keeps anything still waiting or in flight. */
export function clearFinished() {
  for (let i = uploads.items.length - 1; i >= 0; i--) {
    const item = uploads.items[i];
    if (item && (item.status === 'done' || item.status === 'error')) {
      release(item);
      uploads.items.splice(i, 1);
    }
  }
}

export function busy(): boolean {
  return uploads.items.some((i) => i.status === 'queued' || i.status === 'uploading' || i.status === 'processing');
}

// ---------------------------------------------------------------------------
// Window-wide input: drag & drop and paste.

function pastedName(file: File): File {
  const generic = /^(image|blob|screenshot)\.(png|jpe?g|gif|webp|avif)$/i.test(file.name) || file.name === '';
  if (!generic) return file;
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, '0');
  const stamp = `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`;
  const ext = extOf(file.name) || (file.type.split('/')[1] ?? 'png').replace('jpeg', 'jpg');
  return new File([file], `pasted-${stamp}.${ext}`, { type: file.type, lastModified: file.lastModified });
}

/** Files can only be watched on the library screen, so a drop or paste elsewhere goes there. */
function goToLibrary() {
  if (!isLibraryPath(router.path)) void navigate('/library');
}

/** Install on authenticated routes. */
export function installGlobalInput(): () => void {
  let heartbeat: ReturnType<typeof setTimeout> | undefined;

  const hasFiles = (e: DragEvent) => !!e.dataTransfer && Array.from(e.dataTransfer.types).includes('Files');
  const stopDrag = () => {
    clearTimeout(heartbeat);
    uploads.drag.active = false;
    uploads.drag.count = 0;
  };

  const onDragEnter = (e: DragEvent) => {
    if (!hasFiles(e) || dropBlocked()) return;
    e.preventDefault();
    uploads.drag.active = true;
    const items = e.dataTransfer?.items;
    uploads.drag.count = items ? Array.from(items).filter((i) => i.kind === 'file').length : 0;
  };
  const onDragOver = (e: DragEvent) => {
    if (!hasFiles(e)) return;
    e.preventDefault(); // required to allow a drop
    if (e.dataTransfer) e.dataTransfer.dropEffect = dropBlocked() ? 'none' : 'copy';
    if (dropBlocked()) return;
    uploads.drag.active = true;
    // Some browsers never fire a final dragleave when the pointer exits the window.
    clearTimeout(heartbeat);
    heartbeat = setTimeout(stopDrag, 700);
  };
  const onDragLeave = (e: DragEvent) => {
    if (hasFiles(e) && e.relatedTarget === null) stopDrag();
  };
  const onDrop = (e: DragEvent) => {
    if (!hasFiles(e)) return;
    e.preventDefault(); // never let the browser navigate to a dropped file
    stopDrag();
    if (dropBlocked()) return;
    const files = Array.from(e.dataTransfer?.files ?? []);
    if (files.length === 0) return;
    addFiles(files);
    goToLibrary();
  };
  const onPaste = (e: ClipboardEvent) => {
    if (isTypingTarget(e.target) || dropBlocked()) return;
    const files = Array.from(e.clipboardData?.files ?? []).filter((f) => f.type.startsWith('image/'));
    if (files.length === 0) return;
    e.preventDefault();
    addFiles(files.map(pastedName));
    goToLibrary();
  };
  const onBeforeUnload = (e: BeforeUnloadEvent) => {
    if (!busy()) return;
    e.preventDefault();
    e.returnValue = '';
  };

  window.addEventListener('dragenter', onDragEnter);
  window.addEventListener('dragover', onDragOver);
  window.addEventListener('dragleave', onDragLeave);
  window.addEventListener('drop', onDrop);
  document.addEventListener('paste', onPaste);
  window.addEventListener('beforeunload', onBeforeUnload);
  return () => {
    stopDrag();
    window.removeEventListener('dragenter', onDragEnter);
    window.removeEventListener('dragover', onDragOver);
    window.removeEventListener('dragleave', onDragLeave);
    window.removeEventListener('drop', onDrop);
    document.removeEventListener('paste', onPaste);
    window.removeEventListener('beforeunload', onBeforeUnload);
  };
}
