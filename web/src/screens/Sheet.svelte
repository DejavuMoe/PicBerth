<script lang="ts">
  // 底片 / 回收站: the contact sheet. Images are frames on film strips, grouped into one roll per day;
  // this session's uploads develop in their own roll on top (docs/ui.md §5.1–5.3).
  import { flushSync, onDestroy, onMount, tick, untrack } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import Button from '../components/Button.svelte';
  import Chip from '../components/Chip.svelte';
  import ConfirmDialog from '../components/ConfirmDialog.svelte';
  import DevelopFrame from '../components/DevelopFrame.svelte';
  import FilmRoll from '../components/FilmRoll.svelte';
  import Frame from '../components/Frame.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import Segmented from '../components/Segmented.svelte';
  import Select from '../components/Select.svelte';
  import SelectionBar from '../components/SelectionBar.svelte';
  import TextField from '../components/TextField.svelte';
  import { api, errorMessage, isAbort } from '../lib/api';
  import { copyText } from '../lib/clipboard';
  import { formatBytes, formatInt } from '../lib/format';
  import { health, refreshHealth } from '../lib/health.svelte';
  import { isTypingTarget } from '../lib/modal';
  import { openPicker } from '../lib/picker';
  import { router, setQuery } from '../lib/router.svelte';
  import { toast } from '../lib/toast.svelte';
  import type { BatchAction, Image, ImageFormat, ImageQuery, ImageSort, LinkFormat, Visibility } from '../lib/types';
  import {
    clearFinished,
    freshIds,
    LINK_FORMATS,
    linkOf,
    onUploaded,
    setLinkFormat,
    uploads,
  } from '../lib/upload.svelte';
  import Lightbox from './Lightbox.svelte';

  interface Props {
    mode: 'library' | 'trash';
  }
  let { mode }: Props = $props();
  const isTrash = $derived(mode === 'trash');

  // ---- filters live in the URL -------------------------------------------------------------
  const FORMATS: ImageFormat[] = ['jpeg', 'png', 'gif', 'webp', 'avif'];
  const FORMAT_LABEL: Record<ImageFormat, string> = { jpeg: 'JPEG', png: 'PNG', gif: 'GIF', webp: 'WebP', avif: 'AVIF' };
  const SORTS: ImageSort[] = ['new', 'old', 'large', 'small'];

  const query = $derived(router.query);
  const f = $derived.by(() => {
    const format = query.get('format') ?? '';
    const visibility = query.get('visibility') ?? '';
    const sort = query.get('sort') ?? 'new';
    return {
      q: query.get('q') ?? '',
      format: (FORMATS as string[]).includes(format) ? (format as ImageFormat) : ('' as const),
      visibility: visibility === 'public' || visibility === 'private' ? (visibility as Visibility) : ('' as const),
      gps: query.get('gps') === 'true',
      sort: (SORTS as string[]).includes(sort) ? (sort as ImageSort) : ('new' as const),
    };
  });
  const filtered = $derived(!!(f.q || f.format || f.visibility || f.gps));

  const VIS_OPTIONS: { value: '' | Visibility; label: string }[] = [
    { value: '', label: '全部' },
    { value: 'public', label: '公开' },
    { value: 'private', label: '私有' },
  ];
  const SORT_OPTIONS = [
    { value: 'new', label: '最新在前' },
    { value: 'old', label: '最早在前' },
    { value: 'large', label: '最大在前' },
    { value: 'small', label: '最小在前' },
  ];

  // Debounced search box. Enter applies immediately.
  let qInput = $state(router.query.get('q') ?? '');
  let searchEl = $state<HTMLInputElement | null>(null);
  $effect(() => {
    const v = qInput.trim();
    if (v === f.q) return;
    const t = setTimeout(() => setQuery({ q: v || undefined }), 250);
    return () => clearTimeout(t);
  });
  $effect(() => {
    const urlQ = f.q;
    untrack(() => {
      if (urlQ !== qInput.trim()) qInput = urlQ;
    });
  });

  function clearFilters() {
    qInput = '';
    setQuery({ q: undefined, format: undefined, visibility: undefined, gps: undefined });
  }

  // ---- data ----------------------------------------------------------------------------------
  let items = $state<Image[]>([]);
  let cursor = $state<string | null>(null);
  let phase = $state<'loading' | 'ready' | 'error'>('loading');
  let more = $state<'idle' | 'loading' | 'error'>('idle');
  let errorText = $state('');
  let seq = 0;
  let ctrl: AbortController | null = null;
  let alive = true;
  onDestroy(() => {
    alive = false;
    ctrl?.abort();
  });

  function pageQuery(cursorArg?: string | null): ImageQuery {
    return {
      state: isTrash ? 'trashed' : 'active',
      q: f.q || undefined,
      format: f.format || undefined,
      visibility: f.visibility || undefined,
      gps: f.gps || undefined,
      sort: f.sort === 'new' ? undefined : f.sort,
      cursor: cursorArg ?? undefined,
    };
  }

  async function reload() {
    ctrl?.abort();
    ctrl = new AbortController();
    const my = ++seq;
    phase = 'loading';
    items = [];
    cursor = null;
    more = 'idle';
    selected.clear();
    anchorId = null;
    try {
      const page = await api.listImages(pageQuery(), ctrl.signal);
      if (my !== seq) return;
      items = page.items;
      cursor = page.nextCursor;
      phase = 'ready';
      void fillViewport();
    } catch (e) {
      if (isAbort(e) || my !== seq) return;
      errorText = errorMessage(e);
      phase = 'error';
    }
  }

  async function loadMore() {
    if (!cursor || more === 'loading' || phase !== 'ready') return;
    const my = seq;
    more = 'loading';
    try {
      const page = await api.listImages(pageQuery(cursor), ctrl?.signal);
      if (my !== seq) return;
      const known = new Set(items.map((i) => i.id));
      items.push(...page.items.filter((i) => !known.has(i.id)));
      cursor = page.nextCursor;
      more = 'idle';
      void fillViewport();
    } catch (e) {
      if (isAbort(e) || my !== seq) return;
      more = 'error';
      toast(errorMessage(e));
    }
  }

  const filterKey = $derived(JSON.stringify([mode, f.q, f.format, f.visibility, f.gps, f.sort]));
  $effect(() => {
    filterKey;
    untrack(() => void reload());
  });

  // Freshly archived uploads join today's roll when the sheet shows the plain newest-first view.
  let statsTimer: ReturnType<typeof setTimeout> | undefined;
  onMount(() => {
    void refreshHealth();
    if (mode === 'trash') {
      api
        .getSettings()
        .then((s) => (retention = s.trashRetentionDays))
        .catch(() => {});
    }
    const off = onUploaded((img) => {
      clearTimeout(statsTimer);
      statsTimer = setTimeout(() => void refreshHealth(), 600);
      if (isTrash || filtered || f.sort !== 'new' || phase !== 'ready') return;
      if (!items.some((i) => i.id === img.id)) items.unshift(img);
    });
    return () => {
      off();
      clearTimeout(statsTimer);
    };
  });

  let retention = $state<number | null>(null);
  const stats = $derived(health.status?.images ?? null);

  function daysLeft(img: Image): string | null {
    if (retention === null || retention <= 0 || !img.trashedAt) return null;
    const ms = new Date(img.trashedAt).getTime() + retention * 86_400_000 - Date.now();
    const days = Math.ceil(ms / 86_400_000);
    return days <= 0 ? '即将清除' : `${days} 天后清除`;
  }

  // ---- layout: strips and rolls ----------------------------------------------------------------
  let sheetWidth = $state(0);
  const cols = $derived.by(() => {
    const w = sheetWidth;
    if (!w) return 4;
    const min = w < 560 ? 148 : 208;
    return Math.max(2, Math.floor((w - 16 + 6) / (min + 6)));
  });

  const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
  function dayOf(iso: string) {
    const d = new Date(iso);
    return { key: `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`, d };
  }
  function rollTitle(d: Date): { title: string; sub: string } {
    const now = new Date();
    const start = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
    const days = Math.round((start(now) - start(d)) / 86_400_000);
    const date = `${d.getFullYear() !== now.getFullYear() ? `${d.getFullYear()}年` : ''}${d.getMonth() + 1}月${d.getDate()}日`;
    if (days === 0) return { title: '今天', sub: `${date} ${WEEK[d.getDay()]}` };
    if (days === 1) return { title: '昨天', sub: `${date} ${WEEK[d.getDay()]}` };
    return { title: date, sub: WEEK[d.getDay()] ?? '' };
  }

  interface Roll {
    key: string;
    title: string;
    sub: string;
    items: Image[];
  }
  const rolls = $derived.by((): Roll[] => {
    if (f.sort === 'large' || f.sort === 'small') {
      return [{ key: f.sort, title: f.sort === 'large' ? '按大小' : '按大小', sub: f.sort === 'large' ? '从大到小' : '从小到大', items }];
    }
    const out: Roll[] = [];
    for (const img of items) {
      const { key, d } = dayOf(img.createdAt);
      let roll = out[out.length - 1];
      if (!roll || roll.key !== key) {
        roll = { key, ...rollTitle(d), items: [] };
        out.push(roll);
      }
      roll.items.push(img);
    }
    return out;
  });

  // ---- infinite scroll -----------------------------------------------------------------------
  let sentinel = $state<HTMLElement | null>(null);
  $effect(() => {
    if (!sentinel) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) void loadMore();
      },
      { rootMargin: '800px 0px' },
    );
    io.observe(sentinel);
    return () => io.disconnect();
  });
  /** IntersectionObserver only reports changes; keep loading while the sentinel is still near the viewport. */
  async function fillViewport() {
    await tick();
    if (!alive || !sentinel || !cursor || more === 'loading') return;
    if (sentinel.getBoundingClientRect().top < window.innerHeight + 800) void loadMore();
  }

  // ---- selection: the grease pencil ----------------------------------------------------------
  const selected = new SvelteSet<string>();
  let anchorId: string | null = null;

  function setSelected(id: string, on: boolean) {
    if (on) selected.add(id);
    else selected.delete(id);
    anchorId = id;
  }
  function selectRange(toId: string) {
    const a = items.findIndex((i) => i.id === anchorId);
    const b = items.findIndex((i) => i.id === toId);
    if (a < 0 || b < 0) return;
    for (let i = Math.min(a, b); i <= Math.max(a, b); i++) {
      const it = items[i];
      if (it) selected.add(it.id);
    }
  }
  function clearSelection() {
    selected.clear();
    anchorId = null;
  }

  function activate(e: MouseEvent | KeyboardEvent, img: Image) {
    if (e.shiftKey && anchorId) selectRange(img.id);
    else if (e.ctrlKey || e.metaKey || (selected.size > 0 && !(e instanceof KeyboardEvent))) {
      setSelected(img.id, !selected.has(img.id));
    } else void openFrame(img.id);
  }
  function toggle(e: MouseEvent | KeyboardEvent | PointerEvent, img: Image) {
    if (e.shiftKey && anchorId) selectRange(img.id);
    else setSelected(img.id, !selected.has(img.id));
  }

  // ---- keyboard: '/' search, arrows between frames, Esc clears -------------------------------
  function frameButtons(): HTMLElement[] {
    return Array.from(document.querySelectorAll<HTMLElement>('[data-frame-btn]'));
  }
  function moveFocus(from: HTMLElement, key: string) {
    const all = frameButtons();
    const i = all.indexOf(from);
    if (i < 0) return;
    if (key === 'ArrowLeft') return all[i - 1]?.focus();
    if (key === 'ArrowRight') return all[i + 1]?.focus();
    const r = from.getBoundingClientRect();
    const cx = r.left + r.width / 2;
    const down = key === 'ArrowDown';
    let best: HTMLElement | null = null;
    let bestScore = Infinity;
    for (const b of all) {
      const br = b.getBoundingClientRect();
      const dy = down ? br.top - r.bottom : r.top - br.bottom;
      if (dy < 0) continue;
      const score = dy * 4 + Math.abs(br.left + br.width / 2 - cx);
      if (score < bestScore) {
        bestScore = score;
        best = b;
      }
    }
    best?.focus();
    best?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
  }

  function onwindowkeydown(e: KeyboardEvent) {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey || openId) return;
    if (document.querySelector('dialog[open]')) return;
    if (e.key === '/' && !isTypingTarget(e.target)) {
      e.preventDefault();
      searchEl?.focus();
      return;
    }
    if (e.key === 'Escape') {
      if (isTypingTarget(e.target) && e.target === searchEl && qInput) {
        qInput = '';
        return;
      }
      if (selected.size > 0) clearSelection();
      return;
    }
    const t = e.target;
    if (t instanceof HTMLElement && t.hasAttribute('data-frame-btn') && e.key.startsWith('Arrow')) {
      e.preventDefault();
      moveFocus(t, e.key);
    }
  }

  // ---- lightbox ------------------------------------------------------------------------------
  let openId = $state<string | null>(null);
  let morphId = $state<string | null>(null);
  const openIndex = $derived(openId ? items.findIndex((i) => i.id === openId) : -1);
  const current = $derived(openIndex >= 0 ? (items[openIndex] ?? null) : null);
  const canPrev = $derived(openIndex > 0);
  const canNext = $derived(openIndex >= 0 && (openIndex < items.length - 1 || cursor !== null));
  const total = $derived(isTrash ? stats?.trashed : filtered ? null : stats?.active);
  const position = $derived(
    openIndex >= 0 ? `${String(openIndex + 1).padStart(2, '0')} / ${total && !filtered ? formatInt(total) : formatInt(items.length) + (cursor ? '+' : '')}` : '',
  );

  function frameImg(id: string): HTMLElement | null {
    return document.querySelector<HTMLElement>(`[data-frame="${CSS.escape(id)}"] img.ready`);
  }
  function canMorph(el: HTMLElement | null): el is HTMLElement {
    if (!el || !('startViewTransition' in document)) return false;
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return false;
    const r = el.getBoundingClientRect();
    return r.bottom > 0 && r.top < innerHeight;
  }

  async function openFrame(id: string) {
    const el = frameImg(id);
    if (!canMorph(el)) {
      openId = id;
      return;
    }
    flushSync(() => (morphId = id));
    const vt = document.startViewTransition(() => {
      flushSync(() => {
        morphId = null;
        openId = id;
      });
    });
    await vt.finished.catch(() => {});
  }

  function focusFrame(id: string | null) {
    if (!id) return;
    document.querySelector<HTMLElement>(`[data-frame="${CSS.escape(id)}"] [data-frame-btn]`)?.focus({ preventScroll: true });
  }

  async function closeLightbox(focusId?: string | null) {
    const id = focusId ?? openId;
    if (id === openId && id && canMorph(frameImg(id))) {
      const vt = document.startViewTransition(() => {
        flushSync(() => {
          openId = null;
          morphId = id;
        });
      });
      await vt.finished.catch(() => {});
      morphId = null;
    } else {
      openId = null;
      await tick();
    }
    focusFrame(id);
  }

  async function step(delta: number) {
    if (openIndex < 0) return;
    let target = items[openIndex + delta];
    if (!target && delta > 0 && cursor) {
      await loadMore();
      target = items[openIndex + delta];
    }
    if (target) {
      openId = target.id;
      frameImg(target.id)?.scrollIntoView({ block: 'nearest' });
    }
    if (delta > 0 && openIndex + delta >= items.length - 3) void loadMore();
  }

  // ---- actions -------------------------------------------------------------------------------
  interface Removed {
    image: Image;
    index: number;
  }

  async function runBatch(action: BatchAction, ids: string[], visibility?: Visibility) {
    try {
      const results = await api.batch(action, ids, visibility);
      const ok = results.filter((r) => r.ok).map((r) => r.id);
      return { ok, failed: results.length - ok.length };
    } catch (e) {
      toast(errorMessage(e));
      return null;
    }
  }

  function takeFromList(ids: string[]): Removed[] {
    const set = new Set(ids);
    const removed: Removed[] = [];
    const kept: Image[] = [];
    items.forEach((img, index) => {
      if (set.has(img.id)) removed.push({ image: $state.snapshot(img) as Image, index });
      else kept.push(img);
    });
    items = kept;
    for (const id of ids) selected.delete(id);
    return removed;
  }

  /** Reverse a trash / restore: ask the server for the inverse and put the frames back where they were. */
  async function undo(inverse: 'restore' | 'trash', removed: Removed[], forSeq: number) {
    const r = await runBatch(inverse, removed.map((x) => x.image.id));
    if (!r) return;
    if (alive) {
      if (seq === forSeq) {
        const okIds = new Set(r.ok);
        const back = removed.filter((x) => okIds.has(x.image.id)).sort((a, b) => a.index - b.index);
        for (const x of back) items.splice(Math.min(x.index, items.length), 0, x.image);
      } else {
        void reload();
      }
    }
    void refreshHealth();
    if (r.failed > 0) toast(`${r.failed} 张未能撤销`);
  }

  /** A frame that is about to disappear from under the lightbox hands focus to its neighbour. */
  function neighbourOf(ids: string[]): string | null {
    const set = new Set(ids);
    const i = items.findIndex((x) => set.has(x.id));
    for (let k = i + 1; k < items.length; k++) if (items[k] && !set.has(items[k]!.id)) return items[k]!.id;
    for (let k = i - 1; k >= 0; k--) if (items[k] && !set.has(items[k]!.id)) return items[k]!.id;
    return null;
  }

  async function moveToTrash(ids: string[]) {
    const focusAfter = neighbourOf(ids);
    const r = await runBatch('trash', ids);
    if (!r) return;
    const forSeq = seq;
    if (openId && r.ok.includes(openId)) await closeLightbox(focusAfter);
    const removed = takeFromList(r.ok);
    if (r.ok.length > 0) {
      const tail = r.failed > 0 ? `，${r.failed} 张失败` : '';
      toast(`已移到回收站 ${r.ok.length} 张${tail}`, {
        action: { label: '撤销', run: () => void undo('restore', removed, forSeq) },
      });
    } else if (r.failed > 0) {
      toast(`${r.failed} 张未能移到回收站`);
    }
    void refreshHealth();
  }

  async function restore(ids: string[]) {
    const focusAfter = neighbourOf(ids);
    const r = await runBatch('restore', ids);
    if (!r) return;
    const forSeq = seq;
    if (openId && r.ok.includes(openId)) await closeLightbox(focusAfter);
    const removed = takeFromList(r.ok);
    if (r.ok.length > 0) {
      const tail = r.failed > 0 ? `，${r.failed} 张失败` : '';
      toast(`已恢复 ${r.ok.length} 张${tail}`, {
        action: { label: '撤销', run: () => void undo('trash', removed, forSeq) },
      });
    } else if (r.failed > 0) {
      toast(`${r.failed} 张未能恢复`);
    }
    void refreshHealth();
  }

  // Purge is irreversible, so it is the only action that asks first.
  let purgeOpen = $state(false);
  let purgeIds = $state<string[]>([]);
  function askPurge(ids: string[]) {
    purgeIds = ids;
    purgeOpen = true;
  }
  async function purge() {
    const ids = purgeIds;
    const focusAfter = neighbourOf(ids);
    const r = await runBatch('purge', ids);
    if (!r) return;
    if (openId && r.ok.includes(openId)) await closeLightbox(focusAfter);
    takeFromList(r.ok);
    toast(r.failed > 0 ? `已永久删除 ${r.ok.length} 张，${r.failed} 张失败` : `已永久删除 ${r.ok.length} 张`);
    void refreshHealth();
  }

  const pendingVisibility = new SvelteSet<string>();
  /** Optimistic: flip now, roll back when the server refuses. */
  async function setVisibility(img: Image, v: Visibility) {
    if (pendingVisibility.has(img.id)) return;
    const row = items.find((i) => i.id === img.id);
    if (!row) return;
    const before = row.visibility;
    row.visibility = v;
    pendingVisibility.add(img.id);
    try {
      const updated = await api.setVisibility(img.id, v);
      const at = items.findIndex((i) => i.id === img.id);
      if (at >= 0) items[at] = updated;
    } catch (e) {
      const target = items.find((i) => i.id === img.id);
      if (target) target.visibility = before;
      toast(errorMessage(e));
    } finally {
      pendingVisibility.delete(img.id);
    }
  }

  async function setSelectedVisibility(v: Visibility) {
    const ids = [...selected];
    const r = await runBatch('visibility', ids, v);
    if (!r) return;
    const ok = new Set(r.ok);
    for (const it of items) if (ok.has(it.id)) it.visibility = v;
    const label = v === 'public' ? '公开' : '私有';
    toast(r.failed > 0 ? `已设为${label} ${r.ok.length} 张，${r.failed} 张失败` : `已设为${label} ${r.ok.length} 张`);
  }

  async function copyLines(lines: string[]) {
    if (lines.length === 0) return;
    const r = await copyText(lines.join('\n'));
    toast(r === 'copied' ? `已复制 ${lines.length} 条链接` : '当前连接不安全，无法写入剪贴板');
  }
  function copySelectedLinks() {
    void copyLines(items.filter((i) => selected.has(i.id)).map((i) => i.links[uploads.format]));
  }

  // ---- the developing roll -------------------------------------------------------------------
  const tray = $derived(isTrash ? [] : uploads.items);
  const trayDone = $derived(tray.filter((i) => i.status === 'done'));
  const trayFailed = $derived(tray.filter((i) => i.status === 'error').length);
  const trayActive = $derived(tray.length - trayDone.length - trayFailed);
  const traySettled = $derived(tray.length > 0 && trayActive === 0);
  const traySummary = $derived(
    [
      trayActive > 0 ? `${trayActive} 张显影中` : '',
      trayDone.length > 0 ? `${trayDone.length} 张已存档` : '',
      trayFailed > 0 ? `${trayFailed} 张失败` : '',
    ]
      .filter(Boolean)
      .join(' · '),
  );
  function copyTray() {
    void copyLines(trayDone.map((i) => linkOf(i, uploads.format)).filter(Boolean));
  }

  const showEmpty = $derived(phase === 'ready' && items.length === 0 && !filtered && !isTrash && tray.length === 0);
