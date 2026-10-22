<script lang="ts">
  import { filterCount, type Library } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import Icon from '../Icon.svelte';
  import BulkBar from './BulkBar.svelte';
  import FilterChips from './FilterChips.svelte';
  import FilterMenus from './FilterMenus.svelte';
  import FilterPopover from './FilterPopover.svelte';
  import SearchBox from './SearchBox.svelte';
  import SortMenu from './SortMenu.svelte';
  import ViewToggle from './ViewToggle.svelte';

  /** `width` is the space the library page content gets, not the viewport. */
  let {
    lib,
    width,
    searchEl = $bindable(null),
    onopenfilters,
  }: { lib: Library; width: number; searchEl?: HTMLInputElement | null; onopenfilters: () => void } = $props();

  const n = $derived(filterCount(lib.filters));
  const bare = $derived(lib.phase === 'ready' && lib.items.length === 0 && !lib.filtered);
  const roomy = $derived(width >= 940);
</script>

{#if lib.selecting && !phone.matches}
  <!-- On desktop the bulk bar takes the toolbar's place. -->
  <BulkBar {lib} />
{:else if !bare}
  {#if phone.matches}
    <div class="toolbar">
      <SearchBox {lib} bind:el={searchEl} placeholder="搜索图片" />
      <button
        type="button"
        class="btn btn-secondary filter-btn btn-icon-sm"
        class:active={n > 0}
        aria-label={n > 0 ? `筛选（${n}）` : '筛选'}
        onclick={onopenfilters}
      >
        <Icon name="filter" />
        {#if n > 0}<span class="filter-n">{n}</span>{/if}
      </button>
      <button
        type="button"
        class="btn btn-secondary btn-icon"
        class:active-soft={lib.selecting}
        aria-label="选择"
        aria-pressed={lib.selecting}
        onclick={() => (lib.selecting ? lib.clearSel() : (lib.selectMode = true))}
      >
        <Icon name="select" />
      </button>
      <ViewToggle {lib} />
    </div>
  {:else}
    <div class="toolbar">
      <SearchBox {lib} bind:el={searchEl} />
      {#if roomy}<FilterMenus {lib} />{:else}<FilterPopover {lib} />{/if}
      <span class="grow"></span>
      <SortMenu {lib} iconOnly={width < 720} />
      <ViewToggle {lib} />
    </div>
  {/if}
  <FilterChips {lib} />
{/if}

{#if lib.selecting && phone.matches}
  <!-- On phones the bar floats above the tab bar while the toolbar stays. -->
  <BulkBar {lib} />
{/if}
