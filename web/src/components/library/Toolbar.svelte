<script lang="ts">
  import { filterCount, type Library } from '../../lib/library.svelte';
  import { phone } from '../../lib/media.svelte';
  import Icon from '../Icon.svelte';
  import BulkBar from './BulkBar.svelte';
  import FilterChips from './FilterChips.svelte';
  import FilterMenus from './FilterMenus.svelte';
  import SearchBox from './SearchBox.svelte';
  import SortMenu from './SortMenu.svelte';
  import ViewToggle from './ViewToggle.svelte';

  let {
    lib,
    searchEl = $bindable(null),
    onopenfilters,
  }: { lib: Library; searchEl?: HTMLInputElement | null; onopenfilters: () => void } = $props();

  const n = $derived(filterCount(lib.filters));
  const bare = $derived(lib.phase === 'ready' && lib.items.length === 0 && !lib.filtered);
</script>

{#if lib.selecting}
  <BulkBar {lib} />
{:else if !bare}
  <div class="toolbar">
    <SearchBox {lib} bind:el={searchEl} />
    {#if phone.matches}
      <button type="button" class="btn btn-secondary filter-btn" class:active={n > 0} onclick={onopenfilters}>
        <Icon name="filter" />
        <span>筛选</span>
        {#if n > 0}<span class="filter-n">{n}</span>{/if}
      </button>
      <span class="grow"></span>
      <button type="button" class="btn btn-ghost" onclick={() => (lib.selectMode = true)}>选择</button>
      <ViewToggle {lib} />
    {:else}
      <FilterMenus {lib} />
      <span class="grow"></span>
      <SortMenu {lib} />
      <ViewToggle {lib} />
    {/if}
  </div>
  <FilterChips {lib} />
{/if}
