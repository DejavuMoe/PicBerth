<script lang="ts">
  import { filterCount, SORTS, type Library } from '../../lib/library.svelte';
  import Sheet from '../Sheet.svelte';
  import FacetPanel from './FacetPanel.svelte';

  let { lib, onclose }: { lib: Library; onclose: () => void } = $props();
</script>

<Sheet label="筛选" {onclose}>
  <div class="sheet-head">
    <h2>筛选</h2>
    {#if filterCount(lib.filters) > 0}
      <button type="button" class="btn btn-ghost btn-sm" onclick={() => lib.clearFilters()}>清除</button>
    {/if}
  </div>
  <div class="sheet-body">
    <div class="facet">
      <h3>排序</h3>
      {#each SORTS as s (s.key)}
        <button type="button" class="facet-opt" class:on={lib.sort === s.key} aria-pressed={lib.sort === s.key} onclick={() => lib.setSort(s.key)}>
          <span class="radio" class:on={lib.sort === s.key} aria-hidden="true"></span>
          {s.label}
        </button>
      {/each}
    </div>
    <FacetPanel {lib} />
  </div>
  <div class="sheet-foot">
    <button type="button" class="btn btn-primary btn-block" onclick={onclose}>
      显示 {lib.items.length}{lib.cursor ? '+' : ''} 张图片
    </button>
  </div>
</Sheet>
