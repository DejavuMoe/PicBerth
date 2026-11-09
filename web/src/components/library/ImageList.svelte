<script lang="ts">
  import type { Library } from '../../lib/library.svelte';
  import { openPicker } from '../../lib/picker';
  import ImageGrid from './ImageGrid.svelte';
  import ImageTable from './ImageTable.svelte';

  let { lib }: { lib: Library } = $props();

  const trash = $derived(lib.mode === 'trash');
  const empty = $derived(lib.phase === 'ready' && lib.items.length === 0);
  const failure = $derived(lib.phase === 'error' ? (lib.errorKind === 'folders' ? '文件夹加载失败' : '图片加载失败') : '');
</script>

{#if lib.phase === 'loading'}
  <div aria-busy="true" aria-label="加载中">
    {#each Array.from({ length: 8 }, (_, i) => i) as i (i)}
      <div
        class="row"
        style="height: 48px; padding: 0 20px; gap: 16px; border-bottom: var(--bw) solid var(--hairline)"
      >
        <div class="skeleton" style="width: 32px; height: 32px; margin-left: 36px"></div>
        <div class="skeleton" style="width: {30 + ((i * 17) % 30)}%; height: 10px"></div>
        <span class="grow"></span>
        <div class="skeleton" style="width: 56px; height: 10px"></div>
        <div class="skeleton" style="width: 72px; height: 10px; margin-right: 96px"></div>
      </div>
    {/each}
  </div>
{:else if lib.phase === 'error'}
  <div class="state" role="alert">
    <span class="t">{failure}</span>
    <span class="d">{lib.errorText}</span>
    <button
      type="button"
      class="btn btn-secondary"
      onclick={() => (lib.errorKind === 'folders' ? lib.loadFolders() : lib.reload())}>重试</button
    >
  </div>
{:else if empty && lib.filtered}
  <div class="state">
    <span class="t">没有匹配的图片</span>
    <button type="button" class="btn btn-secondary" onclick={() => lib.clearFilters(false)}>清除筛选</button>
  </div>
{:else if empty}
  {#if trash}
    <div class="state">
      <span class="t">回收站是空的</span>
    </div>
  {:else}
    <div class="state">
      <span class="t">{lib.hasAnyImages ? '这里没有图片' : '还没有图片'}</span>
      <span class="d">拖入、粘贴或选择文件上传</span>
      <button type="button" class="btn btn-secondary" onclick={openPicker}>选择文件</button>
    </div>
  {/if}
{:else if lib.view === 'grid'}
  <ImageGrid {lib} />
{:else}
  <ImageTable {lib} />
{/if}
