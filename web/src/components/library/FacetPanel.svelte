<script lang="ts">
  import { FORMAT_NAMES } from '../../lib/format';
  import { sourceLabel } from '../../lib/image';
  import { DATE_RANGES, FORMATS, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  /** `menu`: inside the filter popover, where the options are menu items the arrow keys can walk through. */
  let { lib, menu = false }: { lib: Library; menu?: boolean } = $props();

  const f = $derived(lib.filters);
  const facets = $derived(lib.facets);

  const toggleIn = <T,>(arr: T[], v: T): T[] => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
</script>

{#snippet head(title: string, active: boolean, clear: () => void)}
  <h3>
    {title}
    {#if active}<button type="button" role={menu ? 'menuitem' : undefined} onclick={clear}>清除</button>{/if}
  </h3>
{/snippet}

{#snippet option(on: boolean, label: string, count: number | undefined, multi: boolean, onclick: () => void)}
  <button
    type="button"
    class="facet-opt"
    class:on
    role={menu ? (multi ? 'menuitemcheckbox' : 'menuitemradio') : undefined}
    aria-checked={menu ? on : undefined}
    aria-pressed={menu ? undefined : on}
    {onclick}
  >
    {#if multi}
      <span class="menu-check" class:on aria-hidden="true">{#if on}<Icon name="check" size={12} />{/if}</span>
    {:else}
      <span class="radio" class:on aria-hidden="true"></span>
    {/if}
    <span class="truncate">{label}</span>
    {#if count !== undefined}<span class="count">{count}</span>{/if}
  </button>
{/snippet}

<div>
  <div class="facet">
    {@render head('格式', f.formats.length > 0, () => lib.setFilters({ formats: [] }))}
    {#each FORMATS as k (k)}
      {@render option(f.formats.includes(k), FORMAT_NAMES[k] ?? k, facets?.formats[k], true, () => lib.setFilters({ formats: toggleIn(f.formats, k) }))}
    {/each}
  </div>

  <div class="facet">
    {@render head('可见性', !!f.vis, () => lib.setFilters({ vis: '' }))}
    {@render option(!f.vis, '全部', facets?.total, false, () => lib.setFilters({ vis: '' }))}
    {@render option(f.vis === 'public', '公开', facets?.visibility.public, false, () => lib.setFilters({ vis: f.vis === 'public' ? '' : 'public' }))}
    {@render option(f.vis === 'private', '私有', facets?.visibility.private, false, () => lib.setFilters({ vis: f.vis === 'private' ? '' : 'private' }))}
  </div>

  <div class="facet">
    {@render head('上传时间', !!f.date, () => lib.setFilters({ date: '' }))}
    {@render option(!f.date, '不限', undefined, false, () => lib.setFilters({ date: '' }))}
    {#each DATE_RANGES as d (d.key)}
      {@render option(f.date === d.key, d.label, undefined, false, () => lib.setFilters({ date: f.date === d.key ? '' : d.key }))}
    {/each}
  </div>

  {#if (facets?.sources.length ?? 0) > 0}
    <div class="facet">
      {@render head('来源', f.sources.length > 0, () => lib.setFilters({ sources: [] }))}
      {#each facets?.sources ?? [] as s (s.key)}
        {@render option(f.sources.includes(s.key), sourceLabel(s), s.count, true, () => lib.setFilters({ sources: toggleIn(f.sources, s.key) }))}
      {/each}
    </div>
  {/if}

  {#if (facets?.tags.length ?? 0) > 0}
    <div class="facet">
      {@render head('标签', f.tags.length > 0, () => lib.setFilters({ tags: [] }))}
      {#each facets?.tags ?? [] as t (t.tag)}
        {@render option(f.tags.includes(t.tag), t.tag, t.count, true, () => lib.setFilters({ tags: toggleIn(f.tags, t.tag) }))}
      {/each}
    </div>
  {/if}
</div>
