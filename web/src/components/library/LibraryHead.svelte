<script lang="ts">
  import { filterCount, type Library } from '../../lib/library.svelte';
  import { phone, tablet } from '../../lib/media.svelte';
  import { uploadDialog } from '../../lib/upload.svelte';
  import Icon from '../Icon.svelte';
  import PanelHeader from '../PanelHeader.svelte';
  import BulkBar from './BulkBar.svelte';
  import FilterPopover from './FilterPopover.svelte';
  import SearchBox from './SearchBox.svelte';
  import SortMenu from './SortMenu.svelte';
  import ViewToggle from './ViewToggle.svelte';

  interface Props {
    lib: Library;
    /** Width the page content gets (the detail column takes some of it), not the viewport. */
    width: number;
    searchEl?: HTMLInputElement | null;
    searchOpen?: boolean;
    onadjust: () => void;
  }
  let { lib, width, searchEl = $bindable(null), searchOpen = $bindable(false), onadjust }: Props = $props();

  const trash = $derived(lib.mode === 'trash');
  const title = $derived(trash ? '回收站' : '全部图片');
  const n = $derived(filterCount(lib.filters));
  // Nothing to search, filter or upload into yet: the empty state carries the one primary action.
  const bare = $derived(lib.phase === 'ready' && lib.items.length === 0 && !lib.filtered);
  const hasTools = $derived(!bare);
  const trashCount = $derived(lib.facets?.total ?? lib.items.length);
  const collapsed = $derived(tablet.matches || width < (trash ? 660 : 640));
</script>

{#if phone.matches}
  {#if searchOpen || lib.filters.q}
    <header class="page-head">
      <SearchBox {lib} bind:el={searchEl} bind:open={searchOpen} collapsed keepOpen />
      <button
        type="button"
        class="btn btn-ghost"
        onclick={() => {
          lib.setFilters({ q: '' });
          searchOpen = false;
        }}
      >
        取消
      </button>
    </header>
  {:else}
    <PanelHeader {title}>
      {#if hasTools}
        <SearchBox {lib} bind:el={searchEl} bind:open={searchOpen} collapsed keepOpen />
        <button
          type="button"
          class="btn btn-ghost btn-icon"
          class:active-soft={lib.selecting}
          aria-label="选择"
          aria-pressed={lib.selecting}
          onclick={() => (lib.selecting ? lib.clearSel() : (lib.selectMode = true))}
        >
          <Icon name="select" />
        </button>
        <button
          type="button"
          class="btn btn-ghost btn-icon"
          class:active-soft={n > 0}
          aria-label={n > 0 ? `筛选（${n}）` : '筛选'}
          onclick={onadjust}
        >
          <Icon name="more" />
        </button>
      {/if}
    </PanelHeader>
  {/if}
{:else}
  <PanelHeader {title}>
    {#if lib.selecting}
      <BulkBar {lib} compact={width < 620} />
    {:else}
      {#if hasTools}
        <SearchBox {lib} bind:el={searchEl} bind:open={searchOpen} {collapsed} />
        <FilterPopover {lib} />
        <SortMenu {lib} />
        <ViewToggle {lib} />
      {/if}
      {#if trash}
        {#if trashCount > 0}
          <button type="button" class="btn btn-danger-ghost" onclick={() => lib.emptyTrash()}>清空回收站</button>
        {/if}
      {:else if hasTools}
        <button type="button" class="btn btn-primary" title="上传 (U)" onclick={() => (uploadDialog.open = true)}>
          <Icon name="upload" />
          上传
        </button>
      {/if}
    {/if}
  </PanelHeader>
{/if}
