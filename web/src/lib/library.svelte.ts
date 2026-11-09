// State and actions of the image list (library and trash). One instance per page; filters live in the URL,
// the current folder comes from the path.

import { SvelteSet } from 'svelte/reactivity';
import { api, ApiError, errorMessage, isAbort, rawUrl } from './api';
import { copyText } from './clipboard';
import { facetsFor, refreshFacets } from './facets.svelte';
import { startOfDay } from './format';
import {
  MAX_DEPTH,
  ROOT_LABEL,
  childrenOf,
  folderById,
  folderChain,
  folderDepth,
  folderLabel,
  folderNameError,
  folderNames,
  fitsUnder,
  isInside,
  resolvePath,
  subtreeCount,
} from './folders';
import { refreshHealth } from './health.svelte';
import { libraryPath, librarySegments, navigate, router, updateQuery } from './router.svelte';
import { toast } from './toast.svelte';
import type {
  BatchAction,
  Folder,
  Image,
  ImageFormat,
  ImageQuery,
  ImageSort,
  ImageState,
  LinkFormat,
} from './types';
import { LINK_FORMATS, setLinkFormat, uploads } from './upload.svelte';

export const FORMATS: ImageFormat[] = ['jpeg', 'png', 'gif', 'webp', 'avif'];

export const DATE_RANGES = [
  { key: 'today', label: '今天' },
  { key: '7d', label: '最近 7 天' },
  { key: '30d', label: '最近 30 天' },
  { key: 'older', label: '30 天以前' },
] as const;
export type DateKey = (typeof DATE_RANGES)[number]['key'];

export const SORTS: { key: ImageSort; label: string }[] = [
  { key: 'new', label: '最新上传' },
  { key: 'old', label: '最早上传' },
  { key: 'large', label: '文件最大' },
  { key: 'small', label: '文件最小' },
  { key: 'name', label: '文件名' },
];

export interface Filters {
  q: string;
  formats: ImageFormat[];
  date: '' | DateKey;
  sources: string[];
}

export const EMPTY_FILTERS: Filters = { q: '', formats: [], date: '', sources: [] };

function parseFilters(p: URLSearchParams): Filters {
  const date = p.get('date');
  return {
    q: p.get('q') ?? '',
    formats: FORMATS.filter((f) => p.getAll('format').includes(f)),
    date: DATE_RANGES.find((d) => d.key === date)?.key ?? '',
    sources: [...new Set(p.getAll('source').filter(Boolean))],
  };
}

function parseSort(p: URLSearchParams): ImageSort {
  const s = p.get('sort');
  return SORTS.find((x) => x.key === s)?.key ?? 'new';
}

/** Number of active filters, not counting the search text. */
export function filterCount(f: Filters): number {
  return f.formats.length + (f.date ? 1 : 0) + f.sources.length;
}

function dateRange(key: DateKey, now = Date.now()): { since?: string; until?: string } {
  const DAY = 86_400_000;
  if (key === 'today') return { since: new Date(startOfDay(now)).toISOString() };
  if (key === '7d') return { since: new Date(now - 7 * DAY).toISOString() };
  if (key === '30d') return { since: new Date(now - 30 * DAY).toISOString() };
  return { until: new Date(now - 30 * DAY).toISOString() };
}

/** Map a folder request failure onto the inline message the tree shows. */
function folderError(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.code === 'folder-exists') return '已有同名文件夹';
    if (e.code === 'folder-depth') return `最多 ${MAX_DEPTH} 层`;
    if (e.code === 'folder-not-empty') return '文件夹不为空';
  }
  return errorMessage(e);
}

/** The drag payload images use when dropped on a folder node or a breadcrumb. */
export const IMAGE_DRAG_TYPE = 'application/x-picberth-images';

const VIEW_KEY = 'picberth.view';
export type ViewMode = 'table' | 'grid';
function loadView(): ViewMode {
  try {
    return localStorage.getItem(VIEW_KEY) === 'grid' ? 'grid' : 'table';
  } catch {
    return 'table';
  }
}

interface Removed {
  image: Image;
  index: number;
}

export function downloadImage(img: Image) {
  const a = document.createElement('a');
  a.href = rawUrl(img.id);
  a.download = img.filename;
  document.body.append(a);
  a.click();
  a.remove();
  toast(`已开始下载 ${img.filename}`);
}

const COPY_FAILED = '当前连接不安全，无法写入剪贴板';

function formatLabel(f: LinkFormat): string {
  return LINK_FORMATS.find((x) => x.value === f)?.label ?? '';
}

