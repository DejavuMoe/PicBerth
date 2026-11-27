// State and actions of the image list (library and trash). One instance per page; filters live in the URL,
// the current folder comes from the path.

import { tick } from 'svelte';
import { SvelteSet } from 'svelte/reactivity';
import { api, ApiError, errorMessage, isAbort, rawUrl } from './api';
import { copyText } from './clipboard';
import { facetsFor, refreshFacets } from './facets.svelte';
import { startOfDay } from './format';
import { folderById, resolvePath, subtreeCount } from './folders';
import { refreshHealth } from './health.svelte';
import { modalOpen } from './modal';
import { librarySegments, router, updateQuery } from './router.svelte';
import { toast } from './toast.svelte';
import { tree } from './tree.svelte';
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

/** The drag payload images use when dropped on a folder node or a breadcrumb. */
export const IMAGE_DRAG_TYPE = 'application/x-picberth-images';

const VIEW_KEY = 'picberth.view';
/** How many further pages opening one image may load before giving up. */
const REVEAL_PAGES = 20;
export type ViewMode = 'grid' | 'list';
/** Grid unless the list was chosen; the old value `table` counts as unset. */
function loadView(): ViewMode {
  try {
    return localStorage.getItem(VIEW_KEY) === 'list' ? 'list' : 'grid';
  } catch {
    return 'grid';
  }
}

interface Removed {
  image: Image;
  index: number;
}

// Toast actions outlive the screen that created them.
let currentLibrary: Library | null = null;
const pendingChanges = new SvelteSet<string>();

async function focusList(trigger: Element | null) {
  await tick();
  if (!currentLibrary || modalOpen() || (document.activeElement !== document.body && document.activeElement !== trigger)) return;
  const target = document.querySelector<HTMLElement>('#main .head h1') ?? document.getElementById('main');
  if (target) {
    target.tabIndex = -1;
    target.focus({ preventScroll: true });
  }
}