</script>

<svelte:window onkeydown={onwindowkeydown} />

<div class="page page-wide sheet" class:has-bar={selected.size > 0}>
  <PageHeader
    title={isTrash ? '回收站' : '底片'}
    figure={stats ? formatInt(isTrash ? stats.trashed : stats.active) : undefined}
    figureUnit={stats
      ? isTrash
        ? `帧${retention === null ? '' : retention === 0 ? ' · 不保留' : ` · 保留 ${retention} 天`}`
        : `帧 · ${formatBytes(stats.bytes)}`
      : undefined}
  />

  <div class="finder" role="search">
    <div class="search">
      <TextField
        label="查找"
        hideLabel
        variant="line"
        icon="search"
        placeholder="按文件名、ID 或 SHA-256 查找"
        autocomplete="off"
        spellcheck={false}
        enterkeyhint="search"
        bind:value={qInput}
        bind:input={searchEl}
        onkeydown={(e) => {
          if (e.key === 'Enter') setQuery({ q: qInput.trim() || undefined });
        }}
      >
        {#snippet trail()}
          {#if qInput}
            <button type="button" class="clear" onclick={() => (qInput = '')} aria-label="清除查找">清除</button>
          {:else}
            <kbd title="按 / 开始查找">/</kbd>
          {/if}
        {/snippet}
      </TextField>
    </div>

    <div class="filters">
      <div class="fmts" role="group" aria-label="格式">
        {#each FORMATS as fmt (fmt)}
          <button
            type="button"
            class="fmt"
            aria-pressed={f.format === fmt}
            onclick={() => setQuery({ format: f.format === fmt ? undefined : fmt })}>{FORMAT_LABEL[fmt]}</button
          >
        {/each}
      </div>
      <Segmented
        label="可见性"
        options={VIS_OPTIONS}
        value={f.visibility}
        onchange={(v) => setQuery({ visibility: v || undefined })}
      />
      <Chip pressed={f.gps} icon="pin" onclick={() => setQuery({ gps: f.gps ? undefined : 'true' })}>含位置</Chip>
      <span class="spacer"></span>
      {#if filtered}
        <Button variant="text" size="sm" onclick={clearFilters}>清除筛选</Button>
      {/if}
      <Select
        label="排序"
        hideLabel
        quiet
        options={SORT_OPTIONS}
        value={f.sort}
        onchange={(v) => setQuery({ sort: v === 'new' ? undefined : v })}
      />
    </div>
  </div>

  <div class="sr-only" role="status" aria-live="polite">{uploads.announcement}</div>

  <div class="contact" bind:clientWidth={sheetWidth}>
    {#if tray.length > 0}
      <section class="roll tray" aria-label="冲洗盘">
        <header class="roll-head">
          <h2>
            <span class="lamp" class:warn={trayActive > 0} class:danger={trayActive === 0 && trayFailed > 0}></span>
            冲洗盘
          </h2>
          <span class="roll-sub">{traySummary}</span>
          <span class="tray-tools">
            <Segmented label="链接格式" options={LINK_FORMATS} value={uploads.format} onchange={(v: LinkFormat) => setLinkFormat(v)} />
            <Button size="sm" icon="copy" disabled={trayDone.length === 0} onclick={copyTray}>复制全部链接</Button>
            {#if traySettled}<Button size="sm" variant="text" onclick={clearFinished}>收起</Button>{/if}
          </span>
        </header>
        <FilmRoll items={tray} {cols} key={(i) => i.id}>
          {#snippet frame(item)}
            <DevelopFrame {item} format={uploads.format} />
          {/snippet}
        </FilmRoll>
      </section>
    {/if}

    {#if phase === 'loading'}
      <div class="roll" aria-busy="true" aria-label="正在载入">
        <header class="roll-head"><span class="skel-line"></span></header>
        <FilmRoll items={Array.from({ length: cols * 2 }, (_, i) => i)} {cols} key={(i) => i}>
          {#snippet frame()}
            <span class="skel-frame"></span>
          {/snippet}
        </FilmRoll>
      </div>
    {:else if phase === 'error'}
      <div class="message">
        <p class="msg-title display">底片没能取出来</p>
        <p class="muted">{errorText}</p>
        <Button variant="primary" onclick={() => void reload()}>重试</Button>
      </div>
    {:else if showEmpty}
      <div class="ready">
        <p class="ready-title display">暗房已就绪</p>
        <p class="ready-text">把照片拖进窗口、粘贴截图，或者</p>
        <Button variant="primary" icon="plus" onclick={openPicker}>选择照片</Button>
        <div class="blank film" aria-hidden="true">
          {#each { length: Math.min(cols, 6) } as _, i (i)}
            <button type="button" class="blank-frame" tabindex="-1" onclick={openPicker}>
              <span class="edge">{i + 1}<span class="tri">▸</span></span>
            </button>
          {/each}
        </div>
      </div>
    {:else if items.length === 0}
      <div class="message">
        {#if filtered}
          <p class="msg-title display">没有符合条件的底片</p>
          <Button onclick={clearFilters}>清除筛选</Button>
        {:else if isTrash}
          <p class="msg-title display">回收站是空的</p>
          <p class="muted">移到回收站的照片会在这里保留一段时间，然后自动清除。</p>
        {/if}
      </div>
    {:else}
      {#each rolls as roll (roll.key)}
        <section class="roll" aria-label="{roll.title} {roll.sub}">
          <header class="roll-head">
            <h2>{roll.title}</h2>
            <span class="roll-sub">{roll.sub}</span>
            <span class="roll-count edge">{roll.items.length} 帧</span>
          </header>
          <FilmRoll items={roll.items} {cols} key={(i) => i.id}>
            {#snippet frame(img, n)}
              <Frame
                image={img}
                number={n + 1}
                selected={selected.has(img.id)}
                selecting={selected.size > 0}
                fresh={freshIds.has(img.id)}
                note={isTrash ? daysLeft(img) : null}
                muted={isTrash}
                linkFormat={uploads.format}
                morph={morphId === img.id}
                onactivate={(e) => activate(e, img)}
                ontoggle={(e) => toggle(e, img)}
              />
            {/snippet}
          </FilmRoll>
        </section>
      {/each}

      <div class="foot" bind:this={sentinel}>
        {#if more === 'loading'}
          <span class="edge muted">正在取出更多底片…</span>
        {:else if cursor}
          <Button onclick={() => void loadMore()}>{more === 'error' ? '加载失败，重试' : '加载更多'}</Button>
        {:else}
          <span class="end edge">— 全部 {formatInt(items.length)} 帧 —</span>
        {/if}
      </div>
    {/if}
  </div>
</div>

{#if selected.size > 0}
  <SelectionBar count={selected.size} onclear={clearSelection}>
    {#if isTrash}
      <Button variant="inverse" size="sm" icon="restore" onclick={() => void restore([...selected])}>恢复</Button>
      <Button variant="inverse" size="sm" onclick={() => askPurge([...selected])}>永久删除</Button>
    {:else}
      <Button variant="inverse" size="sm" icon="copy" onclick={copySelectedLinks}>复制链接</Button>
      <Button variant="inverse" size="sm" onclick={() => void setSelectedVisibility('public')}>设为公开</Button>
      <Button variant="inverse" size="sm" onclick={() => void setSelectedVisibility('private')}>设为私有</Button>
      <Button variant="inverse" size="sm" icon="trash" onclick={() => void moveToTrash([...selected])}>移到回收站</Button>
    {/if}
  </SelectionBar>
{/if}

<Lightbox
  image={current}
  {mode}
  {position}
  {canPrev}
  {canNext}
  visibilityBusy={current ? pendingVisibility.has(current.id) : false}
  linkFormat={uploads.format}
  note={current && isTrash ? daysLeft(current) : null}
  onformat={setLinkFormat}
  onprev={() => void step(-1)}
  onnext={() => void step(1)}
  onclose={() => void closeLightbox()}
  ontrash={() => current && void moveToTrash([current.id])}
  onrestore={() => current && void restore([current.id])}
  onpurge={() => current && askPurge([current.id])}
  onvisibility={(v) => current && void setVisibility(current, v)}
/>

<ConfirmDialog
  bind:open={purgeOpen}
  title={purgeIds.length > 1 ? `永久删除 ${purgeIds.length} 张照片？` : '永久删除这张照片？'}
  message="永久删除后无法恢复。已被浏览器或 CDN 缓存的副本可能仍可访问一段时间。"
  confirmLabel="永久删除"
  danger
  onconfirm={purge}
/>

<style>
  .sheet.has-bar {
    padding-bottom: calc(var(--s7) + 80px + var(--tabbar-h));
  }

  /* ---- finder: the sleeve label ---- */
  .finder {
    display: flex;
    flex-direction: column;
    gap: var(--s3);
    margin-bottom: var(--s6);
  }
  .search {
    max-width: 640px;
  }
  .search kbd {
    margin-right: 2px;
  }
  .clear {
    color: var(--ink-3);
    font-size: var(--t-12);
    font-weight: 600;
  }
  .clear:hover {
    color: var(--ink);
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--s2);
  }
  .spacer {
    flex: 1;
  }
  .fmts {
    display: inline-flex;
    border: 1px solid var(--line-strong);
    border-radius: var(--r-ctl);
    overflow: hidden;
  }
  .fmt {
    height: calc(var(--control-h) - 4px);
    padding: 0 10px;
    font-family: var(--font-mono);
    font-stretch: 87.5%;
    font-size: 11.5px;
    font-weight: 600;
    letter-spacing: 0.04em;
    color: var(--ink-2);
    transition:
      background-color var(--dur) var(--ease),
      color var(--dur) var(--ease);
  }
  .fmt + .fmt {
    border-left: 1px solid var(--line-strong);
  }
  .fmt:hover {
    background: var(--surface);
    color: var(--ink);
  }
  .fmt[aria-pressed='true'] {
    background: var(--ink);
    color: var(--bg);
  }
  .fmt:focus-visible {
    outline-offset: -3px;
    outline-color: var(--mark);
  }

  /* ---- rolls ---- */
  .contact {
    display: flex;
    flex-direction: column;
    gap: 44px;
  }
  .roll-head {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 4px 12px;
    margin-bottom: 12px;
  }
  .roll-head h2 {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    font-size: 17px;
    font-weight: 700;
  }
  .roll-sub {
    color: var(--ink-3);
    font-size: var(--t-13);
  }
  .roll-count {
    margin-left: auto;
    color: var(--ink-3);
  }
  .tray-tools {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--s2);
    margin-left: auto;
    align-self: center;
  }
  .tray {
    padding-bottom: 36px;
    border-bottom: 1px dashed var(--line-strong);
  }

  .skel-line {
    display: block;
    width: 120px;
    height: 16px;
    background: var(--surface-2);
  }
  .skel-frame {
    display: block;
    aspect-ratio: 3 / 2;
    background: var(--film-2);
    animation: skel 1.4s ease-in-out infinite;
  }
  @keyframes skel {
    50% {
      opacity: 0.55;
    }
  }

  .message {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--s3);
    padding: var(--s6) 0;
  }
  .msg-title {
    font-size: var(--t-20);
  }

  /* ---- the empty darkroom ---- */
  .ready {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--s3);
    padding: var(--s6) 0 var(--s4);
    text-align: center;
  }
  .ready-title {
    font-size: clamp(30px, 4.4vw, 44px);
  }
  .ready-text {
    color: var(--ink-2);
    font-size: var(--t-16);
  }
  .blank {
    display: flex;
    gap: 6px;
    width: 100%;
    max-width: 1040px;
    margin-top: var(--s6);
  }
  .blank-frame {
    position: relative;
    flex: 1;
    aspect-ratio: 3 / 2;
    background: var(--film-2);
    outline: 1px dashed rgb(167 156 139 / 0.28);
    outline-offset: -8px;
    cursor: pointer;
    transition: outline-color var(--dur) var(--ease);
  }
  .blank-frame:hover {
    outline-color: var(--edge);
  }
  .blank-frame .edge {
    position: absolute;
    top: -16px;
    left: 1px;
    color: var(--edge);
    font-weight: 650;
  }
  .tri {
    font-size: 8px;
  }

  .foot {
    display: flex;
    justify-content: center;
    padding: var(--s2) 0 0;
  }
  .end {
    color: var(--ink-3);
  }

  @media (max-width: 760px) {
    .filters {
      flex-wrap: nowrap;
      overflow-x: auto;
      margin: 0 calc(-1 * var(--gutter));
      padding: 2px var(--gutter);
      scrollbar-width: none;
    }
    .filters::-webkit-scrollbar {
      display: none;
    }
    .filters > :global(*) {
      flex: none;
    }
    .spacer {
      display: none;
    }
    .contact {
      gap: 32px;
    }
    .tray-tools {
      margin-left: 0;
    }
    .blank-frame:nth-child(n + 4) {
      display: none;
    }
  }

  :global(::view-transition-group(pb-print)) {
    animation-duration: 320ms;
    animation-timing-function: cubic-bezier(0.16, 1, 0.3, 1);
  }
  :global(::view-transition-old(root)),
  :global(::view-transition-new(root)) {
    animation-duration: 220ms;
  }
</style>