/** Copy one value (an ID, a hash) and say what was copied. */
export async function copyValue(text: string, label: string) {
  const r = await copyText(text);
  toast(r === 'copied' ? `已复制 ${label}` : COPY_FAILED);
}

/** Copy one link in the given format and remember that format for the next copy. */
export async function copyLink(img: Pick<Image, 'links'>, format: LinkFormat = uploads.format) {
  setLinkFormat(format);
  await copyValue(img.links[format], formatLabel(format));
}

/** Copy every selected link, one per line, in the last-used format. */
export async function copyLinks(images: Pick<Image, 'links'>[]) {
  if (images.length === 0) return;
  const format = uploads.format;
  const r = await copyText(images.map((i) => i.links[format]).join('\n'));
  toast(r === 'copied' ? `已复制 ${images.length} 条 ${formatLabel(format)}` : COPY_FAILED);
}

export class Library {
  readonly mode: 'library' | 'trash' = 'library';
  readonly state: ImageState = 'active';

  items = $state<Image[]>([]);
  cursor = $state<string | null>(null);
  phase = $state<'loading' | 'ready' | 'error'>('loading');
  more = $state<'idle' | 'loading' | 'error'>('idle');
  errorText = $state('');
  errorKind = $state<'images' | 'folders'>('images');
  retention = $state<number | null>(null);

  folders = $state<Folder[]>([]);
  foldersPhase = $state<'loading' | 'ready' | 'error'>('loading');
  foldersError = $state('');

  view = $state<ViewMode>(loadView());
  sel = new SvelteSet<string>();
  selectMode = $state(false);
  activeId = $state<string | null>(null);
  /** An image to open once the list for its folder has loaded (e.g. 查看 on an upload elsewhere). */
  pendingOpen = $state<string | null>(null);
  purgeIds = $state<string[] | null>(null);

  /** Decoded folder names from the path (empty at the root). */
  pathNames = $derived(librarySegments(router.path));
  /** The folder id for the path: null at the root, undefined when no such folder exists. */
  resolved = $derived(this.mode === 'library' && this.foldersPhase === 'ready' ? resolvePath(this.folders, this.pathNames) : null);
  missing = $derived(this.mode === 'library' && this.resolved === undefined);
  folderId = $derived(this.resolved ?? null);

  filters = $derived(parseFilters(router.query));
  sort = $derived(parseSort(router.query));
  filtered = $derived(!!this.filters.q || filterCount(this.filters) > 0);
  selecting = $derived(this.sel.size > 0 || this.selectMode);
  active = $derived(this.activeId ? (this.items.find((i) => i.id === this.activeId) ?? null) : null);
  activeIndex = $derived(this.activeId ? this.items.findIndex((i) => i.id === this.activeId) : -1);

  #seq = 0;
  #ctrl: AbortController | null = null;
  #anchor: string | null = null;
  #suppress = false;
  alive = true;

  constructor(mode: 'library' | 'trash') {
    this.mode = mode;
    this.state = mode === 'trash' ? 'trashed' : 'active';
  }

  get facets() {
    return facetsFor(this.state);
  }

  /** Reload trigger: changes whenever the query, sort or folder would. */
  get key() {
    return JSON.stringify([this.mode, this.filters, this.sort, this.folderId, this.missing, this.foldersPhase]);
  }

  destroy() {
    this.alive = false;
    this.#ctrl?.abort();
  }

  // ---- folders -----------------------------------------------------------------------------------

  async loadFolders(quiet = false) {
    if (!quiet) this.foldersPhase = 'loading';
    try {
      const page = await api.listFolders();
      this.folders = page.items;
      this.foldersPhase = 'ready';
      this.foldersError = '';
    } catch (e) {
      this.foldersError = errorMessage(e);
      if (!quiet || this.foldersPhase !== 'ready') this.foldersPhase = 'error';
    }
  }

  refreshFolders(): Promise<void> {
    return this.loadFolders(true);
  }

  /** Direct image counts for the tree, including the root's own count. */
  get counts(): Map<string | null, number> {
    const m = new Map<string | null, number>();
    for (const f of this.folders) m.set(f.id, f.imageCount);
    m.set(null, this.rootCount);
    return m;
  }

  get rootCount(): number {
    const total = facetsFor('active')?.total;
    if (total === undefined) return 0;
    return Math.max(0, total - this.folders.reduce((n, f) => n + f.imageCount, 0));
  }

