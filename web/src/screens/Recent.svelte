<script lang="ts">
  import { onDestroy, onMount, tick } from 'svelte';
  import Drawer from '../components/Drawer.svelte';
  import Icon from '../components/Icon.svelte';
  import Menu from '../components/Menu.svelte';
  import Seg from '../components/Seg.svelte';
  import Sheet from '../components/Sheet.svelte';
  import UploadList from '../components/UploadList.svelte';
  import DetailPanel from '../components/library/DetailPanel.svelte';
  import FolderTargets from '../components/library/FolderTargets.svelte';
  import PurgeConfirm from '../components/library/PurgeConfirm.svelte';
  import RowMenu from '../components/library/RowMenu.svelte';
  import Viewer from '../components/library/Viewer.svelte';
  import { refreshFacets } from '../lib/facets.svelte';
  import { formatBytes, formatRelative, startOfDay } from '../lib/format';
  import { sourceLabel, thumbSrc } from '../lib/image';
  import { copyLink, copyLinks, Library } from '../lib/library.svelte';
  import { phone, wide } from '../lib/media.svelte';
  import { isTypingTarget } from '../lib/modal';
  import { openPalette } from '../lib/palette.svelte';
  import { openPicker } from '../lib/picker';
  import { tree } from '../lib/tree.svelte';
  import type { Image, LinkFormat } from '../lib/types';
  import {
    clearFinished,
    LINK_FORMATS,
    onUploaded,
    setLinkFormat,
    setUploadTarget,
    uploads,
    type UploadItem,
  } from '../lib/upload.svelte';

  // Every active image, newest first, grouped by day. Uploads started here go to `dest` (the root by default).
  const lib = new Library('recent');
  let dest = $state<string | null>(null);
  let scroller = $state<HTMLElement | null>(null);

  onMount(() => {
    void tree.ensure();
    void refreshFacets();
    void lib.reload();
    void lib.loadRetention();
    return onUploaded((img) => {
      void refreshFacets();
      void lib.refreshFolders();
      lib.addUploaded(img);
    });
  });
  onDestroy(() => {
    lib.destroy();
    setUploadTarget(null);
  });

  // Drops and pastes on this screen use the same destination as the upload button.
  $effect(() => {
    setUploadTarget(dest === null ? null : { id: dest, label: tree.label(dest) });
  });
  // A destination folder that disappears falls back to the root.
  $effect(() => {
    if (dest !== null && tree.phase === 'ready' && !tree.folders.some((f) => f.id === dest)) dest = null;
  });

  const DAY = 86_400_000;
  function dayLabel(t: number, now: number): string {
    const start = startOfDay(now);
    if (t >= start) return '今天';
    if (t >= start - DAY) return '昨天';
    const d = new Date(t);
    const md = `${d.getMonth() + 1} 月 ${d.getDate()} 日`;
    return d.getFullYear() === new Date(now).getFullYear() ? md : `${d.getFullYear()} 年 ${md}`;
  }

  const groups = $derived.by(() => {
    const now = Date.now();
    const out: { key: string; label: string; items: Image[] }[] = [];
    for (const im of lib.items) {
      const t = new Date(im.createdAt).getTime();
      const key = String(startOfDay(t));
      let g = out[out.length - 1];
      if (!g || g.key !== key) out.push((g = { key, label: dayLabel(t, now), items: [] }));
      g.items.push(im);
    }
    return out;
  });

  const queue = $derived(uploads.items);
  const doneItems = $derived(queue.filter((u) => u.status === 'done' && u.image));
  const busy = $derived(queue.some((u) => u.status === 'queued' || u.status === 'uploading' || u.status === 'processing'));
  const queueTitle = $derived(busy ? '正在上传' : doneItems.length > 0 ? '上传完成' : '上传失败');

  /** Where an image is: the folder path without the root, or 图片 for the root itself. */
  function place(id: string | null): string {
    return id === null ? tree.label(null) : tree.names(id).join(' / ') || tree.label(id);
  }

  const formatOptions = LINK_FORMATS.map((f) => ({ key: f.value, label: f.label }));
  const variant = $derived(wide.matches && !phone.matches ? 'docked' : phone.matches ? 'sheet' : 'drawer');
  const openImg = $derived(lib.active);
  const empty = $derived(lib.phase === 'ready' && lib.items.length === 0 && queue.length === 0);

  function openUpload(u: UploadItem) {
    if (!u.image) return;
    if (lib.items.some((i) => i.id === u.image!.id)) lib.open(u.image.id);
    else lib.pendingOpen = u.image.id;
  }

  function maybeLoadMore() {
    if (!scroller || !lib.cursor || lib.more === 'loading' || lib.phase !== 'ready') return;
    if (scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 600) void lib.loadMore();
  }
  $effect(() => {
    lib.items.length;
    lib.cursor;
    lib.phase;
    void tick().then(() => maybeLoadMore());
  });

  function onkeydown(e: KeyboardEvent) {
    if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return;
    const typing = isTypingTarget(e.target) || document.querySelector('.scrim') !== null;
    if (typing) return;
    if (e.key === 'Escape') {
      if (lib.active && variant === 'docked') lib.close();
      return;
    }
    const overlays = document.querySelectorAll('[data-overlay]').length;
    if (overlays !== (lib.active && variant !== 'docked' ? 1 : 0) + (lib.viewing ? 1 : 0)) return;
    if (e.key === 'j' || e.key === 'k') {
      e.preventDefault();
      if (!lib.active) {
        if (lib.items[0]) lib.open(lib.items[0].id);
      } else if (e.key === 'j') void lib.step(1);
      else if (lib.prev) lib.open(lib.prev.id);
      return;
    }
    if (!lib.active || (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight')) return;
    e.preventDefault();
    if (e.key === 'ArrowLeft') {
      if (lib.prev) lib.open(lib.prev.id);
    } else void lib.step(1);
  }
</script>

<svelte:window {onkeydown} />

<div class="panel-main">
  <div class="head">
    <div class="head-title">
      <h1>最近</h1>
    </div>
    {#if !phone.matches && !empty}
      <span class="muted">复制为</span>
      <Seg label="复制为" options={formatOptions} value={uploads.format} onchange={(f: LinkFormat) => setLinkFormat(f)} />
    {/if}
    {#if phone.matches}
      <button type="button" class="btn btn-ghost btn-icon" aria-label="搜索" title="搜索" onclick={openPalette}>
        <Icon name="search" />
      </button>
    {/if}
    {#if dest !== null && !phone.matches}
      <span class="rc-dest muted"><Icon name="folder" size="s" /><span class="ellipsis">{place(dest)}</span></span>
    {/if}
    <div class="split">
      <button type="button" class="btn btn-primary" aria-label="上传" title="上传 (U)" onclick={() => openPicker()}>
        <Icon name="upload" />
        {#if !phone.matches}上传{/if}
      </button>
      {#if !phone.matches}
        <Menu align="right" width={220} label="位置">
          {#snippet trigger(open, toggle)}
            <button type="button" class="btn btn-primary split-caret" aria-label="位置" title="位置" aria-haspopup="menu" aria-expanded={open} onclick={toggle}>
              <Icon name="down" size="s" />
            </button>
          {/snippet}
          {#snippet children(close)}
            <div class="menu-group">位置</div>
            <FolderTargets currentId={dest} onpick={(id) => (dest = id)} {close} />
          {/snippet}
        </Menu>
      {/if}
    </div>
  </div>
  {#if phone.matches && !empty}
    <div class="subbar">
      <Seg block label="复制为" options={formatOptions} value={uploads.format} onchange={(f: LinkFormat) => setLinkFormat(f)} />
    </div>
  {/if}

  <div class="scroll rc-scroll" bind:this={scroller} onscroll={maybeLoadMore}>
    {#if queue.length > 0}
      <section class="rc-group rc-queue" aria-label={queueTitle}>
        <h2>
          {queueTitle}
          <span class="n num">{doneItems.length} / {queue.length}</span>
          <span class="grow"></span>
          {#if doneItems.length > 1}
            <button type="button" class="btn btn-ghost btn-sm" onclick={() => void copyLinks(doneItems.map((u) => u.image!))}>
              <Icon name="copy" size="s" />
              复制全部链接
            </button>
          {/if}
          {#if !busy}
            <button type="button" class="btn btn-ghost btn-sm btn-icon" aria-label="关闭" title="关闭" onclick={() => clearFinished()}>
              <Icon name="x" />
            </button>
          {/if}
        </h2>
        <div class="rc-queue-list"><UploadList open={openUpload} /></div>
      </section>
    {/if}

    {#if lib.phase === 'loading'}
      <div aria-busy="true" aria-label="加载中">
        {#each [220, 160, 260, 190, 240, 170] as w, i (i)}
          <div class="rc-row rc-skel">
            <span class="skeleton rc-thumb"></span>
            <span class="skeleton" style:width={`${w}px`} style:height="12px"></span>
          </div>
        {/each}
      </div>
    {:else if lib.phase === 'error'}
      <div class="state" role="alert">
        <span class="t">加载失败</span>
        <span class="d">{lib.errorText}</span>
        <button type="button" class="btn btn-secondary" onclick={() => void lib.reload()}>重试</button>
      </div>
    {:else if empty}
      <div class="state">
        <span class="t">还没有图片</span>
        <span class="d">拖入、粘贴或选择文件上传</span>
      </div>
    {:else}
      {#each groups as g (g.key)}
        <section class="rc-group" aria-label={g.label}>
          <h2>
            {g.label}
            <span class="n num">{g.items.length}</span>
            <span class="grow"></span>
            {#if !phone.matches}
              <button type="button" class="btn btn-ghost btn-sm rc-copyall" onclick={() => void copyLinks(g.items)}>
                <Icon name="copy" size="s" />
                复制全部链接
              </button>
            {/if}
          </h2>
          {#each g.items as im (im.id)}
            <div class="rc-row" class:is-open={lib.activeId === im.id}>
              <button type="button" class="rc-thumb-btn" aria-label={im.filename} onclick={() => lib.open(im.id)} ondblclick={() => lib.openViewer(im.id)}>
                {#if im.hasThumb && !im.broken}
                  <img class="rc-thumb" src={thumbSrc(im)} alt="" loading="lazy" />
                {:else}
                  <span class="rc-thumb"></span>
                {/if}
              </button>
              <button type="button" class="rc-name" tabindex="-1" onclick={() => lib.open(im.id)}>
                <span class="ellipsis medium" title={im.filename}>{im.filename}</span>
                <span class="rc-sub">
                  <span class="ellipsis">{place(im.folderId ?? null)}</span>
                  {#if phone.matches}
                    <span>{formatRelative(im.createdAt)}</span>
                  {:else}
                    <span class="ellipsis">{sourceLabel(im.source)}</span>
                  {/if}
                </span>
              </button>
              {#if !phone.matches}
                <span class="rc-meta num">
                  <span class="rc-dims">{im.width} × {im.height}</span>
                  <span>{formatBytes(im.size)}</span>
                  <span>{formatRelative(im.createdAt)}</span>
                </span>
              {/if}
              <span class="rc-act">
                <button type="button" class="btn btn-ghost btn-sm" class:btn-icon={phone.matches} aria-label={`复制 ${im.filename} 的链接`} title="复制" onclick={() => void copyLink(im)}>
                  <Icon name="copy" size="s" />
                  {#if !phone.matches}复制{/if}
                </button>
                {#if !phone.matches}<RowMenu {lib} image={im} />{/if}
              </span>
            </div>
          {/each}
        </section>
      {/each}
      {#if lib.more === 'error'}
        <div class="loadmore">
          <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.loadMore()}>加载失败，重试</button>
        </div>
      {/if}
    {/if}
  </div>
</div>

{#if openImg && variant === 'docked'}
  <DetailPanel {lib} image={openImg} variant="docked" onclose={() => lib.close()} />
{:else if openImg && variant === 'sheet'}
  <Sheet label={openImg.filename} onclose={() => lib.close()}>
    {#snippet children(close)}
      <DetailPanel {lib} image={openImg} variant="sheet" onclose={close} />
    {/snippet}
  </Sheet>
{:else if openImg}
  <Drawer label={openImg.filename} onclose={() => lib.close()}>
    {#snippet children(close)}
      <DetailPanel {lib} image={openImg} variant="drawer" onclose={close} />
    {/snippet}
  </Drawer>
{/if}

{#if openImg && lib.viewing}
  <Viewer {lib} image={openImg} />
{/if}

{#if lib.purgeIds}
  <PurgeConfirm {lib} />
{/if}
