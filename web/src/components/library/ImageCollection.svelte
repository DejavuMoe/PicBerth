<script lang="ts">
  import { tick } from 'svelte';
  import { formatBytes } from '../../lib/format';
  import type { Library } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import { openPicker } from '../../lib/picker';
  import Icon from '../Icon.svelte';
  import ImageGrid from './ImageGrid.svelte';
  import ImageRows from './ImageRows.svelte';
  import ImageTable from './ImageTable.svelte';

  let { lib }: { lib: Library } = $props();

  const grid = $derived(lib.view === 'grid');
  const total = $derived(lib.facets?.total);
  const bytes = $derived(lib.items.reduce((s, i) => s + i.size, 0));
  const empty = $derived(lib.phase === 'ready' && lib.items.length === 0);

  // Infinite scroll: load the next page when the footer comes near the viewport.
  let footer = $state<HTMLElement | null>(null);
  $effect(() => {
    if (!footer) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) void lib.loadMore();
      },
      { rootMargin: '800px 0px' },
    );
    io.observe(footer);
    return () => io.disconnect();
  });
  // The observer only reports changes: keep loading while the footer is still near the viewport.
  $effect(() => {
    lib.items.length;
    lib.cursor;
    void tick().then(() => {
      if (footer && lib.cursor && lib.more === 'idle' && footer.getBoundingClientRect().top < window.innerHeight + 800) {
        void lib.loadMore();
      }
    });
  });
</script>

{#if lib.phase === 'loading'}
  {#if grid}
    <div class="grid" aria-busy="true" aria-label="正在加载">
      {#each { length: 12 } as _, i (i)}
        <div class="tile" style="cursor: default">
          <div class="skeleton" style="aspect-ratio: 4 / 3; border-radius: 0"></div>
          <div class="tile-meta">
            <div class="skeleton" style="height: 12px; width: 70%; margin: 3px 0"></div>
            <div class="skeleton" style="height: 10px; width: 45%; margin: 3px 0"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="rows" aria-busy="true" aria-label="正在加载">
      {#each { length: 8 } as _, i (i)}
        <div class="row">
          <div class="skeleton" style="width: 40px; height: 40px; flex: none"></div>
          <div style="flex: 1; display: flex; flex-direction: column; gap: 6px">
            <div class="skeleton" style="height: 12px; width: {40 + ((i * 17) % 35)}%"></div>
            <div class="skeleton" style="height: 10px; width: 22%"></div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
{:else if lib.phase === 'error'}
  <div class="alert danger" role="alert">
    <Icon name="alert" />
    <div class="grow">
      <strong>无法加载图片列表</strong>
      <div>{lib.errorText}</div>
    </div>
    <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.reload()}>
      <Icon name="refresh" />
      重试
    </button>
  </div>
{:else if empty && !lib.filtered}
  {#if lib.mode === 'trash'}
    <div class="empty">
      <Icon name="trash" size={28} />
      <strong>回收站是空的</strong>
      {#if lib.retention !== null}<p>移到回收站的图片会保留 {lib.retention} 天，之后自动永久删除。</p>{/if}
    </div>
  {:else}
    <div class="empty">
      <Icon name="image" size={28} />
      <strong>还没有图片</strong>
      <p>把图片拖到此页面、直接粘贴，或选择文件上传。</p>
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
    <Icon name="search" size={28} />
    <strong>没有符合条件的图片</strong>
    <p>修改搜索词或清除筛选条件后再试。</p>
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
    <ImageTable {lib} />
  {/if}
  <div class="pager" bind:this={footer}>
    {#if lib.cursor}
      <span>{lib.items.length}{total !== undefined && !lib.filtered ? ` / ${total}` : ''} 张 · {formatBytes(bytes)}</span>
      {#if lib.more === 'loading'}
        <span>加载中…</span>
      {:else}
        <button type="button" class="btn btn-secondary btn-sm" onclick={() => lib.loadMore()}>
          {lib.more === 'error' ? '加载失败，重试' : '加载更多'}
        </button>
      {/if}
    {:else}
      <span>
        {total !== undefined && lib.items.length < total ? `${lib.items.length} / ${total} 张` : `共 ${lib.items.length} 张`} · {formatBytes(bytes)}
      </span>
      <span>已全部加载</span>
    {/if}
  </div>
{/if}