function moveMessage(action: 'trash' | 'restore', count: number, failed: number): string {
  const message = action === 'trash'
    ? (count === 1 ? '已移到回收站' : `已将 ${count} 张图片移到回收站`)
    : (count === 1 ? '已恢复' : `已恢复 ${count} 张图片`);
  return failed ? `${message}，${failed} 张失败` : message;
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

/** `recent` lists every active image newest first, whatever folder it is in. */
export type LibraryMode = 'library' | 'trash' | 'recent';

export class Library {
  readonly mode: LibraryMode = 'library';
  readonly state: ImageState = 'active';

  items = $state<Image[]>([]);
  cursor = $state<string | null>(null);
  phase = $state<'loading' | 'ready' | 'error'>('loading');
  more = $state<'idle' | 'loading' | 'error'>('idle');
  errorText = $state('');
  errorKind = $state<'images' | 'folders'>('images');
  retention = $state<number | null>(null);
  retentionFailed = $state(false);
  deleteLabel = $derived(this.retention === null ? '删除' : this.retention === 0 ? '永久删除' : '移到回收站');

  view = $state<ViewMode>(loadView());
  sel = new SvelteSet<string>();
  readonly pending = pendingChanges;
  selectMode = $state(false);
  #activeId = $state<string | null>(null);
  /** An image to open once the list for its folder has loaded (e.g. 查看 on an upload elsewhere). */
  pendingOpen = $state<string | null>(null);
  purgeIds = $state<string[] | null>(null);
  purgeBusy = $state(false);
  purgeFailed = $state(false);
  /** The viewer (the image at window size) is open on the active image. */
  viewing = $state(false);

  /** Decoded folder names from the path (empty at the root). */
  pathNames = $derived(librarySegments(router.path));
  /** The folder id for the path: null at the root, undefined when no such folder exists. */
  resolved = $derived(this.mode === 'library' && tree.phase === 'ready' ? resolvePath(tree.folders, this.pathNames) : null);
  missing = $derived(this.mode === 'library' && this.resolved === undefined);
  folderId = $derived(this.resolved ?? null);

  filters = $derived(this.mode === 'recent' ? EMPTY_FILTERS : parseFilters(router.query));
  sort = $derived<ImageSort>(this.mode === 'recent' ? 'new' : parseSort(router.query));
  filtered = $derived(!!this.filters.q || filterCount(this.filters) > 0);
  selecting = $derived(this.sel.size > 0 || this.selectMode);
  active = $derived(this.activeId ? (this.items.find((i) => i.id === this.activeId) ?? null) : null);
  activeIndex = $derived(this.activeId ? this.items.findIndex((i) => i.id === this.activeId) : -1);

  #seq = 0;
  #ctrl: AbortController | null = null;
  #retentionSeq = 0;
  #anchor: string | null = null;
  #suppress = false;

  /** The image whose detail is open. Without one there is no viewer either. */
  get activeId(): string | null {
    return this.#activeId;
  }

  set activeId(id: string | null) {
    this.#activeId = id;
    if (id === null) this.viewing = false;
  }

  constructor(mode: LibraryMode) {
    this.mode = mode;
    this.state = mode === 'trash' ? 'trashed' : 'active';
    currentLibrary = this;
  }

  get facets() {
    return facetsFor(this.state);
  }

  /** Reload trigger: changes whenever the query, sort or folder would. */
  get key() {
    return JSON.stringify([this.mode, this.filters, this.sort, this.folderId, this.missing, this.foldersPhase]);
  }

  destroy() {
    if (currentLibrary === this) currentLibrary = null;
    this.#ctrl?.abort();
  }

  // ---- folders (shared with the sidebar: see lib/tree.svelte.ts) -----------------------------------

  get folders(): Folder[] {
    return tree.folders;
  }

  get foldersPhase(): 'loading' | 'ready' | 'error' {
    return tree.phase;
  }

  get foldersError(): string {
    return tree.error;
  }

  loadFolders(): Promise<void> {
    return tree.load();
  }

  refreshFolders(): Promise<void> {
    return tree.refresh();
  }

  /** Direct image counts per folder, including the root's own count. */
  get counts(): Map<string | null, number> {
    return tree.counts;
  }

  get currentCount(): number {
    if (this.mode === 'trash') return facetsFor('trashed')?.total ?? this.items.length;
    if (this.folderId === null) return tree.rootCount;
    return folderById(tree.folders, this.folderId)?.imageCount ?? 0;
  }

  /** The unfiltered size of what the current folder covers (for `n / total`). */
  get scopeTotal(): number {
    if (this.mode === 'trash') return facetsFor('trashed')?.total ?? 0;
    if (this.folderId === null) return facetsFor('active')?.total ?? 0;
    return subtreeCount(tree.folders, this.folderId);
  }

  get countText(): string {
    return this.filtered ? `${this.items.length} / ${this.scopeTotal}` : String(this.currentCount);
  }

  get hasAnyImages(): boolean {
    return (facetsFor('active')?.total ?? 0) > 0;
  }

  label(id: string | null): string {
    return tree.label(id);
  }

  kids(id: string | null): Folder[] {
    return tree.kids(id);
  }

  chain(id: string | null): Folder[] {
    return tree.chain(id);
  }

  openFolder(id: string | null) {
    tree.openFolder(id);
  }

  /** The direct subfolders shown above the images (grid view): not while searching, filtering or in the trash. */
  get subfolders(): Folder[] {
    if (this.mode !== 'library' || this.filtered || tree.phase !== 'ready') return [];
    return tree.kids(this.folderId);
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
      this.activeId = null;
      this.clearSel();
    }
    this.errorKind = 'images';
    try {
      const page = await api.listImages(this.#query(), this.#ctrl.signal);
      if (my !== this.#seq) return;
      this.items = page.items;
      this.cursor = page.nextCursor;
      this.more = 'idle';
      if (quiet && !page.nextCursor && this.sel.size) {
        const visible = new Set(page.items.map((im) => im.id));
        for (const id of this.sel) if (!visible.has(id)) this.sel.delete(id);
        if (!this.sel.size) this.clearSel();
      }
      this.phase = 'ready';
      if (this.pendingOpen) void this.reveal(this.pendingOpen, true);
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

  /**
   * Open an image's detail, loading further pages until it is in the list (at most REVEAL_PAGES of them).
   * The request stays in `pendingOpen` while a reload is under way or may still follow a folder change;
   * the reload ends with a `settled` call, after which an image that was not found stays closed.
   */
  async reveal(id: string, settled = false) {
    this.pendingOpen = id;
    if (this.phase !== 'ready') return;
    const my = this.#seq;
    for (let n = 0; n < REVEAL_PAGES && !this.items.some((i) => i.id === id); n++) {
      const before = this.cursor;
      if (!before) break;
      await this.loadMore();
      if (my !== this.#seq) return;
      if (this.cursor === before) break;
    }
    const found = this.items.some((i) => i.id === id);
    if (this.pendingOpen === id && (found || settled)) this.pendingOpen = null;
    if (found) this.activeId = id;
  }

  async loadRetention(refocus = false) {
    const trigger = document.activeElement;
    const seq = ++this.#retentionSeq;
    this.retention = null;
    this.retentionFailed = false;
    try {
      const settings = await api.getSettings();
      if (seq === this.#retentionSeq) this.retention = settings.trashRetentionDays;
    } catch {
      if (seq === this.#retentionSeq) this.retentionFailed = true;
    }
    if (refocus && seq === this.#retentionSeq && currentLibrary === this) void focusList(trigger);
  }

  /** A fresh upload joins the top of the plain newest-first list when it belongs here. */
  addUploaded(img: Image) {
    if (this.mode === 'trash' || this.filtered || this.sort !== 'new') return;
    if (this.mode === 'library' && (img.folderId ?? null) !== this.folderId) return;
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

  #refresh(reloadList = false) {
    void refreshFacets();
    void refreshHealth();
    void this.refreshFolders();
    if (reloadList || currentLibrary !== this) void currentLibrary?.reload(true);
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

  /** Double-click: the detail and the viewer open together. */
  openViewer(id: string) {
    if (this.selecting) return;
    this.activeId = id;
    this.viewing = true;
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

  async #batch(action: BatchAction, ids: string[], folderId?: string | null, expectedRetention?: number) {
    const { results, error, unconfirmed } = await api.batch(action, ids, folderId, expectedRetention);
    // A confirmed delete may have committed before its response was lost.
    const ok = new Set(results.filter((r) => r.ok || (expectedRetention === 0 && r.code === 'not-found')).map((r) => r.id));
    let reloadList = false;
    const ambiguous = error instanceof ApiError && (error.status === 0 || error.status === 200 || error.status >= 500);
    if (action === 'trash' && expectedRetention !== undefined && expectedRetention > 0 && ambiguous && unconfirmed?.length) {
      // Replaying trash could delete an image restored by someone else in the meantime.
      try {
        if (unconfirmed.length === 1) {
          const image = await api.getImage(unconfirmed[0]!);
          if (image.state === 'trashed') ok.add(image.id);
        } else {
          const pending = new Set(unconfirmed);
          let cursor: string | undefined;
          do {
            const page = await api.listImages({ state: 'trashed', limit: 200, cursor });
            for (const image of page.items) {
              if (pending.delete(image.id)) ok.add(image.id);
            }
            cursor = page.nextCursor ?? undefined;
          } while (cursor && pending.size);
        }
      } catch {
        reloadList = true;
      }
    }
    if (action === 'restore') {
      const check = results.filter((r) => r.code === 'conflict').map((r) => r.id);
      if (ambiguous) {
        check.push(...(unconfirmed ?? []));
      }
      // One image can be verified directly. For a batch, refresh the visible list once.
      if (check.length === 1) {
        const id = check[0]!;
        try {
          if ((await api.getImage(id)).state === 'active') ok.add(id);
        } catch {
          reloadList = true;
        }
      } else reloadList = check.length > 1;
    }
    const unfinished = ids.length - ok.size;
    const policyChanged = error instanceof ApiError && error.code === 'trash-retention-changed';
    if (policyChanged) {
      toast('回收站保留时间已更改，请重新确认。');
      void currentLibrary?.loadRetention(true);
    } else if (error && unfinished > 0 && expectedRetention !== 0) {
      toast(errorMessage(error));
      if (results.length === 0 && ok.size === 0 && !reloadList) return null;
    }
    const rejected = results.filter((r) => !ok.has(r.id) && !(reloadList && r.code === 'conflict')).length;
    if (reloadList && !error && !ok.size && !rejected) toast(errorMessage(new ApiError({ status: 409, code: 'conflict' })));
    return { ok: [...ok], unfinished, rejected, policyChanged, reloadList };
  }

  async #changeState(action: 'trash' | 'restore' | 'purge', ids: string[], expectedRetention?: number) {
    const targets = ids.filter((id) => !this.pending.has(id));
    if (!targets.length) return null;
    for (const id of targets) this.pending.add(id);
    try {
      return await this.#batch(action, targets, undefined, expectedRetention);
    } finally {
      for (const id of targets) this.pending.delete(id);
    }
  }

  #take(ids: string[]): Removed[] {
    if (!ids.length) return [];
    const set = new Set(ids);
    const removed: Removed[] = [];
    const kept: Image[] = [];
    this.items.forEach((img, index) => {
      if (set.has(img.id)) removed.push({ image: $state.snapshot(img) as Image, index });
      else kept.push(img);
    });
    this.items = kept;
    if (this.activeId && set.has(this.activeId)) this.activeId = null;
    const hadSelection = this.sel.size > 0;
    for (const id of ids) this.sel.delete(id);
    if (hadSelection && this.sel.size === 0) this.clearSel();
    return removed;
  }

  async #undo(inverse: 'restore' | 'trash', ids: string[], removed: Removed[], forSeq: number, expectedRetention?: number) {
    const trigger = document.activeElement;
    const r = await this.#changeState(inverse, ids, expectedRetention);
    if (!r) return;
    let reloadList = r.reloadList;
    if (r.ok.length && currentLibrary === this) {
      if (this.#seq === forSeq && removed.length === ids.length) {
        const okIds = new Set(r.ok);
        for (const x of removed) {
          if (okIds.has(x.image.id)) this.items.splice(Math.min(x.index, this.items.length), 0, x.image);
        }
      } else {
        reloadList = true;
      }
    } else if (currentLibrary && currentLibrary.mode !== this.mode) {
      currentLibrary.#take(r.ok);
    }
    this.#refresh(reloadList);
    if (r.ok.length) {
      toast(moveMessage(inverse, r.ok.length, r.rejected));
      void focusList(trigger);
    } else if (r.rejected && !r.policyChanged) toast(`${r.rejected} 张未能撤销`);
  }

  async #move(action: 'trash' | 'restore', ids: string[]) {
    const trigger = document.activeElement;
    const retention = this.retention ?? undefined;
    const forSeq = this.#seq;
    const r = await this.#changeState(action, ids, action === 'trash' ? retention : undefined);
    if (!r) return;
    const removed = this.#take(r.ok);
    if (r.ok.length > 0) {
      const undo = action === 'trash'
        ? () => void this.#undo('restore', r.ok, removed, forSeq)
        : retention !== undefined && retention > 0
          ? () => void this.#undo('trash', r.ok, removed, forSeq, retention)
          : undefined;
      toast(moveMessage(action, r.ok.length, r.rejected), undo ? { action: { label: '撤销', run: undo } } : {});
      if (currentLibrary === this) void focusList(trigger);
    } else if (r.rejected > 0 && !r.policyChanged) {
      toast(action === 'trash' ? `${r.rejected} 张未能移到回收站` : `${r.rejected} 张未能恢复`);
    }
    this.#refresh(r.reloadList || this.#seq !== forSeq);
  }

  trash(ids: string[]) {
    if (this.retention === null || this.purgeBusy) return;
    if (this.retention === 0) return this.requestPurge(ids);
    return this.#move('trash', ids);
  }

  restore(ids: string[]) {
    return this.#move('restore', ids);
  }

  requestPurge(ids: string[]) {
    const targets = ids.filter((id) => !this.pending.has(id));
    if (targets.length > 0 && !this.purgeBusy) {
      this.purgeFailed = false;
      this.purgeIds = targets;
    }
  }

  async confirmPurge() {
    const trigger = document.activeElement;
    const ids = this.purgeIds;
    if (!ids || this.purgeBusy) return;
    const forSeq = this.#seq;
    const fromLibrary = this.mode !== 'trash';
    if (fromLibrary) {
      this.purgeBusy = true;
      this.purgeFailed = false;
    } else {
      this.purgeIds = null;
    }
    const r = await this.#changeState(fromLibrary ? 'trash' : 'purge', ids, fromLibrary ? 0 : undefined);
    this.purgeBusy = false;
    if (!r) return;
    this.#take(r.ok);
    if (fromLibrary) {
      const done = new Set(r.ok);
      this.purgeIds = r.policyChanged || !r.unfinished ? null : ids.filter((id) => !done.has(id));
      this.purgeFailed = this.purgeIds !== null;
    }
    if (r.ok.length === 0) {
      if (!this.purgeFailed && !r.policyChanged) toast(`${r.unfinished} 张未能删除`);
    } else if (r.rejected > 0) toast(`已永久删除 ${r.ok.length} 张，${r.rejected} 张失败`);
    else toast(r.ok.length > 1 ? `已永久删除 ${r.ok.length} 张图片` : '已永久删除');
    if (r.ok.length && currentLibrary === this) void focusList(trigger);
    this.#refresh(this.#seq !== forSeq || (this.mode === 'trash' && !this.items.length && this.cursor !== null));
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
    } else if (r.unfinished > 0) {
      toast(`${r.unfinished} 张未能移动`);
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
