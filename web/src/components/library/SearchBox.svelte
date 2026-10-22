<script lang="ts">
  import { untrack } from 'svelte';
  import type { Library } from '../../lib/library.svelte';
  import Icon from '../Icon.svelte';

  interface Props {
    lib: Library;
    el?: HTMLInputElement | null;
    /** Tablet and phone headers: an icon button that expands into the field. */
    collapsed?: boolean;
    /** Whether the collapsed field is expanded (owned by the header, so the `/` shortcut can open it). */
    open?: boolean;
    /** Phones keep the field open until 取消 is pressed. */
    keepOpen?: boolean;
  }
  let { lib, el = $bindable(null), collapsed = false, open = $bindable(false), keepOpen = false }: Props = $props();

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

  // Expanding from the icon moves focus into the field.
  const shown = $derived(!collapsed || open || value !== '');
  $effect(() => {
    if (collapsed && open && el) el.focus();
  });

  function clear() {
    value = '';
    lib.setFilters({ q: '' });
  }
</script>

{#if !shown}
  <button type="button" class="btn btn-ghost btn-icon" aria-label="搜索" title="搜索 (/)" onclick={() => (open = true)}>
    <Icon name="search" />
  </button>
{:else}
  <label class="search">
    <Icon name="search" />
    <input
      bind:this={el}
      class="input"
      type="text"
      role="searchbox"
      placeholder="搜索"
      aria-label="搜索图片"
      autocomplete="off"
      spellcheck="false"
      bind:value
      onblur={() => {
        if (collapsed && !value.trim() && !keepOpen) open = false;
      }}
      onkeydown={(e) => {
        if (e.key === 'Enter') lib.setFilters({ q: value.trim() });
        if (e.key === 'Escape') {
          // Leave the field without closing the detail behind it.
          e.stopPropagation();
          if (value) clear();
          open = false;
          e.currentTarget.blur();
        }
      }}
    />
    {#if value}
      <button
        type="button"
        class="search-clear"
        aria-label="清除"
        onmousedown={(e) => e.preventDefault()}
        onclick={() => {
          clear();
          el?.focus();
        }}
      >
        <Icon name="x" size={12} />
      </button>
    {:else if !collapsed}
      <kbd class="hide-touch">/</kbd>
    {/if}
  </label>
{/if}
