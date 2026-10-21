<script lang="ts">
  import { FORMAT_NAMES } from '../../lib/format';
  import { sourceLabel } from '../../lib/image';
  import { DATE_RANGES, FORMATS, type Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';
  import Menu from '../Menu.svelte';
  import MenuItem from '../MenuItem.svelte';

  let { lib }: { lib: Library } = $props();

  const f = $derived(lib.filters);
  const facets = $derived(lib.facets);

  const toggleIn = <T,>(arr: T[], v: T): T[] => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
</script>

{#snippet filterButton(label: string, n: number, open: boolean, toggle: () => void)}
  <button type="button" class="btn btn-secondary filter-btn" class:active={n > 0} aria-expanded={open} onclick={toggle}>
    <span>{label}</span>
    {#if n > 0}<span class="filter-n">{n}</span>{/if}
    <Icon name="chevron-down" size={14} />
  </button>
{/snippet}

<Menu>
  {#snippet trigger(open, toggle)}{@render filterButton('格式', f.formats.length, open, toggle)}{/snippet}
  {#snippet children()}
    {#each FORMATS as k (k)}
      <MenuItem checked={f.formats.includes(k)} end={facets?.formats[k]} onclick={() => lib.setFilters({ formats: toggleIn(f.formats, k) })}>
        {FORMAT_NAMES[k]}
      </MenuItem>
    {/each}
  {/snippet}
</Menu>

<Menu>
  {#snippet trigger(open, toggle)}{@render filterButton('可见性', f.vis ? 1 : 0, open, toggle)}{/snippet}
  {#snippet children(close)}
    <MenuItem selected={!f.vis} end={facets?.total} onclick={() => (lib.setFilters({ vis: '' }), close())}>全部</MenuItem>
    <MenuItem selected={f.vis === 'public'} end={facets?.visibility.public} onclick={() => (lib.setFilters({ vis: 'public' }), close())}>公开</MenuItem>
    <MenuItem selected={f.vis === 'private'} end={facets?.visibility.private} onclick={() => (lib.setFilters({ vis: 'private' }), close())}>私有</MenuItem>
  {/snippet}
</Menu>

<Menu>
  {#snippet trigger(open, toggle)}{@render filterButton('上传时间', f.date ? 1 : 0, open, toggle)}{/snippet}
  {#snippet children(close)}
    <MenuItem selected={!f.date} onclick={() => (lib.setFilters({ date: '' }), close())}>不限</MenuItem>
    {#each DATE_RANGES as d (d.key)}
      <MenuItem selected={f.date === d.key} onclick={() => (lib.setFilters({ date: d.key }), close())}>{d.label}</MenuItem>
    {/each}
  {/snippet}
</Menu>

<Menu width={220}>
  {#snippet trigger(open, toggle)}{@render filterButton('来源', f.sources.length, open, toggle)}{/snippet}
  {#snippet children()}
    {#each facets?.sources ?? [] as s (s.key)}
      <MenuItem checked={f.sources.includes(s.key)} end={s.count} onclick={() => lib.setFilters({ sources: toggleIn(f.sources, s.key) })}>
        {sourceLabel(s)}
      </MenuItem>
    {/each}
  {/snippet}
</Menu>

<Menu>
  {#snippet trigger(open, toggle)}{@render filterButton('标签', f.tags.length, open, toggle)}{/snippet}
  {#snippet children()}
    {#each facets?.tags ?? [] as t (t.tag)}
      <MenuItem checked={f.tags.includes(t.tag)} end={t.count} onclick={() => lib.setFilters({ tags: toggleIn(f.tags, t.tag) })}>
        {t.tag}
      </MenuItem>
    {/each}
  {/snippet}
</Menu>

<button type="button" class="btn btn-secondary filter-btn" class:active={f.gps} aria-pressed={f.gps} onclick={() => lib.setFilters({ gps: !f.gps })}>
  <Icon name="pin" />
  <span>含位置</span>
</button>
