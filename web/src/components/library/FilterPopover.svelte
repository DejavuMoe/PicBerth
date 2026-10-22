<script lang="ts">
  import { filterCount, type Library } from '../../lib/library.svelte';
  import Menu from '../Menu.svelte';
  import FacetPanel from './FacetPanel.svelte';
  import FilterButton from './FilterButton.svelte';

  let { lib }: { lib: Library } = $props();
</script>

<!-- One 筛选 button with every facet in a popover, for toolbars too narrow for a menu per facet. -->
<Menu class="menu-facets" width={300}>
  {#snippet trigger(open, toggle)}
    <FilterButton label="筛选" icon="filter" n={filterCount(lib.filters)} {open} {toggle} />
  {/snippet}
  {#snippet children()}
    <FacetPanel {lib} />
  {/snippet}
</Menu>