  get currentCount(): number {
    if (this.mode === 'trash') return facetsFor('trashed')?.total ?? this.items.length;
    if (this.folderId === null) return this.rootCount;
    return folderById(this.folders, this.folderId)?.imageCount ?? 0;
  }

  /** The unfiltered size of what the current folder covers (for `n / total`). */
  get scopeTotal(): number {
    if (this.mode === 'trash') return facetsFor('trashed')?.total ?? 0;
    if (this.folderId === null) return facetsFor('active')?.total ?? 0;
    return subtreeCount(this.folders, this.folderId);
  }

  get countText(): string {
    return this.filtered ? `${this.items.length} / ${this.scopeTotal}` : String(this.currentCount);
  }

  get hasAnyImages(): boolean {
    return (facetsFor('active')?.total ?? 0) > 0;
  }

  label(id: string | null): string {
    return folderLabel(this.folders, id);
  }

  names(id: string | null): string[] {
    return folderNames(this.folders, id);
  }

  chain(id: string | null): Folder[] {
    return folderChain(this.folders, id);
  }

  kids(id: string | null): Folder[] {
    return childrenOf(this.folders, id);
  }

  openFolder(id: string | null) {
    // Filters and sort live in the query and stay with the folder being opened.
    void navigate(libraryPath(folderNames(this.folders, id)) + router.search);
  }

  get canCreateInCurrent(): boolean {
    return folderDepth(this.folders, this.folderId) < MAX_DEPTH;
  }

  async createFolder(parentId: string | null, name: string): Promise<string> {
    const e = folderNameError(this.folders, parentId, name);
    if (e) return e;
    if (folderDepth(this.folders, parentId) >= MAX_DEPTH) return `最多 ${MAX_DEPTH} 层`;
    try {
      await api.createFolder({ name: name.trim(), parentId });
      await this.loadFolders(true);
      toast('已创建文件夹');
      return '';
    } catch (err) {
      return folderError(err);
    }
  }

  async renameFolder(id: string, name: string): Promise<string> {
    const f = folderById(this.folders, id);
    if (!f) return '文件夹不存在';
    const e = folderNameError(this.folders, f.parentId ?? null, name, id);
    if (e) return e;
    const here = isInside(this.folders, id, this.folderId);
    const currentId = this.folderId;
    const search = router.search;
    try {
      await api.patchFolder(id, { name: name.trim() });
      await this.loadFolders(true);
      if (here) void navigate(libraryPath(folderNames(this.folders, currentId)) + search, { replace: true });
      toast('已重命名');
      return '';
    } catch (err) {
      return folderError(err);
    }
  }

  async moveFolder(id: string, to: string | null) {
    const f = folderById(this.folders, id);
    if (!f) return;
    if (f.parentId === to) return;
    if (folderNameError(this.folders, to, f.name, id)) return toast('目标位置已有同名文件夹');
    if (!fitsUnder(this.folders, id, to)) return toast(`最多 ${MAX_DEPTH} 层`);
    const before = f.parentId ?? null;
    const here = isInside(this.folders, id, this.folderId);
    const currentId = this.folderId;
    const search = router.search;
    const where = folderLabel(this.folders, to);
    try {
      await api.patchFolder(id, { parentId: to });
      await this.loadFolders(true);
      if (here) void navigate(libraryPath(folderNames(this.folders, currentId)) + search, { replace: true });
      toast(`已将 ${f.name} 移到 ${where}`, {
        action: {
          label: '撤销',
          run: () => {
            void (async () => {
              try {
                await api.patchFolder(id, { parentId: before });
                await this.loadFolders(true);
                if (here) void navigate(libraryPath(folderNames(this.folders, currentId)) + router.search, { replace: true });
              } catch (e) {
                toast(folderError(e));
              }
            })();
          },
        },
      });
    } catch (e) {
      toast(folderError(e));
    }
  }

  async deleteFolder(id: string) {
    const f = folderById(this.folders, id);
    if (!f) return;
    const nonEmpty = childrenOf(this.folders, id).length > 0 || (this.counts.get(id) ?? 0) > 0;
    if (nonEmpty) return toast('文件夹不为空');
    const parentId = f.parentId ?? null;
    const name = f.name;
    const leaving = this.folderId === id;
    const search = router.search;
    try {
      await api.deleteFolder(id);
      await this.loadFolders(true);
      if (leaving) void navigate(libraryPath(folderNames(this.folders, parentId)) + search, { replace: true });
      toast('已删除文件夹', {
        action: {
          label: '撤销',
          run: () => {
            void (async () => {
              try {
                await api.createFolder({ name, parentId });
                await this.loadFolders(true);
                if (leaving) {
                  void navigate(libraryPath([...folderNames(this.folders, parentId), name]) + router.search, { replace: true });
                }
              } catch (e) {
                toast(folderError(e));
              }
            })();
          },
        },
      });
    } catch (e) {
      toast(folderError(e));
    }
  }

