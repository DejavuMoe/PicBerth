<script lang="ts">
  import { tick } from 'svelte';
  import type { Library } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import { openPicker } from '../../lib/picker';
  import Icon from '../Icon.svelte';
  import ImageGrid from './ImageGrid.svelte';
  import ImageList from './ImageList.svelte';
  import ImageRows from './ImageRows.svelte';

  let { lib }: { lib: Library } = $props();

  const grid = $derived(lib.view === 'grid');
  const empty = $derived(lib.phase === 'ready' && lib.items.length === 0);

  /** The element that scrolls the list: the page body on desktop, the document on phones. */
  function scroller(el: HTMLElement): HTMLElement | null {
    const body = el.closest<HTMLElement>('.page-body');
    return body && /auto|scroll/.test(getComputedStyle(body).overflowY) ? body : null;
  }

  // Infinite scroll: load the next page when the footer comes near the visible area.
  let footer = $state<HTMLElement | null>(null);
  $effect(() => {
    if (!footer) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) void lib.loadMore();
      },
      { root: scroller(footer), rootMargin: '800px 0px' },
    );
    io.observe(footer);
    return () => io.disconnect();
  });
  // The observer only reports changes: keep loading while the footer is still near the visible area.
  $effect(() => {
    lib.items.length;
    lib.cursor;
    void tick().then(() => {
      if (!footer || !lib.cursor || lib.more !== 'idle') return;
      const root = scroller(footer);
      const bottom = root ? root.getBoundingClientRect().bottom : window.innerHeight;
      if (footer.getBoundingClientRect().top < bottom + 800) void lib.loadMore();
    });
  });
</script>

{#if lib.phase === 'loading'}
  {#if grid}
    <div class="grid" aria-busy="true" aria-label="正在加载">
      {#each { length: 12 } as _, i (i)}
        <div class="tile is-skeleton">
          <div class="skeleton" style="aspect-ratio: 4 / 3; border-radius: 0"></div>
          <div class="skeleton" style="height: 12px; width: 70%; margin: 8px"></div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="rows rows-skeleton" class:desk={!phone.matches} aria-busy="true" aria-label="正在加载">
      {#each { length: 8 } as _, i (i)}
        <div class="row">
          <div class="skeleton" style="width: {phone.matches ? 36 : 28}px; height: {phone.matches ? 36 : 28}px; flex: none"></div>
          <div class="skeleton" style="height: 12px; width: {30 + ((i * 17) % 35)}%"></div>
        </div>
      {/each}
    </div>
  {/if}
{:else if lib.phase === 'error'}
  <div class="alert danger" role="alert" style="margin: 16px">
    <Icon name="alert" />
    <span class="grow">{lib.errorText}</span>
    <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.reload()}>
      <Icon name="refresh" />
      重试
    </button>
  </div>
{:else if empty && !lib.filtered}
  {#if lib.mode === 'trash'}
    <div class="empty">
      <Icon name="trash" size={32} />
      <strong>回收站是空的</strong>
    </div>
  {:else}
    <div class="empty">
      <Icon name="upload" size={32} />
      <strong>还没有图片</strong>
      <div class="actions">
        <button type="button" class="btn btn-primary" onclick={openPicker}>
          <Icon name="upload" />
          上传图片
        </button>
        <a class="btn btn-secondary" href="/tokens">
          <Icon name="key" />
          新建令牌
        </a>
      </div>
    </div>
  {/if}
{:else if empty}
  <div class="empty">
    <Icon name="search" size={32} />
    <strong>没有符合条件的图片</strong>
    <div class="actions">
      <button type="button" class="btn btn-secondary" onclick={() => lib.clearFilters(false)}>清除搜索和筛选</button>
    </div>
  </div>
{:else}
  {#if grid}
    <ImageGrid {lib} />
  {:else if phone.matches}
    <ImageRows {lib} />
  {:else}
    <ImageList {lib} />
  {/if}
  <!-- Sentinel for infinite scroll; the button appears only when auto-loading failed or is unavailable. -->
  <div class="pager" bind:this={footer}>
    {#if lib.cursor}
      {#if lib.more === 'loading'}
        <span>加载中…</span>
      {:else if lib.more === 'error' || typeof IntersectionObserver === 'undefined'}
        <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.loadMore()}>
          {lib.more === 'error' ? '加载失败，重试' : '加载更多'}
        </button>
      {/if}
    {/if}
  </div>
{/if}