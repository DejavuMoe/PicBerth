<script lang="ts">
  import { filterCount, SORTS, type Library, type ViewMode } from '../../lib/library.svelte';
  import Seg from '../Seg.svelte';
  import Sheet from '../Sheet.svelte';
  import FacetPanel from './FacetPanel.svelte';

  /** Phone adjustments: sort, view and the same facets as the popover, applied live. */
  let { lib, onclose }: { lib: Library; onclose: () => void } = $props();
</script>

<Sheet label="筛选" {onclose}>
  {#snippet children(close)}
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
      <div class="facet">
        <h3>视图</h3>
        <div class="facet-seg">
          <Seg
            fill
            label="视图"
            value={lib.view}
            onchange={(v: ViewMode) => lib.setView(v)}
            options={[
              { key: 'table', icon: 'list', label: '列表' },
              { key: 'grid', icon: 'grid', label: '网格' },
            ]}
          />
        </div>
      </div>
      <FacetPanel {lib} />
    </div>
    <div class="sheet-foot">
      <button type="button" class="btn btn-primary btn-block btn-lg" onclick={close}>
        显示 {lib.items.length}{lib.cursor ? '+' : ''} 张图片
      </button>
    </div>
  {/snippet}
</Sheet>
