<script lang="ts">
  import { FORMAT_NAMES } from '../../lib/format';
  import { sourceKeyLabel } from '../../lib/image';
  import { DATE_RANGES, EMPTY_FILTERS, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  interface Props {
    lib: Library;
    /** The phone search field already shows the query. */
    omitQuery?: boolean;
  }
  let { lib, omitQuery = false }: Props = $props();

  const chips = $derived.by(() => {
    const f = lib.filters;
    const out: { key: string; label: string; remove: () => void }[] = [];
    if (f.q && !omitQuery) out.push({ key: 'q', label: `“${f.q}”`, remove: () => lib.setFilters({ q: '' }) });
    for (const v of f.formats) {
      out.push({
        key: `f${v}`,
        label: FORMAT_NAMES[v] ?? v,
        remove: () => lib.setFilters({ formats: f.formats.filter((x) => x !== v) }),
      });
    }
    if (f.date) {
      out.push({
        key: 'date',
        label: DATE_RANGES.find((d) => d.key === f.date)?.label ?? '',
        remove: () => lib.setFilters({ date: '' }),
      });
    }
    for (const v of f.sources) {
      out.push({
        key: `s${v}`,
        label: sourceKeyLabel(v, lib.facets?.sources ?? []),
        remove: () => lib.setFilters({ sources: f.sources.filter((x) => x !== v) }),
      });
    }
    for (const v of f.tags) {
      out.push({ key: `t${v}`, label: v, remove: () => lib.setFilters({ tags: f.tags.filter((x) => x !== v) }) });
    }
    return out;
  });
</script>

{#if chips.length > 0}
  <div class="filters">
    {#each chips as c (c.key)}
      <span class="chip" style:color="var(--text)">
        {c.label}
        <button type="button" class="x" aria-label={`移除 ${c.label}`} onclick={c.remove}>
          <Icon name="x" size="s" />
        </button>
      </span>
    {/each}
    <button type="button" class="btn btn-ghost btn-sm muted" onclick={() => lib.setFilters({ ...EMPTY_FILTERS })}>
      清除
    </button>
  </div>
{/if}