  // ---- query -------------------------------------------------------------------------------------

  setFilters(patch: Partial<Filters>) {
    const f = { ...this.filters, ...patch };
    updateQuery((p) => {
      for (const k of ['q', 'format', 'date', 'source']) p.delete(k);
      if (f.q) p.set('q', f.q);
      for (const x of f.formats) p.append('format', x);
      if (f.date) p.set('date', f.date);
      for (const x of f.sources) p.append('source', x);
    });
  }

  clearFilters(keepQuery = true) {
    this.setFilters({ ...EMPTY_FILTERS, q: keepQuery ? this.filters.q : '' });
  }

  setSort(sort: ImageSort) {
    updateQuery((p) => (sort === 'new' ? p.delete('sort') : p.set('sort', sort)));
  }

  setView(v: ViewMode) {
    this.view = v;
    try {
      localStorage.setItem(VIEW_KEY, v);
    } catch {
      // storage unavailable: the choice lasts for this page only
    }
  }

  #query(cursor?: string | null): ImageQuery {
    const f = this.filters;
    const range = f.date ? dateRange(f.date) : {};
    return {
      state: this.state,
      folder: this.mode === 'library' ? (this.folderId ?? 'root') : undefined,
      // Browsing shows direct images; searching or filtering covers the folder and everything below it.
      recursive: this.mode === 'library' && this.filtered ? true : undefined,
      q: f.q || undefined,
      format: f.formats.length ? f.formats : undefined,
      source: f.sources.length ? f.sources : undefined,
      ...range,
      sort: this.sort === 'new' ? undefined : this.sort,
      cursor: cursor ?? undefined,
    };
  }

  // ---- data --------------------------------------------------------------------------------------

  async reload(quiet = false) {
    if (this.mode === 'library' && this.foldersPhase !== 'ready') {
      if (this.foldersPhase === 'error') {
        this.errorKind = 'folders';
        this.errorText = this.foldersError;
        this.phase = 'error';
      } else {
        this.phase = 'loading';
      }
      return;
    }
    if (this.missing) {
      this.items = [];
      this.cursor = null;
      this.phase = 'ready';
      this.activeId = null;
      this.clearSel();
      return;
    }
    this.#ctrl?.abort();
    this.#ctrl = new AbortController();
    const my = ++this.#seq;
    if (!quiet) this.phase = 'loading';
    if (!quiet) {
      this.items = [];
      this.cursor = null;
      this.more = 'idle';
      this.activeId = null;
      this.clearSel();
    }
    this.errorKind = 'images';
    try {
      const page = await api.listImages(this.#query(), this.#ctrl.signal);
      if (my !== this.#seq) return;
      this.items = page.items;
      this.cursor = page.nextCursor;
      this.phase = 'ready';
      if (this.pendingOpen) {
        if (this.items.some((i) => i.id === this.pendingOpen)) this.activeId = this.pendingOpen;
        this.pendingOpen = null;
      }
      if (this.activeId && !this.items.some((i) => i.id === this.activeId)) this.activeId = null;
    } catch (e) {
      if (isAbort(e) || my !== this.#seq) return;
      this.errorText = errorMessage(e);
      this.errorKind = 'images';
      this.phase = 'error';
    }
  }

  async loadMore() {
    if (!this.cursor || this.more === 'loading' || this.phase !== 'ready') return;
    const my = this.#seq;
    this.more = 'loading';
    try {
      const page = await api.listImages(this.#query(this.cursor), this.#ctrl?.signal);
      if (my !== this.#seq) return;
      const known = new Set(this.items.map((i) => i.id));
      this.items.push(...page.items.filter((i) => !known.has(i.id)));
      this.cursor = page.nextCursor;
      this.more = 'idle';
    } catch (e) {
      if (isAbort(e) || my !== this.#seq) return;
      this.more = 'error';
      toast(errorMessage(e));
    }
  }

  loadRetention() {
    api
      .getSettings()
      .then((s) => (this.retention = s.trashRetentionDays))
      .catch(() => {});
  }

  /** A fresh upload joins the top of the plain newest-first list when it belongs here. */
  addUploaded(img: Image) {
    if (this.mode !== 'library' || this.filtered || this.sort !== 'new') return;
    if ((img.folderId ?? null) !== this.folderId) return;
    // An upload can finish while the folder is still loading; fetch the list again rather than leave
    // the new image invisible until the next navigation.
    if (this.phase !== 'ready') {
      void this.reload(true);
      return;
    }
    if (!this.items.some((i) => i.id === img.id)) this.items.unshift(img);
  }

  #replace(img: Image) {
    const i = this.items.findIndex((x) => x.id === img.id);
    if (i >= 0) this.items[i] = img;
  }

  #refresh() {
    void refreshFacets();
    void refreshHealth();
    void this.refreshFolders();
  }

  // ---- selection and opening ---------------------------------------------------------------------

  clearSel() {
    this.sel.clear();
    this.selectMode = false;
    this.#anchor = null;
  }

  selectAll() {
    for (const i of this.items) this.sel.add(i.id);
  }

  /** Uncheck every row but stay in selection mode. */
  selectNone() {
    this.sel.clear();
  }

  toggle(id: string, range = false) {
    if (range && this.#anchor) {
      const ids = this.items.map((i) => i.id);
      const a = ids.indexOf(this.#anchor);
      const b = ids.indexOf(id);
      if (a >= 0 && b >= 0) {
        for (let i = Math.min(a, b); i <= Math.max(a, b); i++) this.sel.add(ids[i]!);
        this.#anchor = id;
        return;
      }
    }
    if (this.sel.has(id)) this.sel.delete(id);
    else this.sel.add(id);
    this.#anchor = id;
  }

  /** The touch long-press that started a selection also fires a click; swallow it. */
  suppressNextClick() {
    this.#suppress = true;
  }

  clickItem(id: string, e: { metaKey: boolean; ctrlKey: boolean; shiftKey: boolean }) {
    if (this.#suppress) {
      this.#suppress = false;
      return;
    }
    if (e.metaKey || e.ctrlKey) return this.toggle(id);
    if (e.shiftKey) return this.toggle(id, true);
    if (this.selecting) return this.toggle(id);
    this.open(id);
  }

  open(id: string) {
    this.activeId = id;
  }

  close() {
    this.activeId = null;
  }

  get prev(): Image | null {
    return this.activeIndex > 0 ? (this.items[this.activeIndex - 1] ?? null) : null;
  }

  get canNext(): boolean {
    return this.activeIndex >= 0 && (this.activeIndex < this.items.length - 1 || this.cursor !== null);
  }

  async step(delta: 1 | -1) {
    const at = this.activeIndex;
    if (at < 0) return;
    let target = this.items[at + delta];
    if (!target && delta > 0 && this.cursor) {
      await this.loadMore();
      target = this.items[at + delta];
    }
    if (target) this.activeId = target.id;
    if (delta > 0 && at + delta >= this.items.length - 3) void this.loadMore();
  }

  // ---- batch actions -----------------------------------------------------------------------------

  async #batch(action: BatchAction, ids: string[], folderId?: string | null) {
    try {
      const results = await api.batch(action, ids, folderId);
      const ok = results.filter((r) => r.ok).map((r) => r.id);
      return { ok, failed: results.length - ok.length };
    } catch (e) {
      toast(errorMessage(e));
      return null;
    }
  }

  #take(ids: string[]): Removed[] {
    const set = new Set(ids);
    const removed: Removed[] = [];
    const kept: Image[] = [];
    this.items.forEach((img, index) => {
      if (set.has(img.id)) removed.push({ image: $state.snapshot(img) as Image, index });
      else kept.push(img);
    });
    this.items = kept;
    if (this.activeId && set.has(this.activeId)) this.activeId = null;
    return removed;
  }

  async #undo(inverse: 'restore' | 'trash', removed: Removed[], forSeq: number) {
    const r = await this.#batch(inverse, removed.map((x) => x.image.id));
    if (!r) return;
    if (this.alive) {
      if (this.#seq === forSeq) {
        const okIds = new Set(r.ok);
        const back = removed.filter((x) => okIds.has(x.image.id)).sort((a, b) => a.index - b.index);
        for (const x of back) this.items.splice(Math.min(x.index, this.items.length), 0, x.image);
      } else {
        void this.reload();
      }
    }
    this.#refresh();
    if (r.failed > 0) toast(`${r.failed} 张未能撤销`);
  }

  async #move(action: 'trash' | 'restore', ids: string[]) {
    const r = await this.#batch(action, ids);
    if (!r) return;
    const forSeq = this.#seq;
    const removed = this.#take(r.ok);
    this.clearSel();
    const tail = r.failed > 0 ? `，${r.failed} 张失败` : '';
    if (r.ok.length > 0) {
      const text =
        action === 'trash'
          ? r.ok.length > 1
            ? `已将 ${r.ok.length} 张图片移到回收站${tail}`
            : `已移到回收站${tail}`
          : r.ok.length > 1
            ? `已恢复 ${r.ok.length} 张图片${tail}`
            : `已恢复${tail}`;
      toast(text, { action: { label: '撤销', run: () => void this.#undo(action === 'trash' ? 'restore' : 'trash', removed, forSeq) } });
    } else if (r.failed > 0) {
      toast(action === 'trash' ? `${r.failed} 张未能移到回收站` : `${r.failed} 张未能恢复`);
    }
    this.#refresh();
  }

  trash(ids: string[]) {
    return this.#move('trash', ids);
  }

  restore(ids: string[]) {
    return this.#move('restore', ids);
  }

  requestPurge(ids: string[]) {
    if (ids.length > 0) this.purgeIds = ids;
  }

  async confirmPurge() {
    const ids = this.purgeIds;
    this.purgeIds = null;
    if (!ids) return;
    const r = await this.#batch('purge', ids);
    if (!r) return;
    this.#take(r.ok);
    this.clearSel();
    if (r.ok.length === 0) toast(`${r.failed} 张未能删除`);
    else if (r.failed > 0) toast(`已永久删除 ${r.ok.length} 张，${r.failed} 张失败`);
    else toast(r.ok.length > 1 ? `已永久删除 ${r.ok.length} 张图片` : '已永久删除');
    if (this.mode === 'trash' && this.items.length === 0 && this.cursor) void this.reload();
    this.#refresh();
  }

  // ---- moving images -----------------------------------------------------------------------------

  /** Dragging a selected image drags the whole selection; the ghost counts them. */
  dragImages(im: Image, e: DragEvent) {
    const dt = e.dataTransfer;
    if (!dt) return;
    const ids = this.sel.has(im.id) ? [...this.sel] : [im.id];
    dt.setData(IMAGE_DRAG_TYPE, JSON.stringify(ids));
    dt.effectAllowed = 'move';
    if (ids.length > 1) {
      const ghost = document.createElement('div');
      ghost.className = 'drag-ghost';
      ghost.textContent = `${ids.length} 张图片`;
      document.body.append(ghost);
      dt.setDragImage(ghost, 12, 12);
      setTimeout(() => ghost.remove(), 0);
    }
  }

  /**
   * Move images into a folder (null = root). The rows leave the current list when the destination is
   * outside what this view shows; a quiet reload keeps folder labels and counts honest either way.
   */
  async moveImages(ids: string[], to: string | null) {
    const moving = this.items.filter((im) => ids.includes(im.id) && im.state === 'active' && (im.folderId ?? null) !== to);
    if (moving.length === 0) return;
    const before = new Map(moving.map((im) => [im.id, im.folderId ?? null]));
    const r = await this.#batch(
      'move',
      moving.map((im) => im.id),
      to,
    );
    if (!r) return;
    this.clearSel();
    if (this.activeId && r.ok.includes(this.activeId)) this.activeId = null;
    const where = this.label(to);
    if (r.ok.length > 0) {
      toast(r.ok.length > 1 ? `已将 ${r.ok.length} 张图片移到 ${where}` : `已移到 ${where}`, {
        action: {
          label: '撤销',
          run: () => {
            void (async () => {
              const groups = new Map<string | null, string[]>();
              for (const id of r.ok) {
                const to0 = before.get(id) ?? null;
                const list = groups.get(to0) ?? [];
                list.push(id);
                groups.set(to0, list);
              }
              for (const [folderId, group] of groups) await this.#batch('move', group, folderId);
              await this.reload(true);
              this.#refresh();
            })();
          },
        },
      });
    } else if (r.failed > 0) {
      toast(`${r.failed} 张未能移动`);
    }
    await this.reload(true);
    this.#refresh();
  }

  // ---- single-image edits ------------------------------------------------------------------------

  async rename(id: string, filename: string): Promise<boolean> {
    try {
      this.#replace(await api.patchImage(id, { filename }));
      toast('已重命名');
      return true;
    } catch (e) {
      toast(errorMessage(e));
      return false;
    }
  }
}
