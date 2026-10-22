<script lang="ts">
  import { FORMAT_NAMES } from '../../lib/format';
  import { sourceKeyLabel } from '../../lib/image';
  import { DATE_RANGES, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  let { lib }: { lib: Library } = $props();

  const f = $derived(lib.filters);

  const chips = $derived.by(() => {
    const out: { key: string; label: string; remove: () => void }[] = [];
    for (const k of f.formats) {
      out.push({ key: `f${k}`, label: FORMAT_NAMES[k] ?? k, remove: () => lib.setFilters({ formats: f.formats.filter((x) => x !== k) }) });
    }
    if (f.vis) out.push({ key: 'vis', label: f.vis === 'public' ? '公开' : '私有', remove: () => lib.setFilters({ vis: '' }) });
    if (f.date) {
      out.push({ key: 'date', label: DATE_RANGES.find((d) => d.key === f.date)?.label ?? '', remove: () => lib.setFilters({ date: '' }) });
    }
    for (const s of f.sources) {
      out.push({ key: `s${s}`, label: sourceKeyLabel(s, lib.facets?.sources), remove: () => lib.setFilters({ sources: f.sources.filter((x) => x !== s) }) });
    }
    for (const t of f.tags) {
      out.push({ key: `t${t}`, label: `#${t}`, remove: () => lib.setFilters({ tags: f.tags.filter((x) => x !== t) }) });
    }
    if (f.gps) out.push({ key: 'gps', label: '含位置', remove: () => lib.setFilters({ gps: false }) });
    return out;
  });
</script>

{#if chips.length > 0}
  <div class="chips">
    {#each chips as c (c.key)}
      <span class="chip">
        {c.label}
        <button type="button" aria-label={`移除筛选 ${c.label}`} onclick={c.remove}><Icon name="x" size={12} /></button>
      </span>
    {/each}
    <button type="button" class="chip-clear" onclick={() => lib.clearFilters()}>清除筛选</button>
  </div>
{/if}
