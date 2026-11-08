// State and actions of the image list (library and trash). One instance per page; filters live in the URL.

import { SvelteSet } from 'svelte/reactivity';
import { api, errorMessage, isAbort, rawUrl } from './api';
import { copyText } from './clipboard';
import { facetsFor, refreshFacets } from './facets.svelte';
import { startOfDay } from './format';
import { refreshHealth } from './health.svelte';
import { router, updateQuery } from './router.svelte';
import { toast } from './toast.svelte';
import type { BatchAction, Image, ImageFormat, ImageQuery, ImageSort, ImageState, LinkFormat } from './types';
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
  tags: string[];
}

export const EMPTY_FILTERS: Filters = { q: '', formats: [], date: '', sources: [], tags: [] };

function parseFilters(p: URLSearchParams): Filters {
  const date = p.get('date');
  return {
    q: p.get('q') ?? '',
    formats: FORMATS.filter((f) => p.getAll('format').includes(f)),
    date: DATE_RANGES.find((d) => d.key === date)?.key ?? '',
    sources: [...new Set(p.getAll('source').filter(Boolean))],
    tags: [...new Set(p.getAll('tag').filter(Boolean))],
  };
}

function parseSort(p: URLSearchParams): ImageSort {
  const s = p.get('sort');
  return SORTS.find((x) => x.key === s)?.key ?? 'new';
}

/** Number of active filters, not counting the search text. */
export function filterCount(f: Filters): number {
  return f.formats.length + (f.date ? 1 : 0) + f.sources.length + f.tags.length;
}

function dateRange(key: DateKey, now = Date.now()): { since?: string; until?: string } {
  const DAY = 86_400_000;
  if (key === 'today') return { since: new Date(startOfDay(now)).toISOString() };
  if (key === '7d') return { since: new Date(now - 7 * DAY).toISOString() };
  if (key === '30d') return { since: new Date(now - 30 * DAY).toISOString() };
  return { until: new Date(now - 30 * DAY).toISOString() };
}

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
  readonly mode: 'library' | 'trash';
  readonly state: ImageState;

  items = $state<Image[]>([]);
  cursor = $state<string | null>(null);
  phase = $state<'loading' | 'ready' | 'error'>('loading');
  more = $state<'idle' | 'loading' | 'error'>('idle');
  errorText = $state('');
  retention = $state<number | null>(null);

  view = $state<ViewMode>(loadView());
  sel = new SvelteSet<string>();
  selectMode = $state(false);
  activeId = $state<string | null>(null);
  purgeIds = $state<string[] | null>(null);

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

  /** Reload trigger: changes whenever the query would. */
  get key() {
    return JSON.stringify([this.mode, this.filters, this.sort]);
  }

  destroy() {
    this.alive = false;
    this.#ctrl?.abort();
  }

  // ---- query -------------------------------------------------------------------------------------

  setFilters(patch: Partial<Filters>) {
    const f = { ...this.filters, ...patch };
    updateQuery((p) => {
      for (const k of ['q', 'format', 'date', 'source', 'tag']) p.delete(k);
      if (f.q) p.set('q', f.q);
      for (const x of f.formats) p.append('format', x);
      if (f.date) p.set('date', f.date);
      for (const x of f.sources) p.append('source', x);
      for (const x of f.tags) p.append('tag', x);
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
      q: f.q || undefined,
      format: f.formats.length ? f.formats : undefined,
      tag: f.tags.length ? f.tags : undefined,
      source: f.sources.length ? f.sources : undefined,
      ...range,
      sort: this.sort === 'new' ? undefined : this.sort,
      cursor: cursor ?? undefined,
    };
  }

  // ---- data --------------------------------------------------------------------------------------

  async reload() {
    this.#ctrl?.abort();
    this.#ctrl = new AbortController();
    const my = ++this.#seq;
    this.phase = 'loading';
    this.items = [];
    this.cursor = null;
    this.more = 'idle';
    this.activeId = null;
    this.clearSel();
    try {
      const page = await api.listImages(this.#query(), this.#ctrl.signal);
      if (my !== this.#seq) return;
      this.items = page.items;
      this.cursor = page.nextCursor;
      this.phase = 'ready';
    } catch (e) {
      if (isAbort(e) || my !== this.#seq) return;
      this.errorText = errorMessage(e);
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

  /** A fresh upload joins the top of the plain newest-first list. */
  addUploaded(img: Image) {
    if (this.mode !== 'library' || this.filtered || this.sort !== 'new' || this.phase !== 'ready') return;
    if (!this.items.some((i) => i.id === img.id)) this.items.unshift(img);
  }

  #replace(img: Image) {
    const i = this.items.findIndex((x) => x.id === img.id);
    if (i >= 0) this.items[i] = img;
  }

  #refresh() {
    void refreshFacets();
    void refreshHealth();
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

  async #batch(action: BatchAction, ids: string[]) {
    try {
      const results = await api.batch(action, ids);
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

  async setTags(id: string, tags: string[]) {
    const row = this.items.find((i) => i.id === id);
    if (!row) return;
    const before = [...row.tags];
    row.tags = tags;
    try {
      this.#replace(await api.patchImage(id, { tags }));
      void refreshFacets();
    } catch (e) {
      const cur = this.items.find((i) => i.id === id);
      if (cur) cur.tags = before;
      toast(errorMessage(e));
    }
  }
}
