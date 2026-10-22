<script lang="ts">
  import { untrack } from 'svelte';
  import type { Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  let {
    lib,
    el = $bindable(null),
    placeholder = '搜索文件名、标签、ID 或 SHA-256',
  }: { lib: Library; el?: HTMLInputElement | null; placeholder?: string } = $props();

  // The box edits local text; the URL follows after a short pause. Enter applies at once.
  let value = $state(untrack(() => lib.filters.q));
  $effect(() => {
    const v = value.trim();
    if (v === lib.filters.q) return;
    const t = setTimeout(() => lib.setFilters({ q: v }), 250);
    return () => clearTimeout(t);
  });
  $effect(() => {
    const urlQ = lib.filters.q;
    untrack(() => {
      if (urlQ !== value.trim()) value = urlQ;
    });
  });
</script>

<label class="search">
  <Icon name="search" />
  <input
    bind:this={el}
    class="input"
    type="search"
    {placeholder}
    aria-label="搜索图片"
    autocomplete="off"
    spellcheck="false"
    bind:value
    onkeydown={(e) => {
      if (e.key === 'Enter') lib.setFilters({ q: value.trim() });
      if (e.key === 'Escape') {
        if (value) {
          value = '';
          lib.setFilters({ q: '' });
        }
        e.currentTarget.blur();
      }
    }}
  />
  {#if !value}<kbd class="hide-touch">/</kbd>{/if}
</label>
