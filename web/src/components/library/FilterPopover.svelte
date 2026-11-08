<script lang="ts">
  import { FORMAT_NAMES } from '../../lib/format';
  import { sourceLabel } from '../../lib/image';
  import { DATE_RANGES, FORMATS, filterCount, type Library } from '../../lib/library.svelte';
  import type { ImageFormat } from '../../lib/types';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';

  let { lib }: { lib: Library } = $props();

  const n = $derived(filterCount(lib.filters));

  function toggleFormat(v: ImageFormat) {
    const a = lib.filters.formats;
    lib.setFilters({ formats: a.includes(v) ? a.filter((x) => x !== v) : [...a, v] });
  }

  function toggleSource(v: string) {
    const a = lib.filters.sources;
    lib.setFilters({ sources: a.includes(v) ? a.filter((x) => x !== v) : [...a, v] });
  }

  function toggleTag(v: string) {
    const a = lib.filters.tags;
    lib.setFilters({ tags: a.includes(v) ? a.filter((x) => x !== v) : [...a, v] });
  }
</script>

<Menu align="right" class="popover" label="筛选">
  {#snippet trigger(open, toggle)}
    <button
      type="button"
      class="btn btn-ghost"
      aria-haspopup="dialog"
      aria-expanded={open}
      aria-label={n ? `筛选，已选 ${n} 项` : '筛选'}
      onclick={toggle}
    >
      <Icon name="filter" />
      <span class="hide-phone">筛选</span>
      {#if n > 0}<span class="num muted">{n}</span>{/if}
    </button>
  {/snippet}
  {#snippet children()}
    <div class="fp-section">
      <div class="fp-label">格式</div>
      <div class="fp-chips">
        {#each FORMATS as k (k)}
          <button
            type="button"
            class="chip"
            aria-pressed={lib.filters.formats.includes(k)}
            disabled={lib.facets ? !lib.facets.formats[k] : false}
            onclick={() => toggleFormat(k)}
          >
            {FORMAT_NAMES[k] ?? k}
          </button>
        {/each}
      </div>
    </div>
    <div class="fp-section">
      <div class="fp-label">上传时间</div>
      <div class="fp-chips">
        {#each DATE_RANGES as d (d.key)}
          <button
            type="button"
            class="chip"
            aria-pressed={lib.filters.date === d.key}
            onclick={() => lib.setFilters({ date: lib.filters.date === d.key ? '' : d.key })}
          >
            {d.label}
          </button>
        {/each}
      </div>
    </div>
    <div class="fp-section">
      <div class="fp-label">来源</div>
      <div class="fp-list">
        {#each lib.facets?.sources ?? [] as s (s.key)}
          {@const on = lib.filters.sources.includes(s.key)}
          <button type="button" class="fp-opt" role="checkbox" aria-checked={on} onclick={() => toggleSource(s.key)}>
            <span class="check" data-on={String(on)}>{#if on}<Icon name="check" />{/if}</span>
            {sourceLabel(s)}
            <span class="n">{s.count}</span>
          </button>
        {/each}
      </div>
    </div>
    <div class="fp-section">
      <div class="fp-label">标签</div>
      <div class="fp-list">
        {#each lib.facets?.tags ?? [] as t (t.tag)}
          {@const on = lib.filters.tags.includes(t.tag)}
          <button type="button" class="fp-opt" role="checkbox" aria-checked={on} onclick={() => toggleTag(t.tag)}>
            <span class="check" data-on={String(on)}>{#if on}<Icon name="check" />{/if}</span>
            {t.tag}
            <span class="n">{t.count}</span>
          </button>
        {/each}
      </div>
    </div>
  {/snippet}
</Menu>
