<script lang="ts">
  import { filterCount, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import FacetPanel from './FacetPanel.svelte';

  let { lib }: { lib: Library } = $props();

  const n = $derived(filterCount(lib.filters));
</script>

<!-- The single 筛选 button: every facet lives in one popover. -->
<Menu align="right" class="menu-facets" width={300} label="筛选">
  {#snippet trigger(open, toggle)}
    <button type="button" class="btn btn-ghost filter-btn" aria-haspopup="menu" aria-expanded={open} onclick={toggle}>
      <Icon name="filter" />
      <span>筛选</span>
      {#if n > 0}<span class="filter-n">{n}</span>{/if}
    </button>
  {/snippet}
  {#snippet children()}
    <FacetPanel {lib} menu />
  {/snippet}
</Menu>
